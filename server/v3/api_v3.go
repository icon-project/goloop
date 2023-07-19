package v3

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/btp/ntm"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/trace"
	"github.com/icon-project/goloop/service/txresult"
)

const (
	ConfigShowPatchTransaction = false
)

func MethodRepository(mtr *metric.JsonrpcMetric) *jsonrpc.MethodRepository {
	mr := jsonrpc.NewMethodRepository(mtr)
	RegisterValidationRule(mr.Validator())

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
	mr.RegisterMethod("icx_getProofForEvents", getProofForEvents)
	mr.RegisterMethod("icx_getScoreStatus", getScoreStatus)
	mr.RegisterMethod("icx_getNetworkInfo", getNetworkInfo)

	mr.RegisterMethod("btp_getNetworkInfo", getBTPNetworkInfo)
	mr.RegisterMethod("btp_getNetworkTypeInfo", getBTPNetworkTypeInfo)
	mr.RegisterMethod("btp_getMessages", getBTPMessages)
	mr.RegisterMethod("btp_getHeader", getBTPHeader)
	mr.RegisterMethod("btp_getProof", getBTPProof)
	mr.RegisterMethod("btp_getSourceInformation", getBTPSourceInformation)

	mr.SetAllowedNotification("icx_sendTransaction")
	mr.SetAllowedNotification("icx_sendTransactionAndWait")
	return mr
}

func fillTransactions(blockJson interface{}, b module.Block, v module.JSONVersion) error {
	result := blockJson.(map[string]interface{})

	if ConfigShowPatchTransaction {
		if txs, err := convertTransactionList(b.PatchTransactions(), v); err != nil {
			return err
		} else {
			if len(txs) > 0 {
				result["patch_transaction_list"] = txs
			}
		}
	}

	if txs, err := convertTransactionList(b.NormalTransactions(), v); err != nil {
		return err
	} else {
		result["confirmed_transaction_list"] = txs
	}
	return nil
}

type contextWithChain struct {
	*jsonrpc.Context
	debug bool
	chain module.Chain
}

func (c *contextWithChain) Init(ctx *jsonrpc.Context) error {
	c.Context = ctx
	c.debug = ctx.IncludeDebug()

	var err error
	c.chain, err = ctx.Chain()
	if err != nil {
		return jsonrpc.ErrorCodeServer.Wrap(err, c.debug)
	}
	return nil
}

// AsRPCError ensure err to be *jsonrpc.Error.
// If debug flag is on, then it would include debug information.
// It returns jsonrpc.ErrorCodeNotFound for errors.NotFoundError.
// It returns jsonrpc.ErrorCodeSystem for others.
func (c *contextWithChain) AsRPCError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*jsonrpc.Error); ok {
		return err
	}
	if errors.NotFoundError.Equals(err) {
		return jsonrpc.ErrorCodeNotFound.Wrap(err, c.debug)
	}
	return jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
}

// CheckBaseHeight returns jsonrpc.ErrorCodeNotFound for lower height
// than the base height in genesis.
func (c *contextWithChain) CheckBaseHeight(height int64) error {
	if height < 0 {
		return jsonrpc.ErrorCodeNotFound.Errorf("NegativeHeight(height=%d)", height)
	}
	base := c.chain.GenesisStorage().Height()
	if height < base {
		return jsonrpc.ErrorCodeNotFound.Errorf(
			"PrunedBlock(height=%d,base=%d)", height, base)
	}
	return nil
}

type contextWithBM struct {
	contextWithChain
	bm module.BlockManager
}

func (c *contextWithBM) Init(ctx *jsonrpc.Context) error {
	if err := c.contextWithChain.Init(ctx); err != nil {
		return err
	}

	c.bm = c.chain.BlockManager()
	if c.bm == nil {
		return jsonrpc.ErrorCodeServer.New("Stopped")
	}
	return nil
}

func (c *contextWithBM) GetBlockByHeight(height jsonrpc.HexInt) (module.Block, error) {
	if height == "" {
		blk, err := c.bm.GetLastBlock()
		return blk, c.AsRPCError(err)
	} else {
		h, err := height.Int64()
		if err != nil {
			return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
		}
		if err = c.CheckBaseHeight(h); err != nil {
			return nil, err
		}
		blk, err := c.bm.GetBlockByHeight(h)
		return blk, c.AsRPCError(err)
	}
}

func (c *contextWithBM) GetBlockByID(id []byte) (module.Block, error) {
	blk, err := c.bm.GetBlock(id)
	if err != nil {
		return nil, c.AsRPCError(err)
	}
	if err = c.CheckBaseHeight(blk.Height()); err != nil {
		return nil, err
	}
	return blk, nil
}

type contextWithSM struct {
	contextWithBM
	sm module.ServiceManager
}

func (c *contextWithSM) Init(ctx *jsonrpc.Context) error {
	if err := c.contextWithBM.Init(ctx); err != nil {
		return err
	}
	c.sm = c.chain.ServiceManager()
	if c.sm == nil {
		return jsonrpc.ErrorCodeServer.New("Stopped")
	}
	return nil
}

type contextWithCS struct {
	contextWithBM
	cs module.Consensus
}

func (c *contextWithCS) Init(ctx *jsonrpc.Context) error {
	if err := c.contextWithBM.Init(ctx); err != nil {
		return err
	}
	c.cs = c.chain.Consensus()
	if c.cs == nil {
		return jsonrpc.ErrorCodeServer.New("Stopped")
	}
	return nil
}

func getLastBlock(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithBM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param struct{}
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	blk, err := c.bm.GetLastBlock()
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	blockJson, err := blk.ToJSON(module.JSONVersion3)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	if err = fillTransactions(blockJson, blk, module.JSONVersion3); err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	return blockJson, nil
}

func getBlockByHeight(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithBM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param BlockHeightParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	blk, err := c.GetBlockByHeight(param.Height)
	if err != nil {
		return nil, err
	}

	blockJson, err := blk.ToJSON(module.JSONVersion3)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	if err = fillTransactions(blockJson, blk, module.JSONVersion3); err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	return blockJson, nil
}

func getBlockByHash(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithBM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param BlockHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	blk, err := c.GetBlockByID(param.Hash.Bytes())
	if err != nil {
		return nil, err
	}

	blockJson, err := blk.ToJSON(module.JSONVersion3)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	if err = fillTransactions(blockJson, blk, module.JSONVersion3); err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	return blockJson, nil
}

func call(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param CallParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	blk, err := c.GetBlockByHeight(param.Height)
	if err != nil {
		return nil, err
	}

	bi := common.NewBlockInfo(blk.Height(), blk.Timestamp())
	result, err := c.sm.Call(blk.Result(), blk.NextValidators(), params.RawMessage(), bi)
	if err != nil {
		if service.InvalidQueryError.Equals(err) {
			return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
		} else if scoreresult.IsValid(err) {
			return nil, jsonrpc.ErrScore(err, c.debug)
		} else {
			return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
		}
	} else {
		return result, nil
	}
}

func getBalance(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param AddressParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	var balance common.HexInt
	blk, err := c.GetBlockByHeight(param.Height)
	if err != nil {
		return nil, err
	}

	b, err := c.sm.GetBalance(blk.Result(), param.Address.Address())
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	balance.Set(b)
	return &balance, nil
}

func getScoreApi(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param ScoreAddressParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	b, err := c.GetBlockByHeight(param.Height)
	if err != nil {
		return nil, err
	}
	info, err := c.sm.GetAPIInfo(b.Result(), param.Address.Address())
	if service.NoActiveContractError.Equals(err) {
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, c.debug)
	}
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	if jso, err := info.ToJSON(module.JSONVersion3); err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	} else {
		return jso, nil
	}
}

func getTotalSupply(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}
	var param *HeightParam
	var height jsonrpc.HexInt
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	} else {
		if param != nil {
			height = param.Height
		}
	}

	b, err := c.GetBlockByHeight(height)
	if err != nil {
		return nil, err
	}
	var tsValue common.HexInt
	ts, err := c.sm.GetTotalSupply(b.Result())
	if err != nil {
		return nil, c.AsRPCError(err)
	}
	tsValue.Set(ts)

	return &tsValue, nil
}

func getTransactionResult(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param TransactionHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	txInfo, err := c.bm.GetTransactionInfo(param.Hash.Bytes())
	if errors.NotFoundError.Equals(err) {
		if c.sm.HasTransaction(param.Hash.Bytes()) {
			return nil, jsonrpc.ErrorCodePending.New("Pending")
		}
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, c.debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	blk := txInfo.Block()
	if err = c.CheckBaseHeight(blk.Height()); err != nil {
		return nil, err
	}
	receipt, err := txInfo.GetReceipt()
	if block.ResultNotFinalizedError.Equals(err) {
		return nil, jsonrpc.ErrorCodeExecuting.New("Executing")
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	res, err := receipt.ToJSON(module.JSONVersion3)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	result := res.(map[string]interface{})
	result["blockHash"] = "0x" + hex.EncodeToString(blk.ID())
	result["blockHeight"] = "0x" + strconv.FormatInt(blk.Height(), 16)
	result["txIndex"] = "0x" + strconv.FormatInt(int64(txInfo.Index()), 16)
	result["txHash"] = "0x" + hex.EncodeToString(param.Hash.Bytes())

	return result, nil
}

func getTransactionByHash(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithBM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param TransactionHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	txInfo, err := c.bm.GetTransactionInfo(param.Hash.Bytes())
	if err != nil {
		return nil, c.AsRPCError(err)
	}

	tx, err := txInfo.Transaction()
	if err != nil {
		return nil, c.AsRPCError(err)
	}
	res, err := tx.ToJSON(module.JSONVersion3)
	if err != nil {
		return nil, c.AsRPCError(err)
	}

	blk := txInfo.Block()
	if err = c.CheckBaseHeight(blk.Height()); err != nil {
		return nil, err
	}
	result := res.(map[string]interface{})
	result["blockHash"] = "0x" + hex.EncodeToString(blk.ID())
	result["blockHeight"] = "0x" + strconv.FormatInt(blk.Height(), 16)
	result["txIndex"] = "0x" + strconv.FormatInt(int64(txInfo.Index()), 16)

	return result, nil
}

func sendTransaction(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param TransactionParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	var state []byte
	var height int64
	if c.chain.ValidateTxOnSend() {
		blk, err := c.bm.GetLastBlock()
		if err != nil {
			return nil, jsonrpc.ErrorCodeServer.Wrap(err, c.debug)
		}
		state = blk.Result()
		height = blk.Height() + 1
	}

	hash, err := c.sm.SendTransaction(state, height, params.RawMessage())
	if err != nil {
		if service.TransactionPoolOverflowError.Equals(err) {
			return nil, jsonrpc.ErrorCodeTxPoolOverflow.Wrap(err, c.debug)
		}
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	result := "0x" + hex.EncodeToString(hash)

	return result, nil
}

func getDataByHash(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithChain
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param DataHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	var ret error
	var value []byte
	c.chain.DoDBTask(func(database db.Database) {
		bucket, err := database.GetBucket(db.BytesByHash)
		if err != nil {
			ret = jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
			return
		}
		value, err = bucket.Get(param.Hash.Bytes())
		if err != nil {
			ret = jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
			return
		}
	})
	if ret != nil {
		return nil, ret
	}

	if value == nil {
		return nil, jsonrpc.ErrorCodeNotFound.New("Fail to find data")
	}

	return value, nil
}

func getBlockHeaderByHeight(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithBM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param BlockHeightParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	blk, err := c.GetBlockByHeight(param.Height)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)
	if err = blk.MarshalHeader(buf); err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	return buf.Bytes(), nil
}

func getVotesByHeight(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithChain
	if err := c.Init(ctx); err != nil {
		return nil, err
	}
	cs := c.chain.Consensus()
	if cs == nil {
		return nil, jsonrpc.ErrorCodeServer.New("AlreadyStopped")
	}

	var param BlockHeightParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}
	height, err := param.Height.ParseInt(64)
	if err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}
	if err = c.CheckBaseHeight(height); err != nil {
		return nil, err
	}

	votes, err := cs.GetVotesByHeight(height)
	if err != nil {
		return nil, c.AsRPCError(err)
	}
	return votes.Bytes(), nil
}

func getProofForResult(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param ProofResultParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}
	var idx int
	if v64, err := param.Index.ParseInt(int(unsafe.Sizeof(idx)) * 8); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	} else {
		idx = int(v64)
	}

	blk, err := c.GetBlockByID(param.BlockHash.Bytes())
	if err != nil {
		return nil, err
	}

	receiptList, err := c.sm.ReceiptListFromResult(blk.Result(), module.TransactionGroupNormal)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	proofs, err := receiptList.GetProof(idx)
	if err != nil {
		return nil, c.AsRPCError(err)
	}

	return proofs, nil
}

func getProofForEvents(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param ProofEventsParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}
	var idx int
	if v64, err := param.Index.ParseInt(int(unsafe.Sizeof(idx)) * 8); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	} else {
		idx = int(v64)
	}
	blk, err := c.GetBlockByID(param.BlockHash.Bytes())
	if err != nil {
		return nil, err
	}

	receiptList, err := c.sm.ReceiptListFromResult(blk.Result(), module.TransactionGroupNormal)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	receipt, err := receiptList.Get(idx)
	if err != nil {
		err = errors.NotFoundError.Wrapf(err,
			"fail to get a receipt for index=%d", idx)
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, c.debug)
	}
	var proofs [][][]byte
	rProof, err := receiptList.GetProof(idx)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	proofs = append(proofs, rProof)
	for _, ei := range param.Events {
		proof, err := receipt.GetProofOfEvent(int(ei.Value()))
		if err != nil {
			if errors.InvalidStateError.Equals(err) {
				return nil, jsonrpc.ErrorCodeSystem.Errorf(
					"unable to get proof from current receipt index=%d", idx)
			}
			if errors.NotFoundError.Equals(err) {
				return nil, jsonrpc.ErrorCodeNotFound.Errorf(
					"no proof for receipt index=%d, event index=%d", idx, ei.Value())
			}
			return nil, c.AsRPCError(err)
		}
		proofs = append(proofs, proof)
	}
	return proofs, nil
}

func getScoreStatus(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}
	var param ScoreAddressParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	b, err := c.GetBlockByHeight(param.Height)
	if err != nil {
		return nil, err
	}
	s, err := c.sm.GetSCOREStatus(b.Result(), param.Address.Address())
	if err != nil {
		return nil, c.AsRPCError(err)
	}
	jso, err := s.ToJSON(b.Height(), module.JSONVersion3)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	return jso, nil
}

type NetworkInfo struct {
	Platform  string         `json:"platform"`
	NID       jsonrpc.HexInt `json:"nid"`
	Channel   string         `json:"channel"`
	Earliest  jsonrpc.HexInt `json:"earliest"`
	Latest    jsonrpc.HexInt `json:"latest"`
	StepPrice jsonrpc.HexInt `json:"stepPrice"`
}

func getNetworkInfo(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}
	blk, err := c.bm.GetLastBlock()
	if err != nil {
		return nil, c.AsRPCError(err)
	}
	price, err := c.sm.GetStepPrice(blk.Result())
	if err != nil {
		return nil, c.AsRPCError(err)
	}
	earliest := c.chain.GenesisStorage().Height()
	return &NetworkInfo{
		Platform:  c.chain.PlatformName(),
		NID:       jsonrpc.HexInt(intconv.FormatInt(int64(c.chain.NID()))),
		Channel:   c.chain.Channel(),
		Earliest:  jsonrpc.HexInt(intconv.FormatInt(earliest)),
		Latest:    jsonrpc.HexInt(intconv.FormatInt(blk.Height())),
		StepPrice: jsonrpc.HexInt(intconv.FormatBigInt(price)),
	}, nil
}

func getBTPNetworkInfo(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param BTPQueryParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	nid, err := param.Id.Int64()
	if err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	blk, err := c.GetBlockByHeight(param.Height)
	if err != nil {
		return nil, err
	}

	blockResult := blk.Result()
	nw, err := c.sm.BTPNetworkFromResult(blockResult, nid)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	nt, err := c.sm.BTPNetworkTypeFromResult(blockResult, nw.NetworkTypeID())
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	res := nw.ToJSON()
	res["networkID"] = intconv.FormatInt(nid)
	res["networkTypeName"] = nt.UID()
	return res, nil
}

func getBTPNetworkTypeInfo(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param BTPQueryParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	ntid, err := param.Id.Int64()
	if err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	blk, err := c.GetBlockByHeight(param.Height)
	if err != nil {
		return nil, err
	}

	blockResult := blk.Result()
	nt, err := c.sm.BTPNetworkTypeFromResult(blockResult, ntid)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	res := nt.ToJSON()
	res["networkTypeID"] = intconv.FormatInt(ntid)
	return res, nil
}

func getBTPMessages(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param BTPMessagesParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	nid, err := param.NetworkId.Int64()
	if err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	blk, err := c.GetBlockByHeight(param.Height)
	if err != nil {
		return nil, err
	}

	res := make([]string, 0)
	blockResult := blk.Result()
	bDigest, err := c.sm.BTPDigestFromResult(blockResult)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	if bDigest == nil {
		return res, nil
	}
	nw, err := c.sm.BTPNetworkFromResult(blockResult, nid)
	if err != nil || nw == nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	ntid := nw.NetworkTypeID()
	nt, err := c.sm.BTPNetworkTypeFromResult(blockResult, ntid)
	if err != nil || nt == nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	ntDigest := bDigest.NetworkTypeDigestFor(ntid)
	if ntDigest == nil {
		return res, nil
	}
	nwDigest := ntDigest.NetworkDigestFor(nid)
	if nwDigest == nil {
		return res, nil
	}
	ml, err := nwDigest.MessageList(c.chain.Database(), ntm.ForUID(nt.UID()))
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	size := int(ml.Len())
	for i := 0; i < size; i++ {
		msg, err := ml.Get(i)
		if err != nil {
			return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
		}
		res = append(res, base64.StdEncoding.EncodeToString(msg.Bytes()))
	}
	return res, nil
}

func getBTPHeader(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithCS
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param BTPMessagesParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	nid, err := param.NetworkId.Int64()
	if err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}
	blk, err := c.GetBlockByHeight(param.Height)
	if err != nil {
		return nil, err
	}
	btpBlock, _, err := c.cs.GetBTPBlockHeaderAndProof(blk, nid, module.FlagBTPBlockHeader)
	if errors.NotFoundError.Equals(err) {
		err = errors.NotFoundError.Wrapf(
			err, "fail to get a BTP block header for height=%d, nid=%d", blk.Height(), nid)
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, c.debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	return base64.StdEncoding.EncodeToString(btpBlock.HeaderBytes()), nil
}

func getBTPProof(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithCS
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param BTPMessagesParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	nid, err := param.NetworkId.Int64()
	if err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	blk, err := c.GetBlockByHeight(param.Height)
	_, proof, err := c.cs.GetBTPBlockHeaderAndProof(blk, nid, module.FlagBTPBlockProof)
	if errors.NotFoundError.Equals(err) {
		err = errors.NotFoundError.Wrapf(
			err, "fail to get a BTP block proof for height=%d, nid=%d", blk.Height(), nid)
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, c.debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	return base64.StdEncoding.EncodeToString(proof), nil
}

func getBTPSourceInformation(ctx *jsonrpc.Context, _ *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	blk, err := c.bm.GetLastBlock()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, c.debug)
	}
	ntids, err := c.sm.BTPNetworkTypeIDsFromResult(blk.Result())
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, c.debug)
	}

	ontids := make([]interface{}, len(ntids))
	for i, ntid := range ntids {
		ontids[i] = intconv.FormatInt(ntid)
	}

	return map[string]interface{}{
		"srcNetworkUID":  intconv.FormatInt(int64(c.chain.NID())) + ".icon",
		"networkTypeIDs": ontids,
	}, nil
}

// convert TransactionList to []Transaction
func convertTransactionList(txs module.TransactionList, version module.JSONVersion) ([]interface{}, error) {
	list := []interface{}{}

	for it := txs.Iterator(); it.Has(); it.Next() {
		tx, _, err := it.Get()
		if err != nil {
			return nil, err
		}

		res, err := tx.ToJSON(version)
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

func sendTransactionAndWait(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithBM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	dt := c.chain.DefaultWaitTimeout()
	if dt <= 0 {
		return nil, jsonrpc.ErrorCodeMethodNotFound.Errorf("NotEnabled(waitTimeout=%d)", dt)
	}

	ut := ctx.GetTimeout(dt)
	if ut <= 0 {
		return nil, jsonrpc.ErrorCodeInvalidRequest.Errorf("InvalidTimeout(%d)", ut)
	}
	mt := c.chain.MaxWaitTimeout()
	timeout := ut
	maxLimit := false
	if timeout > mt {
		timeout = mt
		maxLimit = true
	}

	var param TransactionParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	var state []byte
	var height int64
	if c.chain.ValidateTxOnSend() {
		blk, err := c.bm.GetLastBlock()
		if err != nil {
			return nil, jsonrpc.ErrorCodeServer.Wrap(err, c.debug)
		}
		state = blk.Result()
		height = blk.Height() + 1
	}

	hash, fc, err := c.bm.SendTransactionAndWait(state, height, params.RawMessage())
	if err != nil {
		if service.TransactionPoolOverflowError.Equals(err) {
			return nil, jsonrpc.ErrorCodeTxPoolOverflow.Wrap(err, c.debug)
		}
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	return waitTransactionResultOnChannel(&c, hash, timeout, maxLimit, fc)
}

func waitTransactionResult(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithBM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	dt := c.chain.DefaultWaitTimeout()
	if dt <= 0 {
		return nil, jsonrpc.ErrorCodeMethodNotFound.Errorf("NotEnabled(waitTimeout=%d)", dt)
	}

	ut := ctx.GetTimeout(dt)
	if ut <= 0 {
		return nil, jsonrpc.ErrorCodeInvalidParams.Errorf("InvalidTimeout(%d)", ut)
	}
	mt := c.chain.MaxWaitTimeout()
	timeout := ut
	maxLimit := false
	if timeout > mt {
		timeout = mt
		maxLimit = true
	}

	var param TransactionHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	hash := param.Hash.Bytes()
	fc, err := c.bm.WaitTransactionResult(hash)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	return waitTransactionResultOnChannel(&c, hash, timeout, maxLimit, fc)
}

func waitTransactionResultOnChannel(c *contextWithBM, id []byte, timeout time.Duration, maxLimit bool, fc <-chan interface{}) (interface{}, error) {
	tc := time.After(timeout)

	var err error
	var txInfo module.TransactionInfo
	var receipt module.Receipt
	select {
	case result := <-fc:
		switch ro := result.(type) {
		case error:
			return nil, jsonrpc.ErrorCodeSystem.Wrap(ro, c.debug)
		case module.TransactionInfo:
			txInfo = ro
			receipt, err = txInfo.GetReceipt()
			if err != nil {
				return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
			}
		case module.Receipt:
			txInfo, err = c.bm.GetTransactionInfo(id)
			if err != nil {
				return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
			}
			receipt = ro
		default:
			return nil, jsonrpc.ErrorCodeSystem.New("Unknown resulting object")
		}
	case <-tc:
		if maxLimit {
			return nil, jsonrpc.ErrorCodeSystemTimeout.New(
				fmt.Sprintf("SystemTimeoutExpire(dur=%s)", timeout),
				"0x"+hex.EncodeToString(id),
			)
		}
		return nil, jsonrpc.ErrorCodeTimeout.New(
			fmt.Sprintf("UserTimeoutExpire(dur=%s)", timeout),
			"0x"+hex.EncodeToString(id),
		)
	case <-c.Request().Context().Done():
		return nil, nil
	}

	blk := txInfo.Block()
	if err = c.CheckBaseHeight(blk.Height()); err != nil {
		return nil, err
	}
	res, err := receipt.ToJSON(module.JSONVersion3)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	result := res.(map[string]interface{})
	result["blockHash"] = "0x" + hex.EncodeToString(blk.ID())
	result["blockHeight"] = "0x" + strconv.FormatInt(blk.Height(), 16)
	result["txIndex"] = "0x" + strconv.FormatInt(int64(txInfo.Index()), 16)
	result["txHash"] = "0x" + hex.EncodeToString(id)

	return result, nil
}

func DebugMethodRepository(mtr *metric.JsonrpcMetric) *jsonrpc.MethodRepository {
	mr := jsonrpc.NewMethodRepository(mtr)
	RegisterValidationRule(mr.Validator())

	mr.RegisterMethod("debug_getTrace", getTrace)
	mr.RegisterMethod("debug_estimateStep", estimateStep)

	return mr
}

func getTrace(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param TransactionHashParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	txInfo, err := c.bm.GetTransactionInfo(param.Hash.Bytes())
	if errors.NotFoundError.Equals(err) {
		if c.sm.HasTransaction(param.Hash.Bytes()) {
			return nil, jsonrpc.ErrorCodePending.New("Pending")
		}
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, c.debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	if txInfo.Group() == module.TransactionGroupPatch {
		return nil, jsonrpc.ErrorCodeInvalidParams.New("Patch transaction can't be replayed")
	}

	blk := txInfo.Block()
	if err = c.CheckBaseHeight(blk.Height()); err != nil {
		return nil, err
	}
	_, err = txInfo.GetReceipt()
	if block.ResultNotFinalizedError.Equals(err) {
		return nil, jsonrpc.ErrorCodeExecuting.New("Executing")
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	csi, err := c.bm.NewConsensusInfo(blk)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	nblk, err := c.bm.GetBlockByHeight(blk.Height() + 1)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	tr1, err := c.sm.CreateInitialTransition(blk.Result(), blk.NextValidators())
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	tr2, err := c.sm.CreateTransition(tr1, blk.NormalTransactions(), blk, csi, true)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	tr2 = c.sm.PatchTransition(tr2, nblk.PatchTransactions(), nblk)

	cb := &traceCallback{
		logs:    make([]interface{}, 0, 100),
		channel: make(chan interface{}, 10),
	}
	ti := module.TraceInfo{
		TraceMode: module.TraceModeInvoke,
		Range:     module.TraceRangeTransaction,
		Group:     txInfo.Group(),
		Index:     txInfo.Index(),
		Callback:  cb,
	}
	canceller, err := tr2.ExecuteForTrace(ti)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	timer := time.After(time.Second * 5)
	for {
		select {
		case <-timer:
			canceller()
			return nil, jsonrpc.ErrorCodeSystemTimeout.Errorf(
				"Not enough time to get result of %x", param.Hash.Bytes())
		case <-cb.channel:
			return cb.invokeTraceToJSON(), nil
		}
	}
}

func estimateStep(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param TransactionParamForEstimate
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	// get last block
	blk, err := c.bm.GetLastBlock()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, c.debug)
	}

	// new block information based on the last
	oldTS := blk.Timestamp()
	newTS := common.UnixMicroFromTime(time.Now())
	if newTS <= oldTS {
		newTS = oldTS + 1
	}
	bi := common.NewBlockInfo(blk.Height()+1, newTS)

	// execute transaction
	rct, err := c.sm.ExecuteTransaction(
		blk.Result(),
		blk.NextValidators().Hash(),
		params.RawMessage(),
		bi,
	)
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, c.debug)
	}
	if status := rct.Status(); status != module.StatusSuccess {
		if rctex, ok := rct.(txresult.Receipt); ok {
			if err = rctex.Reason(); err != nil {
				return nil, jsonrpc.ErrScore(rctex.Reason(), c.debug)
			}
		}
		return nil, jsonrpc.ErrScoreWithStatus(status)
	}
	steps := new(common.HexInt)
	steps.Set(rct.StepUsed())
	return steps, nil
}

type MissingTransactionInfo interface {
	ReplaceID(height int64, id []byte) []byte
	GetLocationOf(id []byte) (int64, int, bool)
}

func findMissingTransactionInfoOf(cid int) MissingTransactionInfo {
	// for ICON Mainnet
	if cid == 0x1 {
		return iconMissedTransactions
	}
	return nil
}

func getTraceForRosetta(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	var c contextWithSM
	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	var param RosettaTraceParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, c.debug)
	}

	blk, txInfo, err := findBlockAndTxInfoByRosettaTraceParam(&c, param)
	if err != nil {
		return nil, err
	}

	csi, err := c.bm.NewConsensusInfo(blk)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	nblk, err := c.bm.GetBlockByHeight(blk.Height() + 1)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	tr1, err := c.sm.CreateInitialTransition(blk.Result(), blk.NextValidators())
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	tr2, err := c.sm.CreateTransition(tr1, blk.NormalTransactions(), blk, csi, true)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}
	tr2 = c.sm.PatchTransition(tr2, nblk.PatchTransactions(), nblk)

	rl, err := c.sm.ReceiptListFromResult(nblk.Result(), module.TransactionGroupNormal)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	var replacer trace.TxHashReplacer
	if mt := findMissingTransactionInfoOf(c.chain.CID()); mt != nil {
		replacer = mt.ReplaceID
	}
	cb := &traceCallback{
		channel: make(chan interface{}, 10),
		bt:      trace.NewBalanceTracer(10, replacer),
	}
	ti := module.TraceInfo{
		TraceMode:  module.TraceModeBalanceChange,
		TraceBlock: trace.NewTraceBlock(blk.ID(), rl),
		Callback:   cb,
	}
	if txInfo != nil {
		ti.Range = module.TraceRangeTransaction
		ti.Group = module.TransactionGroupNormal
		ti.Index = txInfo.Index()
	} else {
		if len(param.Tx) > 0 {
			ti.Range = module.TraceRangeBlockTransaction
		} else {
			ti.Range = module.TraceRangeBlock
		}
	}
	canceller, err := tr2.ExecuteForTrace(ti)
	if err != nil {
		return nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
	}

	timer := time.After(time.Second * 60)
	for {
		select {
		case <-timer:
			canceller()
			return nil, jsonrpc.ErrorCodeSystemTimeout.Errorf(
				"Not enough time to get result of %+v", param)
		case <-cb.channel:
			return cb.balanceChangeToJSON(blk), nil
		}
	}
}

func findBlockAndTxInfoByRosettaTraceParam(
	c *contextWithSM,
	param RosettaTraceParam,
) (module.Block, module.TransactionInfo, error) {
	var blk module.Block
	var txInfo module.TransactionInfo
	var err error

	if len(param.Tx) > 0 {
		if strings.HasPrefix(string(param.Tx), "bx") {
			if blk, err = c.GetBlockByID(param.Tx.Bytes()); err != nil {
				return nil, nil, jsonrpc.ErrorCodeNotFound.Wrap(err, c.debug)
			}
		} else {
			txInfo, err = getTransactionInfo(param.Tx.Bytes(), c)
			if errors.NotFoundError.Equals(err) {
				if c.sm.HasTransaction(param.Tx.Bytes()) {
					return nil, nil, jsonrpc.ErrorCodePending.New("Pending")
				}
				return nil, nil, jsonrpc.ErrorCodeNotFound.Wrap(err, c.debug)
			} else if err != nil {
				return nil, nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
			}
			if txInfo.Group() == module.TransactionGroupPatch {
				return nil, nil, jsonrpc.ErrorCodeInvalidParams.New("Patch transaction can't be replayed")
			}
			_, err = txInfo.GetReceipt()
			if block.ResultNotFinalizedError.Equals(err) {
				return nil, nil, jsonrpc.ErrorCodeExecuting.New("Executing")
			} else if err != nil {
				return nil, nil, jsonrpc.ErrorCodeSystem.Wrap(err, c.debug)
			}
			blk = txInfo.Block()
			if err = c.CheckBaseHeight(blk.Height()); err != nil {
				return nil, nil, err
			}
		}
	} else if len(param.Block) > 0 {
		blk, err = c.GetBlockByID(param.Block.Bytes())
	} else if len(param.Height) > 0 {
		blk, err = c.GetBlockByHeight(param.Height)
	} else {
		// Last block
		if blk, err = c.bm.GetLastBlock(); err == nil && blk.Height() > 0 {
			// Transactions in the last block are not finalized in onTheNext blockchain,
			// so the previous one of the last block is actually considered the last block in rosetta_getTrace()
			blk, err = c.bm.GetBlockByHeight(blk.Height() - 1)
		}
		err = c.AsRPCError(err)
	}
	return blk, txInfo, err
}

type missingTransactionInfo struct {
	blk   module.Block
	index int
	sm    module.ServiceManager
	nblk  module.Block
}

func (m *missingTransactionInfo) Block() module.Block {
	return m.blk
}

func (m *missingTransactionInfo) Index() int {
	return m.index
}

func (m *missingTransactionInfo) Group() module.TransactionGroup {
	return module.TransactionGroupNormal
}

func (m *missingTransactionInfo) Transaction() (module.Transaction, error) {
	return nil, errors.UnsupportedError.Errorf("Not implemented")
}

func (m *missingTransactionInfo) GetReceipt() (module.Receipt, error) {
	if m.nblk != nil {
		rl, err := m.sm.ReceiptListFromResult(
			m.nblk.Result(), m.Group())
		if err != nil {
			return nil, err
		}
		rct, err := rl.Get(m.index)
		if err != nil {
			return nil, err
		}
		return rct, nil
	} else {
		return nil, block.ErrResultNotFinalized
	}
}

func getTransactionInfo(
	txHash []byte, c *contextWithSM) (module.TransactionInfo, error) {
	if mt := findMissingTransactionInfoOf(c.chain.CID()); mt != nil {
		height, index, ok := mt.GetLocationOf(txHash)
		if ok {
			if blk, err := c.bm.GetBlockByHeight(height); err == nil {
				nblk, _ := c.bm.GetBlockByHeight(height + 1)
				return &missingTransactionInfo{blk, index, c.sm, nblk}, nil
			}
		}
	}
	return c.bm.GetTransactionInfo(txHash)
}

func RosettaMethodRepository(mtr *metric.JsonrpcMetric) *jsonrpc.MethodRepository {
	mr := jsonrpc.NewMethodRepository(mtr)

	mr.RegisterMethod("rosetta_getTrace", getTraceForRosetta)

	return mr
}
