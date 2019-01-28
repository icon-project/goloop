package service

import (
	"math/big"

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

func NewTransactionHandler(wc WorldContext, from, to module.Address,
	value, stepLimit *big.Int, dataType *string, data []byte,
) (TransactionHandler, error) {
	th := &transactionHandler{
		from:      from,
		to:        to,
		value:     value,
		stepLimit: stepLimit,
		dataType:  dataType,
		data:      data,
	}
	ctype := ctypeNone // invalid contract type
	if dataType == nil {
		if wc.GetAccountState(to.ID()).IsContract() {
			ctype = ctypeTransferAndCall
		} else {
			ctype = ctypeTransfer
		}
	} else {
		switch *dataType {
		case dataTypeMessage:
			if wc.GetAccountState(to.ID()).IsContract() {
				ctype = ctypeTransferAndCall
			} else {
				ctype = ctypeTransfer
			}
		case dataTypeDeploy:
			ctype = ctypeDeploy
		case dataTypeCall:
			if value != nil && value.Sign() == 1 { //value > 0
				ctype = ctypeTransferAndCall
			} else {
				ctype = ctypeCall
			}
		default:
			return nil, errors.Errorf("IllegalDataType(type=%s)", *dataType)
		}
	}

	th.receipt = NewReceipt(to)
	th.cc = newCallContext(th.receipt)
	th.handler = wc.ContractManager().GetHandler(th.cc, from, to, value, stepLimit, ctype, data)
	if th.handler == nil {
		return nil, errors.New("NoSuitableHandler")
	}
	return th, nil
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
	status = module.StatusSuccess

	cnt, err := countBytesOfData(th.data)
	if err != nil {
		status = module.StatusSystemError
		stepUsed = th.stepLimit
	} else {
		if !th.handler.ApplySteps(wc, StepTypeDefault, 1) ||
			!th.handler.ApplySteps(wc, StepTypeInput, cnt) {
			status = module.StatusOutOfStep
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
	}

	// Try to charge fee
	stepPrice := wc.StepPrice()
	fee := big.NewInt(0).Mul(stepUsed, stepPrice)

	as := wc.GetAccountState(th.from.ID())
	bal := as.GetBalance()
	for bal.Cmp(fee) < 0 {
		if status == module.StatusSuccess {
			// rollback all changes
			status = module.StatusOutOfBalance
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

func (th *transactionHandler) Dispose() {
	th.cc.Dispose()
}
