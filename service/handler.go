package service

import (
	"math/big"
	"time"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
)

const (
	transactionTimeLimit = time.Duration(2 * time.Second)

	ctypeTransfer = 0x100
	ctypeNone     = iota
	ctypeMessage
	ctypeDeploy
	ctypeAccept
	ctypeCall
	ctypeGovCall
	ctypeTransferAndMessage = ctypeTransfer | ctypeMessage
	ctypeTransferAndCall    = ctypeTransfer | ctypeCall
	ctypeTransferAndDeploy  = ctypeTransfer | ctypeDeploy
)

type (
	ContractHandler interface {
		StepLimit() *big.Int
		Prepare(wc WorldContext) (WorldContext, error)
	}

	SyncContractHandler interface {
		ContractHandler
		ExecuteSync(wc WorldContext) (module.Status, *big.Int, []byte, module.Address)
	}

	AsyncContractHandler interface {
		ContractHandler
		ExecuteAsync(wc WorldContext) error
		SendResult(status module.Status, steps *big.Int, result interface{}) error
		Cancel()

		EEType() string
		eeproxy.CallContext
	}
)

type CommonHandler struct {
	from, to         module.Address
	value, stepLimit *big.Int
}

func newCommonHandler(from, to module.Address, value, stepLimit *big.Int) *CommonHandler {
	return &CommonHandler{from: from, to: to, value: value, stepLimit: stepLimit}
}

func (h *CommonHandler) StepLimit() *big.Int {
	return h.stepLimit
}

func (h *CommonHandler) Prepare(wc WorldContext) (WorldContext, error) {
	lq := []LockRequest{
		{string(h.from.ID()), AccountWriteLock},
		{string(h.to.ID()), AccountWriteLock},
	}
	return wc.WorldStateChanged(wc.WorldVirtualState().GetFuture(lq)), nil
}
