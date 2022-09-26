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
	"github.com/icon-project/goloop/service/txresult"
)

type Handler interface {
	Prepare(ctx contract.Context) (state.WorldContext, error)
	// Execute executes transaction in the Handler.
	// wcs is a snapshot of the current state.
	// estimate would be true if it's executed for estimating steps.
	Execute(ctx contract.Context, wcs state.WorldSnapshot, estimate bool) (txresult.Receipt, error)
	Dispose()
}

type transactionHandler struct {
	group     module.TransactionGroup
	from      module.Address
	to        module.Address
	value     *big.Int
	stepLimit *big.Int
	dataType  *string
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
		dataType:  dataType,
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
	if th.to.IsContract() && contract.IsCallableDataType(th.dataType) {
		as2 := cc.GetAccountState(th.to.ID())
		if !as2.CheckDeposit(cc) {
			// ICON throws InvalidRequestError and it's mapped to IllegalFormatError.
			// This may be changed to proper one, but now it throws same error.
			return scoreresult.IllegalFormatError.New("EmptyDeposit")
		}
	}
	return nil
}

var blockedAccount = map[string]bool {
	"hx76dcc464a27d74ca7798dd789d2e1da8193219b4": true,
	"hxac5c6e6f7a6e8ae1baba5f0cb512f7596b95f1fe": true,
	"hx966f5f9e2ab5b80a0f2125378e85d17a661352f4": true,
	"hxad2bc6446ee3ae23228889d21f1871ed182ca2ca": true,
	"hxc39a4c8438abbcb6b49de4691f07ee9b24968a1b": true,
	"hx96505aac67c4f9033e4bac47397d760f121bcc44": true,
	"hxf5bbebeb7a7d37d2aee5d93a8459e182cbeb725d": true,
	"hx4602589eb91cf99b27296e5bd712387a23dd8ce5": true,
	"hxa67e30ec59e73b9e15c7f2c4ddc42a13b44b2097": true,
	"hx985cf67b563fb908543385da806f297482f517b4": true,
	"hxc0567bbcba511b84012103a2360825fddcd058ab": true,
	"hx52c32d0b82f46596f697d8ba2afb39105f3a6360": true,
	"hx20be21b8afbbc0ba46f0671508cfe797c7bb91be": true,
	"hx19e551eae80f9b9dcfed1554192c91c96a9c71d1": true,
	"hx0607341382dee5e039a87562dcb966e71881f336": true,
	"hxdea6fe8d6811ec28db095b97762fdd78b48c291f": true,
	"hxaf3a561e3888a2b497941e464f82fd4456db3ebf": true,
	"hx061b01c59bd9fc1282e7494ff03d75d0e7187f47": true,
	"hx10d12d5726f50e4cf92c5fad090637b403516a41": true,
	"hx10e8a7289c3989eac07828a840905344d8ed559b": true,
}

func (th *transactionHandler) checkBlocked(cc contract.CallContext) error {
	as := cc.GetAccountState(th.from.ID())
	if as.IsBlocked() {
		return scoreresult.AccessDeniedError.Errorf("BlockedAccount(addr=%s)", th.from.String())
	}
	if cc.ChainID() == ICONMainNetCID {
		if blocked, _ := blockedAccount[th.from.String()]; blocked {
			return scoreresult.AccessDeniedError.Errorf("ICONBlockedAccount(addr=%s)", th.from.String())
		}
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
	if !isPatch {
		if err := th.checkBlocked(cc); err != nil {
			return err, nil, nil
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

func (th *transactionHandler) Execute(ctx contract.Context, wcs state.WorldSnapshot, estimate bool) (txresult.Receipt, error) {
	isPatch := th.group == module.TransactionGroupPatch
	limit := th.stepLimit
	if invokeLimit := ctx.GetStepLimit(state.StepLimitTypeInvoke); isPatch || estimate || limit.Cmp(invokeLimit) > 0 {
		limit = invokeLimit
	}

	// Set up
	cc := contract.NewCallContext(ctx, limit, false)
	th.cc = cc
	logger := cc.FrameLogger()
	logger.TSystemf("TRANSACTION start from=%s to=%s id=%#x", th.from, th.to, th.cc.TransactionID())

	status, addr, err := th.DoExecute(cc, estimate, isPatch)
	if err != nil {
		return nil, err
	}

	isTrace := logger.TraceMode() != module.TraceModeNone
	if !estimate && !isTrace && (cc.ResultFlags()&contract.ResultForceRerun) != 0 {
		return nil, errors.CriticalRerunError.New("NeedToRerunTheTX")
	}

	// Try to charge fee
	stepPrice := ctx.StepPrice()
	stepUsed := cc.StepUsed()
	if isPatch {
		stepPrice = new(big.Int)
		logger.TSystem("TRANSACTION reset stepPrice=0 msg=\"patch tx\"")
	}
	minSteps := big.NewInt(cc.StepsFor(state.StepTypeDefault, 1))
	if stepUsed.Cmp(minSteps) == -1 {
		old := stepUsed
		stepUsed = minSteps
		logger.TSystemf("STEP reset value=%d old=%d msg=\"sustain minimum\"",
			minSteps, old)
	}

	stepToPay := stepUsed
	var redeemed *big.Int
	if cc.FeeSharingEnabled() && stepPrice.Sign() > 0 {
		var err error
		redeemed, err = cc.RedeemSteps(stepToPay)
		if err != nil {
			logger.TSystem("TRANSACTION failed on RedeemSteps")
			return nil, err
		} else if redeemed != nil {
			old := stepToPay
			stepToPay = new(big.Int).Sub(stepToPay, redeemed)
			logger.TSystemf("STEP redeemed value=%d redeemed=%d old=%d",
				stepToPay, redeemed, old)
		}
	}
	if stepPrice == nil {
		logger.Debugf("MKSONG StepPrice is NIL")
	}
	if stepToPay == nil {
		logger.Debugf("MKSONG StepToPay is NIL")
	}
	fee := new(big.Int).Mul(stepToPay, stepPrice)

	as := ctx.GetAccountState(th.from.ID())
	bal := as.GetBalance()
	for bal.Cmp(fee) < 0 {
		if cc.Revision().LegacyFeeCharge() {
			logger.TSystemf("STEP reset value=0 reason=OutOfBalance balance=%d fee=%d", bal, fee)
			if redeemed != nil {
				cc.ClearRedeemLogs()
			}
			stepToPay = new(big.Int)
			stepUsed = new(big.Int)
			fee.SetInt64(0)
			break
		}
		if status == nil {
			// rollback all changes
			logger.TSystemf("TRANSACTION rollback reason=OutOfBalance balance=%d fee=%d",
				bal, fee)
			status = scoreresult.ErrOutOfBalance
			ctx.Reset(wcs)
			bal = as.GetBalance()
			if redeemed != nil {
				cc.ClearRedeemLogs()
				logger.TSystemf("STEP rollback value=%d", stepUsed)
				stepToPay = stepUsed
			}
			fee.Mul(stepToPay, stepPrice)
		} else {
			if redeemed != nil {
				ctx.Reset(wcs)
				bal = as.GetBalance()
				cc.ClearRedeemLogs()
				logger.TSystemf("STEP rollback value=%d", stepUsed)
				stepToPay = stepUsed
			}
			status = scoreresult.ErrOutOfBalance
			logger.TSystemf("TRANSACTION setprice price=0 reason=OutOfBalance balance=%d fee=%d", bal, fee)
			stepPrice = new(big.Int)
			fee.SetInt64(0)
		}
	}
	logger.TSystemf("TRANSACTION charge fee=%d steps=%d price=%d", fee, stepToPay, stepPrice)
	as.SetBalance(new(big.Int).Sub(bal, fee))

	// Make a receipt
	receipt := txresult.NewReceipt(ctx.Database(), ctx.Revision(), th.to)
	s, _ := scoreresult.StatusOf(status)
	if status == nil {
		cc.GetEventLogs(receipt)
	}
	if redeemed := cc.GetRedeemLogs(receipt); redeemed && stepToPay.Sign() != 0 {
		receipt.AddPayment(th.from, stepToPay, stepToPay)
	}
	receipt.SetResult(s, stepUsed, stepPrice, addr)
	receipt.SetReason(status)

	logger.TSystemf("TRANSACTION done status=%s steps=%s price=%s", s, stepUsed, stepPrice)
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
