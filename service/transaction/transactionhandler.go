package transaction

import (
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"

	"github.com/go-errors/errors"

	"github.com/icon-project/goloop/module"
)

const (
	DataTypeMessage = "message"
	DataTypeCall    = "call"
	DataTypeDeploy  = "deploy"
)

type TransactionHandler interface {
	Prepare(ctx contract.Context) (state.WorldContext, error)
	Execute(ctx contract.Context) (txresult.Receipt, error)
	Dispose()
}

type transactionHandler struct {
	from      module.Address
	to        module.Address
	value     *big.Int
	stepLimit *big.Int
	dataType  *string
	data      []byte

	handler contract.ContractHandler
	cc      contract.CallContext
	receipt txresult.Receipt
}

func NewTransactionHandler(cm contract.ContractManager, from, to module.Address,
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
	ctype := contract.CTypeNone // invalid contract type
	if dataType == nil {
		if th.to.IsContract() {
			ctype = contract.CTypeTransferAndCall
		} else {
			ctype = contract.CTypeTransfer
		}
	} else {
		switch *dataType {
		case DataTypeMessage:
			if th.to.IsContract() {
				ctype = contract.CTypeTransferAndCall
			} else {
				ctype = contract.CTypeTransfer
			}
		case DataTypeDeploy:
			ctype = contract.CTypeDeploy
		case DataTypeCall:
			if value != nil && value.Sign() == 1 { //value > 0
				ctype = contract.CTypeTransferAndCall
			} else {
				ctype = contract.CTypeCall
			}
		default:
			return nil, errors.Errorf("IllegalDataType(type=%s)", *dataType)
		}
	}

	th.receipt = txresult.NewReceipt(to)
	th.cc = contract.NewCallContext(th.receipt, false)
	th.handler = cm.GetHandler(th.cc, from, to, value, stepLimit, ctype, data)
	if th.handler == nil {
		return nil, errors.New("NoSuitableHandler")
	}
	return th, nil
}

func (th *transactionHandler) Prepare(ctx contract.Context) (state.WorldContext, error) {
	return th.handler.Prepare(ctx)
}

func (th *transactionHandler) Execute(ctx contract.Context) (txresult.Receipt, error) {
	// Make a copy of initial state
	wcs := ctx.GetSnapshot()

	// Set up
	th.cc.Setup(ctx)
	if th.handler.StepLimit().Cmp(ctx.GetStepLimit(LimitTypeInvoke)) > 0 {
		th.handler.ResetSteps(ctx.GetStepLimit(LimitTypeInvoke))
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
		if !th.handler.ApplySteps(ctx, state.StepTypeDefault, 1) ||
			!th.handler.ApplySteps(ctx, state.StepTypeInput, cnt) {
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
				ctx.Reset(wcs)
			}
		}
	}

	// Try to charge fee
	stepPrice := ctx.StepPrice()
	fee := big.NewInt(0).Mul(stepUsed, stepPrice)

	as := ctx.GetAccountState(th.from.ID())
	bal := as.GetBalance()
	for bal.Cmp(fee) < 0 {
		if status == module.StatusSuccess {
			// rollback all changes
			status = module.StatusOutOfBalance
			ctx.Reset(wcs)
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

func ParseCallData(data []byte) (*contract.DataCallJSON, error) {
	var jso contract.DataCallJSON
	if json.Unmarshal(data, &jso) != nil || jso.Method == "" {
		return nil, state.ErrInvalidDataValue
	} else {
		return &jso, nil
	}
}
