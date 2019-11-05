package v3

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
	"unsafe"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/service"
)

const jsonRpcApiVersion = jsonrpc.APIVersion3

const (
	ConfigShowPatchTransaction = false
)

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
	mr.RegisterMethod("icx_sendTransactionAndWait", sendTransactionAndWait)
	mr.RegisterMethod("icx_waitTransactionResult", waitTransactionResult)

	mr.RegisterMethod("icx_getDataByHash", getDataByHash)
	mr.RegisterMethod("icx_getBlockHeaderByHeight", getBlockHeaderByHeight)
	mr.RegisterMethod("icx_getVotesByHeight", getVotesByHeight)
	mr.RegisterMethod("icx_getProofForResult", getProofForResult)

	return mr
}

func fillTransactions(blockJson interface{}, b module.Block) error {
	result := blockJson.(map[string]interface{})

	if ConfigShowPatchTransaction {
		if txs, err := convertTransactionList(b.PatchTransactions()); err != nil {
			return err
		} else {
			if len(txs) > 0 {
				result["patch_transaction_list"] = txs
			}
		}
	}

	if txs, err := convertTransactionList(b.NormalTransactions()); err != nil {
		return err
	} else {
		result["confirmed_transaction_list"] = txs
	}
	return nil
}

func getLastBlock(ctx *jsonrpc.Context, _ *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	bm := chain.BlockManager()

	block, err := bm.GetLastBlock()
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	blockJson, err := block.ToJSON(jsonRpcApiVersion)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	if err := fillTransactions(blockJson, block); err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}
	return blockJson, nil
}

func getBlockByHeight(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param BlockHeightParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}
	height, err := param.Height.ParseInt(64)
	if err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	bm := chain.BlockManager()

	block, err := bm.GetBlockByHeight(height)
	if errors.NotFoundError.Equals(err) {
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	blockJson, err := block.ToJSON(jsonRpcApiVersion)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	if err := fillTransactions(blockJson, block); err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}
	return blockJson, nil
}

func getBlockByHash(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param BlockHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	bm := chain.BlockManager()

	block, err := bm.GetBlock(param.Hash.Bytes())
	if errors.NotFoundError.Equals(err) {
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	blockJson, err := block.ToJSON(jsonRpcApiVersion)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	if err := fillTransactions(blockJson, block); err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}
	return blockJson, nil
}

func call(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param CallParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	bm := chain.BlockManager()
	sm := chain.ServiceManager()

	block, err := bm.GetLastBlock()
	result, err := sm.Call(block.Result(), block.NextValidators(), params.RawMessage(), block)
	if err != nil {
		return nil, jsonrpc.ErrScore(err, debug)
	} else {
		return result, nil
	}
}

func getBalance(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param AddressParam
	debug := ctx.IncludeDebug()
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	bm := chain.BlockManager()
	sm := chain.ServiceManager()

	var balance common.HexInt
	block, err := bm.GetLastBlock()
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}
	b, err := sm.GetBalance(block.Result(), param.Address.Address())
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}
	balance.Set(b)
	return &balance, nil
}

func getScoreApi(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var param ScoreAddressParam
	debug := ctx.IncludeDebug()
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}
	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}
	bm := chain.BlockManager()
	b, err := bm.GetLastBlock()
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}
	sm := chain.ServiceManager()
	info, err := sm.GetAPIInfo(b.Result(), param.Address.Address())
	if err != nil {
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, debug)
	}
	if jso, err := info.ToJSON(jsonRpcApiVersion); err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	} else {
		return jso, nil
	}
}

func getTotalSupply(ctx *jsonrpc.Context, _ *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()
	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}
	bm := chain.BlockManager()
	b, err := bm.GetLastBlock()
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}
	sm := chain.ServiceManager()

	var tsValue common.HexInt
	ts, err := sm.GetTotalSupply(b.Result())
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}
	tsValue.Set(ts)

	return &tsValue, nil
}

func getTransactionResult(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param TransactionHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	bm := chain.BlockManager()
	sm := chain.ServiceManager()

	txInfo, err := bm.GetTransactionInfo(param.Hash.Bytes())
	if errors.NotFoundError.Equals(err) {
		if sm.HasTransaction(param.Hash.Bytes()) {
			return nil, jsonrpc.ErrorCodePending.New("Pending")
		}
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	blk := txInfo.Block()
	receipt, err := txInfo.GetReceipt()
	if block.ResultNotFinalizedError.Equals(err) {
		return nil, jsonrpc.ErrorCodeExecuting.New("Executing")
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}
	res, err := receipt.ToJSON(jsonRpcApiVersion)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	result := res.(map[string]interface{})
	result["blockHash"] = "0x" + hex.EncodeToString(blk.ID())
	result["blockHeight"] = "0x" + strconv.FormatInt(int64(blk.Height()), 16)
	result["txIndex"] = "0x" + strconv.FormatInt(int64(txInfo.Index()), 16)
	result["txHash"] = "0x" + hex.EncodeToString(txInfo.Transaction().ID())

	return result, nil
}

func getTransactionByHash(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param TransactionHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	bm := chain.BlockManager()

	txInfo, err := bm.GetTransactionInfo(param.Hash.Bytes())
	if errors.NotFoundError.Equals(err) {
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	tx := txInfo.Transaction()

	var res interface{}
	switch tx.Version() {
	case module.TransactionVersion2:
		res, err = tx.ToJSON(jsonrpc.APIVersion2)
		if err != nil {
			return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
		}
	case module.TransactionVersion3:
		res, err = tx.ToJSON(jsonrpc.APIVersion3)
		if err != nil {
			return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
		}
	default:
		return nil, jsonrpc.ErrorCodeSystem.Errorf(
			"Unknown transaction version=%d", tx.Version())
	}

	blk := txInfo.Block()
	result := res.(map[string]interface{})
	result["blockHash"] = "0x" + hex.EncodeToString(blk.ID())
	result["blockHeight"] = "0x" + strconv.FormatInt(int64(blk.Height()), 16)
	result["txIndex"] = "0x" + strconv.FormatInt(int64(txInfo.Index()), 16)

	return result, nil
}

func sendTransaction(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param TransactionParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	sm := chain.ServiceManager()

	hash, err := sm.SendTransaction(params.RawMessage())
	if err != nil {
		if service.TransactionPoolOverflowError.Equals(err) {
			return nil, jsonrpc.ErrorCodeTxPoolOverflow.Wrap(err, debug)
		}
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	result := "0x" + hex.EncodeToString(hash)

	return result, nil
}

func getDataByHash(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param DataHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	dbm := chain.Database()

	bucket, err := dbm.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}
	value, err := bucket.Get(param.Hash.Bytes())
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	if value == nil {
		return nil, jsonrpc.ErrorCodeNotFound.New("Fail to find data")
	}

	return value, nil
}

func getBlockHeaderByHeight(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param BlockHeightParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}
	height, err := param.Height.ParseInt(64)
	if err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	bm := chain.BlockManager()

	block, err := bm.GetBlockByHeight(height)
	if errors.NotFoundError.Equals(err) {
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	buf := bytes.NewBuffer(nil)
	if err := block.MarshalHeader(buf); err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	return buf.Bytes(), nil
}

func getVotesByHeight(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param BlockHeightParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}
	height, err := param.Height.ParseInt(64)
	if err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	cs := chain.Consensus()

	votes, err := cs.GetVotesByHeight(height)
	if errors.NotFoundError.Equals(err) {
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	return votes.Bytes(), nil
}

func getProofForResult(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param ProofResultParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}
	var idx int
	if v64, err := param.Index.ParseInt(int(unsafe.Sizeof(idx)) * 8); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	} else {
		idx = int(v64)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	bm := chain.BlockManager()
	sm := chain.ServiceManager()

	block, err := bm.GetBlock(param.BlockHash.Bytes())
	if errors.NotFoundError.Equals(err) {
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	blockResult := block.Result()
	receiptList, err := sm.ReceiptListFromResult(blockResult, module.TransactionGroupNormal)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}
	proofs, err := receiptList.GetProof(idx)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	return proofs, nil
}

// convert TransactionList to []Transaction
func convertTransactionList(txs module.TransactionList) ([]interface{}, error) {
	list := []interface{}{}

	for it := txs.Iterator(); it.Has(); it.Next() {
		tx, _, err := it.Get()
		if err != nil {
			return nil, err
		}
		switch tx.Version() {
		case module.TransactionVersion2:
			res, err := tx.ToJSON(jsonrpc.APIVersion2)
			list = append(list, res)
			if err != nil {
				return nil, err
			}
		case module.TransactionVersion3:
			res, err := tx.ToJSON(jsonrpc.APIVersion3)
			list = append(list, res)
			if err != nil {
				return nil, err
			}
		}
	}
	return list, nil
}

func sendTransactionAndWait(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	dt := chain.DefaultWaitTimeout()
	if dt <= 0 {
		return nil, jsonrpc.ErrorCodeMethodNotFound.Errorf("NotEnabled(waitTimeout=%d)", dt)
	}

	ut := ctx.GetTimeout(dt)
	if ut <= 0 {
		return nil, jsonrpc.ErrorCodeInvalidParams.Errorf("InvalidTimeout(%d)", ut)
	}
	mt := chain.MaxWaitTimeout()
	timeout := ut
	maxLimit := false
	if timeout > mt {
		timeout = mt
		maxLimit = true
	}

	var param TransactionParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	bm := chain.BlockManager()

	hash, fc, err := bm.SendTransactionAndWait(params.RawMessage())
	if err != nil {
		if service.TransactionPoolOverflowError.Equals(err) {
			return nil, jsonrpc.ErrorCodeTxPoolOverflow.Wrap(err, debug)
		}
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	return waitTransactionResultOnChannel(ctx, bm, hash, debug, timeout, maxLimit, fc)
}

func waitTransactionResult(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	dt := chain.DefaultWaitTimeout()
	if dt <= 0 {
		return nil, jsonrpc.ErrorCodeMethodNotFound.Errorf("NotEnabled(defaultWaitTimeout=%d)", dt)
	}

	ut := ctx.GetTimeout(dt)
	if ut <= 0 {
		return nil, jsonrpc.ErrorCodeInvalidParams.Errorf("InvalidTimeout(%d)", ut)
	}
	mt := chain.MaxWaitTimeout()
	timeout := ut
	maxLimit := false
	if timeout > mt {
		timeout = mt
		maxLimit = true
	}

	var param TransactionHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	bm := chain.BlockManager()

	hash := param.Hash.Bytes()
	fc, err := bm.WaitTransactionResult(hash)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}

	return waitTransactionResultOnChannel(ctx, bm, hash, debug, timeout, maxLimit, fc)
}

func waitTransactionResultOnChannel(ctx *jsonrpc.Context, bm module.BlockManager,
	id []byte, debug bool, timeout time.Duration, maxLimit bool,
	fc <-chan interface{},
) (interface{}, error) {
	tc := time.After(timeout)

	var err error
	var txInfo module.TransactionInfo
	var receipt module.Receipt
	select {
	case result := <-fc:
		switch ro := result.(type) {
		case error:
			return nil, jsonrpc.ErrorCodeSystem.Wrap(ro, debug)
		case module.TransactionInfo:
			txInfo = ro
			receipt, err = txInfo.GetReceipt()
			if err != nil {
				return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
			}
		case module.Receipt:
			txInfo, err = bm.GetTransactionInfo(id)
			if err != nil {
				return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
			}
			receipt = ro
		default:
			return nil, jsonrpc.ErrorCodeSystem.New("Unknown resulting object")
		}
	case <-tc:
		if maxLimit {
			return nil, jsonrpc.ErrorCodeSystemTimeout.NewWithData(
				fmt.Sprintf("SystemTimeout(dur=%s)", timeout),
				"0x"+hex.EncodeToString(id),
			)
		}
		return nil, jsonrpc.ErrorCodeTimeout.NewWithData(
			fmt.Sprintf("Timeout(dur=%s)", timeout),
			"0x"+hex.EncodeToString(id),
		)
	case <-ctx.Request().Context().Done():
		return nil, nil
	}

	res, err := receipt.ToJSON(jsonRpcApiVersion)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, debug)
	}
	result := res.(map[string]interface{})
	blk := txInfo.Block()
	result["blockHash"] = "0x" + hex.EncodeToString(blk.ID())
	result["blockHeight"] = "0x" + strconv.FormatInt(int64(blk.Height()), 16)
	result["txIndex"] = "0x" + strconv.FormatInt(int64(txInfo.Index()), 16)
	result["txHash"] = "0x" + hex.EncodeToString(id)

	return result, nil
}
