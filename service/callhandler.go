package service

import (
	"encoding/json"
	"log"
	"math/big"

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

	cc CallContext

	// set in ExecuteAsync()
	as   AccountState
	cm   ContractManager
	conn eeproxy.Proxy
}

// TODO data is not always JSON string, so consider it
func newCallHandler(ch *CommonHandler, data []byte, cc CallContext,
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
	}
}

func (h *CallHandler) Prepare(wc WorldContext) (WorldContext, error) {
	c := h.as.ActiveContract()
	if c == nil {
		return nil, errors.New("No active contract")
	}
	wc.ContractManager().PrepareContractStore(wc, c)

	lq := []LockRequest{{"", AccountWriteLock}}
	return wc.WorldStateChanged(wc.WorldVirtualState().GetFuture(lq)), nil
}

func (h *CallHandler) ExecuteAsync(wc WorldContext) error {
	h.as = wc.GetAccountState(h.to.ID())

	h.cm = wc.ContractManager()
	h.conn = h.cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	c := h.as.ActiveContract()
	if c == nil {
		return errors.New("No active contract")
	}
	ch := wc.ContractManager().PrepareContractStore(wc, c)
	select {
	case r := <-ch:
		if r.err != nil {
			return r.err
		}
		info := h.as.APIInfo()
		paramObj, err := info.ConvertParamsToTypedObj(h.method, h.params)
		if err != nil {
			return err
		}
		err = h.conn.Invoke(h, r.path, false, h.from, h.to,
			h.value, h.stepLimit, h.method, paramObj)
		return err
	default:
		go func() {
			select {
			case r := <-ch:
				if r.err == nil {
					info := h.as.APIInfo()
					if paramObj, err := info.ConvertParamsToTypedObj(h.method, h.params); err == nil {
						if err = h.conn.Invoke(h, r.path, false, h.from, h.to,
							h.value, h.stepLimit, h.method, paramObj); err == nil {
							return
						}
					}
				}
				h.OnResult(module.StatusSystemError, h.stepLimit, nil)
			}
		}()
	}
	return nil
}

func (h *CallHandler) SendResult(status module.Status, steps *big.Int, result *codec.TypedObj) error {
	if h.conn == nil {
		return errors.New("Don't have a connection of (" + h.EEType() + ")")
	}
	return h.conn.SendResult(h, uint16(status), steps, result)
}

func (h *CallHandler) Cancel() {
	// Do nothing
}

func (h *CallHandler) EEType() string {
	// TODO resolve it at run time
	return "python"
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
	h.cc.OnEvent(indexed, data)
}

func (h *CallHandler) OnResult(status uint16, steps *big.Int, result *codec.TypedObj) {
	h.cc.OnResult(module.Status(status), steps, result, nil)
}

func (h *CallHandler) OnCall(from, to module.Address, value,
	limit *big.Int, method string, params *codec.TypedObj,
) {
	ctype := ctypeNone
	if method != "" {
		ctype |= ctypeCall
	}
	if value.Sign() == 1 { // value >= 0
		ctype |= ctypeTransfer
	}
	if ctype == ctypeNone {
		log.Println("Invalid call:", from, to, value, method)

		if conn := h.cc.GetConnection(h.EEType()); conn != nil {
			conn.SendResult(h, uint16(module.StatusSystemError), h.stepLimit, nil)
		} else {
			// It can't be happened
			log.Println("FAIL to get connection of (", h.EEType(), ")")
		}
		return
	}

	// TODO need to prepare shortcut to make contract handler with
	//  *codec.TypedObj
	paramBytes, err := json.Marshal(common.MustDecodeAny(params))
	if err != nil {
		log.Panicf("Fail to marshal object to JSON err=%+v", err)
	}

	jso := dataCallJSON{Method: method, Params: paramBytes}
	data, err := json.Marshal(jso)
	if err != nil {
		log.Panicln("Wrong params: FAIL to create data JSON string")
	}
	handler := h.cm.GetHandler(h.cc, from, to, value, limit, ctype, data)
	h.cc.OnCall(handler)
}

func (h *CallHandler) OnAPI(info *scoreapi.Info) {
	h.as.SetAPIInfo(info)
}

type TransferAndCallHandler struct {
	th *TransferHandler
	*CallHandler
}

func (h *TransferAndCallHandler) Prepare(wc WorldContext) (WorldContext, error) {
	if wc, err := h.th.Prepare(wc); err == nil {
		return h.CallHandler.Prepare(wc)
	} else {
		return wc, err
	}
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
