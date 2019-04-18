package transaction

import (
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"

	"github.com/icon-project/goloop/common/errors"
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

	chandler contract.ContractHandler
	receipt  txresult.Receipt

	// Assigned at Execute()
	cc contract.CallContext
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
			return nil, InvalidFormat.Errorf("IllegalDataType(type=%s)", *dataType)
		}
	}

	th.receipt = txresult.NewReceipt(to)
	th.chandler = cm.GetHandler(from, to, value, stepLimit, ctype, data)
	if th.chandler == nil {
		return nil, errors.InvalidStateError.New("NoSuitableHandler")
	}
	return th, nil
}

func (th *transactionHandler) Prepare(ctx contract.Context) (state.WorldContext, error) {
	return th.chandler.Prepare(ctx)
}

func (th *transactionHandler) Execute(ctx contract.Context) (txresult.Receipt, error) {
	// Make a copy of initial state
	wcs := ctx.GetSnapshot()

	// Set up
	th.cc = contract.NewCallContext(ctx, th.receipt, false)
	if th.chandler.StepLimit().Cmp(ctx.GetStepLimit(LimitTypeInvoke)) > 0 {
		th.chandler.ResetSteps(ctx.GetStepLimit(LimitTypeInvoke))
	}

	// Calculate common steps
	var status module.Status
	var addr module.Address
	status = module.StatusSuccess

	if !th.chandler.ApplySteps(ctx, state.StepTypeDefault, 1) {
		status = module.StatusOutOfStep
	} else {
		cnt, err := countBytesOfData(th.data)
		if err != nil {
			status = module.StatusSystemError
		} else {
			if !th.chandler.ApplySteps(ctx, state.StepTypeInput, cnt) {
				status = module.StatusOutOfStep
			}

			// Execute
			if status == module.StatusSuccess {
				status, _, _, addr = th.cc.Call(th.chandler)

				// If it's not successful, roll back the state.
				if status != module.StatusSuccess {
					ctx.Reset(wcs)
				}
			}
		}
	}

	// Try to charge fee
	stepPrice := ctx.StepPrice()
	fee := big.NewInt(0).Mul(th.chandler.StepUsed(), stepPrice)

	as := ctx.GetAccountState(th.from.ID())
	bal := as.GetBalance()
	for bal.Cmp(fee) < 0 {
		if status == module.StatusSuccess {
			// rollback all changes
			status = module.StatusOutOfBalance
			ctx.Reset(wcs)
			bal = as.GetBalance()

			fee.Mul(th.chandler.StepUsed(), stepPrice)
		} else {
			stepPrice.SetInt64(0)
			fee.SetInt64(0)
		}
	}
	bal.Sub(bal, fee)
	as.SetBalance(bal)

	// Make a receipt
	th.receipt.SetResult(status, th.chandler.StepUsed(), stepPrice, addr)

	return th.receipt, nil
}

func (th *transactionHandler) Dispose() {
	// Actually it is called after calling Execute(), so cc can't be nil.
	if th.cc != nil {
		th.cc.Dispose()
	}
}

func ParseCallData(data []byte) (*contract.DataCallJSON, error) {
	var jso contract.DataCallJSON
	if json.Unmarshal(data, &jso) != nil || jso.Method == "" {
		return nil, InvalidValue.Errorf("NoSpecifiedMethod(%s)", string(data))
	} else {
		return &jso, nil
	}
}
