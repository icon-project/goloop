package contract

import (
	"math/big"

	"github.com/icon-project/goloop/service/txresult"

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

func (h *TransferHandler) ExecuteSync(cc CallContext) (module.Status, *big.Int, *codec.TypedObj, module.Address) {
	as1 := cc.GetAccountState(h.from.ID())
	bal1 := as1.GetBalance()
	if bal1.Cmp(h.value) < 0 {
		msg, _ := common.EncodeAny(string(module.StatusOutOfBalance))
		return module.StatusOutOfBalance, h.stepLimit, msg, nil
	}
	bal1.Sub(bal1, h.value)
	as1.SetBalance(bal1)

	as2 := cc.GetAccountState(h.to.ID())
	bal2 := as2.GetBalance()
	bal2.Add(bal2, h.value)
	as2.SetBalance(bal2)

	if h.from.IsContract() {
		indexed := make([][]byte, 4, 4)
		indexed[0] = []byte(txresult.EventLogICXTransfer)
		indexed[1] = h.from.Bytes()
		indexed[2] = h.to.Bytes()
		indexed[3] = h.value.Bytes()
		cc.OnEvent(h.from, indexed, make([][]byte, 0))
	}

	return module.StatusSuccess, h.stepUsed, nil, nil
}

type TransferAndMessageHandler struct {
	*TransferHandler
	data []byte
}
