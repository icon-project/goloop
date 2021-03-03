package contract

import (
	"encoding/json"

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
	patch *Patch
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
		h.Log.Warn("RoundLimitFactor is not enabled")
		return false
	}
	vs := cc.GetValidatorState()
	round := RoundLimitFactorToRound(vs.Len(), f)
	nid := scoredb.NewVarDB(as, state.VarNetwork).Int64()
	if err := p.Verify(vs.GetSnapshot(), round, int(nid)); err != nil {
		h.Log.Warnf("FailToVerifySkipTxPatch(err=%v)", err)
		return false
	}
	return true
}

func (h *patchHandler) handleSkipTransaction(cc CallContext) error {
	decode := cc.PatchDecoder()
	if decode == nil {
		h.Log.Warn("PatchHandler: patch decoder isn't set")
		return scoreresult.InvalidParameterError.New("PatchDecoderIsNil")
	}
	pd, err := decode(h.patch.Type, h.patch.Data)
	if err != nil {
		h.Log.Warnf("PatchHandler: decode fail err=%+v", err)
		return scoreresult.InvalidParameterError.Wrap(err, "DecodeFail")
	}
	p := pd.(module.SkipTransactionPatch)
	if cc.BlockHeight() != p.Height() || p.Height() < 1 {
		h.Log.Warnf("PatchHandler: invalid height block.height=%d patch.height=%d",
			cc.BlockHeight(), p.Height())
		return scoreresult.InvalidParameterError.Errorf("InvalidHeight(bh=%d,ph=%d)",
			cc.BlockHeight(), p.Height())
	}
	if !h.verifySkipTransactionPatch(cc, p) {
		return scoreresult.InvalidParameterError.New("VerifySkipTransactionPatchFail")
	}
	cc.EnableSkipTransaction()
	h.Log.Warnf("PatchHandler: SKIP TRANSACTION height=%d", p.Height())
	return nil
}

func (h *patchHandler) ExecuteSync(cc CallContext) (error, *codec.TypedObj, module.Address) {
	vs := cc.GetValidatorState()
	if idx := vs.IndexOf(h.From); idx < 0 {
		h.Log.Warnf("PatchHandler: %s isn't validator", h.From)
		return scoreresult.AccessDeniedError.Errorf("InvalidProposer(%s)", h.From), nil, nil
	}
	if h.Value != nil && h.Value.Sign() == 1 {
		return scoreresult.InvalidParameterError.New("ValueMustBeZero"), nil, nil
	}
	if !h.To.Equal(state.SystemAddress) {
		return scoreresult.InvalidParameterError.Errorf("TargetInNotSystem(target=%s)", h.To.String()), nil, nil
	}
	switch h.patch.Type {
	case module.PatchTypeSkipTransaction:
		s := h.handleSkipTransaction(cc)
		return s, nil, nil
	default:
		return scoreresult.InvalidParameterError.Errorf("InvalidDataType(%s)", h.patch.Type), nil, nil
	}
}

func newPatchHandler(ch *CommonHandler, data []byte) (ContractHandler, error) {
	patch, err := ParsePatchData(data)
	if err != nil {
		return nil, err
	}
	handler := &patchHandler{
		CommonHandler: ch,
		patch:         patch,
	}
	return handler, nil
}

func ParsePatchData(data []byte) (*Patch, error) {
	p := new(Patch)
	if err := json.Unmarshal(data, p); err != nil {
		return nil, scoreresult.InvalidParameterError.Wrapf(err,
			"InvalidJSON(json=%s)", data)
	}
	switch p.Type {
	case module.PatchTypeSkipTransaction:
		// do nothing
	default:
		return nil, scoreresult.InvalidParameterError.Errorf(
			"UnknownPatchType(%s)", p.Type)
	}
	return p, nil
}
