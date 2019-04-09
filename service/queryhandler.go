package service

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/pkg/errors"
)

type QueryHandler struct {
	to   module.Address
	data []byte
}

func (qh *QueryHandler) Query(ctx contract.Context) (module.Status, interface{}) {
	// check if function is read-only
	jso, err := transaction.ParseCallData(qh.data)
	if err != nil {
		return module.StatusMethodNotFound, err.Error()
	}
	as := ctx.GetAccountSnapshot(qh.to.ID())
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
	cc := contract.NewCallContext(ctx, nil, true)
	handler := ctx.ContractManager().GetHandler(nil, qh.to,
		big.NewInt(0), ctx.GetStepLimit(transaction.LimitTypeCall), contract.CTypeCall, qh.data)

	// Execute
	status, _, result, _ := cc.Call(handler)
	cc.Dispose()
	msg, _ := common.DecodeAny(result)
	return status, msg
}

func NewQueryHandler(cm contract.ContractManager, to module.Address, dataType *string, data []byte) (*QueryHandler, error) {
	if *dataType != transaction.DataTypeCall {
		return nil, errors.Errorf("IllegalDataType(type=%s)", *dataType)
	}

	qh := &QueryHandler{
		to:   to,
		data: data,
	}
	return qh, nil
}
