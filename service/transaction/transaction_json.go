package transaction

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
)

type transactionV3JSON struct {
	transactionV3Data
	Fee      common.HexInt   `json:"fee"`               // V2 only
	TxHash   common.HexBytes `json:"txHash,omitempty"`  // V3 only
	TxHashV2 common.HexBytes `json:"tx_hash,omitempty"` // V2 only

	raw    []byte
	txHash []byte
}

var (
	v2FieldInclusion = map[string]bool(nil)
	v2FieldExclusion = map[string]bool{
		"method":    true,
		"signature": true,
		"tx_hash":   true,
	}
)

func (tx *transactionV3JSON) calcHash() ([]byte, error) {
	var bs []byte
	var data map[string]interface{}
	var err error
	if err = json.Unmarshal(tx.raw, &data); err != nil {
		return nil, err
	}
	bs, err = SerializeMap(data, v2FieldInclusion, v2FieldExclusion)
	if err != nil {
		log.Println("Serialize FAILs")
		log.Println("JSON", string(tx.raw))
		return nil, err
	}
	bs = append([]byte("icx_sendTransaction."), bs...)

	return crypto.SHA3Sum256(bs), nil
}

func (tx *transactionV3JSON) ID() []byte {
	if err := tx.updateTxHash(); err != nil {
		log.Panicf("Fail to calculate TxHash err=%+v", err)
	}
	return tx.txHash
}

func (tx *transactionV3JSON) updateTxHash() error {
	if tx.txHash == nil {
		h, err := tx.calcHash()
		if err != nil {
			return err
		}
		tx.txHash = h
	}
	return nil
}

func (tx *transactionV3JSON) verifySignature() error {
	pk, err := tx.Signature.RecoverPublicKey(tx.txHash)
	if err != nil {
		return err
	}
	addr := common.NewAccountAddressFromPublicKey(pk)
	if addr.Equal(&tx.From) {
		return nil
	}
	return errors.New("InvalidSignature")
}

func (tx *transactionV3JSON) Timestamp() int64 {
	return tx.TimeStamp.Value
}

func newTransactionV2V3FromJSON(js []byte) (Transaction, error) {
	genjs := new(genesisV3JSON)
	if err := json.Unmarshal(js, genjs); err != nil {
		return nil, err
	}
	if len(genjs.Accounts) != 0 {
		genjs.raw = make([]byte, len(js))
		copy(genjs.raw, js)

		return &genesisV3{genesisV3JSON: genjs}, nil
	}

	txjs := new(transactionV3JSON)
	txjs.Version.Value = 2
	if err := json.Unmarshal(js, txjs); err != nil {
		return nil, err
	}
	txjs.raw = make([]byte, len(js))
	copy(txjs.raw, js)

	switch txjs.Version.Value {
	case 2:
		return &transactionV2{transactionV3JSON: txjs}, nil
	case 3:
		tx, err := newTransactionV3FromJSON(txjs)
		return tx, err
	default:
		return nil, errors.New("IllegalVersion:" + txjs.Version.String())
	}
}
