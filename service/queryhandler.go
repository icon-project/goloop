package service

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
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
	jso, err := contract.ParseCallData(qh.data)
	if err != nil {
		return nil, scoreresult.InvalidParameterError.Wrap(err,
			"InvalidCallData")
	}
	as := ctx.GetAccountSnapshot(qh.to.ID())
	if as == nil {
		return nil, scoreresult.ErrContractNotFound
	}
	apiInfo, err := as.APIInfo()
	if err != nil {
		return nil, err
	}
	if apiInfo == nil {
		return nil, scoreresult.ErrContractNotFound
	} else {
		m := apiInfo.GetMethod(jso.Method)
		if m == nil {
			return nil, scoreresult.ErrMethodNotFound
		}
		if !m.IsReadOnly() {
			return nil, scoreresult.ErrAccessDenied
		}
	}

	limit := ctx.GetStepLimit(state.StepLimitTypeQuery)
	cc := contract.NewCallContext(ctx, limit, true)

	if !cc.ApplySteps(state.StepTypeDefault, 1) {
		return nil, scoreresult.OutOfStepError.New("NotEnoughSteps(Default)")
	}
	cnt, err := transaction.MeasureBytesOfData(ctx.Revision(), qh.data)
	if err != nil {
		return nil, scoreresult.InvalidParameterError.Wrap(err, "InvalidCallData")
	}
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

func NewQueryHandler(cm contract.ContractManager, to module.Address, data []byte) (*QueryHandler, error) {
	handler, err := cm.GetHandler(nil, to, big.NewInt(0), contract.CTypeCall, data)
	if err != nil {
		return nil, errors.InvalidStateError.Wrap(err, "NoSuitableHandler")
	}
	return &QueryHandler{
		to:   to,
		data: data,

		contractHandler: handler,
	}, nil
}
