package contract

import (
	"math/big"

	"github.com/icon-project/goloop/common/codec"
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
		h.from, h.to, h.value)
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
	as1 := cc.GetAccountState(h.from.ID())
	if as1.IsContract() != h.from.IsContract() {
		return scoreresult.InvalidParameterError.Errorf(
			"InvalidAddress(%s)", h.from.String()), nil, nil
	}
	bal1 := as1.GetBalance()
	if bal1.Cmp(h.value) < 0 {
		return scoreresult.ErrOutOfBalance, nil, nil
	}
	as1.SetBalance(new(big.Int).Sub(bal1, h.value))

	as2 := cc.GetAccountState(h.to.ID())
	if as2.IsContract() != h.to.IsContract() {
		return scoreresult.InvalidParameterError.Errorf(
			"InvalidAddress(%s)", h.to.String()), nil, nil
	}
	bal2 := as2.GetBalance()
	as2.SetBalance(new(big.Int).Add(bal2, h.value))

	if h.from.IsContract() {
		indexed := make([][]byte, 4, 4)
		indexed[0] = []byte(txresult.EventLogICXTransfer)
		indexed[1] = h.from.Bytes()
		indexed[2] = h.to.Bytes()
		indexed[3] = h.value.Bytes()
		cc.OnEvent(h.from, indexed, make([][]byte, 0))
	}

	return nil, nil, nil
}

type TransferAndMessageHandler struct {
	*TransferHandler
	data []byte
}
