package service

import (
	"math/big"
	"time"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/state"
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
		ApplySteps(state.WorldContext, state.StepType, int) bool
		ResetSteps(*big.Int)
		Prepare(ctx Context) (state.WorldContext, error)
	}

	SyncContractHandler interface {
		ContractHandler
		ExecuteSync(ctx Context) (module.Status, *big.Int, *codec.TypedObj, module.Address)
	}

	AsyncContractHandler interface {
		ContractHandler
		ExecuteAsync(ctx Context) error
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

func (h *CommonHandler) ApplySteps(wc state.WorldContext, stepType state.StepType, n int) bool {
	h.stepUsed.Add(h.stepUsed, big.NewInt(wc.StepsFor(stepType, n)))
	if h.stepUsed.Cmp(h.stepLimit) > 0 {
		h.stepUsed = h.stepLimit
		return false
	}
	return true
}

// ResetSteps resets stepLimit and stepUsed. Actually, it is used to set stepLimit
// when the current stepLimit exceeds system stepLimit just before execution.
func (h *CommonHandler) ResetSteps(limit *big.Int) {
	h.stepLimit = limit
	h.stepUsed = big.NewInt(0)
}

func (h *CommonHandler) Prepare(ctx Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{string(h.from.ID()), state.AccountWriteLock},
		{string(h.to.ID()), state.AccountWriteLock},
	}
	return ctx.GetFuture(lq), nil
}

func (h *CommonHandler) StepAvail() *big.Int {
	return big.NewInt(0).Sub(h.stepLimit, h.stepUsed)
}
