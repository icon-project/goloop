package service

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"strconv"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
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
		if sig, err := t3.Signature.SerializeRSV(); err == nil {
			t = &transaction{
				source:    t3,
				group:     module.TransactionGroupNormal,
				version:   2,
				from:      t3.From,
				to:        t3.To,
				value:     &t3.Value.Int,
				stepLimit: &t3.Fee.Int,
				timestamp: t3.TimeStamp.Value,
				nid:       0,
				nonce:     t3.Nonce.Value,
				signature: sig,
				hash:      t3.Tx_Hash.ToBytes(),
				bytes:     t3.raw,
			}
		} else {
			return nil, err
		}
	case 3:
		if sig, err := t3.Signature.SerializeRSV(); err == nil {
			t = &transaction{
				source:    t3,
				group:     module.TransactionGroupNormal,
				version:   3,
				from:      t3.From,
				to:        t3.To,
				value:     &t3.Value.Int,
				stepLimit: &t3.StepLimit.Int,
				timestamp: t3.TimeStamp.Value,
				nid:       int(t3.NID.Value),
				nonce:     t3.Nonce.Value,
				signature: sig,
				hash:      t3.Tx_Hash.ToBytes(),
				bytes:     t3.raw,
			}
		} else {
			return nil, err
		}
	default:
		return nil, errors.New("Wrong transaction version: " + strconv.Itoa(int(t3.Version.Value)))
	}
	return t, nil
}

func (t *transactionV3) bytes() []byte {
	// TODO
	return nil
}

func (t *transactionV3) hash() []byte {
	// TODO calculate when hash is nil
	switch t.Version.Value {
	case 2:
		return t.Tx_Hash.ToBytes()
	case 3:
		return t.TxHash.ToBytes()
	default:
		return nil
	}
}

var (
	v2FieldInclusion = map[string]bool(nil)
	v2FieldExclusion = map[string]bool{
		"method":    true,
		"signature": true,
		"tx_hash":   true,
	}
	v3FieldInclusion = map[string]bool(nil)
	v3FieldExclusion = map[string]bool{
		"signature": true,
		"txHash":    true,
	}
)

func (t *transactionV3) verifySignature() error {
	var data map[string]interface{}
	var err error
	if err = json.Unmarshal(t.raw, &data); err != nil {
		log.Println("JSON Parse FAILS")
		log.Println("JSON", string(t.raw))
		return err
	}
	var bs []byte
	var txHash []byte
	if t.Version.Value == 2 {
		bs, err = SerializeMap(data, v2FieldInclusion, v2FieldExclusion)
		if err == nil {
			bs = append([]byte("icx_sendTransaction."), bs...)
		}
		txHash = t.Tx_Hash
	} else {
		bs, err = SerializeMap(data, v3FieldInclusion, v3FieldExclusion)
		if err == nil {
			bs = append([]byte("icx_sendTransaction."), bs...)
		}
		txHash = t.TxHash
	}
	if err != nil {
		log.Println("Serialize FAILs")
		log.Println("JSON", string(t.raw))
		return err
	}
	h := crypto.SHA3Sum256(bs)
	if bytes.Compare(h, txHash) != 0 {
		log.Println("Hashes are different")
		log.Println("JSON.TxHash", hex.EncodeToString(txHash))
		log.Println("Calc.TxHash", hex.EncodeToString(h))
		log.Println("TxPhrase", string(bs))
		return errors.New("txHash value is different from real")
	}

	if pk, err := t.Signature.RecoverPublicKey(h); err != nil {
		log.Println("FAIL Recovering public key")
		log.Println("Signature", t.Signature)
		return err
	} else {
		addr := common.NewAccountAddressFromPublicKey(pk).String()
		if err != nil {
			log.Println("FAIL to recovering address from public key")
			return err
		}
		if addr != t.From.String() {
			log.Println("FROM is different from signer")
			log.Println("SIGNER", addr)
			log.Println("FROM", t.From)
			return errors.New("FROM is different from signer")
		}
	}
	log.Println("TX verified")
	return nil
}

// TODO temporary for compile with block_V1.go
func NewTransactionV3(b []byte) (module.Transaction, error) {
	return newTransactionLegacy(b)
}
