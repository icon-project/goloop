package v3

import (
	"context"
	"encoding/hex"
	"log"
	"reflect"
	"strconv"

	"github.com/icon-project/goloop/module"
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	client "github.com/ybbus/jsonrpc"
)

// ICON TestNet v3
const apiEndPoint string = "https://testwallet.icon.foundation/api/v3"

var rpcClient = client.NewClient(apiEndPoint)

// getLastBlock
type getLastBlockHandler struct {
	bm module.BlockManager
}

func (h getLastBlockHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	//var result blockV2
	result := blockV2{}

	if jsonRpcV3 == 0 {
		err := rpcClient.CallFor(&result, getLastBlock)
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
	} else {
		block, err := h.bm.GetLastBlock()
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
		if block != nil {
			jsonMap, err := block.ToJSON(jsonRpcV3)
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

	return result, nil
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

	if jsonRpcV3 == 0 {
		err := rpcClient.CallFor(&result, getBlockByHeight, param)
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
	} else {
		height, err := strconv.ParseInt(param.BlockHeight, 0, 64)
		log.Printf("GetBlockByHeight(%d)", height)
		block, err := h.bm.GetBlockByHeight(height)
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
		if block != nil {
			jsonMap, err := block.ToJSON(jsonRpcV3)
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

	return result, nil
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

	if jsonRpcV3 == 0 {
		err := rpcClient.CallFor(&result, getBlockByHash, param)
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
	} else {
		hash, err := hex.DecodeString(param.BlockHash[2:])
		block, err := h.bm.GetBlock(hash)
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
		if block != nil {
			jsonMap, err := block.ToJSON(jsonRpcV3)
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

	return result, nil
}

// call
type callHandler struct{}

func (h callHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param callParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	// SCORE external function call
	var result interface{}
	result = "0x2961fff8ca4a62327800000"

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

	var result string

	err := rpcClient.CallFor(&result, getBalance, param)
	if err != nil {
		log.Println(err.Error())
		return nil, jsonrpc.ErrInternal()
	}

	return result, nil
}

// getScoreApi
type getScoreApiHandler struct{}

func (h getScoreApiHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param getScoreApiParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	var result []getScoreApiResult

	err := rpcClient.CallFor(&result, getScoreApi, param)
	if err != nil {
		log.Println(err.Error())
		return nil, jsonrpc.ErrInternal()
	}

	return result, nil
}

// getTotalSupply
type getTotalSupplyeHandler struct{}

func (h getTotalSupplyeHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var result string

	err := rpcClient.CallFor(&result, getTotalSupply)
	if err != nil {
		log.Println(err.Error())
		return nil, jsonrpc.ErrInternal()
	}

	return result, nil
}

// getTransactionResult
type getTransactionResultHandler struct{}

func (h getTransactionResultHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param transactionHashParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	var result transactionResult

	err := rpcClient.CallFor(&result, getTransactionResult, param)
	if err != nil {
		log.Println(err.Error())
		return nil, jsonrpc.ErrInternal()
	}

	return result, nil
}

// getTransactionByHash
type getTransactionByHashHandler struct{}

func (h getTransactionByHashHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param transactionHashParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	var result transactionV3

	err := rpcClient.CallFor(&result, getTransactionByHash, param)
	if err != nil {
		log.Println(err.Error())
		return nil, jsonrpc.ErrInternal()
	}

	return result, nil
}

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
		log.Println(err.Error())
		return nil, jsonrpc.ErrInternal()
	}
	// txHash
	log.Printf("txHash : %x", txHash)
	result := "0x4bf74e6aeeb43bde5dc8d5b62537a33ac8eb7605ebbdb51b015c1881b45b3aed"

	return result, nil
}

// getStatus
type getStatusHandler struct{}

func (h getStatusHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param getStatusParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	var result interface{}

	err := rpcClient.CallFor(&result, getStatus, param)
	if err != nil {
		log.Println(err.Error())
		return nil, jsonrpc.ErrInternal()
	}

	return result, nil
}
