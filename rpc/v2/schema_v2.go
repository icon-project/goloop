package v2

import (
	"log"

	"github.com/asaskevich/govalidator"
	"github.com/osamingo/jsonrpc"
)

// JSON-RPC Request Params
type SendTransactionParam struct {
	FromAddress     string `json:"from" valid:"t_addr_eoa,required"`
	ToAddress       string `json:"to" valid:"t_addr_eoa,required"`
	Value           string `json:"value" valid:"t_int,required"`
	Fee             string `json:"fee" valid:"t_int,required"`
	Timestamp       string `json:"timestamp" valid:"int,required"`
	Nonce           string `json:"nonce" valid:"int,optional"`
	TransactionHash string `json:"tx_hash" valid:"t_hash_v2,required"`
	Signature       string `json:"signature" valid:"t_sig,required"`
}

type getTransactionResultParam struct {
	TransactionHash string `json:"tx_hash" valid:"t_hash_v2,required"`
}

type getBalanceParam struct {
	Address string `json:"address" valid:"t_addr,required"`
}

type getBlockByHashParam struct {
	BlockHash string `json:"hash" valid:"t_hash_v2,required"`
}

type getBlockByHeightParam struct {
	BlockHeight string `json:"height" valid:"int,required"`
}

// JSON-RPC Response Result
type sendTranscationResult struct {
	ResponseCode    int64  `json:"response_code"`
	TransactionHash string `json:"tx_hash,omitempty"`
	Message         string `json:"message,omitempty"`
}

type blockResult struct {
	ResponseCode int64   `json:"response_code"`
	Block        blockV2 `json:"block"`
}

type blockV2 struct {
	Version            string          `json:"version"`
	PrevBlockHash      string          `json:"prev_block_hash"`
	MerkleTreeRootHash string          `json:"merkle_tree_root_hash"`
	Timestamp          uint64          `json:"time_stamp"`
	Transactions       []transactionV2 `json:"confirmed_transaction_list"`
	BlockHash          string          `json:"block_hash"`
	Height             int64           `json:"height"`
	PeerID             string          `json:"peer_id"`
	Signature          string          `json:"signature"`
}

type transactionV2 struct {
	FromAddress     string `json:"from"`
	ToAddress       string `json:"to"`
	Value           string `json:"value,omitempty"`
	Fee             string `json:"fee"`
	Timestamp       string `json:"timestamp"`
	TransactionHash string `json:"tx_hash"`
	Signature       string `json:"signature"`
	Method          string `json:"method"`
}

type getTotalSupplyResult struct {
	ResponseCode int64  `json:"response_code"`
	Response     string `json:"response"`
}

type getBalanceResult struct {
	ResponseCode int64  `json:"response_code"`
	Response     string `json:"response"`
}

type getTransactionResultResult struct {
	ResponseCode string                       `json:"response_code"`
	Response     getTransactionResultResponse `json:"response,omitempty"`
	Message      string                       `json:"message,omitempty"`
}

type getTransactionResultResponse struct {
	Code int `json:"code"`
}

// JSON-RPC Request Params Validator
func validateParam(s interface{}) *jsonrpc.Error {
	_, err := govalidator.ValidateStruct(s)
	if err != nil {
		log.Println(err.Error())
		return jsonrpc.ErrInvalidParams()
	}
	return nil
}
