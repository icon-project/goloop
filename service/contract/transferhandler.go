package contract

import (
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/txresult"
)

type TransferHandler struct {
	*CommonHandler
}

func newTransferHandler(ch *CommonHandler) *TransferHandler {
	return &TransferHandler{ch}
}

func (h *TransferHandler) ExecuteSync(cc CallContext) (err error, ro *codec.TypedObj, addr module.Address) {
	h.log.TSystemf("TRANSFER start from=%s to=%s value=%s",
		h.From, h.To, h.Value)
	defer func() {
		if err != nil {
			h.log.TSystemf("TRANSFER done status=%s msg=%v", err.Error(), err)
		}
	}()

	if !h.ApplyStepsForInterCall(cc) {
		return scoreresult.OutOfStepError.New("OutOfStepForInterCall"), nil, nil
	}
	return h.DoExecuteSync(cc)
}

func (h *TransferHandler) DoExecuteSync(cc CallContext) (err error, ro *codec.TypedObj, addr module.Address) {
	if cc.QueryMode() {
		return scoreresult.AccessDeniedError.New("TransferIsNotAllowed"), nil, nil
	}
	as1 := cc.GetAccountState(h.From.ID())
	if as1.IsContract() != h.From.IsContract() {
		return scoreresult.InvalidParameterError.Errorf(
			"InvalidAddress(%s)", h.From.String()), nil, nil
	}
	bal1 := as1.GetBalance()
	if bal1.Cmp(h.Value) < 0 {
		return scoreresult.ErrOutOfBalance, nil, nil
	}
	as1.SetBalance(new(big.Int).Sub(bal1, h.Value))

	as2 := cc.GetAccountState(h.To.ID())
	if as2.IsContract() != h.To.IsContract() {
		return scoreresult.InvalidParameterError.Errorf(
			"InvalidAddress(%s)", h.To.String()), nil, nil
	}
	bal2 := as2.GetBalance()
	as2.SetBalance(new(big.Int).Add(bal2, h.Value))

	if h.From.IsContract() {
		indexed := make([][]byte, 4, 4)
		indexed[0] = []byte(txresult.EventLogICXTransfer)
		indexed[1] = h.From.Bytes()
		indexed[2] = h.To.Bytes()
		indexed[3] = intconv.BigIntToBytes(h.Value)
		cc.OnEvent(h.From, indexed, make([][]byte, 0))
	}

	return nil, nil, nil
}

type TransferAndMessageHandler struct {
	*TransferHandler
	data []byte
}
