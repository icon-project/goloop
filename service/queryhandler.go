package service

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

type QueryHandler struct {
	to   module.Address
	data []byte

	contractHandler contract.ContractHandler
}

func (qh *QueryHandler) Query(ctx contract.Context) (interface{}, error) {
	// check if function is read-only
	jso, err := transaction.ParseCallData(qh.data)
	if err != nil {
		return nil, scoreresult.ErrMethodNotFound
	}
	as := ctx.GetAccountSnapshot(qh.to.ID())
	if as == nil {
		return nil, scoreresult.ErrContractNotFound
	}
	apiInfo := as.APIInfo()
	if apiInfo == nil {
		return nil, scoreresult.ErrContractNotFound
	} else {
		m := apiInfo.GetMethod(jso.Method)
		if m == nil {
			return nil, scoreresult.ErrMethodNotFound
		}
		if !m.IsReadOnly() {
			return nil, scoreresult.ErrMethodNotFound
		}
	}

	limit := ctx.GetStepLimit(transaction.LimitTypeCall)
	cc := contract.NewCallContext(ctx, limit, true)

	if !cc.ApplySteps(state.StepTypeDefault, 1) {
		return nil, scoreresult.OutOfStepError.New("NotEnoughSteps(Default)")
	}
	cnt, err := transaction.MeasureBytesOfData(ctx.Revision(), qh.data)
	if !cc.ApplySteps(state.StepTypeInput, cnt) {
		return nil, scoreresult.OutOfStepError.New("NotEnoughSteps(Input)")
	}

	// Execute
	status, _, result, _ := cc.Call(qh.contractHandler, cc.StepAvailable())
	cc.Dispose()
	if status != nil {
		return nil, scoreresult.Validate(status)
	}
	value, err := common.DecodeAnyForJSON(result)
	if err != nil {
		return nil, InvalidResultError.Wrap(err, "FailToDecodeOutput")
	}
	return value, nil
}

func NewQueryHandler(cm contract.ContractManager, to module.Address, data []byte) *QueryHandler {
	return &QueryHandler{
		to:   to,
		data: data,

		contractHandler: cm.GetHandler(nil, to, big.NewInt(0), contract.CTypeCall, data),
	}
}
