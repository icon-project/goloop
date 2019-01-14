package service

import (
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/common"

	"github.com/go-errors/errors"

	"github.com/icon-project/goloop/module"
)

const (
	dataTypeMessage = "message"
	dataTypeCall    = "call"
	dataTypeDeploy  = "deploy"
)

type TransactionHandler interface {
	Prepare(wc WorldContext) (WorldContext, error)
	Execute(wc WorldContext) (Receipt, error)
	Dispose()
}

type transactionHandler struct {
	from      module.Address
	to        module.Address
	value     *big.Int
	stepLimit *big.Int
	dataType  *string
	data      []byte

	handler ContractHandler
	cc      CallContext
	receipt Receipt
}

func NewTransactionHandler(cm ContractManager, from, to module.Address,
	value, stepLimit *big.Int, dataType *string, data []byte,
) (TransactionHandler, error) {
	tc := &transactionHandler{
		from:      from,
		to:        to,
		value:     value,
		stepLimit: stepLimit,
		dataType:  dataType,
		data:      data,
	}
	ctype := ctypeNone // invalid contract type
	if dataType == nil {
		ctype = ctypeTransfer
	} else {
		switch *dataType {
		case dataTypeMessage:
			ctype = ctypeTransferAndMessage
		case dataTypeDeploy:
			ctype = ctypeTransferAndDeploy
		case dataTypeCall:
			ctype = ctypeTransferAndCall
		default:
			return nil, errors.Errorf("IllegalDataType(type=%s)", *dataType)
		}
	}

	tc.receipt = NewReceipt(to)
	tc.cc = newCallContext(tc.receipt)
	tc.handler = cm.GetHandler(tc.cc, from, to, value, stepLimit, ctype, data)
	if tc.handler == nil {
		return nil, errors.New("NoSuitableHandler")
	}
	return tc, nil
}

func (th *transactionHandler) Prepare(wc WorldContext) (WorldContext, error) {
	return th.handler.Prepare(wc)
}

func (th *transactionHandler) Execute(wc WorldContext) (Receipt, error) {
	// Make a copy of initial state
	wcs := wc.GetSnapshot()

	// Set up
	th.cc.Setup(wc)

	// Calculate common steps
	var status module.Status
	var stepUsed *big.Int
	var addr module.Address
	var iData interface{}
	if err := json.Unmarshal(th.data, &iData); err == nil {
		status = module.StatusSuccess

		if !th.handler.ApplySteps(wc, StepTypeDefault, 1) ||
			!th.handler.ApplySteps(wc, StepTypeInput, th.countBytesOfData(iData)) {
			status = module.StatusNotPayable
			stepUsed = th.handler.StepLimit()
		}

		// Execute
		if status == module.StatusSuccess {
			status, stepUsed, _, addr = th.cc.Call(th.handler)

			// If it's not successful, roll back the state.
			if status != module.StatusSuccess {
				// In case of timeout, returned stepUsed may not be same as stepLimit.
				// So set it again.
				stepUsed.Set(th.stepLimit)
				wc.Reset(wcs)
			}
		}
	} else {
		status = module.StatusSystemError
		stepUsed = th.stepLimit
	}

	// Try to charge fee
	stepPrice := wc.StepPrice()
	fee := big.NewInt(0).Mul(stepUsed, stepPrice)

	as := wc.GetAccountState(th.from.ID())
	bal := as.GetBalance()
	for bal.Cmp(fee) < 0 {
		if status == module.StatusSuccess {
			// rollback all changes
			status = module.StatusNotPayable
			wc.Reset(wcs)
			bal = as.GetBalance()

			stepUsed.Set(th.stepLimit)
			fee.Mul(stepUsed, stepPrice)
		} else {
			stepPrice.SetInt64(0)
			fee.SetInt64(0)
		}
	}
	bal.Sub(bal, fee)
	as.SetBalance(bal)

	// Make a receipt
	th.receipt.SetResult(status, stepUsed, stepPrice, addr)

	return th.receipt, nil
}

func (h *transactionHandler) countBytesOfData(data interface{}) int {
	switch o := data.(type) {
	case string:
		if len(o) > 2 && o[:2] == "0x" {
			o = o[2:]
		}
		bs := []byte(o)
		for _, b := range bs {
			if (b < '0' || b > '9') && (b < 'a' || b > 'f') {
				return len(bs)
			}
		}
		return (len(bs) + 1) / 2
	case []interface{}:
		var count int
		for _, i := range o {
			count += h.countBytesOfData(i)
		}
		return count
	case map[string]interface{}:
		var count int
		for _, i := range o {
			count += h.countBytesOfData(i)
		}
		return count
	case bool:
		return 1
	case float64:
		return len(common.Int64ToBytes(int64(o)))
	default:
		return 0
	}
}

func (th *transactionHandler) Dispose() {
	th.cc.Dispose()
}
