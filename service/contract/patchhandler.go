package contract

import (
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

type Patch struct {
	Type string `json:"type"`
	Data []byte `json:"data"`
}

type patchHandler struct {
	*CommonHandler
	data Patch
}

func (h *patchHandler) Prepare(ctx Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{state.WorldIDStr, state.AccountWriteLock},
	}
	wc := ctx.GetFuture(lq)
	wc.WorldVirtualState().Ensure()

	return wc, nil
}

func RoundLimitFactorToRound(validator int, factor int64) int64 {
	return (int64(validator)*factor + 2) / 3
}

func (h *patchHandler) verifySkipTransactionPatch(cc CallContext, p module.SkipTransactionPatch) bool {
	as := cc.GetAccountState(state.SystemID)
	f := scoredb.NewVarDB(as, state.VarRoundLimitFactor).Int64()
	if f == 0 {
		h.log.Warn("RoundLimitFactor is not enabled")
		return false
	}
	vs := cc.GetValidatorState()
	round := RoundLimitFactorToRound(vs.Len(), f)
	nid := scoredb.NewVarDB(as, state.VarNetwork).Int64()
	if err := p.Verify(vs.GetSnapshot(), round, int(nid)); err != nil {
		h.log.Warnf("FailToVerifySkipTxPatch(err=%v)", err)
		return false
	}
	return true
}

func (h *patchHandler) handleSkipTransaction(cc CallContext) error {
	decode := cc.PatchDecoder()
	if decode == nil {
		h.log.Warn("PatchHandler: patch decoder isn't set")
		return scoreresult.InvalidParameterError.New("PatchDecoderIsNil")
	}
	pd, err := decode(h.data.Type, h.data.Data)
	if err != nil {
		h.log.Warnf("PatchHandler: decode fail err=%+v", err)
		return scoreresult.InvalidParameterError.Wrap(err, "DecodeFail")
	}
	p := pd.(module.SkipTransactionPatch)
	if cc.BlockHeight() != p.Height() || p.Height() < 1 {
		h.log.Warnf("PatchHandler: invalid height block.height=%d patch.height=%d",
			cc.BlockHeight(), p.Height())
		return scoreresult.InvalidParameterError.Errorf("InvalidHeight(bh=%d,ph=%d)",
			cc.BlockHeight(), p.Height())
	}
	if !h.verifySkipTransactionPatch(cc, p) {
		return scoreresult.InvalidParameterError.New("VerifySkipTransactionPatchFail")
	}
	cc.EnableSkipTransaction()
	h.log.Warnf("PatchHandler: SKIP TRANSACTION height=%d", p.Height())
	return nil
}

func (h *patchHandler) ExecuteSync(cc CallContext) (error, *big.Int, *codec.TypedObj, module.Address) {
	vs := cc.GetValidatorState()
	if idx := vs.IndexOf(h.from); idx < 0 {
		h.log.Warnf("PatchHandler: %s isn't validator", h.from)
		return scoreresult.AccessDeniedError.Errorf("InvalidProposer(%s)", h.from), big.NewInt(0), nil, nil
	}
	switch h.data.Type {
	case module.PatchTypeSkipTransaction:
		s := h.handleSkipTransaction(cc)
		return s, big.NewInt(0), nil, nil
	default:
		return scoreresult.InvalidParameterError.Errorf("InvalidDataType(%s)", h.data.Type), big.NewInt(0), nil, nil
	}
}

func newPatchHandler(ch *CommonHandler, data []byte) ContractHandler {
	handler := &patchHandler{
		CommonHandler: ch,
	}
	err := json.Unmarshal(data, &handler.data)
	if err != nil {
		return nil
	}
	return handler
}
