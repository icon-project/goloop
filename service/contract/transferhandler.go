package contract

import (
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

type TransferHandler struct {
	*CommonHandler
}

func newTransferHandler(ch *CommonHandler) *TransferHandler {
	return &TransferHandler{ch}
}

func (h *TransferHandler) ExecuteSync(cc CallContext) (error, *codec.TypedObj, module.Address) {
	as1 := cc.GetAccountState(h.from.ID())
	if as1.IsContract() != h.from.IsContract() {
		return scoreresult.InvalidParameterError.Errorf(
			"InvalidAddress(%s)", h.from.String()), nil, nil
	}
	bal1 := as1.GetBalance()
	if bal1.Cmp(h.value) < 0 {
		return scoreresult.ErrOutOfBalance, nil, nil
	}
	bal1.Sub(bal1, h.value)
	as1.SetBalance(bal1)

	as2 := cc.GetAccountState(h.to.ID())
	if as2.IsContract() != h.to.IsContract() {
		return scoreresult.InvalidParameterError.Errorf(
			"InvalidAddress(%s)", h.to.String()), nil, nil
	}
	bal2 := as2.GetBalance()
	bal2.Add(bal2, h.value)
	as2.SetBalance(bal2)

	h.log.TSystemf("TRANSFER from=%s to=%s value=%s",
		h.from, h.to, h.value)

	if h.from.IsContract() {
		if !h.to.IsContract() && !cc.ApplySteps(state.StepTypeContractCall, 1) {
			return scoreresult.ErrOutOfStep, nil, nil
		}
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
