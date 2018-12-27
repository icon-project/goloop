package service

import (
	"encoding/json"
	"log"
	"math/big"
	"sync"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/pkg/errors"
)

type dataCallJSON struct {
	method string          `json:"method"`
	params json.RawMessage `json:"params"`
}

type CallHandler struct {
	// Don't embed TransferHandler because it should not be an instance of
	// SyncContractHandler.
	th *TransferHandler

	method string
	params []byte

	cc  CallContext
	csp *contractStoreProxy

	// set in ExecuteAsync()
	as   AccountState
	cm   ContractManager
	conn eeproxy.Proxy
}

// TODO data is not always JSON string, so consider it
func newCallHandler(from, to module.Address, value, stepLimit *big.Int,
	data []byte, cc CallContext,
) *CallHandler {
	var jso dataCallJSON
	if err := json.Unmarshal(data, &jso); err != nil {
		log.Println("FAIL to parse 'data' of transaction")
		return nil
	}
	return &CallHandler{
		th:     &TransferHandler{from: from, to: to, value: value, stepLimit: stepLimit},
		method: jso.method,
		params: jso.params,
		cc:     cc,
		csp:    newContractStoreProxy(),
	}
}

func (h *CallHandler) StepLimit() *big.Int {
	return h.th.stepLimit
}

func (h *CallHandler) Prepare(wc WorldContext) (WorldContext, error) {
	h.csp.prepare(wc, h.as.ActiveContract())

	lq := []LockRequest{{"", AccountWriteLock}}
	return wc.WorldStateChanged(wc.WorldVirtualState().GetFuture(lq)), nil
}

func (h *CallHandler) ExecuteAsync(wc WorldContext) error {
	// TODO check if contract is active
	h.as = wc.GetAccountState(h.th.to.ID())

	h.cm = wc.ContractManager()
	h.conn = h.cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}

	path, err := h.csp.check(wc, h.as.ActiveContract())
	if err != nil {
		return err
	}

	err = h.conn.Invoke(h, path, false, h.th.from, h.th.to, h.th.value,
		h.th.stepLimit, h.method, h.params)
	if err != nil {
		return err
	}

	return nil
}

func (h *CallHandler) SendResult(status module.Status, steps *big.Int, result []byte) error {
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
		return nil, errors.New("GetValue: No Account(" + h.th.to.String() + ") exists")
	}
}

func (h *CallHandler) SetValue(key, value []byte) error {
	if h.as != nil {
		return h.as.SetValue(key, value)
	} else {
		return errors.New("SetValue: No Account(" + h.th.to.String() + ") exists")
	}
}

func (h *CallHandler) DeleteValue(key []byte) error {
	if h.as != nil {
		return h.as.DeleteValue(key)
	} else {
		return errors.New("DeleteValue: No Account(" + h.th.to.String() + ") exists")
	}
}

func (h *CallHandler) GetInfo() map[string]interface{} {
	return h.cc.GetInfo()
}

func (h *CallHandler) GetBalance(addr module.Address) *big.Int {
	return h.cc.GetBalance(addr)
}

func (h *CallHandler) OnEvent(addr module.Address, indexed, data [][]byte) {
	h.cc.OnEvent(indexed, data)
}

func (h *CallHandler) OnResult(status uint16, steps *big.Int, result interface{}) {
	h.cc.OnResult(module.Status(status), steps, result, nil)
}

func (h *CallHandler) OnCall(from, to module.Address, value,
	limit *big.Int, method string, params []byte,
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
			conn.SendResult(h, uint16(module.StatusSystemError), h.th.stepLimit, nil)
		} else {
			// It can't be happened
			log.Println("FAIL to get connection of (", h.EEType(), ")")
		}
		return
	}

	jso := dataCallJSON{method: method, params: params}
	data, err := json.Marshal(jso)
	if err != nil {
		log.Panicln("Wrong params: FAIL to create data JSON string")
	}
	handler := h.cm.GetHandler(h.cc, from, to, value, limit, ctype, data)
	h.cc.OnCall(handler)
}

func (h *CallHandler) OnAPI(obj interface{}) {
	log.Panicln("Unexpected OnAPI() call from Invoke()")
}

type TransferAndCallHandler struct {
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

type contractStoreProxy struct {
	contract Contract
	started  bool
	path     string
	err      error
	cv       *sync.Cond
}

func newContractStoreProxy() *contractStoreProxy {
	return &contractStoreProxy{cv: sync.NewCond(new(sync.Mutex))}
}

func (p *contractStoreProxy) prepare(wc WorldContext, contract Contract) {
	p.cv.L.Lock()
	if contract != nil && contract.Equal(p.contract) {
		// avoid to call PrepareContractStore() more than once for same contract
		return
	}
	p.cv.L.Unlock()

	if contract == nil {
		p.cv.L.Lock()
		p.err = errors.New("No contract exists")
		p.cv.L.Unlock()
		return
	}

	p.contract = contract
	wc.ContractManager().PrepareContractStore(wc, contract)
}

func (p *contractStoreProxy) check(wc WorldContext, contract Contract) (string, error) {
	p.cv.L.Lock()
	defer p.cv.L.Unlock()

	if contract != nil && contract.Equal(p.contract) && (p.err != nil || p.path != "") {
		return p.path, p.err
	}

	p.prepare(wc, contract)
	p.cv.Wait()
	return p.path, p.err
}

func (p *contractStoreProxy) onStoreCompleted(path string, err error) {
	p.cv.L.Lock()
	p.path = path
	p.err = err
	p.cv.Broadcast()
	p.cv.L.Unlock()
}
