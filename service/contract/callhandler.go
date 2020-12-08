package contract

import (
	"encoding/json"
	"math/big"
	"strings"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/trace"
)

type DataCallJSON struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type CallHandler struct {
	*CommonHandler

	name   string
	params []byte
	// nil paramObj means it needs to convert params to *codec.TypedObj.
	paramObj *codec.TypedObj

	forDeploy bool
	external  bool
	disposed  bool
	lock      sync.Mutex

	// set in ExecuteAsync()
	cc        CallContext
	as        state.AccountState
	info      *scoreapi.Info
	method    *scoreapi.Method
	cm        ContractManager
	conn      eeproxy.Proxy
	cs        ContractStore
	isSysCall bool
	isQuery   bool
	charged   bool
	codeID    []byte
}

func newCallHandlerWithData(ch *CommonHandler, data []byte) (*CallHandler, error) {
	jso, err := ParseCallData(data)
	if err != nil {
		return nil, scoreresult.InvalidParameterError.Wrap(err,
			"CallDataInvalid")
	}
	return &CallHandler{
		CommonHandler: ch,
		external:      true,
		name:          jso.Method,
		params:        jso.Params,
	}, nil
}

func newCallHandlerWithTypedObj(
	ch *CommonHandler,
	data *codec.TypedObj,
) (*CallHandler, error) {
	if data.Type != codec.TypeDict {
		return nil, scoreresult.InvalidParameterError.New("InvalidDataType")
	}
	dataReal := data.Object.(map[string]*codec.TypedObj)
	method := common.DecodeAsString(dataReal["method"], scoreapi.FallbackMethodName)
	paramObj := dataReal["params"]

	return &CallHandler{
		CommonHandler: ch,
		name:          method,
		paramObj:      paramObj,
	}, nil
}

func newCallHandlerWithParams(ch *CommonHandler, method string,
	paramObj *codec.TypedObj, forDeploy bool,
) *CallHandler {
	return &CallHandler{
		CommonHandler: ch,
		name:          method,
		paramObj:      paramObj,
		forDeploy:     forDeploy,
		isSysCall:     false,
	}
}

func (h *CallHandler) prepareWorldContextAndAccount(ctx Context) (state.WorldContext, state.AccountState) {
	lq := []state.LockRequest{
		{string(h.to.ID()), state.AccountWriteLock},
		{string(h.from.ID()), state.AccountWriteLock},
	}
	wc := ctx.GetFuture(lq)
	wc.WorldVirtualState().Ensure()

	as := wc.GetAccountState(h.to.ID())

	info, err := as.APIInfo()
	if err != nil || info == nil {
		return wc, as
	}

	method := info.GetMethod(h.name)
	if method == nil || method.IsIsolated() {
		return wc, as
	}

	// Making new world context with locking the world
	lq = []state.LockRequest{
		{state.WorldIDStr, state.AccountWriteLock},
	}
	wc = ctx.GetFuture(lq)
	as = wc.GetAccountState(h.to.ID())

	return wc, as
}

func (h *CallHandler) prepareContractStore(ctx Context, wc state.WorldContext, c state.Contract) error {
	h.lock.Lock()
	defer h.lock.Unlock()
	if cs, err := ctx.ContractManager().PrepareContractStore(wc, c); err != nil {
		return err
	} else {
		h.cs = cs
	}
	return nil
}

func (h *CallHandler) Prepare(ctx Context) (state.WorldContext, error) {
	wc, as := h.prepareWorldContextAndAccount(ctx)

	c := h.contract(as)
	if c == nil {
		return wc, nil
	}
	h.prepareContractStore(ctx, wc, c)

	return wc, nil
}

func (h *CallHandler) contract(as state.AccountState) state.Contract {
	if as == nil || !as.IsContract() {
		return nil
	}
	if !h.forDeploy {
		return as.ActiveContract()
	} else {
		return as.NextContract()
	}
}

func (h *CallHandler) ExecuteAsync(cc CallContext) (err error) {
	h.log = trace.LoggerOf(cc.Logger())

	h.log.TSystemf("INVOKE start score=%s method=%s", h.to, h.name)
	defer func() {
		if err != nil {
			if !h.ApplyCallSteps(cc) {
				err = scoreresult.OutOfStepError.Wrap(err, "OutOfStepForCall")
			}
			h.log.TSystemf("INVOKE done status=%s msg=%v", err.Error(), err)
		}
	}()

	return h.DoExecuteAsync(cc)
}

func (h *CallHandler) ApplyCallSteps(cc CallContext) bool {
	if !h.charged {
		h.charged = true
		if !cc.ApplySteps(state.StepTypeContractCall, 1) {
			return false
		}
	}
	return true
}

func (h *CallHandler) DoExecuteAsync(cc CallContext) (err error) {
	h.cc = cc
	h.cm = cc.ContractManager()

	// Prepare
	if !h.to.IsContract() {
		return scoreresult.InvalidParameterError.Errorf("InvalidAddressForCall(%s)", h.to.String())
	}
	h.as = cc.GetAccountState(h.to.ID())
	c := h.contract(h.as)
	if c == nil {
		return scoreresult.New(module.StatusContractNotFound, "NotAContractAccount")
	}
	cc.SetContractInfo(&state.ContractInfo{Owner: h.as.ContractOwner()})
	h.codeID = c.CodeID()
	// Before we set the codeID, it gets the last frame.
	// Otherwise it would return current frameID for the code.
	cc.SetFrameCodeID(h.codeID)

	// Calculate steps
	isSystem := strings.Compare(c.ContentType(), state.CTAppSystem) == 0
	if !h.forDeploy && !isSystem {
		if !h.ApplyCallSteps(cc) {
			return scoreresult.OutOfStepError.New("OutOfStepForCall")
		}
	}

	if err := h.ensureMethodAndParams(c.EEType()); err != nil {
		return err
	}

	if isSystem {
		return h.invokeSystemMethod(cc, c)
	}
	return h.invokeEEMethod(cc, c)
}

func (h *CallHandler) invokeEEMethod(cc CallContext, c state.Contract) error {
	h.conn = cc.GetProxy(h.EEType())
	if h.conn == nil {
		return errors.ExecutionFailError.Errorf(
			"FAIL to get connection for (%s)", h.EEType())
	}

	// Set up contract files
	if err := h.prepareContractStore(cc, cc, c); err != nil {
		h.log.Warnf("FAIL to prepare contract. err=%+v\n", err)
		return errors.CriticalIOError.Wrap(err, "FAIL to prepare contract")
	}
	path, err := h.cs.WaitResult()
	if err != nil {
		h.log.Warnf("FAIL to prepare contract. err=%+v\n", err)
		return errors.CriticalIOError.Wrap(err, "FAIL to prepare contract")
	}

	last := cc.GetLastEIDOf(h.codeID)
	var state *eeproxy.CodeState
	if next, objHash, _, err := h.as.GetObjGraph(h.codeID, false); err == nil {
		state = &eeproxy.CodeState{
			NexHash:   next,
			GraphHash: objHash,
			PrevEID:   last,
		}
	} else {
		if !errors.NotFoundError.Equals(err) {
			return err
		}
	}
	eid := cc.NewExecution()
	// Execute
	h.lock.Lock()
	if !h.disposed {
		h.log.Tracef("Execution INVOKE last=%d eid=%d", last, eid)
		err = h.conn.Invoke(h, path, cc.QueryMode(), h.from, h.to,
			h.value, cc.StepAvailable(), h.method.Name, h.paramObj,
			h.codeID, eid, state)
	}
	h.lock.Unlock()

	return err
}

func (h *CallHandler) invokeSystemMethod(cc CallContext, c state.Contract) error {
	h.isSysCall = true

	var cid string
	if code, err := c.Code(); err != nil {
		if len(code) == 0 {
			cid = CID_CHAIN
		} else {
			cid = string(cid)
		}
	}

	score, err := cc.ContractManager().GetSystemScore(cid, cc, h.from, h.value)
	if err != nil {
		return err
	}

	status, result, step := Invoke(score, h.method.Name, h.paramObj)
	go func() {
		h.OnResult(status, step, result)
	}()

	return nil
}

func (h *CallHandler) ensureMethodAndParams(eeType state.EEType) error {
	info, err := h.as.APIInfo()
	if err != nil {
		return nil
	}
	if info == nil {
		return scoreresult.New(module.StatusContractNotFound, "APIInfo() is null")
	}

	method := info.GetMethod(h.name)
	if method == nil || !method.IsCallable() {
		return scoreresult.MethodNotFoundError.Errorf("Method(%s)NotFound", h.name)
	}
	if !h.forDeploy {
		if method.IsFallback() {
			if h.external {
				return scoreresult.MethodNotFoundError.New(
					"IllegalAccessToFallback")
			}
		} else {
			if eeType.IsInternalMethod(h.name) {
				return scoreresult.MethodNotFoundError.Errorf(
					"InvalidAccessToInternalMethod(%s)", h.name)
			}
			if !method.IsExternal() {
				return scoreresult.MethodNotFoundError.Errorf(
					"InvalidAccessTo(%s)", h.name)
			}
		}
		if method.IsReadOnly() != h.cc.QueryMode() {
			if method.IsReadOnly() {
				h.cc.EnterQueryMode()
			} else {
				return scoreresult.AccessDeniedError.Errorf(
					"AccessingWritableFromReadOnly(%s)", h.name)
			}
		}
	}

	h.method = method
	h.isQuery = method.IsReadOnly()
	h.info = info
	if h.paramObj != nil {
		if params, err := method.EnsureParamsSequential(h.paramObj); err != nil {
			return err
		} else {
			h.paramObj = params
			return nil
		}
	}

	h.paramObj, err = method.ConvertParamsToTypedObj(h.params)
	return err
}

func (h *CallHandler) SendResult(status error, steps *big.Int, result *codec.TypedObj) error {
	if h.log.IsTrace() {
		if status == nil {
			po, _ := common.DecodeAnyForJSON(result)
			h.log.TSystemf("CALL done status=%s steps=%v result=%s",
				module.StatusSuccess, steps, trace.ToJSON(po))
		} else {
			s, _ := scoreresult.StatusOf(status)
			h.log.TSystemf("CALL done status=%s steps=%v msg=%s",
				s, steps, status.Error())
		}
	}
	if !h.isSysCall {
		if h.conn == nil {
			return errors.ExecutionFailError.Errorf(
				"Don't have a connection for (%s)", h.EEType())
		}
		last := h.cc.GetReturnEID()
		eid := h.cc.NewExecution()
		h.log.Tracef("Execution RESULT last=%d eid=%d", last, eid)
		return h.conn.SendResult(h, status, steps, result, eid, last)
	} else {
		h.cc.OnResult(status, steps, result, nil)
		return nil
	}
}

func (h *CallHandler) Dispose() {
	h.lock.Lock()
	h.disposed = true
	if h.cs != nil {
		h.cs.Dispose()
	}
	h.lock.Unlock()
}

func (h *CallHandler) EEType() state.EEType {
	c := h.contract(h.as)
	if c == nil {
		h.log.Debugf("No associated contract exists. forDeploy(%d), Active(%v), Next(%v)\n",
			h.forDeploy, h.as.ActiveContract(), h.as.NextContract())
		return ""
	}
	return c.EEType()
}

func (h *CallHandler) GetValue(key []byte) ([]byte, error) {
	if h.as != nil {
		return h.as.GetValue(key)
	} else {
		return nil, errors.CriticalUnknownError.Errorf(
			"GetValue: No Account(%s) exists", h.to)
	}
}

func (h *CallHandler) SetValue(key []byte, value []byte) ([]byte, error) {
	if h.isQuery {
		return nil, scoreresult.AccessDeniedError.New(
			"DeleteValueInQuery")
	}
	if h.as != nil {
		return h.as.SetValue(key, value)
	} else {
		return nil, errors.CriticalUnknownError.Errorf(
			"SetValue: No Account(%s) exists", h.to)
	}
}

func (h *CallHandler) DeleteValue(key []byte) ([]byte, error) {
	if h.isQuery {
		return nil, scoreresult.AccessDeniedError.New(
			"DeleteValueInQuery")
	}
	if h.as != nil {
		return h.as.DeleteValue(key)
	} else {
		return nil, errors.CriticalUnknownError.Errorf(
			"DeleteValue: No Account(%s) exists", h.to)
	}
}

func (h *CallHandler) GetInfo() *codec.TypedObj {
	return common.MustEncodeAny(h.cc.GetInfo())
}

func (h *CallHandler) GetBalance(addr module.Address) *big.Int {
	return h.cc.GetBalance(addr)
}

func (h *CallHandler) OnEvent(addr module.Address, indexed, data [][]byte) {
	if h.isQuery {
		h.log.Panic("EventLog arrives in query mode")
		return
	}
	if err := h.info.CheckEventData(indexed, data); err != nil {
		h.log.Warnf("DROP InvalidEventData(%s,%+v,%+v) err=%+v",
			addr, indexed, data, err)
		return
	}
	h.cc.OnEvent(addr, indexed, data)
}

func (h *CallHandler) OnResult(status error, steps *big.Int, result *codec.TypedObj) {
	if h.log.IsTrace() {
		if status != nil {
			s, _ := scoreresult.StatusOf(status)
			h.log.TSystemf("INVOKE done status=%s msg=%v steps=%s", s, status, steps)
		} else {
			obj, _ := common.DecodeAnyForJSON(result)
			if err := h.method.EnsureResult(result); err != nil {
				h.log.TSystemf("INVOKE done status=%s steps=%s result=%s warning=%s",
					module.StatusSuccess, steps, trace.ToJSON(obj), err)
			} else {
				h.log.TSystemf("INVOKE done status=%s steps=%s result=%s",
					module.StatusSuccess, steps, trace.ToJSON(obj))
			}
		}
	}
	h.cc.OnResult(status, steps, result, nil)
}

func (h *CallHandler) OnCall(from, to module.Address, value,
	limit *big.Int, dataType string, dataObj *codec.TypedObj,
) {
	if h.log.IsTrace() {
		po, _ := common.DecodeAnyForJSON(dataObj)
		h.log.TSystemf("CALL start from=%v to=%v value=%v steplimit=%v dataType=%s data=%s",
			from, to, value, limit, dataType, trace.ToJSON(po))
	}

	ctype := CTypeNone
	switch dataType {
	case DataTypeCall:
		if to.IsContract() {
			ctype = CTypeCall
		} else {
			ctype = CTypeTransfer
		}
	case DataTypeDeploy:
		ctype = CTypeDeploy
	}

	handler, err := h.cm.GetCallHandler(from, to, value, ctype, dataObj)

	if err != nil {
		steps := big.NewInt(h.cc.StepsFor(state.StepTypeContractCall, 1))
		if steps.Cmp(limit) > 0 {
			steps = limit
		}
		if err := h.SendResult(err, steps, nil); err != nil {
			h.cc.OnResult(err, h.cc.StepAvailable(), nil, nil)
		}
	} else {
		h.cc.OnCall(handler, limit)
	}
}

func (h *CallHandler) OnAPI(status error, info *scoreapi.Info) {
	h.log.Panicln("Unexpected OnAPI() call")
}

func (h *CallHandler) OnSetFeeProportion(addr module.Address, portion int) {
	h.log.TSystemf("CALL setFeeProportion addr=%s portion=%d", addr, portion)
	h.cc.SetFeeProportion(addr, portion)
}

func (h *CallHandler) SetCode(code []byte) error {
	if h.forDeploy == false {
		return errors.InvalidStateError.New("Unexpected call SetCode()")
	}
	c := h.contract(h.as)
	return c.SetCode(code)
}

func (h *CallHandler) GetObjGraph(flags bool) (int, []byte, []byte, error) {
	return h.as.GetObjGraph(h.codeID, flags)
}

func (h *CallHandler) SetObjGraph(flags bool, nextHash int, objGraph []byte) error {
	if h.isQuery {
		return nil
	}
	return h.as.SetObjGraph(h.codeID, flags, nextHash, objGraph)
}

type TransferAndCallHandler struct {
	th *TransferHandler
	*CallHandler
}

func (h *TransferAndCallHandler) Prepare(ctx Context) (state.WorldContext, error) {
	if h.to.IsContract() {
		return h.CallHandler.Prepare(ctx)
	} else {
		return h.th.Prepare(ctx)
	}
}

func (h *TransferAndCallHandler) ExecuteAsync(cc CallContext) (err error) {
	h.log = trace.LoggerOf(cc.Logger())

	h.log.TSystemf("TRANSFER INVOKE start score=%s method=%s", h.to, h.name)
	defer func() {
		if err != nil {
			if !h.ApplyCallSteps(cc) {
				err = scoreresult.OutOfStepError.New("OutOfStepForCall")
			}
			h.log.TSystemf("TRANSFER INVOKE done status=%s msg=%v", err.Error(), err)
		}
	}()

	status, result, addr := h.th.DoExecuteSync(cc)
	if status != nil {
		if !h.ApplyCallSteps(cc) {
			status = scoreresult.OutOfStepError.New("OutOfStepForCall")
		}
		go func() {
			cc.OnResult(status, cc.StepUsed(), result, addr)
		}()
		return nil
	}

	as := cc.GetAccountState(h.to.ID())
	apiInfo, err := as.APIInfo()
	if err != nil {
		return err
	}
	if apiInfo == nil {
		return scoreresult.New(module.StatusContractNotFound, "APIInfo() is null")
	} else {
		m := apiInfo.GetMethod(h.name)
		if m == nil {
			return scoreresult.ErrMethodNotFound
		}
		if !m.IsPayable() {
			return scoreresult.ErrMethodNotPayable
		}
		if m.IsReadOnly() {
			return scoreresult.ErrAccessDenied
		}
	}

	return h.CallHandler.DoExecuteAsync(cc)
}

func newTransferAndCallHandler(ch *CommonHandler, call *CallHandler) *TransferAndCallHandler {
	return &TransferAndCallHandler{
		th:          newTransferHandler(ch),
		CallHandler: call,
	}
}

func ParseCallData(data []byte) (*DataCallJSON, error) {
	jso := new(DataCallJSON)
	if err := json.Unmarshal(data, jso); err != nil {
		return nil, scoreresult.InvalidParameterError.Wrapf(err,
			"InvalidJSON(json=%s)", data)
	}
	if jso.Method == "" {
		return nil, scoreresult.InvalidParameterError.Errorf(
			"NoMethod(json=%s)", data)
	}
	return jso, nil
}
