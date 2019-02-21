package contract

import (
	"encoding/json"
	"log"
	"math/big"
	"strings"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
	"github.com/pkg/errors"
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
	cc   CallContext
	as   state.AccountState
	cm   ContractManager
	conn eeproxy.Proxy
	cs   ContractStore
}

func newCallHandler(ch *CommonHandler, data []byte, forDeploy bool) *CallHandler {
	h := &CallHandler{
		CommonHandler: ch,
		forDeploy:     forDeploy,
		disposed:      false,
	}
	if data != nil {
		var jso DataCallJSON
		if err := json.Unmarshal(data, &jso); err != nil {
			log.Println("FAIL to parse 'data' of transaction")
			return nil
		}
		h.method = jso.Method
		h.params = jso.Params
	} else if ch.to.IsContract() {
		h.method = "fallback"
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
	}
}

func (h *CallHandler) Prepare(ctx Context) (state.WorldContext, error) {
	as := ctx.GetAccountState(h.to.ID())
	c := h.contract(as)
	if c == nil {
		return nil, errors.New("No active contract")
	}

	var err error
	h.lock.Lock()
	if h.cs == nil {
		h.cs, err = ctx.ContractManager().PrepareContractStore(ctx, c)
	}
	h.lock.Unlock()
	if err != nil {
		return nil, err
	}

	lq := []state.LockRequest{{"", state.AccountWriteLock}}
	return ctx.GetFuture(lq), nil
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
	if !h.ApplySteps(cc, state.StepTypeContractCall, 1) {
		h.cc.OnResult(module.StatusOutOfStep, h.stepLimit, nil, nil)
		return nil
	}

	// Prepare
	h.as = cc.GetAccountState(h.to.ID())
	if !h.as.IsContract() {
		return errors.New("FAIL: not a contract account")
	}
	cc.SetContractInfo(&state.ContractInfo{Owner: h.as.ContractOwner()})

	// Set up contract files
	c := h.contract(h.as)
	if c == nil {
		return errors.New("No active contract")
	}

	if strings.Compare(c.ContentType(), state.CTAppSystem) == 0 {
		var status module.Status
		var result *codec.TypedObj
		// TODO add transactionInfo to icx_call

		var from module.Address
		if f := cc.GetInfo()[state.InfoTxFrom]; f != nil {
			from = f.(module.Address)
		}
		sScore := GetSystemScore(from, h.to, cc)
		if err := h.ensureParamObj(); err == nil {
			status, result = sScore.Invoke(h.method, h.paramObj)
		}
		go func() {
			cc.OnResult(module.Status(status), big.NewInt(0), result, nil)
		}()
		// TODO define error
		return nil
	}

	h.cm = cc.ContractManager()
	h.conn = cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}
	h.lock.Lock()
	var err error
	if h.cs == nil {
		h.cs, err = cc.ContractManager().PrepareContractStore(cc, c)
	}
	h.lock.Unlock()
	if err != nil {
		return err
	}
	path, err := h.cs.WaitResult()
	if err != nil {
		return err
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
		return scoreresult.NewError(module.StatusContractNotFound, "APIInfo() is null")
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

func (h *CallHandler) SendResult(status module.Status, steps *big.Int, result *codec.TypedObj) error {
	if h.conn == nil {
		return errors.New("Don't have a connection of (" + h.EEType() + ")")
	}
	return h.conn.SendResult(h, uint16(status), steps, result)
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
		log.Println("No associated contract exists")
		return ""
	}
	return c.EEType()
}

func (h *CallHandler) GetValue(key []byte) ([]byte, error) {
	if h.as != nil {
		return h.as.GetValue(key)
	} else {
		return nil, errors.New("GetValue: No Account(" + h.to.String() + ") exists")
	}
}

func (h *CallHandler) SetValue(key, value []byte) error {
	if h.as != nil {
		return h.as.SetValue(key, value)
	} else {
		return errors.New("SetValue: No Account(" + h.to.String() + ") exists")
	}
}

func (h *CallHandler) DeleteValue(key []byte) error {
	if h.as != nil {
		return h.as.DeleteValue(key)
	} else {
		return errors.New("DeleteValue: No Account(" + h.to.String() + ") exists")
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

func (h *CallHandler) OnResult(status uint16, steps *big.Int, result *codec.TypedObj) {
	h.stepUsed.Add(h.stepUsed, steps)
	h.cc.OnResult(module.Status(status), h.stepUsed, result, nil)
}

func (h *CallHandler) OnCall(from, to module.Address, value,
	limit *big.Int, method string, params *codec.TypedObj,
) {
	h.cc.OnCall(h.cm.GetCallHandler(from, to, value, limit, method, params))
}

func (h *CallHandler) OnAPI(status uint16, obj *scoreapi.Info) {
	log.Panicln("Unexpected OnAPI() call")
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
			return scoreresult.NewError(module.StatusContractNotFound, "APIInfo() is null")
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
	if status == module.StatusSuccess {
		return h.CallHandler.ExecuteAsync(cc)
	}

	go func() {
		cc.OnResult(module.Status(status), stepUsed, result, addr)
	}()
	return nil
}
