package v3

import (
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
	GetLastBlock         string = "icx_getLastBlock"
	GetBlockByHeight     string = "icx_getBlockByHeight"
	GetBlockByHash       string = "icx_getBlockByHash"
	Call                 string = "icx_call"
	GetBalance           string = "icx_getBalance"
	GetScoreApi          string = "icx_getScoreApi"
	GetTotalSupply       string = "icx_getTotalSupply"
	GetTransactionResult string = "icx_getTransactionResult"
	GetTransactionByHash string = "icx_getTransactionByHash"
	SendTransaction      string = "icx_sendTransaction"
	GetStatus            string = "ise_getStatus"
)

func MethodRepository() *jsonrpc.MethodRepository {

	v3 := jsonrpc.NewMethodRepository()

	// api v3
	v3.RegisterMethod(GetLastBlock, GetLastBlockHandler{}, nil, EchoResult{})
	v3.RegisterMethod(GetBlockByHeight, GetBlockByHeightHandler{}, GetBlockByHeightParam{}, EchoResult{})
	v3.RegisterMethod(GetBlockByHash, GetBlockByHashHandler{}, GetBlockByHashParam{}, EchoResult{})
	v3.RegisterMethod(Call, CallHandler{}, nil, EchoResult{})
	v3.RegisterMethod(GetBalance, GetBalanceHandler{}, GetBalanceParam{}, nil)
	v3.RegisterMethod(GetScoreApi, GetScoreApiHandler{}, GetScoreApiParam{}, EchoResult{})
	v3.RegisterMethod(GetTotalSupply, GetTotalSupplyeHandler{}, nil, nil)
	v3.RegisterMethod(GetTransactionResult, GetTransactionResultHandler{}, TransactionHashParam{}, EchoResult{})
	v3.RegisterMethod(GetTransactionByHash, GetTransactionByHashHandler{}, TransactionHashParam{}, EchoResult{})
	v3.RegisterMethod(SendTransaction, SendTransactionHandler{}, nil, EchoResult{})
	v3.RegisterMethod(GetStatus, GetStatusHandler{}, GetStatusParam{}, EchoResult{})

	return v3
}
