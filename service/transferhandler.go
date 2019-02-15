package service

import (
	"math/big"

	"github.com/icon-project/goloop/common"

	"github.com/icon-project/goloop/common/codec"

	"github.com/icon-project/goloop/module"
)

type TransferHandler struct {
	*CommonHandler
}

func newTransferHandler(from, to module.Address, value, stepLimit *big.Int) *TransferHandler {
	return &TransferHandler{
		newCommonHandler(from, to, value, stepLimit),
	}
}

func (h *TransferHandler) ExecuteSync(ctx Context) (module.Status, *big.Int, *codec.TypedObj, module.Address) {
	as1 := ctx.GetAccountState(h.from.ID())
	bal1 := as1.GetBalance()
	if bal1.Cmp(h.value) < 0 {
		msg, _ := common.EncodeAny(string(module.StatusOutOfBalance))
		return module.StatusOutOfBalance, h.stepLimit, msg, nil
	}
	bal1.Sub(bal1, h.value)
	as1.SetBalance(bal1)

	as2 := ctx.GetAccountState(h.to.ID())
	bal2 := as2.GetBalance()
	bal2.Add(bal2, h.value)
	as2.SetBalance(bal2)

	return module.StatusSuccess, h.stepUsed, nil, nil
}

type TransferAndMessageHandler struct {
	*TransferHandler
	data []byte
}
