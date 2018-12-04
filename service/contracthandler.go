package service

import (
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

	dataTypeMessage = "message"
	dataTypeCall    = "call"
	dataTypeDeploy  = "deploy"
)

type (
	ContractManager interface {
		GetHandler(tc TransactionContext, from, to module.Address, value, stepLimit *big.Int, dataType string, data interface{}) ContractHandler
		PrepareContractStore(module.Address)
		CheckContractStore(module.Address) string
	}

	ContractHandler interface {
		Prepare(wvs WorldVirtualState) (WorldVirtualState, error)
		Cancel()
	}

	SyncContractHandler interface {
		ContractHandler
		ExecuteSync(wc WorldContext) (int, *big.Int)
	}

	AsyncContractHandler interface {
		eeproxy.CallContext
		ContractHandler
		ExecuteAsync(wc WorldContext) <-chan interface{}

		EEType() string
	}
)

type contractManager struct {
}

func (cm *contractManager) GetHandler(tc TransactionContext, from, to module.Address, value, stepLimit *big.Int, dataType string, data interface{}) ContractHandler {
	var handler ContractHandler
	switch dataType {
	case dataTypeMessage:
	case dataTypeCall:
		handler = new(MethodCallHandler)
	case dataTypeDeploy:
		handler = new(DeployHandler)
		// TODO simple transfer
	}
	return handler
}

// PrepareContractStore checks if contract codes are ready for a contract runtime
// and starts to download and uncompress otherwise.
func (cm *contractManager) PrepareContractStore(addr module.Address) {
	// TODO implement when meaningful parallel execution can be performed
}

func (cm *contractManager) CheckContractStore(addr module.Address) string {
	// TODO 만약 valid한 contract이 store에 존재하지 않으면, 저장을 마치고 그 path를 리턴한다.
	// TODO 만약 PrepareContractCode()에 의해서 저장 중이면, 저장 완료를 기다린다.
	panic("implement me")
}

type MethodCallHandler struct {
}

func (h *MethodCallHandler) Prepare(wvs WorldVirtualState) (WorldVirtualState, error) {
	panic("implement me")
}

func (h *MethodCallHandler) ExecuteAsync(wc WorldContext) <-chan interface{} {
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

func (h *MethodCallHandler) OnCall(from, to module.Address, value, limit *big.Int, params []byte) {
	panic("implement me")
}

type TransferHandler struct {
}

func (h *TransferHandler) Prepare(wvs WorldVirtualState) (WorldVirtualState, error) {
	panic("implement me")
}

func (h *TransferHandler) ExecuteSync(wc WorldContext) (int, *big.Int) {
	panic("implement me")
}

func (h *TransferHandler) Cancel() {
	panic("implement me")
}

type DeployHandler struct {
}

func (h *DeployHandler) Prepare(wvs WorldVirtualState) (WorldVirtualState, error) {
	panic("implement me")
}

func (h *DeployHandler) ExecuteSync(wc WorldContext) (int, *big.Int) {
	panic("implement me")
}

func (h *DeployHandler) Cancel() {
	panic("implement me")
}
