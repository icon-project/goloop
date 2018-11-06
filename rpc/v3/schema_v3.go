package v3

import (
	"log"

	"github.com/asaskevich/govalidator"
	"github.com/osamingo/jsonrpc"
)

// JSON-RPC Request Params
type getBlockByHeightParam struct {
	BlockHeight string `json:"height" valid:"t_int,required"`
}

type getBlockByHashParam struct {
	BlockHash string `json:"hash" valid:"t_hash,required"`
}

type callParam struct {
	FromAddress string      `json:"from" valid:"t_addr_eoa"`
	ToAddress   string      `json:"to" valid:"t_addr_score,required"`
	DataType    string      `json:"dataType" valid:"required"`
	Data        interface{} `json:"data" valid:"-"`
}

type getBalanceParam struct {
	Address string `json:"address" valid:"t_addr,required"`
}

type getScoreApiParam struct {
	Address string `json:"address" valid:"t_addr_score,required"`
}

type transactionHashParam struct {
	TransactionHash string `json:"txHash" valid:"t_hash,required"`
}

type sendTransactionParam struct {
	Version     string      `json:"version" valid:"t_int,required"`
	FromAddress string      `json:"from" valid:"t_addr_eoa,required"`
	ToAddress   string      `json:"to" valid:"t_addr,optional"`
	Value       string      `json:"value" valid:"t_int,optional"`
	StepLimit   string      `json:"stepLimit" valid:"t_int,required"`
	Timestamp   string      `json:"timestamp" valid:"t_int,required"`
	NetworkID   string      `json:"nid" valid:"t_int,required"`
	Nonce       string      `json:"nonce" valid:"t_int,optional"`
	Signature   string      `json:"signature" valid:"t_sig,required"`
	DataType    string      `json:"dataType" valid:"-"`
	Data        interface{} `json:"data" valid:"-"`
}

type getStatusParam struct {
	StatusFilter []string `json:"filter" valid:"required"`
}

// JSON-RPC Response Result
type blockV2 struct {
	Version            string          `json:"version"`
	PrevBlockHash      string          `json:"prev_block_hash"`
	MerkleTreeRootHash string          `json:"merkle_tree_root_hash"`
	Timestamp          uint64          `json:"time_stamp"`
	Transactions       []transactionV3 `json:"confirmed_transaction_list"`
	BlockHash          string          `json:"block_hash"`
	Height             int64           `json:"height"`
	PeerID             string          `json:"peer_id"`
	Signature          string          `json:"signature"`
}

type transactionV3 struct {
	Version          string      `json:"version"`
	FromAddress      string      `json:"from"`
	ToAddress        string      `json:"to"`
	Value            string      `json:"value,omitempty"`
	StepLimit        string      `json:"stepLimit"`
	Timestamp        string      `json:"timestamp"`
	NetworkID        string      `json:"nid"`
	Nonce            string      `json:"nonce,omitempty"`
	TransactionHash  string      `json:"txHash"`
	TransactionIndex string      `json:"txIndex,omitempty"`
	Signature        string      `json:"signature"`
	DataType         string      `json:"dataType,omitempty"`
	Data             interface{} `json:"data,omitempty"`
}

type getScoreApiResult struct {
	ApiType    string           `json:"type"`
	ApiName    string           `json:"name"`
	Input      []scoreApiInput  `json:"inputs"`
	Output     []scoreApiOutput `json:"outputs"`
	IsReadOnly string           `json:"readonly,omitempty"`
	Payable    string           `json:"payable,omitempty"`
}

type scoreApiInput struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Indexed string `json:"indexed,omitempty"`
}

type scoreApiOutput struct {
	Type string `json:"type"`
}

type transactionResult struct {
	Status             string     `json:"status"`
	ToAddress          string     `json:"to"`
	TxFailure          *txFailure `json:"failure,omitempty"`
	TransactionHash    string     `json:"txHash"`
	TransactionIndex   string     `json:"txIndex"`
	BlockHeight        string     `json:"blockHeight"`
	BlockHash          string     `json:"blockHash"`
	CumulativeStepUsed string     `json:"cumulativeStepUsed"`
	StepUsed           string     `json:"stepUsed"`
	StepPrice          string     `json:"stepPrice"`
	ScoreAddress       string     `json:"scoreAddress,omitempty"`
	EventLogs          []eventLog `json:"eventLogs,omitempty"`
	LogsBloom          string     `json:"logsBloom,omitempty"`
}

type txFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type eventLog struct {
	ScoreAddress string   `json:"scoreAddress"`
	Indexed      []string `json:"indexed"`
	Data         []string `json:"data"`
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
