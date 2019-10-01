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
)

const (
	MethodFallback = "fallback"
)

type DataCallJSON struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type CallHandler struct {
	*CommonHandler

	method string
	params []byte
	// nil paramObj means it needs to convert params to *codec.TypedObj.
	paramObj *codec.TypedObj

	forDeploy bool
	disposed  bool
	lock      sync.Mutex

	// set in ExecuteAsync()
	cc        CallContext
	as        state.AccountState
	cm        ContractManager
	conn      eeproxy.Proxy
	cs        ContractStore
	isSysCall bool
}

func newCallHandler(ch *CommonHandler, data []byte, forDeploy bool) *CallHandler {
	h := &CallHandler{
		CommonHandler: ch,
		forDeploy:     forDeploy,
		disposed:      false,
		isSysCall:     false,
	}
	if data != nil {
		var jso DataCallJSON
		if err := json.Unmarshal(data, &jso); err != nil {
			ch.log.Debugf("FAIL to parse 'data' of transaction err(%+v)\ndata(%s)\n", err, data)
			return nil
		}
		h.method = jso.Method
		h.params = jso.Params
	} else if ch.to.IsContract() {
		h.method = MethodFallback
		h.params = []byte("{}")
	}
	return h
}

func newCallHandlerFromTypedObj(ch *CommonHandler, method string,
	paramObj *codec.TypedObj, forDeploy bool,
) *CallHandler {
	return &CallHandler{
		CommonHandler: ch,
		method:        method,
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

	info := as.APIInfo()
	if info == nil {
		return wc, as
	}

	method := info.GetMethod(h.method)
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
	if !h.forDeploy {
		return as.ActiveContract()
	} else {
		return as.NextContract()
	}
}

func (h *CallHandler) ExecuteAsync(cc CallContext) error {
	h.cc = cc

	// Calculate steps
	if !h.forDeploy {
		if !h.ApplySteps(cc, state.StepTypeContractCall, 1) {
			status := scoreresult.OutOfStepError.New("FailToApplyContractCall")
			h.cc.OnResult(status, h.StepUsed(), nil, nil)
			return nil
		}
	}

	// Prepare
	h.as = cc.GetAccountState(h.to.ID())
	if !h.as.IsContract() {
		return scoreresult.New(module.StatusContractNotFound, "NotAContractAccount")
	}
	cc.SetContractInfo(&state.ContractInfo{Owner: h.as.ContractOwner()})

	// Set up contract files
	c := h.contract(h.as)
	if c == nil {
		return scoreresult.New(module.StatusContractNotFound, "NotActiveContract")
	}

	if strings.Compare(c.ContentType(), state.CTAppSystem) == 0 {
		h.isSysCall = true

		var status error
		var result *codec.TypedObj
		from := h.from
		if from == nil {
			from = common.NewAddress(state.SystemID)
		}
		sScore, err := GetSystemScore(CID_CHAIN, from, cc, h.log)
		if err != nil {
			return err
		}
		err = h.ensureParamObj()
		if err == nil {
			var step *big.Int
			status, result, step = Invoke(sScore, h.method, h.paramObj)
			h.DeductSteps(step)
			go func() {
				cc.OnResult(status, h.StepUsed(), result, nil)
			}()
		}
		return err
	}

	h.cm = cc.ContractManager()
	h.conn = cc.GetProxy(h.EEType())
	if h.conn == nil {
		return errors.ExecutionFailError.Errorf(
			"FAIL to get connection for (%s)", h.EEType())
	}
	if err := h.prepareContractStore(cc, cc, c); err != nil {
		h.log.Warnf("FAIL to prepare contract. err=%+v\n", err)
		return errors.CriticalIOError.Wrap(err, "FAIL to prepare contract")
	}
	path, err := h.cs.WaitResult()
	if err != nil {
		h.log.Warnf("FAIL to prepare contract. err=%+v\n", err)
		return errors.CriticalIOError.Wrap(err, "FAIL to prepare contract")
	}

	// Execute
	h.lock.Lock()
	if !h.disposed {
		if err = h.ensureParamObj(); err == nil {
			err = h.conn.Invoke(h, path, h.cc.QueryMode(), h.from, h.to,
				h.value, h.StepAvail(), h.method, h.paramObj)
		}
	}
	h.lock.Unlock()

	return err
}

func (h *CallHandler) ensureParamObj() error {
	info := h.as.APIInfo()
	if info == nil {
		return scoreresult.New(module.StatusContractNotFound, "APIInfo() is null")
	}

	if h.paramObj != nil {
		if params, err := info.EnsureParamsSequential(h.method, h.paramObj); err != nil {
			return err
		} else {
			h.paramObj = params
			return nil
		}
	}

	var err error
	h.paramObj, err = info.ConvertParamsToTypedObj(h.method, h.params)
	return err
}

func (h *CallHandler) SendResult(status error, steps *big.Int, result *codec.TypedObj) error {
	if !h.isSysCall {
		if h.conn == nil {
			return errors.ExecutionFailError.Errorf(
				"Don't have a connection for (%s)", h.EEType())
		}
		return h.conn.SendResult(h, status, steps, result)
	} else {
		h.DeductSteps(steps)
		h.cc.OnResult(status, h.StepUsed(), result, nil)
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

func (h *CallHandler) EEType() string {
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

func (h *CallHandler) SetValue(key, value []byte) error {
	if h.as != nil {
		return h.as.SetValue(key, value)
	} else {
		return errors.CriticalUnknownError.Errorf(
			"SetValue: No Account(%s) exists", h.to)
	}
}

func (h *CallHandler) DeleteValue(key []byte) error {
	if h.as != nil {
		return h.as.DeleteValue(key)
	} else {
		return errors.CriticalUnknownError.Errorf(
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
	h.cc.OnEvent(addr, indexed, data)
}

func (h *CallHandler) OnResult(status error, steps *big.Int, result *codec.TypedObj) {
	h.DeductSteps(steps)
	h.cc.OnResult(status, h.StepUsed(), result, nil)
}

func (h *CallHandler) OnCall(from, to module.Address, value,
	limit *big.Int, method string, params *codec.TypedObj,
) {
	h.cc.OnCall(h.cm.GetCallHandler(from, to, value, limit, method, params))
}

func (h *CallHandler) OnAPI(status error, info *scoreapi.Info) {
	h.log.Panicln("Unexpected OnAPI() call")
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

func (h *TransferAndCallHandler) ExecuteAsync(cc CallContext) error {
	if h.to.IsContract() {
		as := cc.GetAccountState(h.to.ID())
		apiInfo := as.APIInfo()
		if apiInfo == nil {
			return scoreresult.New(module.StatusContractNotFound, "APIInfo() is null")
		} else {
			m := apiInfo.GetMethod(h.method)
			if m == nil {
				return scoreresult.ErrMethodNotFound
			}
			if m == nil || !m.IsPayable() {
				return scoreresult.ErrMethodNotPayable
			}
		}
	}

	status, stepUsed, result, addr := h.th.ExecuteSync(cc)
	if status == nil {
		return h.CallHandler.ExecuteAsync(cc)
	}
	go func() {
		cc.OnResult(status, stepUsed, result, addr)
	}()
	return nil
}
