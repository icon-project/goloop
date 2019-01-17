package v2

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"reflect"
	"strconv"

	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	client "github.com/ybbus/jsonrpc"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
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

	// sendTransaction Call
	tx, _ := params.MarshalJSON()
	txHash, err := h.sm.SendTransaction(tx)
	if err != nil {
		log.Printf("Fail on sm.SendTransaction err=%+v", err)
		return nil, jsonrpc.ErrInternal()
	}

	result := &sendTranscationResult{
		ResponseCode:    0,
		TransactionHash: fmt.Sprintf("%x", txHash),
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
type getBalanceHandler struct {
	bm module.BlockManager
	sm module.ServiceManager
}

func (h getBalanceHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param getBalanceParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	var result getBalanceResult

	if jsonRpcV2 == 0 {
		err := rpcClient.CallFor(&result, getBalance, param)
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
	} else {
		var addr common.Address
		if err := addr.SetString(param.Address); err != nil {
			return nil, jsonrpc.ErrInvalidParams()
		}
		block, err := h.bm.GetLastBlock()
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
		var balance common.HexInt
		balance.Set(h.sm.GetBalance(block.Result(), &addr))
		result.Response = balance.String()
		result.ResponseCode = 0
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

	// var result blockV2
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
			txList := jsonMap.(map[string]interface{})["confirmed_transaction_list"].(module.TransactionList)
			err = addConfirmedTxList(txList, &result)
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

	// var result blockV2
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
			txList := jsonMap.(map[string]interface{})["confirmed_transaction_list"].(module.TransactionList)
			err = addConfirmedTxList(txList, &result)
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

	// var result blockV2
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
			txList := jsonMap.(map[string]interface{})["confirmed_transaction_list"].(module.TransactionList)
			err = addConfirmedTxList(txList, &result)
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
