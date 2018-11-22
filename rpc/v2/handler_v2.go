package v2

import (
	"context"
	"encoding/hex"
	"log"
	"reflect"
	"strconv"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"

	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	client "github.com/ybbus/jsonrpc"
)

// JSON RPC api v2
const jsonRpcV2 int = 2

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
type getLastBlockHandler struct {
	bm module.BlockManager
}

func (h getLastBlockHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	//var result blockV2
	result := blockV2{}

	if jsonRpcV2 == 0 {
		var blockResult blockResult
		err := rpcClient.CallFor(&blockResult, getLastBlock)
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
		return blockResult, nil
	} else {
		block, err := h.bm.GetLastBlock()
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
		if block != nil {
			jsonMap, err := block.ToJSON(jsonRpcV2)
			err = convertToResult(jsonMap, &result, reflect.TypeOf(result))
			if err != nil {
				log.Println(err.Error())
				return nil, jsonrpc.ErrInternal()
			}
		} else {
			log.Println("Block is nil")
			return nil, jsonrpc.ErrInternal()
		}
	}

	// ResponseCode ??
	return &blockResult{ResponseCode: 0, Block: result}, nil
}

// getBlockByHash
type getBlockByHashHandler struct {
	bm module.BlockManager
}

func (h getBlockByHashHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param getBlockByHashParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	//var result blockV2
	result := blockV2{}

	if jsonRpcV2 == 0 {
		var blockResult blockResult
		err := rpcClient.CallFor(&blockResult, getBlockByHash, param)
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
		return blockResult, nil
	} else {
		hash, err := hex.DecodeString(param.BlockHash[:])
		block, err := h.bm.GetBlock(hash)
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
		if block != nil {
			jsonMap, err := block.ToJSON(jsonRpcV2)
			err = convertToResult(jsonMap, &result, reflect.TypeOf(result))
			if err != nil {
				log.Println(err.Error())
				return nil, jsonrpc.ErrInternal()
			}
		} else {
			log.Println("Block is nil")
			return nil, jsonrpc.ErrInternal()
		}
	}

	// ResponseCode ??
	return &blockResult{ResponseCode: 0, Block: result}, nil
}

// getBlockByHeight
type getBlockByHeightHandler struct {
	bm module.BlockManager
}

func (h getBlockByHeightHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param getBlockByHeightParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	//var result blockV2
	result := blockV2{}

	if jsonRpcV2 == 0 {
		var blockResult blockResult
		err := rpcClient.CallFor(&blockResult, getBlockByHeight, param)
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
		return blockResult, nil
	} else {
		height, err := strconv.ParseInt(param.BlockHeight, 10, 64)
		log.Printf("GetBlockByHeight(%d)", height)
		block, err := h.bm.GetBlockByHeight(height)
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
		if block != nil {
			jsonMap, err := block.ToJSON(jsonRpcV2)
			err = convertToResult(jsonMap, &result, reflect.TypeOf(result))
			if err != nil {
				log.Println(err.Error())
				return nil, jsonrpc.ErrInternal()
			}
		} else {
			log.Println("Block is nil")
			return nil, jsonrpc.ErrInternal()
		}
	}

	// ResponseCode ??
	return &blockResult{ResponseCode: 0, Block: result}, nil
}
