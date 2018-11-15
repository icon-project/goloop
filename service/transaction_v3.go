package service

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

type (
	transactionV3 struct {
		Version   common.HexInt16  `json:"version"` // V3 only
		From      common.Address   `json:"from"`
		To        common.Address   `json:"to"`
		Value     common.HexInt    `json:"value"`
		StepLimit common.HexInt    `json:"stepLimit"` // V3 only
		Fee       common.HexInt    `json:"fee"`       // V2 only
		TimeStamp common.HexInt64  `json:"timestamp"`
		NID       common.HexInt16  `json:"nid"` // V3 only
		Nonce     common.HexInt64  `json:"nonce"`
		TxHash    common.HexBytes  `json:"txHash"`  // V3 only
		Tx_Hash   common.HexBytes  `json:"tx_hash"` // V2 only
		Signature common.Signature `json:"signature"`
		// TODO data?

		raw []byte
	}

	// TODO check
	TransactionData struct {
		Method string `json:"method"`
		// TODO 이건 어떻게 할 건가?
		Params map[string]interface{} `json:"params"`
	}
)

func newTransactionLegacy(b []byte) (*transaction, error) {
	t3 := &transactionV3{Version: common.HexInt16{2}, raw: b[:]}
	if err := json.Unmarshal(b, t3); err != nil {
		return nil, err
	}

	var t *transaction
	switch t3.Version.Value {
	case 2:
		t = &transaction{
			source:    t3,
			group:     module.TransactionGroupNormal,
			version:   2,
			from:      &t3.From,
			to:        &t3.To,
			value:     &t3.Value.Int,
			stepLimit: &t3.Fee.Int,
			timestamp: t3.TimeStamp.Value,
			nid:       0,
			nonce:     t3.Nonce.Value,
			signature: &t3.Signature,
			hash:      t3.Tx_Hash.ToBytes(),
			bytes:     t3.raw,
		}
	case 3:
		t = &transaction{
			source:    t3,
			group:     module.TransactionGroupNormal,
			version:   3,
			from:      &t3.From,
			to:        &t3.To,
			value:     &t3.Value.Int,
			stepLimit: &t3.StepLimit.Int,
			timestamp: t3.TimeStamp.Value,
			nid:       int(t3.NID.Value),
			nonce:     t3.Nonce.Value,
			signature: &t3.Signature,
			hash:      t3.Tx_Hash.ToBytes(),
			bytes:     t3.raw,
		}
	default:
		return nil, errors.New("Wrong transaction version: " + strconv.Itoa(int(t3.Version.Value)))
	}
	return t, nil
}

func (t *transactionV3) bytes() []byte {
	if t.raw == nil {
		panic("It shouldn't be happened")
	}
	return t.raw
}

// TODO temporary for compile with block_V1.go
func NewTransactionV3(b []byte) (module.Transaction, error) {
	return newTransactionLegacy(b)
}
