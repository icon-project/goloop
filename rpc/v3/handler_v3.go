package v3

import (
	"context"

	"github.com/asaskevich/govalidator"
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
)

// GetLastBlock
type GetLastBlockHandler struct{}

func (h GetLastBlockHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	return EchoResult{
		Method: "Call : " + GetLastBlock,
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

// GetBlockByHash
type GetBlockByHashHandler struct{}

func (h GetBlockByHashHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var p GetBlockByHashParam
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
		Method: "Call : " + GetBlockByHash + " (" + p.BlockHash + ")",
	}, nil
}

// Call
type CallHandler struct{}

func (h CallHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	return EchoResult{
		Method: "Call : " + Call,
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

// GetScoreApi
type GetScoreApiHandler struct{}

func (h GetScoreApiHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var p GetScoreApiParam
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	return EchoResult{
		Method: "Call : " + GetScoreApi + "(" + p.Address + ")",
	}, nil
}

// GetTotalSupply
type GetTotalSupplyeHandler struct{}

func (h GetTotalSupplyeHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	return "Call : " + GetTotalSupply, nil
}

// GetTransactionResult
type GetTransactionResultHandler struct{}

func (h GetTransactionResultHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var p TransactionHashParam
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	return EchoResult{
		Method: "Call : " + GetTransactionResult + " (" + p.TransactionHash + ")",
	}, nil
}

// GetTransactionByHash
type GetTransactionByHashHandler struct{}

func (h GetTransactionByHashHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var p TransactionHashParam
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	return EchoResult{
		Method: "Call : " + GetTransactionByHash + " (" + p.TransactionHash + ")",
	}, nil
}

// SendTransaction
type SendTransactionHandler struct{}

func (h SendTransactionHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	return EchoResult{
		Method: "Call : " + SendTransaction,
	}, nil
}

// GetStatus
type GetStatusHandler struct{}

func (h GetStatusHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var p GetStatusParam
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	return EchoResult{
		Method: "Call : " + GetStatus,
	}, nil
}
