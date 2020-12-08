package transaction

import (
	"bytes"
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/trace"
	"github.com/icon-project/goloop/service/txresult"
)

type Handler interface {
	Prepare(ctx contract.Context) (state.WorldContext, error)
	Execute(ctx contract.Context, estimate bool) (txresult.Receipt, error)
	Dispose()
}

type transactionHandler struct {
	from      module.Address
	to        module.Address
	value     *big.Int
	stepLimit *big.Int
	data      []byte

	chandler contract.ContractHandler

	// Assigned at Execute()
	cc contract.CallContext
}

func NewHandler(cm contract.ContractManager, from, to module.Address,
	value, stepLimit *big.Int, dataType *string, data []byte,
) (Handler, error) {
	th := &transactionHandler{
		from:      from,
		to:        to,
		value:     value,
		stepLimit: stepLimit,
		data:      data,
	}
	ctype := contract.CTypeNone // invalid contract type
	if dataType == nil {
		ctype = contract.CTypeTransfer
	} else {
		switch *dataType {
		case contract.DataTypeMessage:
			ctype = contract.CTypeTransfer
		case contract.DataTypeDeploy:
			ctype = contract.CTypeDeploy
		case contract.DataTypeCall:
			ctype = contract.CTypeCall
		case contract.DataTypePatch:
			ctype = contract.CTypePatch
		case contract.DataTypeDeposit:
			ctype = contract.CTypeDeposit
		default:
			return nil, InvalidFormat.Errorf("IllegalDataType(type=%s)", *dataType)
		}
	}

	if handler, err := cm.GetHandler(from, to, value, ctype, data); err != nil {
		return nil, errors.InvalidStateError.Wrap(err, "NoSuitableHandler")
	} else {
		th.chandler = handler
	}
	return th, nil
}

func (th *transactionHandler) Prepare(ctx contract.Context) (state.WorldContext, error) {
	return th.chandler.Prepare(ctx)
}

func (th *transactionHandler) Execute(ctx contract.Context, estimate bool) (txresult.Receipt, error) {
	// Make a copy of initial state
	wcs := ctx.GetSnapshot()

	limit := th.stepLimit
	if invokeLimit := ctx.GetStepLimit(state.StepLimitTypeInvoke); estimate || limit.Cmp(invokeLimit) > 0 {
		limit = invokeLimit
	}

	// Set up
	cc := contract.NewCallContext(ctx, limit, false)
	th.cc = cc
	logger := trace.LoggerOf(cc.Logger())
	th.chandler.ResetLogger(logger)

	logger.TSystemf("TRANSACTION start to=%s from=%s", th.to, th.from)

	// Calculate common steps
	var status error
	var addr module.Address

	if !cc.ApplySteps(state.StepTypeDefault, 1) {
		status = scoreresult.ErrOutOfStep
	} else {
		cnt, err := MeasureBytesOfData(ctx.Revision(), th.data)
		if err != nil {
			return nil, err
		} else {
			if !cc.ApplySteps(state.StepTypeInput, cnt) {
				status = scoreresult.ErrOutOfStep
			}

			// Execute
			if status == nil {
				var used *big.Int
				status, used, _, addr = cc.Call(th.chandler, cc.StepAvailable())
				cc.DeductSteps(used)

				// If it fails for system failure, then it needs to re-run this.
				if code := errors.CodeOf(status); code == errors.ExecutionFailError ||
					errors.IsCriticalCode(code) {
					return nil, status
				} else if code == scoreresult.TimeoutError {
					// it consumes all steps if it meets timeout.
					cc.DeductSteps(cc.StepAvailable())
				}
			}
		}
	}

	// Try to charge fee
	stepPrice := ctx.StepPrice()
	stepUsed := cc.StepUsed()
	minSteps := big.NewInt(cc.StepsFor(state.StepTypeDefault, 1))
	if stepUsed.Cmp(minSteps) == -1 {
		old := stepUsed
		stepUsed = minSteps
		logger.TSystemf("STEP reset value=%d old=%d msg=%q",
			minSteps, old, "sustain minimum")
	}

	stepAll := stepUsed
	var redeemed *big.Int
	if cc.FeeSharingEnabled() && stepPrice.Sign() > 0 {
		var err error
		redeemed, err = cc.RedeemSteps(stepUsed)
		if err != nil {
			logger.TSystemf("TRANSACTION failed on RedeemSteps")
			return nil, err
		} else if redeemed != nil {
			stepUsed = new(big.Int).Sub(stepUsed, redeemed)
			logger.TSystemf("STEP redeemed value=%d redeemed=%d old=%d",
				stepUsed, redeemed, stepAll)
		}
	}
	fee := new(big.Int).Mul(stepUsed, stepPrice)

	as := ctx.GetAccountState(th.from.ID())
	bal := as.GetBalance()
	for bal.Cmp(fee) < 0 {
		if status == nil {
			// rollback all changes
			logger.TSystemf("TRANSACTION rollback reason=OutOfBalance balance=%d fee=%d", bal, fee)
			status = scoreresult.ErrOutOfBalance
			ctx.Reset(wcs)
			bal = as.GetBalance()
			if redeemed != nil {
				cc.ClearRedeemLogs()
				logger.TSystemf("STEP rollback value=%d", stepAll)
				stepUsed = stepAll
			}
			fee.Mul(stepUsed, stepPrice)
		} else {
			if redeemed != nil {
				ctx.Reset(wcs)
				bal = as.GetBalance()
				cc.ClearRedeemLogs()
				logger.TSystemf("STEP rollback value=%d", stepAll)
				stepUsed = stepAll
			}
			status = scoreresult.ErrOutOfBalance
			logger.TSystemf("TRANSACTION setprice price=0 reason=OutOfBalance balance=%d fee=%d", bal, fee)
			stepPrice = new(big.Int)
			fee.SetInt64(0)
		}
	}
	logger.TSystemf("TRANSACTION charge fee=%d steps=%d price=%d", fee, stepUsed, stepPrice)
	as.SetBalance(new(big.Int).Sub(bal, fee))

	// Make a receipt
	receipt := txresult.NewReceipt(ctx.Database(), ctx.Revision(), th.to)
	s, _ := scoreresult.StatusOf(status)
	if status == nil {
		cc.GetEventLogs(receipt)
	}
	if redeemed := cc.GetRedeemLogs(receipt); redeemed && stepUsed.Sign() != 0 {
		receipt.AddPayment(th.from, stepUsed)
	}
	receipt.SetResult(s, stepAll, stepPrice, addr)
	receipt.SetReason(status)

	logger.TSystemf("TRANSACTION done status=%s steps=%s price=%s", s, stepAll, stepPrice)

	return receipt, nil
}

func (th *transactionHandler) Dispose() {
	// Actually it is called after calling Execute(), so cc can't be nil.
	if th.cc != nil {
		th.cc.Dispose()
	}
}

func MeasureBytesOfData(rev module.Revision, data []byte) (int, error) {
	if data == nil {
		return 0, nil
	}

	if rev.InputCostingWithJSON() {
		return countBytesOfData(data)
	} else {
		var idata interface{}
		if err := json.Unmarshal(data, &idata); err != nil {
			return 0, scoreresult.InvalidParameterError.Wrap(err, "InvalidDataField")
		} else {
			return countBytesOfDataValue(idata), nil
		}
	}
}

func countBytesOfData(data []byte) (int, error) {
	if data == nil {
		return 0, nil
	}
	b := bytes.NewBuffer(nil)
	if err := json.Compact(b, data); err != nil {
		return 0, scoreresult.InvalidParameterError.Wrap(err, "InvalidDataField")
	}
	return b.Len(), nil
}

func countBytesOfDataValue(v interface{}) int {
	switch o := v.(type) {
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
			count += countBytesOfDataValue(i)
		}
		return count
	case map[string]interface{}:
		var count int
		for _, i := range o {
			count += countBytesOfDataValue(i)
		}
		return count
	case bool:
		return 1
	case float64:
		return len(intconv.Int64ToBytes(int64(o)))
	default:
		return 0
	}
}
