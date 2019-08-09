package contract

import (
	"math/big"
	"time"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/state"
)

const (
	transactionTimeLimit     = time.Duration(2 * time.Second)
	scoreDecompressTimeLimit = time.Duration(2 * time.Second)

	CTypeTransfer = 0x100
	CTypeNone     = iota
	CTypeDeploy
	CTypeCall
	CTypePatch
	CTypeTransferAndCall = CTypeTransfer | CTypeCall
)

type (
	ContractHandler interface {
		StepLimit() *big.Int
		StepUsed() *big.Int
		ApplySteps(state.WorldContext, state.StepType, int) bool
		DeductSteps(*big.Int) bool
		ResetSteps(*big.Int)
		Prepare(ctx Context) (state.WorldContext, error)
	}

	SyncContractHandler interface {
		ContractHandler
		ExecuteSync(cc CallContext) (module.Status, *big.Int, *codec.TypedObj, module.Address)
	}

	AsyncContractHandler interface {
		ContractHandler
		ExecuteAsync(cc CallContext) error
		SendResult(status module.Status, steps *big.Int, result *codec.TypedObj) error
		Dispose()

		EEType() string
		eeproxy.CallContext
	}
)

type CommonHandler struct {
	from, to                   module.Address
	value, stepLimit, stepUsed *big.Int
	log                        log.Logger
}

func newCommonHandler(from, to module.Address, value, stepLimit *big.Int, log log.Logger) *CommonHandler {
	return &CommonHandler{
		from: from, to: to, value: value, stepLimit: stepLimit,
		stepUsed: big.NewInt(0),
		log:      log}
}

func (h *CommonHandler) StepLimit() *big.Int {
	return h.stepLimit
}

func (h *CommonHandler) StepUsed() *big.Int {
	return h.stepUsed
}

func (h *CommonHandler) ApplySteps(wc state.WorldContext, stepType state.StepType, n int) bool {
	return h.DeductSteps(big.NewInt(wc.StepsFor(stepType, n)))
}

func (h *CommonHandler) DeductSteps(steps *big.Int) bool {
	h.stepUsed.Add(h.stepUsed, steps)
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
