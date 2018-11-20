package v2

import (
	"context"
	"log"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"

	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	client "github.com/ybbus/jsonrpc"
)

// ICON TestNet v2
const apiEndPoint string = "https://testwallet.icon.foundation/api/v2"

var rpcClient = client.NewClient(apiEndPoint)

// sendTransaction
type sendTransactionHandler struct {
	sm module.ServiceManager
}

func (h sendTransactionHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param sendTransactionParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	p, _ := params.MarshalJSON()
	log.Printf("params : %s", p)
	tx, err := service.NewTransaction(p)
	log.Printf("tx : %x", tx.Hash())
	txHash, err := h.sm.SendTransaction(tx)
	if err != nil {

	}

	result := &sendTranscationResult{
		ResponseCode:    0,
		TransactionHash: string(txHash[:]),
	}

	return result, nil
}

// getTransactionResult
type getTransactionResultHandler struct{}

func (h getTransactionResultHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param getTransactionResultParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	var result getTransactionResultResult

	err := rpcClient.CallFor(&result, getTransactionResult, param)
	if err != nil {
		log.Println(err.Error())
		return nil, jsonrpc.ErrInternal()
	}

	return result, nil
}

// getBalance
type getBalanceHandler struct{}

func (h getBalanceHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param getBalanceParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	var result getBalanceResult

	err := rpcClient.CallFor(&result, getBalance, param)
	if err != nil {
		log.Println(err.Error())
		return nil, jsonrpc.ErrInternal()
	}

	return result, nil
}

// getTotalSupply
type getTotalSupplyeHandler struct{}

func (h getTotalSupplyeHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var result getTotalSupplyResult

	err := rpcClient.CallFor(&result, getTotalSupply)
	if err != nil {
		log.Println(err.Error())
		return nil, jsonrpc.ErrInternal()
	}

	return result, nil
}

// getLastBlock
type getLastBlockHandler struct{}

func (h getLastBlockHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var result blockResult

	err := rpcClient.CallFor(&result, getLastBlock)
	if err != nil {
		log.Println(err.Error())
		return nil, jsonrpc.ErrInternal()
	}

	return result, nil
}

// getBlockByHash
type getBlockByHashHandler struct{}

func (h getBlockByHashHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param getBlockByHashParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	var result blockResult

	err := rpcClient.CallFor(&result, getBlockByHash, param)
	if err != nil {
		log.Println(err.Error())
		return nil, jsonrpc.ErrInternal()
	}

	return result, nil
}

// getBlockByHeight
type getBlockByHeightHandler struct{}

func (h getBlockByHeightHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param getBlockByHeightParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	var result blockResult

	err := rpcClient.CallFor(&result, getBlockByHeight, param)
	if err != nil {
		log.Println(err.Error())
		return nil, jsonrpc.ErrInternal()
	}

	return result, nil
}
