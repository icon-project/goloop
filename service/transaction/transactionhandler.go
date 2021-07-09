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
	group     module.TransactionGroup
	from      module.Address
	to        module.Address
	value     *big.Int
	stepLimit *big.Int
	data      []byte

	chandler contract.ContractHandler

	// Assigned at Execute()
	cc contract.CallContext
}

func NewHandler(cm contract.ContractManager, group module.TransactionGroup, from, to module.Address, value, stepLimit *big.Int, dataType *string, data []byte) (Handler, error) {
	th := &transactionHandler{
		group:     group,
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

func (th *transactionHandler) checkBalance(cc contract.CallContext) error {
	value := new(big.Int).Mul(cc.StepPrice(), th.stepLimit)
	if th.value != nil {
		value.Add(value, th.value)
	}
	var bal *big.Int
	if cc.Revision().LegacyBalanceCheck() {
		wcs := cc.GetProperty(contract.PropInitialSnapshot).(state.WorldSnapshot)
		if as := wcs.GetAccountSnapshot(th.from.ID()); as != nil {
			bal = as.GetBalance()
		} else {
			bal = new(big.Int)
		}
	} else {
		as := cc.GetAccountState(th.from.ID())
		bal = as.GetBalance()
	}
	if bal.Cmp(value) < 0 {
		return scoreresult.ErrOutOfBalance
	}
	as2 := cc.GetAccountState(th.to.ID())
	if !as2.CheckDeposit(cc) {
		// ICON throws InvalidRequestError and it's mapped to IllegalFormatError.
		// This may be changed to proper one, but now it throws same error.
		return scoreresult.IllegalFormatError.New("EmptyDeposit")
	}
	return nil
}

func (th *transactionHandler) DoExecute(cc contract.CallContext, estimate, isPatch bool) (
	status error,
	score module.Address,
	err error,
) {
	if !isPatch && !estimate {
		if err := th.checkBalance(cc); err != nil {
			return err, nil, nil
		}
	}

	if !cc.ApplySteps(state.StepTypeDefault, 1) {
		return scoreresult.ErrOutOfStep, nil, nil
	}

	if cnt, err := MeasureBytesOfData(cc.Revision(), th.data); err != nil {
		return nil, nil, err
	} else {
		if !cc.ApplySteps(state.StepTypeInput, cnt) {
			return scoreresult.ErrOutOfStep, nil, nil
		}
	}

	// Execute
	status, used, _, addr := cc.Call(th.chandler, cc.StepAvailable())
	cc.DeductSteps(used)

	// If it fails for system failure, then it needs to re-run this.
	if code := errors.CodeOf(status); code == errors.ExecutionFailError ||
		errors.IsCriticalCode(code) {
		return nil, nil, status
	} else if code == scoreresult.TimeoutError {
		// it consumes all steps if it meets timeout.
		cc.DeductSteps(cc.StepAvailable())
	}
	return status, addr, nil
}

func (th *transactionHandler) Execute(ctx contract.Context, estimate bool) (txresult.Receipt, error) {
	// Make a copy of initial state
	wcs := ctx.GetSnapshot()

	isPatch := th.group == module.TransactionGroupPatch
	limit := th.stepLimit
	if invokeLimit := ctx.GetStepLimit(state.StepLimitTypeInvoke); isPatch || estimate || limit.Cmp(invokeLimit) > 0 {
		limit = invokeLimit
	}

	// Set up
	cc := contract.NewCallContext(ctx, limit, false)
	fid := cc.FrameID()
	th.cc = cc
	logger := trace.LoggerOf(cc.Logger())
	logger.TSystemf("FRAME[%d] TRANSACTION start to=%s from=%s", fid, th.to, th.from)

	status, addr, err := th.DoExecute(cc, estimate, isPatch)
	if err != nil {
		return nil, err
	}

	// Try to charge fee
	stepPrice := ctx.StepPrice()
	stepUsed := cc.StepUsed()
	if isPatch {
		stepPrice = new(big.Int)
		logger.TSystemf("FRAME[%d] TRANSACTION reset stepPrice=0 msg=\"patch tx\"", fid)
	}
	minSteps := big.NewInt(cc.StepsFor(state.StepTypeDefault, 1))
	if stepUsed.Cmp(minSteps) == -1 {
		old := stepUsed
		stepUsed = minSteps
		logger.TSystemf("FRAME[%d] STEP reset value=%d old=%d msg=\"sustain minimum\"",
			fid, minSteps, old)
	}

	stepAll := stepUsed
	var redeemed *big.Int
	if cc.FeeSharingEnabled() && stepPrice.Sign() > 0 {
		var err error
		redeemed, err = cc.RedeemSteps(stepUsed)
		if err != nil {
			logger.TSystemf("FRAME[%d] TRANSACTION failed on RedeemSteps", fid)
			return nil, err
		} else if redeemed != nil {
			stepUsed = new(big.Int).Sub(stepUsed, redeemed)
			logger.TSystemf("FRAME[%d] STEP redeemed value=%d redeemed=%d old=%d",
				fid, stepUsed, redeemed, stepAll)
		}
	}
	fee := new(big.Int).Mul(stepUsed, stepPrice)

	as := ctx.GetAccountState(th.from.ID())
	bal := as.GetBalance()
	for bal.Cmp(fee) < 0 {
		if status == nil {
			// rollback all changes
			logger.TSystemf("FRAME[%d] TRANSACTION rollback reason=OutOfBalance balance=%d fee=%d",
				fid, bal, fee)
			status = scoreresult.ErrOutOfBalance
			ctx.Reset(wcs)
			bal = as.GetBalance()
			if redeemed != nil {
				cc.ClearRedeemLogs()
				logger.TSystemf("FRAME[%d] STEP rollback value=%d", fid, stepAll)
				stepUsed = stepAll
			}
			fee.Mul(stepUsed, stepPrice)
		} else {
			if redeemed != nil {
				ctx.Reset(wcs)
				bal = as.GetBalance()
				cc.ClearRedeemLogs()
				logger.TSystemf("FRAME[%d] STEP rollback value=%d", fid, stepAll)
				stepUsed = stepAll
			}
			status = scoreresult.ErrOutOfBalance
			if cc.Revision().ResetStepOnFailure() {
				logger.TSystemf("FRAME[%d] STEP reset value=0 reason=OutOfBalance balance=%d fee=%d", fid, bal, fee)
				stepUsed = new(big.Int)
				stepAll = new(big.Int)
			} else {
				logger.TSystemf("FRAME[%d] TRANSACTION setprice price=0 reason=OutOfBalance balance=%d fee=%d", fid, bal, fee)
				stepPrice = new(big.Int)
			}
			fee.SetInt64(0)
		}
	}
	logger.TSystemf("FRAME[%d] TRANSACTION charge fee=%d steps=%d price=%d", fid, fee, stepUsed, stepPrice)
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

	logger.TSystemf("FRAME[%d] TRANSACTION done status=%s steps=%s price=%s", fid, s, stepAll, stepPrice)

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

	if rev.Has(module.InputCostingWithJSON) {
		return countBytesOfCompactJSON(data)
	} else if rev.Has(module.LegacyInputJSON) {
		return countBytesOfReEncodedJSON(data)
	} else {
		var idata interface{}
		if err := json.Unmarshal(data, &idata); err != nil {
			return 0, scoreresult.InvalidParameterError.Wrap(err, "InvalidDataField")
		} else {
			return countBytesOfDataValue(idata), nil
		}
	}
}

func countBytesOfReEncodedJSON(data []byte) (int, error) {
	if data == nil {
		return 0, nil
	}
	var jso interface{}
	if err := json.Unmarshal(data, &jso); err != nil {
		return 0, scoreresult.InvalidParameterError.Wrap(err, "InvalidDataField")
	}
	buf := bytes.NewBuffer(nil)
	je := json.NewEncoder(buf)
	je.SetEscapeHTML(false)
	je.SetIndent("", "")
	_ = je.Encode(jso)
	return countBytesOfCompactJSON(buf.Bytes())
}

func countBytesOfCompactJSON(data []byte) (int, error) {
	if len(data) == 0 {
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
				return len(v.(string))
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
