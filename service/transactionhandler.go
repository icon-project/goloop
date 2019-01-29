package service

import (
	"math/big"

	"github.com/go-errors/errors"
	"github.com/icon-project/goloop/common"

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
	Query(wc WorldContext) (module.Status, interface{})
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
		if th.to.IsContract() {
			ctype = ctypeTransferAndCall
		} else {
			ctype = ctypeTransfer
		}
	} else {
		switch *dataType {
		case dataTypeMessage:
			if th.to.IsContract() {
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
	th.handler = cm.GetHandler(th.cc, from, to, value, stepLimit, ctype, data)
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
	th.cc.Setup(wc, false)
	if th.handler.StepLimit().Cmp(wc.GetStepLimit(LimitTypeInvoke)) > 0 {
		th.handler.ResetSteps(wc.GetStepLimit(LimitTypeInvoke))
	}

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

func (th *transactionHandler) Query(wc WorldContext) (module.Status, interface{}) {
	// check if function is read-only
	jso, err := ParseCallData(th.data)
	if err != nil {
		return module.StatusMethodNotFound, err.Error()
	}
	as := wc.GetAccountSnapshot(th.to.ID())
	apiInfo := as.APIInfo()
	if apiInfo == nil {
		return module.StatusContractNotFound, "APIInfo() is null"
	} else {
		m := apiInfo.GetMethod(jso.Method)
		if m == nil {
			return module.StatusMethodNotFound, string(module.StatusMethodNotFound)
		}
		if m == nil || !m.IsReadOnly() {
			return module.StatusMethodNotFound, "Not a read-only API"
		}
	}

	// Set up
	th.cc.Setup(wc, true)
	if th.handler.StepLimit().Cmp(big.NewInt(0)) == 0 ||
		th.handler.StepLimit().Cmp(wc.GetStepLimit(LimitTypeCall)) > 0 {
		th.handler.ResetSteps(wc.GetStepLimit(LimitTypeCall))
	}

	// Execute
	status, _, result, _ := th.cc.Call(th.handler)
	msg, _ := common.DecodeAny(result)
	return status, msg
}
