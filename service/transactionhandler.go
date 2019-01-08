package service

import (
	"github.com/go-errors/errors"
	"math/big"

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

	// Execute
	th.cc.Setup(wc)
	status, stepUsed, _, addr := th.cc.Call(th.handler)

	// If it's not successful, roll back the state.
	if status != module.StatusSuccess {
		// In case of timeout, returned stepUsed may not be same as stepLimit.
		// So set it again.
		stepUsed.Set(th.stepLimit)
		wc.Reset(wcs)
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
	if status == module.StatusSuccess {
		th.receipt.SetResult(status, stepUsed, stepPrice, addr)
	} else {
		th.receipt.SetResult(status, stepUsed, stepPrice, nil)
	}
	return th.receipt, nil
}

func (th *transactionHandler) Dispose() {
	th.cc.Dispose()
}
