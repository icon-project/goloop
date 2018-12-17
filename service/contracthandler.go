package service

import (
	"encoding/json"
	"log"
	"math/big"
	"time"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/pkg/errors"
)

var contractMngr ContractManager

func init() {
	contractMngr = new(contractManager)
}

const (
	transactionTimeLimit = time.Duration(2 * time.Second)

	ctypeTransfer = 0x100
	ctypeNone     = iota
	ctypeMessage
	ctypeCall
	ctypeDeploy
	ctypeTransferAndMessage = ctypeTransfer | ctypeMessage
	ctypeTransferAndCall    = ctypeTransfer | ctypeCall
	ctypeTransferAndDeploy  = ctypeTransfer | ctypeDeploy
)

type (
	ContractManager interface {
		GetHandler(cc CallContext, from, to module.Address,
			value, stepLimit *big.Int, ctype int, data []byte) ContractHandler
		PrepareContractStore(WorldState, module.Address)
		CheckContractStore(WorldState, module.Address) (string, error)
	}

	ContractHandler interface {
		StepLimit() *big.Int
		Prepare(wvs WorldVirtualState) (WorldVirtualState, error)
	}

	SyncContractHandler interface {
		ContractHandler
		ExecuteSync(wc WorldContext) (module.Status, *big.Int, []byte, module.Address)
	}

	AsyncContractHandler interface {
		ContractHandler
		ExecuteAsync(wc WorldContext) error
		SendResult(status module.Status, steps *big.Int, result []byte) error
		Cancel()

		EEType() string
		eeproxy.CallContext
	}
)

type contractManager struct {
}

func (cm *contractManager) GetHandler(cc CallContext,
	from, to module.Address, value, stepLimit *big.Int, ctype int, data []byte,
) ContractHandler {
	var handler ContractHandler
	switch ctype {
	case ctypeTransfer:
		handler = &TransferHandler{
			from:      from,
			to:        to,
			value:     value,
			stepLimit: stepLimit,
		}
	case ctypeTransferAndMessage:
		handler = &TransferAndMessageHandler{
			TransferHandler: TransferHandler{
				from:      from,
				to:        to,
				value:     value,
				stepLimit: stepLimit,
			},
			data: data,
		}
	case ctypeTransferAndDeploy:
		// TODO
		panic("implement me")
	case ctypeTransferAndCall:
		handler = &TransferAndCallHandler{
			*newCallHandler(from, to, value, stepLimit, data, cc),
		}
	case ctypeCall:
		handler = newCallHandler(from, to, value, stepLimit, data, cc)
	}
	return handler
}

// PrepareContractStore checks if contract codes are ready for a contract runtime
// and starts to download and uncompress otherwise.
func (cm *contractManager) PrepareContractStore(ws WorldState, addr module.Address) {
	// TODO implement when meaningful parallel execution can be performed
}

func (cm *contractManager) CheckContractStore(ws WorldState, addr module.Address,
) (string, error) {
	// TODO 만약 valid한 contract이 store에 존재하지 않으면, 저장을 마치고 그 path를 리턴한다.
	// TODO 만약 PrepareContractCode()에 의해서 저장 중이면, 저장 완료를 기다린다.
	panic("implement me")
}

func executeTransfer(wc WorldContext, from, to module.Address,
	value, limit *big.Int,
) (module.Status, *big.Int) {
	stepUsed := big.NewInt(wc.StepsFor(StepTypeDefault, 1))

	if stepUsed.Cmp(limit) > 0 {
		return module.StatusNotPayable, limit
	}

	as1 := wc.GetAccountState(from.ID())
	bal1 := as1.GetBalance()
	if bal1.Cmp(value) < 0 {
		return module.StatusOutOfBalance, limit
	}
	bal1.Sub(bal1, value)
	as1.SetBalance(bal1)

	as2 := wc.GetAccountState(to.ID())
	bal2 := as2.GetBalance()
	bal2.Add(bal2, value)
	as2.SetBalance(bal2)

	return module.StatusSuccess, stepUsed
}

type TransferHandler struct {
	from, to         module.Address
	value, stepLimit *big.Int
}

func (h *TransferHandler) StepLimit() *big.Int {
	return h.stepLimit
}

func (h *TransferHandler) Prepare(wvs WorldVirtualState) (WorldVirtualState, error) {
	lq := []LockRequest{
		{string(h.from.ID()), AccountWriteLock},
		{string(h.to.ID()), AccountWriteLock},
	}
	return wvs.GetFuture(lq), nil
}

func (h *TransferHandler) ExecuteSync(wc WorldContext) (module.Status, *big.Int, module.Address) {
	stepPrice := wc.StepPrice()
	var (
		fee                 big.Int
		status              module.Status
		step, bal1          *big.Int
		stepUsed, stepAvail big.Int
	)
	wcs := wc.GetSnapshot()
	as1 := wc.GetAccountState(h.from.ID())
	stepAvail.Set(h.stepLimit)

	// it tries to execute
	status, step = executeTransfer(wc, h.from, h.to, h.value, &stepAvail)
	stepUsed.Set(step)
	stepAvail.Sub(&stepAvail, step)

	// try to charge fee
	fee.Mul(&stepUsed, stepPrice)
	bal1 = as1.GetBalance()
	for bal1.Cmp(&fee) < 0 {
		if status == 0 {
			// rollback all changes
			status = module.StatusNotPayable
			wc.Reset(wcs)
			bal1 = as1.GetBalance()

			stepUsed.Set(h.stepLimit)
			fee.Mul(&stepUsed, stepPrice)
		} else {
			//stepPrice.SetInt64(0)
			fee.SetInt64(0)
		}
	}
	bal1.Sub(bal1, &fee)
	as1.SetBalance(bal1)

	return status, &stepUsed, nil
}

type TransferAndMessageHandler struct {
	TransferHandler
	data []byte
}

func (h *TransferAndMessageHandler) ExecuteSync(wc WorldContext) (module.Status, *big.Int, module.Address) {
	stepPrice := wc.StepPrice()
	var (
		fee                 big.Int
		status              module.Status
		step, bal1          *big.Int
		stepUsed, stepAvail big.Int
	)
	wcs := wc.GetSnapshot()
	as1 := wc.GetAccountState(h.from.ID())
	stepAvail.Set(h.stepLimit)

	// it tries to execute
	status, step = executeTransfer(wc, h.from, h.to, h.value, &stepAvail)
	stepUsed.Set(step)
	stepAvail.Sub(&stepAvail, step)

	if status == 0 {
		var data interface{}
		if err := json.Unmarshal(h.data, &data); err != nil {
			status = module.StatusSystemError
			step = &stepAvail
		} else {
			var stepsForMessage big.Int
			stepsForMessage.SetInt64(wc.StepsFor(StepTypeInput, countBytesOfData(data)))
			if stepAvail.Cmp(&stepsForMessage) < 0 {
				status = module.StatusNotPayable
				step = &stepAvail
			} else {
				step = &stepsForMessage
			}
		}
		stepUsed.Add(&stepUsed, step)
		stepAvail.Sub(&stepAvail, step)
	}

	// try to charge fee
	fee.Mul(&stepUsed, stepPrice)
	bal1 = as1.GetBalance()
	for bal1.Cmp(&fee) < 0 {
		if status == 0 {
			// rollback all changes
			status = module.StatusNotPayable
			wc.Reset(wcs)
			bal1 = as1.GetBalance()

			stepUsed.Set(h.stepLimit)
			fee.Mul(&stepUsed, stepPrice)
		} else {
			//stepPrice.SetInt64(0)
			fee.SetInt64(0)
		}
	}
	bal1.Sub(bal1, &fee)
	as1.SetBalance(bal1)

	return status, &stepUsed, nil
}

type CallHandler struct {
	TransferHandler

	method string
	params []byte

	cc CallContext

	// set in ExecuteAsync()
	as   AccountState
	conn eeproxy.Proxy
}

func newCallHandler(from, to module.Address, value, stepLimit *big.Int,
	data []byte, cc CallContext,
) *CallHandler {
	var dataJSON struct {
		method string          `json:"method"`
		params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(data, &dataJSON); err != nil {
		log.Println("FAIL to parse 'data' of transaction")
		return nil
	}
	return &CallHandler{
		TransferHandler: TransferHandler{from: from, to: to, value: value, stepLimit: stepLimit},
		method:          dataJSON.method,
		params:          dataJSON.params,
		cc:              cc,
	}
}

func (h *CallHandler) Prepare(wvs WorldVirtualState) (WorldVirtualState, error) {
	return wvs.GetFuture([]LockRequest{{"", AccountWriteLock}}), nil
}

func (h *CallHandler) ExecuteAsync(wc WorldContext) error {
	h.as = wc.GetAccountState(h.to.ID())

	path, err := contractMngr.CheckContractStore(wc, h.to)
	if err != nil {
		return err
	}
	h.conn = h.cc.GetConnection(h.EEType())
	if h.conn == nil {
		return errors.New("FAIL to get connection of (" + h.EEType() + ")")
	}
	err = h.conn.Invoke(h, path, false, h.from, h.to, h.value,
		h.stepLimit, h.method, h.params)
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

func (h *CallHandler) GetInfo() map[string]interface{} {
	return h.cc.GetInfo()
}

func (h *CallHandler) GetBalance(addr module.Address) *big.Int {
	return h.cc.GetBalance(addr)
}

func (h *CallHandler) OnEvent(addr module.Address, indexed, data [][]byte) {
	h.cc.OnEvent(indexed, data)
}

func (h *CallHandler) OnResult(status uint16, steps *big.Int, result []byte) {
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
			conn.SendResult(h, uint16(module.StatusSystemError), h.stepLimit, nil)
		} else {
			// It can't be happened
			log.Println("FAIL to get connection of (", h.EEType(), ")")
		}
		return
	}

	// TODO make data from method and params
	var data []byte
	handler := contractMngr.GetHandler(h.cc, from, to, value, limit, ctype, data)
	h.cc.OnCall(handler)
}

func (h *CallHandler) OnAPI(obj interface{}) {
	// TODO
	panic("implement me")
}

type TransferAndCallHandler struct {
	CallHandler
}

func (h *TransferAndCallHandler) Prepare(wvs WorldVirtualState) (WorldVirtualState, error) {
	if wvs, err := h.TransferHandler.Prepare(wvs); err == nil {
		return h.CallHandler.Prepare(wvs)
	} else {
		return wvs, err
	}
}

func (h *TransferAndCallHandler) ExecuteAsync(wc WorldContext) error {
	if status, stepUsed, _ := h.TransferHandler.ExecuteSync(wc); status == 0 {
		return h.CallHandler.ExecuteAsync(wc)
	} else {
		go func() {
			h.cc.OnResult(module.Status(status), stepUsed, nil, nil)
		}()

		return nil
	}
}

type DeployHandler struct {
}

func (h *DeployHandler) Prepare(wvs WorldVirtualState) (WorldVirtualState, error) {
	// TODO
	panic("implement me")
}

func (h *DeployHandler) ExecuteSync(wc WorldContext) (module.Status, *big.Int, module.Address) {
	// TODO
	panic("implement me")
}
