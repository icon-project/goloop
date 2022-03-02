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
		SetTraceLogger(logger *trace.Logger)
		TraceLogger() *trace.Logger
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
	From, To module.Address
	Value    *big.Int
	Log      *trace.Logger

	isInterCall bool
	callCharged bool
}

func NewCommonHandler(from, to module.Address, value *big.Int, call bool, log log.Logger) *CommonHandler {
	return &CommonHandler{
		From: from, To: to, Value: value, isInterCall: call,
		Log: trace.LoggerOf(log)}
}

func (h *CommonHandler) Prepare(ctx Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{string(h.From.ID()), state.AccountWriteLock},
		{string(h.To.ID()), state.AccountWriteLock},
	}
	return ctx.GetFuture(lq), nil
}

func (h *CommonHandler) ApplyCallSteps(cc CallContext) error {
	if !h.callCharged {
		h.callCharged = true
		return cc.ApplyCallSteps()
	}
	return nil
}

func (h *CommonHandler) ApplyStepsForInterCall(cc CallContext) error {
	if h.isInterCall {
		return h.ApplyCallSteps(cc)
	}
	return nil
}

func (h *CommonHandler) SetTraceLogger(logger *trace.Logger) {
	h.Log = logger
}

func (h *CommonHandler) TraceLogger() *trace.Logger {
	return h.Log
}

func (h *CommonHandler) Logger() log.Logger {
	return h.Log
}
