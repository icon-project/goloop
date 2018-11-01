package v2

import (
	"github.com/osamingo/jsonrpc"
)

/*
	SCHEMA_V2: dict =
		"icx_sendTransaction": icx_sendTransaction_v2,
		"icx_getTransactionResult": icx_getTransactionResult_v2,
		"icx_getBalance": icx_getBalance_v2,
		"icx_getTotalSupply": icx_getTotalSupply,
		"icx_getLastBlock": icx_getLastBlock,
		"icx_getBlockByHash": icx_getBlockByHash_v2,
		"icx_getBlockByHeight": icx_getBlockByHeight_v2,
		"icx_getTransactionByAddress": icx_getTransactionByAddress_v2
	}
*/

const (
	SendTransaction         string = "icx_sendTransaction"
	GetTransactionResult    string = "icx_getTransactionResult"
	GetBalance              string = "icx_getBalance"
	GetTotalSupply          string = "icx_getTotalSupply"
	GetLastBlock            string = "icx_getLastBlock"
	GetBlockByHash          string = "icx_getBlockByHash"
	GetBlockByHeight        string = "icx_getBlockByHeight"
	GetTransactionByAddress string = "icx_getTransactionByAddress"
)

func MethodRepository() *jsonrpc.MethodRepository {

	v2 := jsonrpc.NewMethodRepository()

	// api v2
	v2.RegisterMethod(SendTransaction, SendTransactionHandler{}, nil, EchoResult{})
	v2.RegisterMethod(GetTransactionResult, GetTransactionResultHandler{}, GetTransactionResultParam{}, EchoResult{})
	v2.RegisterMethod(GetBalance, GetBalanceHandler{}, GetBalanceParam{}, nil)
	v2.RegisterMethod(GetTotalSupply, GetTotalSupplyeHandler{}, nil, nil)
	v2.RegisterMethod(GetLastBlock, GetLastBlockHandler{}, nil, EchoResult{})
	v2.RegisterMethod(GetBlockByHash, GetBlockByHashHandler{}, GetBlockByHashParam{}, EchoResult{})
	v2.RegisterMethod(GetBlockByHeight, GetBlockByHeightHandler{}, GetBlockByHeightParam{}, EchoResult{})
	v2.RegisterMethod(GetTransactionByAddress, GetTransactionByAddressHandler{}, GetTransactionByAddressParam{}, EchoResult{})

	return v2
}
