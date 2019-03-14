package v3

import (
	"github.com/icon-project/goloop/module"
	"github.com/osamingo/jsonrpc"
)

/*
	SCHEMA_V3: dict = {
		"icx_getLastBlock": icx_getLastBlock,
		"icx_getBlockByHeight": icx_getBlockByHeight_v3,
		"icx_getBlockByHash": icx_getBlockByHash_v3,
		"icx_call": icx_call_v3,
		"icx_getBalance": icx_getBalance_v3,
		"icx_getScoreApi": icx_getScoreApi_v3,
		"icx_getTotalSupply": icx_getTotalSupply,
		"icx_getTransactionResult": icx_getTransactionResult_v3,
		"icx_getTransactionByHash": icx_getTransactionByHash_v3,
		"icx_sendTransaction": icx_sendTransaction_v3,
		"ise_getStatus": ise_getStatus_v3
	}
*/

const (
	getLastBlock         string = "icx_getLastBlock"
	getBlockByHeight     string = "icx_getBlockByHeight"
	getBlockByHash       string = "icx_getBlockByHash"
	call                 string = "icx_call"
	getBalance           string = "icx_getBalance"
	getScoreApi          string = "icx_getScoreApi"
	getTotalSupply       string = "icx_getTotalSupply"
	getTransactionResult string = "icx_getTransactionResult"
	getTransactionByHash string = "icx_getTransactionByHash"
	sendTransaction      string = "icx_sendTransaction"

	getStatus string = "ise_getStatus"

	getDataByHash          string = "icx_getDataByHash"
	getBlockHeaderByHeight string = "icx_getBlockHeaderByHeight"
	getVotesByHeight       string = "icx_getVotesByHeight"
	getProofForResult      string = "icx_getProofForResult"
)

func MethodRepository(ch module.Chain, bm module.BlockManager, sm module.ServiceManager, cs module.Consensus) *jsonrpc.MethodRepository {

	v3 := jsonrpc.NewMethodRepository()

	// api v3
	v3.RegisterMethod(getLastBlock, getLastBlockHandler{bm: bm}, nil, blockV2{})
	v3.RegisterMethod(getBlockByHeight, getBlockByHeightHandler{bm: bm}, getBlockByHeightParam{}, blockV2{})
	v3.RegisterMethod(getBlockByHash, getBlockByHashHandler{bm: bm}, getBlockByHashParam{}, blockV2{})
	v3.RegisterMethod(call, callHandler{bm: bm, sm: sm}, callParam{}, nil)
	v3.RegisterMethod(getBalance, getBalanceHandler{bm: bm, sm: sm}, getBalanceParam{}, nil)
	v3.RegisterMethod(getScoreApi, getScoreApiHandler{}, getScoreApiParam{}, getScoreApiResult{})
	v3.RegisterMethod(getTotalSupply, getTotalSupplyeHandler{}, nil, nil)
	v3.RegisterMethod(getTransactionResult, getTransactionResultHandler{bm: bm}, transactionHashParam{}, transactionResult{})
	v3.RegisterMethod(getTransactionByHash, getTransactionByHashHandler{bm: bm}, transactionHashParam{}, transactionV3{})
	v3.RegisterMethod(sendTransaction, sendTransactionHandler{sm: sm}, sendTransactionParamV3{}, nil)
	v3.RegisterMethod(getStatus, getStatusHandler{}, getStatusParam{}, nil)

	// api v3.1
	v3.RegisterMethod(getDataByHash, getDataByHashHandler{db: ch.Database()}, nil, nil)
	v3.RegisterMethod(getBlockHeaderByHeight, getBlockHeaderByHeightHandler{bm: bm}, nil, nil)
	v3.RegisterMethod(getVotesByHeight, getVotesByHeightHandler{cs: cs}, nil, nil)
	v3.RegisterMethod(getProofForResult, getProofForResultHandler{bm: bm, sm: sm}, nil, nil)
	return v3
}
