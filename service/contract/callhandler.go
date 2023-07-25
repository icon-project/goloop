package contract

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strings"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
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
	ch        eeproxy.CallContext
	as        state.AccountState
	store     containerdb.BytesStoreState
	info      *scoreapi.Info
	method    *scoreapi.Method
	cm        ContractManager
	conn      eeproxy.Proxy
	cs        ContractStore
	isSysCall bool
	readOnly  bool
	allowEx   bool
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
	dataReal := data.Object.(*codec.TypedDict).Map
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
		{string(h.To.ID()), state.AccountWriteLock},
		{string(h.From.ID()), state.AccountWriteLock},
	}
	wc := ctx.GetFuture(lq)
	wc.WorldVirtualState().Ensure()

	as := wc.GetAccountState(h.To.ID())

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
	as = wc.GetAccountState(h.To.ID())

	return wc, as
}

func (h *CallHandler) prepareContractStore(ctx Context, wc state.WorldContext, c state.ContractState) error {
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
	if err := h.prepareContractStore(ctx, wc, c); err != nil {
		return nil, err
	}
	return wc, nil
}

func (h *CallHandler) contract(as state.AccountState) state.ContractState {
	if as == nil || !as.IsContract() {
		return nil
	}
	if !h.forDeploy {
		return as.ActiveContract()
	} else {
		return as.NextContract()
	}
}

func (h *CallHandler) TLogStart() {
	h.Log.TSystemf("INVOKE start score=%s method=%s", h.To, h.name)
}

func (h *CallHandler) TLogDone(status error, steps *big.Int, result *codec.TypedObj) {
	if h.Log.TraceMode() == module.TraceModeInvoke {
		if status != nil {
			s, _ := scoreresult.StatusOf(status)
			h.Log.TSystemf("INVOKE done status=%s msg=%v steps=%s", s, status, steps)
		} else {
			obj, _ := common.DecodeAnyForJSON(result)
			if err := h.method.EnsureResult(result); err != nil {
				h.Log.TSystemf("INVOKE done status=%s steps=%s result=%s warning=%s",
					module.StatusSuccess, steps, trace.ToJSON(obj), err)
			} else {
				h.Log.TSystemf("INVOKE done status=%s steps=%s result=%s",
					module.StatusSuccess, steps, trace.ToJSON(obj))
			}
		}
	}
}

func (h *CallHandler) ExecuteAsync(cc CallContext) (err error) {
	h.TLogStart()
	defer func() {
		if err != nil {
			if err2 := h.ApplyCallSteps(cc); err2 != nil {
				err = err2
			}
			h.TLogDone(err, cc.StepUsed(), nil)
		}
	}()

	return h.DoExecuteAsync(cc, h, nil)
}

func (h *CallHandler) DoExecuteAsync(cc CallContext, ch eeproxy.CallContext, store containerdb.BytesStoreState) (err error) {
	h.cc = cc
	h.ch = ch
	h.cm = cc.ContractManager()

	// Prepare
	if !h.To.IsContract() {
		return scoreresult.InvalidParameterError.Errorf("InvalidAddressForCall(%s)", h.To.String())
	}
	h.as = cc.GetAccountState(h.To.ID())
	h.store = h.as
	if store != nil {
		h.store = store
	}
	c := h.contract(h.as)
	if c == nil || c.Status() != state.CSActive {
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
		if err2 := h.ApplyCallSteps(cc); err2 != nil {
			return err2
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

func (h *CallHandler) invokeEEMethod(cc CallContext, c state.ContractState) error {
	h.conn = cc.GetProxy(h.EEType())
	if h.conn == nil {
		return errors.ExecutionFailError.Errorf(
			"FAIL to get connection for (%s)", h.EEType())
	}

	// Set up contract files
	if err := h.prepareContractStore(cc, cc, c); err != nil {
		h.Log.Warnf("FAIL to prepare contract. err=%+v\n", err)
		return errors.CriticalIOError.Wrap(err, "FAIL to prepare contract")
	}
	path, err := h.cs.WaitResult()
	if err != nil {
		h.Log.Warnf("FAIL to prepare contract. err=%+v\n", err)
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
		h.Log.Tracef("Execution INVOKE last=%d eid=%d", last, eid)
		err = h.conn.Invoke(h.ch, path, cc.ReadOnlyMode(), h.From, h.To,
			h.Value, cc.StepAvailable(), h.method.Name, h.paramObj,
			h.codeID, eid, state)
	}
	h.lock.Unlock()

	return err
}

func (h *CallHandler) invokeSystemMethod(cc CallContext, c state.ContractState) error {
	h.isSysCall = true

	var cid string
	if code, err := c.Code(); err != nil {
		return scoreresult.ContractNotFoundError.AttachTo(err)
	} else {
		cid = string(code)
	}

	score, err := cc.ContractManager().GetSystemScore(cid, cc, h.From, h.Value)
	if err != nil {
		return err
	}

	status, result, step := Invoke(score, h.method.Name, h.paramObj)
	go func() {
		h.ch.OnResult(status, 0, step, result)
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
		if method.IsReadOnly() != h.cc.ReadOnlyMode() {
			if method.IsReadOnly() {
				h.cc.EnterReadOnlyMode()
			} else {
				return scoreresult.AccessDeniedError.Errorf(
					"AccessingWritableFromReadOnly(%s)", h.name)
			}
		}
	}

	h.method = method
	h.readOnly = method.IsReadOnly()
	h.info = info
	if h.paramObj != nil {
		if params, err := method.EnsureParamsSequential(h.paramObj); err != nil {
			return err
		} else {
			h.paramObj = params
			return nil
		}
	}

	h.paramObj, err = method.ConvertParamsToTypedObj(h.params, h.allowEx)
	return err
}

func (h *CallHandler) GetMethodName() string {
	return h.name
}

func (h *CallHandler) AllowExtra() {
	h.allowEx = true
}

func (h *CallHandler) SendResult(status error, steps *big.Int, result *codec.TypedObj) error {
	if h.Log.TraceMode() == module.TraceModeInvoke {
		if status == nil {
			po, _ := common.DecodeAnyForJSON(result)
			h.Log.TSystemf("CALL done status=%s steps=%v result=%s",
				module.StatusSuccess, steps, trace.ToJSON(po))
		} else {
			s, _ := scoreresult.StatusOf(status)
			h.Log.TSystemf("CALL done status=%s steps=%v msg=%s",
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
		h.Log.Tracef("Execution RESULT last=%d eid=%d", last, eid)
		return h.conn.SendResult(h, status, steps, result, eid, last)
	} else {
		h.cc.OnResult(status, 0, steps, result, nil)
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
		h.Log.Debugf("No associated contract exists. forDeploy(%d), Active(%v), Next(%v)\n",
			h.forDeploy, h.as.ActiveContract(), h.as.NextContract())
		return ""
	}
	return c.EEType()
}

func (h *CallHandler) GetValue(key []byte) ([]byte, error) {
	if h.store != nil {
		var value []byte
		var err error
		h.cc.DoIOTask(func() {
			value, err = h.store.GetValue(key)
		})
		if err != nil {
			h.Log.TSystemf("GETVALUE key=<%x> err=%+v", key, err)
		} else {
			h.Log.TSystemf("GETVALUE key=<%x> value=<%x>", key, value)
		}
		return value, err
	} else {
		return nil, errors.CriticalUnknownError.Errorf(
			"GetValue: No Account(%s) exists", h.To)
	}
}

func (h *CallHandler) SetValue(key []byte, value []byte) ([]byte, error) {
	if h.readOnly {
		return nil, scoreresult.AccessDeniedError.New(
			"DeleteValueInQuery")
	}
	if h.store != nil {
		var old []byte
		var err error
		h.cc.DoIOTask(func() {
			old, err = h.store.SetValue(key, value)
		})
		if err != nil {
			h.Log.TSystemf("SETVALUE key=<%x> value=<%x> err=%+v", key, value, err)
		} else {
			h.Log.TSystemf("SETVALUE key=<%x> value=<%x> old=<%x>", key, value, old)
		}
		return old, err
	} else {
		return nil, errors.CriticalUnknownError.Errorf(
			"SetValue: No Account(%s) exists", h.To)
	}
}

func (h *CallHandler) DeleteValue(key []byte) ([]byte, error) {
	if h.readOnly {
		return nil, scoreresult.AccessDeniedError.New(
			"DeleteValueInQuery")
	}
	if h.store != nil {
		var old []byte
		var err error
		h.cc.DoIOTask(func() {
			old, err = h.store.DeleteValue(key)
		})
		if err != nil {
			h.Log.TSystemf("DELETE key=<%x> err=%+v", key, err)
		} else {
			h.Log.TSystemf("DELETE key=<%x> old=<%x>", key, old)
		}
		return old, err
	} else {
		return nil, errors.CriticalUnknownError.Errorf(
			"DeleteValue: No Account(%s) exists", h.To)
	}
}

func (h *CallHandler) ArrayDBContains(prefix, value []byte, limit int64) (bool, int, int, error) {
	if h.as == nil {
		return false, 0, 0, errors.CriticalUnknownError.Errorf(
			"ArrayDBContains: No Account(%s) exists", h.To)
	}

	get := h.cc.StepsFor(state.StepTypeGet, 1)
	getBase := h.cc.StepsFor(state.StepTypeGetBase, 1)

	var cost int64
	var found bool
	var count, size int
	adb := containerdb.NewArrayDB(h.store, containerdb.NewHashKey(prefix))
	h.cc.DoIOTask(func() {
		s := adb.Size()
		for i := 0; i < s; i++ {
			v := adb.Get(i)
			count += 1
			cost += getBase
			if v != nil {
				bs := v.Bytes()
				bl := len(bs)
				size += bl
				cost += get * int64(bl)
				if bytes.Equal(bs, value) {
					found = true
					return
				}
			}
			if cost > limit {
				return
			}
		}
	})
	h.Log.TSystemf("CONTAINS prefix=<%x> value=<%x> found=%v count=%d size=%d",
		prefix, value, found, count, size)
	return found, count, size, nil
}

func (h *CallHandler) GetInfo() *codec.TypedObj {
	return common.MustEncodeAny(h.cc.GetInfo())
}

func (h *CallHandler) GetBalance(addr module.Address) *big.Int {
	value := h.cc.GetBalance(addr)
	h.Log.TSystemf("GETBALANCE addr=%s value=%s", addr, value)
	return value
}

func (h *CallHandler) OnEvent(addr module.Address, indexed, data [][]byte) error {
	if h.readOnly {
		// It's not allowed to send event message if it's in query mode.
		// It means that the execution environment is in invalid state.
		// Proxy need to be closed.
		h.Log.TSystemf("EVENT drop event=(%s,%+v,%+v) readonly",
			addr, indexed, data)
		h.Log.Debugf("DROP EventLog(%s,%+v,%+v) in ReadOnlyMode",
			addr, indexed, data)
		return errors.InvalidStateError.New("EventInReadOnlyMode")
	}
	if err := h.info.CheckEventData(indexed, data); err != nil {
		// Given data is incorrect. This may not be able to  checked
		// by execution environment. So we just ignore this and let
		// them know the problem.
		h.Log.TSystemf("EVENT drop event=(%s,%+v,%+v) err=%+v",
			addr, indexed, data, err)
		h.Log.Debugf("DROP InvalidEventData(%s,%+v,%+v) err=%+v",
			addr, indexed, data, err)
		return nil
	}
	h.cc.OnEvent(addr, indexed, data)
	return nil
}

func (h *CallHandler) OnResult(status error, flag int, steps *big.Int, result *codec.TypedObj) {
	h.TLogDone(status, steps, result)
	h.cc.OnResult(status, ResultFlag(flag), steps, result, nil)
}

func (h *CallHandler) OnCall(from, to module.Address, value,
	limit *big.Int, dataType string, dataObj *codec.TypedObj,
) {
	if h.Log.TraceMode() == module.TraceModeInvoke {
		po, _ := common.DecodeAnyForJSON(dataObj)
		h.Log.TSystemf("CALL start from=%v to=%v value=%v steplimit=%v dataType=%s data=%s",
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
			h.ch.OnResult(err, 0, h.cc.StepAvailable(), nil)
		}
	} else {
		h.cc.OnCall(handler, limit)
	}
}

func (h *CallHandler) OnAPI(status error, info *scoreapi.Info) {
	h.Log.Panicln("Unexpected OnAPI() call")
}

func (h *CallHandler) OnSetFeeProportion(portion int) {
	var payer module.Address
	if h.as.UseSystemDeposit() {
		payer = state.SystemAddress
	} else {
		payer = h.To
	}
	h.Log.TSystemf("CALL setFeeProportion contract=%s payer=%s portion=%d", h.To, payer, portion)
	h.cc.SetFeeProportion(payer, portion)
}

func (h *CallHandler) SetCode(code []byte) error {
	if h.forDeploy == false {
		return errors.InvalidStateError.New("Unexpected call SetCode()")
	}
	c := h.contract(h.as)
	return c.SetCode(code)
}

func (h *CallHandler) GetObjGraph(flags bool) (int, []byte, []byte, error) {
	nid, hash, graph, err := h.as.GetObjGraph(h.codeID, flags)
	if state.MissingGraphDataError.Equals(err) {
		h.cc.AddSyncRequest(db.BytesByHash, hash)
		return 0, nil, nil, err
	}
	return nid, hash, graph, err
}

func (h *CallHandler) SetObjGraph(flags bool, nextHash int, objGraph []byte) error {
	if h.readOnly {
		return nil
	}
	return h.as.SetObjGraph(h.codeID, flags, nextHash, objGraph)
}

type TransferAndCallHandler struct {
	th *TransferHandler
	*CallHandler
}

func (h *TransferAndCallHandler) Prepare(ctx Context) (state.WorldContext, error) {
	if h.To.IsContract() {
		return h.CallHandler.Prepare(ctx)
	} else {
		return h.th.Prepare(ctx)
	}
}

func (h *TransferAndCallHandler) ExecuteAsync(cc CallContext) (err error) {
	h.TLogStart()
	defer func() {
		if err != nil {
			if err2 := h.ApplyCallSteps(cc); err2 != nil {
				err = err2
			}
			h.TLogDone(err, cc.StepUsed(), nil)
		}
	}()
	return h.DoExecuteAsync(cc, h, nil)
}

func (h *TransferAndCallHandler) DoExecuteAsync(cc CallContext, ch eeproxy.CallContext, store containerdb.BytesStoreState) (err error) {
	h.cc = cc
	h.ch = ch
	status, _, _ := h.th.DoExecuteSync(cc)
	if status != nil {
		return status
	}

	as := cc.GetAccountState(h.To.ID())
	apiInfo, err := as.APIInfo()
	if err != nil {
		return err
	}
	if apiInfo == nil {
		return scoreresult.New(module.StatusContractNotFound, "APIInfo() is null")
	} else {
		m := apiInfo.GetMethod(h.name)
		if h.name == scoreapi.FallbackMethodName {
			payable := m != nil && m.IsPayable()
			if cc.Revision().LegacyFallbackCheck() {
				if h.Value.Sign() > 0 && !payable {
					return scoreresult.ErrMethodNotPayable
				}
				if m == nil {
					if err := h.ApplyCallSteps(cc); err != nil {
						return err
					}
					go func() {
						ch.OnResult(nil, 0, new(big.Int), nil)
					}()
					return nil
				}
			} else {
				if !payable {
					return scoreresult.ErrMethodNotFound
				}
			}
		} else {
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
	}

	return h.CallHandler.DoExecuteAsync(cc, ch, store)
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
