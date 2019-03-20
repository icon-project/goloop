package v3

import (
	"bytes"
	"encoding/hex"
	"strconv"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
)

const jsonRpcApiVersion = 3

func MethodRepository() *jsonrpc.MethodRepository {
	mr := jsonrpc.NewMethodRepository()

	mr.RegisterMethod("icx_getLastBlock", getLastBlock)
	mr.RegisterMethod("icx_getBlockByHeight", getBlockByHeight)
	mr.RegisterMethod("icx_getBlockByHash", getBlockByHash)
	mr.RegisterMethod("icx_call", call)
	mr.RegisterMethod("icx_getBalance", getBalance)
	mr.RegisterMethod("icx_getScoreApi", getScoreApi)
	mr.RegisterMethod("icx_getTotalSupply", getTotalSupply)
	mr.RegisterMethod("icx_getTransactionResult", getTransactionResult)
	mr.RegisterMethod("icx_getTransactionByHash", getTransactionByHash)
	mr.RegisterMethod("icx_sendTransaction", sendTransaction)

	mr.RegisterMethod("icx_getDataByHash", getDataByHash)
	mr.RegisterMethod("icx_getBlockHeaderByHeight", getBlockHeaderByHeight)
	mr.RegisterMethod("icx_getVotesByHeight", getVotesByHeight)
	mr.RegisterMethod("icx_getProofForResult", getProofForResult)

	return mr
}

func getLastBlock(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	if !params.IsEmpty() {
		return nil, jsonrpc.ErrInvalidParams()
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	bm := chain.BlockManager()

	block, err := bm.GetLastBlock()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	blockJson, err := block.ToJSON(jsonRpcApiVersion)
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	result := blockJson.(map[string]interface{})
	txList := result["confirmed_transaction_list"].(module.TransactionList)
	confirmedTxList, err := convertTransactionList(txList)
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	result["confirmed_transaction_list"] = confirmedTxList

	return result, nil
}

func getBlockByHeight(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param BlockHeightParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrInvalidParams(err.Error())
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	bm := chain.BlockManager()

	block, err := bm.GetBlockByHeight(param.Height.Value())
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	blockJson, err := block.ToJSON(jsonRpcApiVersion)
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	result := blockJson.(map[string]interface{})
	txList := result["confirmed_transaction_list"].(module.TransactionList)
	confirmedTxList, err := convertTransactionList(txList)
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	result["confirmed_transaction_list"] = confirmedTxList

	return result, nil
}

func getBlockByHash(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param BlockHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrInvalidParams(err.Error())
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	bm := chain.BlockManager()

	block, err := bm.GetBlock(param.Hash.Bytes())
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	blockJson, err := block.ToJSON(jsonRpcApiVersion)
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	result := blockJson.(map[string]interface{})
	txList := result["confirmed_transaction_list"].(module.TransactionList)
	confirmedTxList, err := convertTransactionList(txList)
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	result["confirmed_transaction_list"] = confirmedTxList

	return result, nil
}

func call(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param CallParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrInvalidParams(err.Error())
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	bm := chain.BlockManager()
	sm := chain.ServiceManager()

	block, err := bm.GetLastBlock()
	status, result, err := sm.Call(block.Result(), params.RawMessage(), block)
	if err != nil {
		return nil, jsonrpc.ErrInternal()
	}
	if status != module.StatusSuccess {
		msg, ok := result.(string)
		if !ok {
			msg = string(status)
		}
		return nil, &jsonrpc.Error{
			// TODO Is it correct if our error code is in application error range?
			Code:    jsonrpc.ErrorCode(-32500 - int(status)),
			Message: msg,
		}
	} else {
		return result, nil
	}
}

func getBalance(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param AddressParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrInvalidParams(err.Error())
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	bm := chain.BlockManager()
	sm := chain.ServiceManager()

	var balance common.HexInt
	block, err := bm.GetLastBlock()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}
	balance.Set(sm.GetBalance(block.Result(), param.Address.Address()))

	return balance, nil
}

func getScoreApi(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param ScoreAddressParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrInvalidParams()
	}
	_, err := ctx.Chain()
	if err != nil {
	}
	// TODO : service interface required
	return nil, nil
}

func getTotalSupply(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	if !params.IsEmpty() {
		return nil, jsonrpc.ErrInvalidParams()
	}
	_, err := ctx.Chain()
	if err != nil {
	}
	// TODO : service interface required
	return nil, nil
}

func getTransactionResult(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param TransactionHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrInvalidParams(err.Error())
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	bm := chain.BlockManager()

	txInfo, err := bm.GetTransactionInfo(param.Hash.Bytes())
	if err != nil {
		return nil, jsonrpc.ErrServer(err.Error())
	}
	block := txInfo.Block()
	receipt := txInfo.GetReceipt()
	if receipt == nil {
		return nil, jsonrpc.ErrServer("No receipt")
	}
	res, err := receipt.ToJSON(jsonRpcApiVersion)
	if err != nil {
		return nil, jsonrpc.ErrServer(err.Error())
	}

	result := res.(map[string]interface{})
	result["blockHash"] = "0x" + hex.EncodeToString(block.ID())
	result["blockHeight"] = "0x" + strconv.FormatInt(int64(block.Height()), 16)
	result["txIndex"] = "0x" + strconv.FormatInt(int64(txInfo.Index()), 16)

	return result, nil
}

func getTransactionByHash(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param TransactionHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrInvalidParams(err.Error())
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	bm := chain.BlockManager()

	txInfo, err := bm.GetTransactionInfo(param.Hash.Bytes())
	if err != nil {
		return nil, jsonrpc.ErrServer(err.Error())
	}

	tx := txInfo.Transaction()

	var result interface{}
	switch tx.Version() {
	case module.TransactionVersion2:
		result, err = tx.ToJSON(module.TransactionVersion2)
		if err != nil {
			return nil, jsonrpc.ErrServer()
		}
	case module.TransactionVersion3:
		result, err = tx.ToJSON(module.TransactionVersion3)
		if err != nil {
			return nil, jsonrpc.ErrServer()
		}
	default:
		return nil, jsonrpc.ErrServer()
	}

	return result, nil
}

func sendTransaction(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param TransactionParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrInvalidParams(err.Error())
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	sm := chain.ServiceManager()

	hash, err := sm.SendTransaction(params.RawMessage())
	if err != nil {
		return nil, jsonrpc.ErrServer(err.Error())
	}

	result := "0x" + hex.EncodeToString(hash)

	return result, nil
}

func getDataByHash(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param DataHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrInvalidParams()
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	dbm := chain.Database()

	bucket, err := dbm.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, jsonrpc.ErrServer(err.Error())
	}
	value, err := bucket.Get(param.Hash.Bytes())
	if err != nil {
		return nil, jsonrpc.ErrServer(err.Error())
	}

	if value == nil {
		return nil, jsonrpc.ErrInvalidParams("no value")
	}

	return value, nil
}

func getBlockHeaderByHeight(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param BlockHeightParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrInvalidParams()
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	bm := chain.BlockManager()

	block, err := bm.GetBlockByHeight(param.Height.Value())
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	buf := bytes.NewBuffer(nil)
	if err := block.MarshalHeader(buf); err != nil {
		return nil, jsonrpc.ErrServer()
	}

	return buf.Bytes(), nil
}

func getVotesByHeight(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param BlockHeightParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrInvalidParams()
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	cs := chain.Consensus()

	votes, err := cs.GetVotesByHeight(param.Height.Value())
	if err != nil {
		return nil, jsonrpc.ErrServer(err.Error())
	}

	return votes.Bytes(), nil
}

func getProofForResult(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param ProofResultParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrInvalidParams()
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	bm := chain.BlockManager()
	sm := chain.ServiceManager()

	block, err := bm.GetBlock(param.BlockHash.Bytes())
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	blockResult := block.Result()
	receiptList := sm.ReceiptListFromResult(blockResult, module.TransactionGroupNormal)
	proofs, err := receiptList.GetProof(int(param.Index.Value()))
	if err != nil {
		return nil, jsonrpc.ErrServer()
	}

	return proofs, nil
}
