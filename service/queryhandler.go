package service

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/transaction"
)

type QueryHandler struct {
	to   module.Address
	data []byte
}

func (qh *QueryHandler) Query(ctx contract.Context) (module.Status, interface{}) {
	// check if function is read-only
	jso, err := transaction.ParseCallData(qh.data)
	if err != nil {
		return module.StatusMethodNotFound, nil
	}
	as := ctx.GetAccountSnapshot(qh.to.ID())
	if as == nil {
		return module.StatusContractNotFound, nil
	}
	apiInfo := as.APIInfo()
	if apiInfo == nil {
		return module.StatusContractNotFound, nil
	} else {
		m := apiInfo.GetMethod(jso.Method)
		if m == nil {
			return module.StatusMethodNotFound, nil
		}
		if !m.IsReadOnly() {
			return module.StatusMethodNotFound, nil
		}
	}

	// Set up
	cc := contract.NewCallContext(ctx, nil, true)
	handler := ctx.ContractManager().GetHandler(nil, qh.to,
		big.NewInt(0), ctx.GetStepLimit(transaction.LimitTypeCall), contract.CTypeCall, qh.data)

	// Execute
	status, _, result, _ := cc.Call(handler)
	cc.Dispose()
	if status != module.StatusSuccess {
		return status, nil
	}
	value, _ := common.DecodeAny(result)
	return status, value
}

func NewQueryHandler(cm contract.ContractManager, to module.Address, data []byte) *QueryHandler {
	return &QueryHandler{
		to:   to,
		data: data,
	}
}
