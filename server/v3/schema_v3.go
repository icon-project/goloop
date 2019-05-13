package v3

import (
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
)

// swagger:model blockHeightParam
type BlockHeightParam struct {
	Height jsonrpc.HexInt `json:"height" validate:"required,t_int"`
}

// swagger:model blockHashParam
type BlockHashParam struct {
	Hash jsonrpc.HexBytes `json:"hash" validate:"required,t_hash"`
}

// swagger:model callParam
type CallParam struct {
	FromAddress jsonrpc.Address `json:"from" validate:"optional,t_addr_eoa"`
	ToAddress   jsonrpc.Address `json:"to" validate:"required,t_addr_score"`
	DataType    string          `json:"dataType" validate:"required,call"`
	Data        interface{}     `json:"data"`
}

// swagger:model addressParam
type AddressParam struct {
	Address jsonrpc.Address `json:"address" validate:"required,t_addr"`
}

// swagger:model scoreAddressParam
type ScoreAddressParam struct {
	Address jsonrpc.Address `json:"address" validate:"required,t_addr_score"`
}

// swagger:model transactionHashParam
type TransactionHashParam struct {
	Hash jsonrpc.HexBytes `json:"txHash" validate:"required,t_hash"`
}

// swagger:model transactionParam
type TransactionParam struct {
	Version     jsonrpc.HexInt  `json:"version" validate:"required,t_int"`
	FromAddress jsonrpc.Address `json:"from" validate:"required,t_addr_eoa"`
	ToAddress   jsonrpc.Address `json:"to" validate:"required,t_addr"`
	Value       jsonrpc.HexInt  `json:"value,omitempty" validate:"optional,t_int"`
	StepLimit   jsonrpc.HexInt  `json:"stepLimit" validate:"required,t_int"`
	Timestamp   jsonrpc.HexInt  `json:"timestamp" validate:"required,t_int"`
	NetworkID   jsonrpc.HexInt  `json:"nid" validate:"required,t_int"`
	Nonce       jsonrpc.HexInt  `json:"nonce,omitempty" validate:"optional,t_int"`
	Signature   string          `json:"signature" validate:"required,t_sig"`
	DataType    string          `json:"dataType,omitempty" validate:"optional,call|deploy|message"`
	Data        interface{}     `json:"data,omitempty"`
}

// swagger:model dataHashParam
type DataHashParam struct {
	Hash jsonrpc.HexBytes `json:"hash" validate:"required,t_hash"`
}

// swagger:model proofResultParam
type ProofResultParam struct {
	BlockHash jsonrpc.HexBytes `json:"hash" validate:"required,t_hash"`
	Index     jsonrpc.HexInt   `json:"index" validate:"required,t_int"`
}

// convert TransactionList to []Transaction
func convertTransactionList(txs module.TransactionList) ([]interface{}, error) {
	list := new([]interface{})
	for it := txs.Iterator(); it.Has(); it.Next() {
		tx, _, err := it.Get()
		switch tx.Version() {
		case module.TransactionVersion2:
			res, err := tx.ToJSON(module.TransactionVersion2)
			*list = append(*list, res)
			if err != nil {
				return nil, jsonrpc.ErrInternal()
			}
		case module.TransactionVersion3:
			res, err := tx.ToJSON(module.TransactionVersion3)
			*list = append(*list, res)
			if err != nil {
				return nil, jsonrpc.ErrInternal()
			}
		}
		if err != nil {
			return nil, jsonrpc.ErrInternal()
		}
	}
	return *list, nil
}

// JSON-RPC Response Result for swagger spec

// swagger:model block
type _blockV2 struct {
	Version            string         `json:"version"`
	PrevBlockHash      string         `json:"prev_block_hash"`
	MerkleTreeRootHash string         `json:"merkle_tree_root_hash"`
	Timestamp          int64          `json:"time_stamp"`
	Transactions       []_transaction `json:"confirmed_transaction_list"`
	BlockHash          string         `json:"block_hash"`
	Height             int64          `json:"height"`
	PeerID             string         `json:"peer_id"`
	Signature          string         `json:"signature"`
}

// swagger:model transaction
type _transaction struct {
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

// swagger:model scoreApi
type _scoreApi struct {
	ApiType    string            `json:"type"`
	ApiName    string            `json:"name"`
	Input      []_scoreApiInput  `json:"inputs"`
	Output     []_scoreApiOutput `json:"outputs"`
	IsReadOnly string            `json:"readonly,omitempty"`
	Payable    string            `json:"payable,omitempty"`
}

// swagger:model scoreApiInput
type _scoreApiInput struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Indexed string `json:"indexed,omitempty"`
}

// swagger:model scoreApiOutput
type _scoreApiOutput struct {
	Type string `json:"type"`
}

// swagger:model transactionResult
type _transactionResult struct {
	Status             string     `json:"status"`
	ToAddress          string     `json:"to"`
	TxFailure          _txFailure `json:"failure,omitempty"`
	TransactionHash    string     `json:"txHash"`
	TransactionIndex   string     `json:"txIndex"`
	BlockHeight        string     `json:"blockHeight"`
	BlockHash          string     `json:"blockHash"`
	CumulativeStepUsed string     `json:"cumulativeStepUsed"`
	StepUsed           string     `json:"stepUsed"`
	StepPrice          string     `json:"stepPrice"`
	ScoreAddress       string     `json:"scoreAddress,omitempty"`
	EventLogs          []eventLog `json:"eventLogs"`
	LogsBloom          string     `json:"logsBloom,omitempty"`
}

// swagger:model transactionFailure
type _txFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// swagger:model eventLog
type eventLog struct {
	ScoreAddress string   `json:"scoreAddress"`
	Indexed      []string `json:"indexed"`
	Data         []string `json:"data"`
}
