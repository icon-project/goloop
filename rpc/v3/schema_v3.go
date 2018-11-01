package v3

import (
	"github.com/asaskevich/govalidator"
)

// JSON-RPC Request Params
type (
	GetBlockByHeightParam struct {
		BlockHeight string `json:"height" valid:"t_int,required"`
	}

	GetBlockByHashParam struct {
		BlockHash string `json:"hash" valid:"t_hash,required"`
	}

	CallParam struct {
		FromAddress string `json:"from" valid:"t_addr_eoa"`
		ToAddress   string `json:"to" valid:"t_addr_score,required"`
		DataType    string `json:"dataType" valid:"required"`
		Data        Data   `json:"data" valid:"required"`
	}

	GetBalanceParam struct {
		Address string `json:"address" valid:"t_addr,required"`
	}

	GetScoreApiParam struct {
		Address string `json:"address" valid:"t_addr_score,required"`
	}

	TransactionHashParam struct {
		TransactionHash string `json:"txHash" valid:"t_hash,required"`
	}

	SendTransactionParam struct {
		Version     string `json:"version" valid:"t_int,required"`
		FromAddress string `json:"from" valid:"t_addr_eoa,required"`
		ToAddress   string `json:"to" valid:"t_addr"`
		Value       string `json:"value" valid:"t_int"`
		Message     string `json:"message"`
		StepLimit   string `json:"stepLimit" valid:"t_int,required"`
		Timestamp   string `json:"timestamp" valid:"t_int,required"`
		NetworkID   string `json:"nid" valid:"t_int,required"`
		Nonce       string `json:"nonce" valid:"t_int"`
		Signature   string `json:"signature" valid:"t_sig,required"`
		DataType    string `json:"dataType"`
		Data        Data   `json:"data"`
	}

	GetStatusParam struct {
		StatusFilter []string `json:"filter"`
	}

	Data struct{}
)

// JSON-RPC Response Result
type EchoResult struct {
	Method string `json:"method"`
}

func validator(s interface{}) (bool, error) {
	return govalidator.ValidateStruct(s)
}
