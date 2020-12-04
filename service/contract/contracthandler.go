package contract

import (
	"math/big"
	"time"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/trace"
)

const (
	transactionTimeLimit     = time.Duration(5 * time.Second)
	scoreDecompressTimeLimit = time.Duration(5 * time.Second)

	CTypeNone = iota
	CTypeTransfer
	CTypeDeploy
	CTypeCall
	CTypePatch
	CTypeDeposit
)

type (
	ContractHandler interface {
		Prepare(ctx Context) (state.WorldContext, error)
		ResetLogger(logger log.Logger)
	}

	SyncContractHandler interface {
		ContractHandler
		ExecuteSync(cc CallContext) (error, *codec.TypedObj, module.Address)
	}

	AsyncContractHandler interface {
		ContractHandler
		ExecuteAsync(cc CallContext) error
		SendResult(status error, steps *big.Int, result *codec.TypedObj) error
		Dispose()

		EEType() state.EEType
		eeproxy.CallContext
	}
)

type CommonHandler struct {
	from, to module.Address
	value    *big.Int
	log      *trace.Logger
	call     bool
}

func NewCommonHandler(from, to module.Address, value *big.Int, call bool, log log.Logger) *CommonHandler {
	return &CommonHandler{
		from: from, to: to, value: value, call: call,
		log: trace.LoggerOf(log)}
}

func (h *CommonHandler) Prepare(ctx Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{string(h.from.ID()), state.AccountWriteLock},
		{string(h.to.ID()), state.AccountWriteLock},
	}
	return ctx.GetFuture(lq), nil
}

func (h *CommonHandler) ApplyStepsForInterCall(cc CallContext) bool {
	if h.call {
		if !cc.ApplySteps(state.StepTypeContractCall, 1) {
			return false
		}
	}
	return true
}

func (h *CommonHandler) Logger() log.Logger {
	return h.log
}

func (h *CommonHandler) ResetLogger(logger log.Logger) {
	h.log = trace.LoggerOf(logger)
}
