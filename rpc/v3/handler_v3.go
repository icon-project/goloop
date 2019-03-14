package v3

import (
	"bytes"
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
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
)

// ICON TestNet v3
const apiEndPoint string = "https://testwallet.icon.foundation/api/v3"

var rpcClient = client.NewClient(apiEndPoint)

func addReason(rerr *jsonrpc.Error, err error) *jsonrpc.Error {
	log.Printf("MKSONG: fail with reason err=%+v", err)
	// rerr.Message = err.Error()
	// rerr.Data = err.Error()
	return rerr
}

// getLastBlock
type getLastBlockHandler struct {
	bm module.BlockManager
}

func (h getLastBlockHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	// _, span := trace.StartSpan(context.Background(), getLastBlock)
	// defer span.End()

	// var result blockV2
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

	// var result blockV2
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

	// var result blockV2
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
type callHandler struct {
	bm module.BlockManager
	sm module.ServiceManager
}

func (h callHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var param callParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	// SCORE external function call
	var result string

	if jsonRpcV3 == 0 {
		err := rpcClient.CallFor(&result, call, param)
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
		tx, _ := params.MarshalJSON()
		// TODO temporary block info
		s, r, err := h.sm.Call(block.Result(), tx, block)
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
		if s != module.StatusSuccess {
			msg, ok := r.(string)
			if !ok {
				msg = string(s)
			}
			return nil, &jsonrpc.Error{
				// TODO Is it correct if our error code is in application error range?
				Code:    jsonrpc.ErrorCode(-32500 - int(s)),
				Message: msg,
			}
		} else {
			return r, nil
		}
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

	var result string

	if jsonRpcV3 == 0 {
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
		result = balance.String()
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
type getTransactionResultHandler struct {
	bm module.BlockManager
}

func (h getTransactionResultHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param transactionHashParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	var result transactionResult

	if jsonRpcV3 == 0 {
		err := rpcClient.CallFor(&result, getTransactionResult, param)
		if err != nil {
			log.Println(err.Error())
			return nil, &jsonrpc.Error{
				Code:    jsonrpc.ErrorCodeInternal,
				Message: "Fail to call external",
				Data:    nil,
			}
		}
	} else {
		hash, err := hex.DecodeString(param.TransactionHash[2:])
		if err != nil {
			log.Printf("Fail on decoding txHash hash=\"%s\" err=%+v",
				param.TransactionHash, err)
			return nil, &jsonrpc.Error{
				Code:    jsonrpc.ErrorCodeInternal,
				Message: "Not a valid transaction hash",
				Data:    nil,
			}
		}
		txinfo, err := h.bm.GetTransactionInfo(hash)
		if err != nil {
			log.Printf("Fail to get transaction info hash=<%x> err=%+v",
				hash, err)
			return nil, &jsonrpc.Error{
				Code:    jsonrpc.ErrorCodeInternal,
				Message: "Transaction not found",
				Data:    nil,
			}
		}
		blk := txinfo.Block()
		rct := txinfo.GetReceipt()
		if rct == nil {
			return nil, &jsonrpc.Error{
				Code:    jsonrpc.ErrorCodeInternal,
				Message: "No receipt",
				Data:    nil,
			}
		}
		rctjson, err := rct.ToJSON(jsonRpcV3)
		if err != nil {
			return nil, &jsonrpc.Error{
				Code:    jsonrpc.ErrorCodeInternal,
				Message: err.Error(),
				Data:    nil,
			}
		}
		rctmap := rctjson.(map[string]interface{})
		rctmap["blockHash"] = "0x" + hex.EncodeToString(blk.ID())
		rctmap["blockHeight"] = "0x" + strconv.FormatInt(
			int64(blk.Height()), 16)
		rctmap["txIndex"] = "0x" + strconv.FormatInt(
			int64(txinfo.Index()), 16)
		return rctmap, nil
	}

	return result, nil
}

// getTransactionByHash
type getTransactionByHashHandler struct {
	bm module.BlockManager
}

func (h getTransactionByHashHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param transactionHashParam
	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	if jsonRpcV3 == 0 {
		var result transactionV3
		err := rpcClient.CallFor(&result, getTransactionByHash, param)
		if err != nil {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
		return result, nil
	} else {
		hash, err := hex.DecodeString(param.TransactionHash[2:])
		log.Printf("TxHash : %x", hash)
		txInfo, err := h.bm.GetTransactionInfo(hash)
		if txInfo != nil {
			tx := txInfo.Transaction()
			var txMap interface{}
			switch tx.Version() {
			case jsonRpcV2:
				// txV2 := transactionV2{}
				txMap, err = tx.ToJSON(jsonRpcV2)
				if err != nil {
					log.Println(err.Error())
				}
				// convertToResult(txMap, &txV2, reflect.TypeOf(txV2))
				return txMap, nil
			case jsonRpcV3:
				// txV3 := transactionV3{}
				txMap, err = tx.ToJSON(jsonRpcV3)
				if err != nil {
					log.Println(err.Error())
				}
				// convertToResult(txMap, &txV3, reflect.TypeOf(txV3))
				return txMap, nil
			}
		}
	}
	return nil, jsonrpc.ErrInternal()
}

// sendTransaction
type sendTransactionHandler struct {
	sm module.ServiceManager
}

func (h sendTransactionHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	var param sendTransactionParamV3

	if err := jsonrpc.Unmarshal(params, &param); err != nil {
		return nil, err
	}
	if err := validateParam(param); err != nil {
		return nil, err
	}

	// sendTransaction Call
	txo, _ := params.MarshalJSON()
	txHash, err := h.sm.SendTransaction(txo)

	if err != nil {
		if err == service.ErrTransactionPoolOverFlow {
			error := &jsonrpc.Error{
				Code:    -32101,
				Message: "TransactionPool Overflow",
			}
			return nil, error
		} else {
			log.Println(err.Error())
			return nil, jsonrpc.ErrInternal()
		}
	}

	result := fmt.Sprintf("0x%x", txHash)

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

type getDataByHashHandler struct {
	db db.Database
}

func (h getDataByHashHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var param struct {
		Hash string `json:"hash" valid:"t_hash,required"`
	}
	if rpcErr := jsonrpc.Unmarshal(params, &param); rpcErr != nil {
		return nil, rpcErr
	}
	if rpcErr := validateParam(&param); rpcErr != nil {
		return nil, rpcErr
	}
	hash, err := hex.DecodeString(param.Hash[2:])
	if err != nil {
		return nil, addReason(jsonrpc.ErrInvalidParams(), err)
	}

	bk, err := h.db.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, addReason(jsonrpc.ErrInternal(), err)
	}

	value, err := bk.Get(hash)
	if err != nil {
		return nil, addReason(jsonrpc.ErrInternal(), err)
	}

	if value == nil {
		return nil, jsonrpc.ErrInvalidParams()
	}
	return value, nil
}

type getBlockHeaderByHeightHandler struct {
	bm module.BlockManager
}

func (h getBlockHeaderByHeightHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var param struct {
		Height string `json:"height" valid:"t_int,required"`
	}
	if rpcErr := jsonrpc.Unmarshal(params, &param); rpcErr != nil {
		return nil, rpcErr
	}
	if rpcErr := validateParam(&param); rpcErr != nil {
		return nil, rpcErr
	}

	height, err2 := strconv.ParseInt(param.Height, 0, 64)
	if err2 != nil {
		return nil, addReason(jsonrpc.ErrInvalidParams(), err2)
	}

	block, err2 := h.bm.GetBlockByHeight(height)
	if err2 != nil {
		return nil, addReason(jsonrpc.ErrInvalidParams(), err2)
	}
	buf := bytes.NewBuffer(nil)
	if err2 := block.MarshalHeader(buf); err2 != nil {
		return nil, addReason(jsonrpc.ErrInternal(), err2)
	}
	return buf.Bytes(), nil
}

type getVotesByHeightHandler struct {
	cs module.Consensus
}

func (h getVotesByHeightHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var param struct {
		Height string `json:"height" valid:"t_int,required"`
	}
	if rpcErr := jsonrpc.Unmarshal(params, &param); rpcErr != nil {
		return nil, rpcErr
	}
	if rpcErr := validateParam(&param); rpcErr != nil {
		return nil, rpcErr
	}

	height, err2 := strconv.ParseInt(param.Height, 0, 64)
	if err2 != nil {
		return nil, addReason(jsonrpc.ErrInvalidParams(), err2)
	}
	votes, err2 := h.cs.GetVotesByHeight(height)
	if err2 != nil {
		return nil, addReason(jsonrpc.ErrInvalidParams(), err2)
	}
	return votes.Bytes(), nil
}

type getProofForResultHandler struct {
	bm module.BlockManager
	sm module.ServiceManager
}

func (h getProofForResultHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (result interface{}, err *jsonrpc.Error) {
	var param struct {
		Hash  string `json:"hash" valid:"t_hash,required"`
		Index string `json:"index" valid:"t_int,required"`
	}
	if rpcErr := jsonrpc.Unmarshal(params, &param); rpcErr != nil {
		return nil, rpcErr
	}
	if rpcErr := validateParam(&param); rpcErr != nil {
		return nil, rpcErr
	}

	index, err2 := strconv.ParseInt(param.Index, 0, 64)
	if err2 != nil {
		return nil, addReason(jsonrpc.ErrInvalidParams(), err2)
	}
	hash, err2 := hex.DecodeString(param.Hash[2:])
	if err2 != nil {
		return nil, addReason(jsonrpc.ErrInvalidParams(), err2)
	}

	block, err2 := h.bm.GetBlock(hash)
	if err2 != nil {
		return nil, addReason(jsonrpc.ErrInvalidParams(), err2)
	}
	blockResult := block.Result()
	rl := h.sm.ReceiptListFromResult(blockResult, module.TransactionGroupNormal)
	if rl == nil {
		return nil, addReason(jsonrpc.ErrInvalidParams(), err2)
	}
	proofs, err2 := rl.GetProof(int(index))
	if err2 != nil {
		return nil, addReason(jsonrpc.ErrInvalidParams(), err2)
	}
	return proofs, nil
}
