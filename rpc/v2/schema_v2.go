package v2

import (
	"github.com/asaskevich/govalidator"
)

// JSON-RPC Request Params
type (
	SendTransactionParam struct {
		FromAddress     string `json:"from" valid:"t_addr_eoa,required"`
		ToAddress       string `json:"to" valid:"t_addr_eoa,required"`
		Value           string `json:"value" valid:"t_int,required"`
		Fee             string `json:"fee" valid:"t_int,required"`
		Timestamp       string `json:"timestamp" valid:"int,required"`
		Nonce           string `json:"nonce" valid:"int"`
		TransactionHash string `json:"tx_hash" valid:"t_hash_v2"`
		Signature       string `json:"signature" valid:"t_sig,required"`
	}

	GetTransactionResultParam struct {
		TransactionHash string `json:"tx_hash" valid:"t_hash_v2,required"`
	}

	GetBalanceParam struct {
		Address string `json:"address" valid:"t_addr,required"`
	}

	GetBlockByHashParam struct {
		BlockHash string `json:"hash" valid:"t_hash_v2,required"`
	}

	GetBlockByHeightParam struct {
		BlockHeight string `json:"height" valid:"int,required"`
	}

	GetTransactionByAddressParam struct {
		Address string `json:"address" valid:"address_eoa,required"`
		Index   int    `json:"index" valid:"int,required"`
	}
)

// JSON-RPC Response Result
type EchoResult struct {
	Method string `json:"method"`
}

func validator(s interface{}) (bool, error) {
	return govalidator.ValidateStruct(s)
}
