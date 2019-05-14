package v3

import (
	"bytes"
	"encoding/hex"
	"strconv"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/service"
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

// swagger:operation POST /api/v3/{nid}/:getLastBlock v3 getLastBlock
//
// icx_getLastBlock
//
// icx_getLastBlock
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: getLastBlock
//     name: getLastBlock
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_getLastBlock
//
// responses:
//   200:
//     description: Success
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcResult'
//         - type: object
//           properties:
//             result:
//               $ref: '#/definitions/block'
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
func getLastBlock(ctx *jsonrpc.Context, _ *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	bm := chain.BlockManager()

	block, err := bm.GetLastBlock()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	blockJson, err := block.ToJSON(jsonRpcApiVersion)
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	result := blockJson.(map[string]interface{})
	txList := result["confirmed_transaction_list"].(module.TransactionList)
	confirmedTxList, err := convertTransactionList(txList)
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	result["confirmed_transaction_list"] = confirmedTxList

	return result, nil
}

// swagger:operation POST /api/v3/{nid}/:getBlockByHeight v3 getBlockByHeight
//
// icx_getBlockByHeight
//
// icx_getBlockByHeight
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: getBlockByHeight
//     name: getBlockByHeight
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//         - type: object
//           properties:
//             params:
//               $ref: '#/definitions/blockHeightParam'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_getBlockByHeight
//         params:
//           height: "0x100"
//
// responses:
//   200:
//     description: Success
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcResult'
//         - type: object
//           properties:
//             result:
//               $ref: '#/definitions/block'
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
func getBlockByHeight(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param BlockHeightParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	bm := chain.BlockManager()

	block, err := bm.GetBlockByHeight(param.Height.Value())
	if errors.NotFoundError.Equals(err) {
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	blockJson, err := block.ToJSON(jsonRpcApiVersion)
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	result := blockJson.(map[string]interface{})
	txList := result["confirmed_transaction_list"].(module.TransactionList)
	confirmedTxList, err := convertTransactionList(txList)
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	result["confirmed_transaction_list"] = confirmedTxList

	return result, nil
}

// swagger:operation POST /api/v3/{nid}/:getBlockByHash v3 getBlockByHash
//
// icx_getBlockByHash
//
// icx_getBlockByHash
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: getBlockByHash
//     name: getBlockByHash
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//         - type: object
//           properties:
//             params:
//               $ref: '#/definitions/blockHashParam'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_getBlockByHash
//         params:
//           hash: "0x3a317563d4eefa52af7733e1ea68fe29daf77e78c8fb1c66e699b6f35673141e"
//
// responses:
//   200:
//     description: Success
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcResult'
//         - type: object
//           properties:
//             result:
//               $ref: '#/definitions/block'
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
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
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	blockJson, err := block.ToJSON(jsonRpcApiVersion)
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	result := blockJson.(map[string]interface{})
	txList := result["confirmed_transaction_list"].(module.TransactionList)
	confirmedTxList, err := convertTransactionList(txList)
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	result["confirmed_transaction_list"] = confirmedTxList

	return result, nil
}

// swagger:operation POST /api/v3/{nid}/:call v3 call
//
// icx_call
//
// icx_call
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: call
//     name: call
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//         - type: object
//           properties:
//             params:
//               $ref: '#/definitions/callParam'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_call
//
// responses:
//   200:
//     description: Success
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
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
		return nil, jsonrpc.ErrScore(err, false)
	} else {
		return result, nil
	}
}

// swagger:operation POST /api/v3/{nid}/:getBalance v3 getBalance
//
// icx_getBalance
//
// icx_getBalance
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: getBalance
//     name: getBalance
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//         - type: object
//           properties:
//             params:
//               $ref: '#/definitions/addressParam'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_getBalance
//         params:
//           address: "hxb0776ee37f5b45bfaea8cff1d8232fbb6122ec32"
//
// responses:
//   200:
//     description: Success
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcResult'
//         - type: object
//           properties:
//             result:
//               $ref: '#/definitions/HexInt'
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
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
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}
	b, err := sm.GetBalance(block.Result(), param.Address.Address())
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}
	balance.Set(b)
	return &balance, nil
}

// swagger:operation POST /api/v3/{nid}/:getScoreApi v3 getScoreApi
//
// icx_getScoreApi
//
// icx_getScoreApi
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: getScoreApi
//     name: getScoreApi
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//         - type: object
//           properties:
//             params:
//               $ref: '#/definitions/scoreAddressParam'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_getScoreApi
//         params:
//           address: "cxb0776ee37f5b45bfaea8cff1d8232fbb6122ec32"
//
// responses:
//   200:
//     description: Success
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcResult'
//         - type: object
//           properties:
//             result:
//               $ref: '#/definitions/scoreApi'
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
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
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}
	sm := chain.ServiceManager()
	info, err := sm.GetAPIInfo(b.Result(), param.Address.Address())
	if err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}
	if jso, err := info.ToJSON(jsonRpcApiVersion); err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	} else {
		return jso, nil
	}
}

// swagger:operation POST /api/v3/{nid}/:getTotalSupply v3 getTotalSupply
//
// icx_getTotalSupply
//
// icx_getTotalSupply
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: getTotalSupply
//     name: getTotalSupply
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_getTotalSupply
//
// responses:
//   200:
//     description: Success
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcResult'
//         - type: object
//           properties:
//             result:
//               $ref: '#/definitions/HexInt'
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
func getTotalSupply(ctx *jsonrpc.Context, _ *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()
	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}
	bm := chain.BlockManager()
	b, err := bm.GetLastBlock()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}
	sm := chain.ServiceManager()

	var tsValue common.HexInt
	ts, err := sm.GetTotalSupply(b.Result())
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}
	tsValue.Set(ts)

	return &tsValue, nil
}

// swagger:operation POST /api/v3/{nid}/:getTransactionResult v3 getTransactionResult
//
// icx_getTransactionResult
//
// icx_getTransactionResult
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: getTransactionResult
//     name: getTransactionResult
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//         - type: object
//           properties:
//             params:
//               $ref: '#/definitions/transactionHashParam'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_getTransactionResult
//         params:
//           txHash: "0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238"
//
// responses:
//   200:
//     description: Success
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcResult'
//         - type: object
//           properties:
//             result:
//               $ref: '#/definitions/transactionResult'
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
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
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	block := txInfo.Block()
	receipt := txInfo.GetReceipt()
	if receipt == nil {
		return nil, jsonrpc.ErrorCodeExecuting.New("Executing")
	}
	res, err := receipt.ToJSON(jsonRpcApiVersion)
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	result := res.(map[string]interface{})
	result["blockHash"] = "0x" + hex.EncodeToString(block.ID())
	result["blockHeight"] = "0x" + strconv.FormatInt(int64(block.Height()), 16)
	result["txIndex"] = "0x" + strconv.FormatInt(int64(txInfo.Index()), 16)

	return result, nil
}

// swagger:operation POST /api/v3/{nid}/:getTransactionByHash v3 getTransactionByHash
//
// icx_getTransactionByHash
//
// icx_getTransactionByHash
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: getTransactionByHash
//     name: getTransactionByHash
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//         - type: object
//           properties:
//             params:
//               $ref: '#/definitions/transactionHashParam'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_getTransactionByHash
//         params:
//           txHash: "0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238"
//
// responses:
//   200:
//     description: Success
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcResult'
//         - type: object
//           properties:
//             result:
//               $ref: '#/definitions/transaction'
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
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
		return nil, jsonrpc.ErrorCodeNotFound.New("Not Found")
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	tx := txInfo.Transaction()

	var result interface{}
	switch tx.Version() {
	case module.TransactionVersion2:
		result, err = tx.ToJSON(module.TransactionVersion2)
		if err != nil {
			return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
		}
	case module.TransactionVersion3:
		result, err = tx.ToJSON(module.TransactionVersion3)
		if err != nil {
			return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
		}
	default:
		return nil, jsonrpc.ErrorCodeServer.Errorf(
			"Unknown transaction version=%d", tx.Version())
	}

	return result, nil
}

// swagger:operation POST /api/v3/{nid}/:sendTransaction v3 sendTransaction
//
// icx_sendTransaction
//
// icx_sendTransaction
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: sendTransaction
//     name: sendTransaction
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//         - type: object
//           properties:
//             params:
//               $ref: '#/definitions/transactionParam'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_sendTransaction
//
// responses:
//   200:
//     description: Success
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcResult'
//         - type: object
//           properties:
//             result:
//               $ref: '#/definitions/HexBytes'
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
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
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	result := "0x" + hex.EncodeToString(hash)

	return result, nil
}

// swagger:operation POST /api/v3/{nid}/:getDataByHash extension getDataByHash
//
// icx_getDataByHash
//
// icx_getDataByHash
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: getDataByHash
//     name: getDataByHash
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//         - type: object
//           properties:
//             params:
//               $ref: '#/definitions/dataHashParam'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_getDataByHash
//
// responses:
//   200:
//     description: Success
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
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
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}
	value, err := bucket.Get(param.Hash.Bytes())
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	if value == nil {
		return nil, jsonrpc.ErrorCodeNotFound.New("Fail to find data")
	}

	return value, nil
}

// swagger:operation POST /api/v3/{nid}/:getBlockHeaderByHeight extension getBlockHeaderByHeight
//
// icx_getBlockHeaderByHeight
//
// icx_getBlockHeaderByHeight
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: getBlockHeaderByHeight
//     name: getBlockHeaderByHeight
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//         - type: object
//           properties:
//             params:
//               $ref: '#/definitions/blockHeightParam'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_getBlockHeaderByHeight
//
// responses:
//   200:
//     description: Success
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
func getBlockHeaderByHeight(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param BlockHeightParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	bm := chain.BlockManager()

	block, err := bm.GetBlockByHeight(param.Height.Value())
	if errors.NotFoundError.Equals(err) {
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	buf := bytes.NewBuffer(nil)
	if err := block.MarshalHeader(buf); err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	return buf.Bytes(), nil
}

// swagger:operation POST /api/v3/{nid}/:getVotesByHeight extension getVotesByHeight
//
// icx_getVotesByHeight
//
// icx_getVotesByHeight
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: getVotesByHeight
//     name: getVotesByHeight
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//         - type: object
//           properties:
//             params:
//               $ref: '#/definitions/blockHeightParam'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_getVotesByHeight
//
// responses:
//   200:
//     description: Success
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
func getVotesByHeight(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param BlockHeightParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
	}

	chain, err := ctx.Chain()
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	cs := chain.Consensus()

	votes, err := cs.GetVotesByHeight(param.Height.Value())
	if errors.NotFoundError.Equals(err) {
		return nil, jsonrpc.ErrorCodeNotFound.Wrap(err, debug)
	} else if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	return votes.Bytes(), nil
}

// swagger:operation POST /api/v3/{nid}/:getProofForResult extension getProofForResult
//
// icx_getProofForResult
//
// icx_getProofForResult
//
// ---
// consumes:
//   - application/json
//
// produces:
//   - application/json
//
// parameters:
//   - description: getProofForResult
//     name: getProofForResult
//     in: body
//     required: true
//     schema:
//       allOf:
//         - $ref: '#/definitions/JsonRpcRequest'
//         - type: object
//           properties:
//             params:
//               $ref: '#/definitions/proofResultParam'
//       example:
//         id: 1001
//         jsonrpc: "2.0"
//         method: icx_getProofForResult
//
// responses:
//   200:
//     description: Success
//   default:
//     description: JSON-RPC Error
//     schema:
//       $ref: '#/definitions/JsonRpcErrorResponse'
func getProofForResult(ctx *jsonrpc.Context, params *jsonrpc.Params) (interface{}, error) {
	debug := ctx.IncludeDebug()

	var param ProofResultParam
	if err := params.Convert(&param); err != nil {
		return nil, jsonrpc.ErrorCodeInvalidParams.Wrap(err, debug)
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
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	blockResult := block.Result()
	receiptList := sm.ReceiptListFromResult(blockResult, module.TransactionGroupNormal)
	proofs, err := receiptList.GetProof(int(param.Index.Value()))
	if err != nil {
		return nil, jsonrpc.ErrorCodeServer.Wrap(err, debug)
	}

	return proofs, nil
}
