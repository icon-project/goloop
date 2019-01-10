package service

import (
	"encoding/json"
	"log"
	"math/big"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/pkg/errors"
)

type dataCallJSON struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type CallHandler struct {
	*CommonHandler

	method string
	params []byte
	// nil paramObj means it needs to convert params to *codec.TypedObj.
	paramObj *codec.TypedObj

	cc        CallContext
	forDeploy bool
	canceled  bool
	lock      sync.Mutex

	// set in ExecuteAsync()
	as   AccountState
	cm   ContractManager
	conn eeproxy.Proxy
}

func newCallHandler(ch *CommonHandler, data []byte, cc CallContext, forDeploy bool,
) *CallHandler {
	var jso dataCallJSON
	if err := json.Unmarshal(data, &jso); err != nil {
		log.Println("FAIL to parse 'data' of transaction")
		return nil
	}
	return &CallHandler{
		CommonHandler: ch,
		method:        jso.Method,
		params:        jso.Params,
		cc:            cc,
		forDeploy:     forDeploy,
		canceled:      false,
	}
}

func newCallHandlerFromTypedObj(ch *CommonHandler, method string,
	paramObj *codec.TypedObj, cc CallContext, forDeploy bool,
) *CallHandler {
	return &CallHandler{
		CommonHandler: ch,
		method:        method,
		paramObj:      paramObj,
		cc:            cc,
		forDeploy:     forDeploy,
	}
}

func (h *CallHandler) Prepare(wc WorldContext) (WorldContext, error) {
	as := wc.GetAccountState(h.to.ID())
	if as == nil {
		return nil, errors.New("No contract account")
	}
	c := h.contract(as)
	if c == nil {
		return nil, errors.New("No active contract")
	}
	wc.ContractManager().PrepareContractStore(wc, c)

	lq := []LockRequest{{"", AccountWriteLock}}
	return wc.GetFuture(lq), nil
}

func (h *CallHandler) contract(as AccountState) Contract {
	if !h.forDeploy {
		return as.ActiveContract()
	} else {
		return as.NextContract()
	}
}

func (h *CallHandler) ExecuteAsync(wc WorldContext) error {
	h.as = wc.GetAccountState(h.to.ID())
	if h.as == nil {
		return errors.New("No contract account")
	}

	wc.SetContractInfo(&ContractInfo{Owner: h.as.ContractOwner()})

	h.cm = wc.ContractManager()
	h.conn = h.cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	c := h.contract(h.as)
	if c == nil {
		return errors.New("No active contract")
	}
	ch := wc.ContractManager().PrepareContractStore(wc, c)

	h.cc.SetTimer()
	select {
	case r := <-ch:
		if r.err != nil {
			return r.err
		}
		var err error
		if err = h.ensureParamObj(); err == nil {
			err = h.conn.Invoke(h, r.path, false, h.from, h.to,
				h.value, h.stepLimit, h.method, h.paramObj)
		}
		return err
	default:
		go func() {
			select {
			case r := <-ch:
				h.lock.Lock()
				if !h.canceled {
					if r.err == nil {
						var err error
						if err = h.ensureParamObj(); err == nil {
							if err = h.conn.Invoke(h, r.path, false,
								h.from, h.to, h.value, h.stepLimit, h.method,
								h.paramObj); err == nil {
								return
							}
						}
					}
					h.cc.OnResult(module.StatusSystemError, h.stepLimit, nil, nil)
				}
				h.lock.Unlock()
			}
		}()
	}
	return nil
}

func (h *CallHandler) ensureParamObj() error {
	info := h.as.APIInfo()
	if info == nil {
		return errors.New("No API Info in " + h.to.String())
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

func (h *CallHandler) Cancel() {
	h.lock.Lock()
	h.canceled = true
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
	h.cc.OnResult(module.Status(status), steps, result, nil)
}

func (h *CallHandler) OnCall(from, to module.Address, value,
	limit *big.Int, method string, params *codec.TypedObj,
) {
	h.cc.OnCall(h.cm.GetCallHandler(h.cc, from, to, value, limit, method, params))
}

func (h *CallHandler) OnAPI(info *scoreapi.Info) {
	log.Panicln("Unexpected OnAPI() call")
}

type TransferAndCallHandler struct {
	th *TransferHandler
	*CallHandler
}

func (h *TransferAndCallHandler) Prepare(wc WorldContext) (WorldContext, error) {
	return h.CallHandler.Prepare(wc)
}

func (h *TransferAndCallHandler) ExecuteAsync(wc WorldContext) error {
	if status, stepUsed, result, addr := h.th.ExecuteSync(wc); status == 0 {
		return h.CallHandler.ExecuteAsync(wc)
	} else {
		go func() {
			h.cc.OnResult(module.Status(status), stepUsed, result, addr)
		}()

		return nil
	}
}
