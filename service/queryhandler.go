package service

import (
	"math/big"

	"github.com/icon-project/goloop/common"

	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

type QueryHandler struct {
	from module.Address
	to   module.Address
	data []byte
}

func (qh *QueryHandler) Query(wc WorldContext) (module.Status, interface{}) {
	// check if function is read-only
	jso, err := ParseCallData(qh.data)
	if err != nil {
		return module.StatusMethodNotFound, err.Error()
	}
	as := wc.GetAccountSnapshot(qh.to.ID())
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
	cc := newCallContext(nil, true)
	cc.Setup(wc)
	handler := wc.ContractManager().GetHandler(cc, qh.from, qh.to,
		big.NewInt(0), wc.GetStepLimit(LimitTypeCall), ctypeCall, qh.data)

	// Execute
	status, _, result, _ := cc.Call(handler)
	cc.Dispose()
	msg, _ := common.DecodeAny(result)
	return status, msg
}

func NewQueryHandler(cm ContractManager, from, to module.Address,
	dataType *string, data []byte,
) (*QueryHandler, error) {
	if *dataType != dataTypeCall {
		return nil, errors.Errorf("IllegalDataType(type=%s)", *dataType)
	}

	qh := &QueryHandler{
		from: from,
		to:   to,
		data: data,
	}
	return qh, nil
}
