package transaction

import (
	"bytes"
	"encoding/json"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/log"
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
		return nil, InvalidFormat.Wrapf(err, "Serialize FAILs(%s)", string(tx.raw))
	}
	bs = append([]byte("icx_sendTransaction."), bs...)

	return crypto.SHA3Sum256(bs), nil
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

func (tx *transactionV3JSON) ID() []byte {
	if err := tx.updateTxHash(); err != nil {
		log.Debugf("Fail to calculate TxHash err=%+v", err)
		return nil
	}
	return tx.txHash
}

func (tx *transactionV3JSON) verifySignature() error {
	pk, err := tx.Signature.RecoverPublicKey(tx.txHash)
	if err != nil {
		return InvalidSignatureError.Wrap(err, "fail to recover public key")
	}
	addr := common.NewAccountAddressFromPublicKey(pk)
	if addr.Equal(&tx.From) {
		return nil
	}
	return InvalidSignatureError.New("fail to verify signature")
}

func (tx *transactionV3JSON) Timestamp() int64 {
	return tx.TimeStamp.Value
}

func newTransactionV2V3FromJSON(js []byte) (Transaction, error) {
	b := bytes.NewBuffer(nil)
	if err := json.Compact(b, js); err != nil {
		return nil, InvalidFormat.Wrap(err, "Fail on json.Compact")
	} else {
		js = b.Bytes()
	}

	genjs, err := newGenesisV3(js)
	if err == nil {
		return genjs, nil
	}

	txjs := new(transactionV3JSON)
	txjs.Version.Value = 2
	if err := json.Unmarshal(js, txjs); err != nil {
		return nil, InvalidFormat.Wrapf(err, "Invalid json for transactionV3(%s)", string(js))
	}
	txjs.raw = js

	switch txjs.Version.Value {
	case 2:
		return newTransactionV2FromJSONObject(txjs)
	case 3:
		return newTransactionV3FromJSONObject(txjs)
	default:
		return nil, InvalidVersion.Errorf("IllegalVersion:" + txjs.Version.String())
	}
}
