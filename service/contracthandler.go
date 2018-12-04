package service

import (
	"math/big"
	"time"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
)

var contractMngr ContractManager

// TODO eeManager는 contractMngr 안에?
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
		eeproxy.CallContext

		Prepare(wvs WorldVirtualState) (WorldVirtualState, error)
		Cancel()
	}

	SyncContractHandler interface {
		ContractHandler
		ExecuteSync(wc WorldContext) (int, *big.Int)
	}

	AsyncContractHandler interface {
		ContractHandler
		ExecuteAsync(wc WorldContext) <-chan interface{}

		EEType() string
	}
)

type contractManager struct {
}

func (cm *contractManager) GetHandler(tc TransactionContext, from, to module.Address, value, stepLimit *big.Int, dataType string, data interface{}) ContractHandler {
	// TODO
	panic("implement me")
}
func (cm *contractManager) PrepareContractCode(module.Address) {
	// TODO 만약 valid한 contract이 store에 존재하지 않으면, 저장을 시도한다.
	// 저장이 완료될 때까지 기다리지 않는다.
	// 이건 일단 빈 구현을 두고 나중에 구현해도 된다.
	panic("implement me")
}

func (cm *contractManager) CheckContractStore(module.Address) string {
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
