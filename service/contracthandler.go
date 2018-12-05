package service

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
)

var contractMngr ContractManager

func init() {
	contractMngr = new(contractManager)
}

const (
	transactionTimeLimit = time.Duration(2 * time.Second)

	dataTypeNone    = ""
	dataTypeMessage = "message"
	dataTypeCall    = "call"
	dataTypeDeploy  = "deploy"
)

type (
	ContractManager interface {
		GetHandler(tc TransactionContext, from, to module.Address,
			value, stepLimit *big.Int, dataType string, data []byte) ContractHandler
		PrepareContractStore(module.Address)
		CheckContractStore(module.Address) (string, error)
	}

	ContractHandler interface {
		Prepare(wvs WorldVirtualState) (WorldVirtualState, error)
	}

	SyncContractHandler interface {
		ContractHandler
		ExecuteSync(wc WorldContext) (module.Status, *big.Int, module.Address)
	}

	AsyncContractHandler interface {
		ContractHandler
		ExecuteAsync(wc WorldContext) (<-chan interface{}, error)
		Cancel()

		EEType() string
		eeproxy.CallContext
	}
)

type contractManager struct {
}

func (cm *contractManager) GetHandler(tc TransactionContext,
	from, to module.Address, value, stepLimit *big.Int, dataType string,
	data []byte,
) ContractHandler {
	var handler ContractHandler
	switch dataType {
	case dataTypeCall:
		// TODO
		handler = new(MethodCallHandler)
	case dataTypeDeploy:
		// TODO
		handler = new(DeployHandler)
	case dataTypeMessage:
		fallthrough
	case dataTypeNone:
		fallthrough
	default:
		handler = &TransferHandler{
			from:       from,
			to:         to,
			value:      value,
			stepLimit:  stepLimit,
			hasMessage: dataType == dataTypeMessage,
			data:       data,
		}
	}
	return handler
}

// PrepareContractStore checks if contract codes are ready for a contract runtime
// and starts to download and uncompress otherwise.
func (cm *contractManager) PrepareContractStore(addr module.Address) {
	// TODO implement when meaningful parallel execution can be performed
}

func (cm *contractManager) CheckContractStore(addr module.Address) (string, error) {
	// TODO 만약 valid한 contract이 store에 존재하지 않으면, 저장을 마치고 그 path를 리턴한다.
	// TODO 만약 PrepareContractCode()에 의해서 저장 중이면, 저장 완료를 기다린다.
	panic("implement me")
}

func ExecuteTransfer(wc WorldContext, from, to module.Address,
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
	hasMessage       bool
	data             []byte
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
	status, step = ExecuteTransfer(wc, h.from, h.to, h.value, &stepAvail)
	stepUsed.Set(step)
	stepAvail.Sub(&stepAvail, step)

	if status == 0 && h.hasMessage {
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

type MethodCallHandler struct {
}

func (h *MethodCallHandler) Prepare(wvs WorldVirtualState) (WorldVirtualState, error) {
	panic("implement me")
}

func (h *MethodCallHandler) ExecuteAsync(wc WorldContext) (<-chan interface{}, error) {
	panic("implement me")
}

func (h *MethodCallHandler) Cancel() {
	panic("implement me")
}

func (h *MethodCallHandler) GetValue(key []byte) ([]byte, error) {
	panic("implement me")
}

func (h *MethodCallHandler) SetValue(key, value []byte) error {
	panic("implement me")
}

func (h *MethodCallHandler) GetInfo() map[string]interface{} {
	panic("implement me")
}

func (h *MethodCallHandler) OnEvent(addr module.Address, indexed, data [][]byte) {
	panic("implement me")
}

func (h *MethodCallHandler) OnResult(status uint16, steps *big.Int, result []byte) {
	panic("implement me")
}

func (h *MethodCallHandler) OnCall(from, to module.Address, value,
	limit *big.Int, params []byte,
) {
	panic("implement me")
}

type DeployHandler struct {
}

func (h *DeployHandler) Prepare(wvs WorldVirtualState) (WorldVirtualState, error) {
	panic("implement me")
}

func (h *DeployHandler) ExecuteSync(wc WorldContext) (module.Status, *big.Int, module.Address) {
	panic("implement me")
}
