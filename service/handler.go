package service

import (
	"math/big"
	"time"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
)

const (
	transactionTimeLimit     = time.Duration(2 * time.Second)
	scoreDecompressTimeLimit = time.Duration(3 * time.Second)

	ctypeTransfer = 0x100
	ctypeNone     = iota
	ctypeDeploy
	ctypeCall
	ctypeGovCall
	ctypeTransferAndCall = ctypeTransfer | ctypeCall
)

type (
	ContractHandler interface {
		StepLimit() *big.Int
		// TODO Not an adequate API here.
		ResetSteps(*big.Int)
		ApplySteps(WorldContext, StepType, int) bool
		Prepare(WorldContext) (WorldContext, error)
	}

	SyncContractHandler interface {
		ContractHandler
		ExecuteSync(wc WorldContext) (module.Status, *big.Int, *codec.TypedObj, module.Address)
	}

	AsyncContractHandler interface {
		ContractHandler
		ExecuteAsync(wc WorldContext) error
		SendResult(status module.Status, steps *big.Int, result *codec.TypedObj) error
		Dispose()

		EEType() string
		eeproxy.CallContext
	}
)

type CommonHandler struct {
	from, to                   module.Address
	value, stepLimit, stepUsed *big.Int
}

func newCommonHandler(from, to module.Address, value, stepLimit *big.Int) *CommonHandler {
	return &CommonHandler{
		from: from, to: to, value: value, stepLimit: stepLimit,
		stepUsed: big.NewInt(0)}
}

func (h *CommonHandler) StepLimit() *big.Int {
	return h.stepLimit
}

func (h *CommonHandler) ResetSteps(limit *big.Int) {
	h.stepLimit = limit
	h.stepUsed = big.NewInt(0)
}

func (h *CommonHandler) ApplySteps(wc WorldContext, stepType StepType, n int) bool {
	h.stepUsed.Add(h.stepUsed, big.NewInt(wc.StepsFor(stepType, n)))
	if h.stepUsed.Cmp(h.stepLimit) > 0 {
		h.stepUsed = h.stepLimit
		return false
	}
	return true
}

func (h *CommonHandler) Prepare(wc WorldContext) (WorldContext, error) {
	lq := []LockRequest{
		{string(h.from.ID()), AccountWriteLock},
		{string(h.to.ID()), AccountWriteLock},
	}
	return wc.GetFuture(lq), nil
}

func (h *CommonHandler) StepAvail() *big.Int {
	return big.NewInt(0).Sub(h.stepLimit, h.stepUsed)
}
