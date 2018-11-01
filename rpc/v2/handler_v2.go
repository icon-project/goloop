package v2

import (
	"context"

	"github.com/asaskevich/govalidator"
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
)

// SendTransaction
type SendTransactionHandler struct{}

func (h SendTransactionHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	return EchoResult{
		Method: "Call : " + SendTransaction,
	}, nil
}

// GetTransactionResult
type GetTransactionResultHandler struct{}

func (h GetTransactionResultHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var p GetTransactionResultParam
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	return EchoResult{
		Method: "Call : " + GetTransactionResult + " (" + p.TransactionHash + ")",
	}, nil
}

// GetBalance
type GetBalanceHandler struct{}

func (h GetBalanceHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var p GetBalanceParam
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	_, err := govalidator.ValidateStruct(p)
	if err != nil {
		e := jsonrpc.ErrInvalidParams()
		e.Data = err.Error()
		return nil, e
	}

	return "Call : " + GetBalance + "(" + p.Address + ")", nil
}

// GetTotalSupply
type GetTotalSupplyeHandler struct{}

func (h GetTotalSupplyeHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	return "Call : " + GetTotalSupply, nil
}

// GetLastBlock
type GetLastBlockHandler struct{}

func (h GetLastBlockHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	return EchoResult{
		Method: "Call : " + GetLastBlock,
	}, nil
}

// GetBlockByHash
type GetBlockByHashHandler struct{}

func (h GetBlockByHashHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var p GetBlockByHashParam
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	_, err := govalidator.ValidateStruct(p)
	if err != nil {
		e := jsonrpc.ErrInvalidParams()
		e.Data = err.Error()
		return nil, e
	}

	return EchoResult{
		Method: "Call : " + GetBlockByHash + " (" + p.BlockHash + ")",
	}, nil
}

// GetBlockByHeight
type GetBlockByHeightHandler struct{}

func (h GetBlockByHeightHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var p GetBlockByHeightParam
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	_, err := validator(p)
	if err != nil {
		e := jsonrpc.ErrInvalidParams()
		e.Data = err.Error()
		return nil, e
	}

	return EchoResult{
		Method: "Call : " + GetBlockByHeight + " (" + p.BlockHeight + ")",
	}, nil
}

// GetTransactionByAddress
type GetTransactionByAddressHandler struct{}

func (h GetTransactionByAddressHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var p GetTransactionByAddressParam
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	return EchoResult{
		Method: "Call : " + GetTransactionByAddress,
	}, nil
}
