package v3

import (
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
)

type BlockHeightParam struct {
	Height jsonrpc.HexInt `json:"height" validate:"required,t_int"`
}

type BlockHashParam struct {
	Hash jsonrpc.HexBytes `json:"hash" validate:"required,t_hash"`
}

type CallParam struct {
	FromAddress jsonrpc.Address `json:"from" validate:"optional,t_addr_eoa"`
	ToAddress   jsonrpc.Address `json:"to" validate:"required,t_addr_score"`
	DataType    string          `json:"dataType" validate:"required,call"`
	Data        interface{}     `json:"data"`
}

type AddressParam struct {
	Address jsonrpc.Address `json:"address" validate:"required,t_addr"`
}

type ScoreAddressParam struct {
	Address jsonrpc.Address `json:"address" validate:"required,t_addr_score"`
}

type TransactionHashParam struct {
	Hash jsonrpc.HexBytes `json:"txHash" validate:"required,t_hash"`
}

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

type DataHashParam struct {
	Hash jsonrpc.HexBytes `json:"hash" validate:"required,t_hash"`
}

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
