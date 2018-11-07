package service

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"

	"github.com/icon-project/goloop/common"
)

type transactionV3 struct {
	Version   common.HexInt16  `json:"version"`
	NID       common.HexInt16  `json:"nid"`
	From      common.Address   `json:"from"`
	To        common.Address   `json:"to"`
	Value     common.HexInt    `json:"value"`
	TimeStamp common.HexInt64  `json:"timestamp"`
	Nonce     common.HexInt64  `json:"nonce"`
	StepLimit common.HexInt    `json:"stepLimit"`
	Fee       common.HexInt    `json:"fee"`
	TxHash    common.HexBytes  `json:"txHash"`
	Tx_Hash   common.HexBytes  `json:"tx_hash"`
	Signature common.Signature `json:"signature"`
	params    []byte
}

func (t *transactionV3) GetID() []byte {
	if t.TxHash != nil {
		return t.TxHash
	} else {
		return t.Tx_Hash
	}
}

func (t *transactionV3) GetVersion() int {
	return int(t.Version.Value)
}

// var version2FieldInclusion = map[string]bool{
// 	"from":      true,
// 	"to":        true,
// 	"value":     true,
// 	"timestamp": true,
// 	"nonce":     true,
// 	"fee":       true,
// }
// var version2FieldExclusion = map[string]bool(nil)
var (
	version2FieldInclusion = map[string]bool(nil)
	version2FieldExclusion = map[string]bool{
		"method":    true,
		"signature": true,
		"tx_hash":   true,
	}
)

// var version3FieldInclusion = map[string]bool{
// 	"from":      true,
// 	"to":        true,
// 	"value":     true,
// 	"timestamp": true,
// 	"nonce":     true,
// 	"stepLimit": true,
// 	"nid":       true,
// 	"version":   true,
// 	"data": true,
// }
// var version3FieldExclusion = map[string]bool(nil)

var (
	version3FieldInclusion = map[string]bool(nil)
	version3FieldExclusion = map[string]bool{
		"signature": true,
		"txHash":    true,
	}
)

func (t *transactionV3) Verify() error {
	var data map[string]interface{}
	var err error
	if err = json.Unmarshal(t.params, &data); err != nil {
		log.Println("JSON Parse FAILS")
		log.Println("JSON", string(t.params))
		return err
	}
	var bs []byte
	var txHash []byte
	if t.Version.Value == 2 {
		bs, err = SerializeMap(data,
			version2FieldInclusion, version2FieldExclusion)
		if err == nil {
			bs = append([]byte("icx_sendTransaction."), bs...)
		}
		txHash = t.Tx_Hash
	} else {
		bs, err = SerializeMap(data,
			version3FieldInclusion, version3FieldExclusion)
		if err == nil {
			bs = append([]byte("icx_sendTransaction."), bs...)
		}
		txHash = t.TxHash
	}
	if err != nil {
		log.Println("Serialize FAILs")
		log.Println("JSON", string(t.params))
		return err
	}
	//fmt.Println("JSON      :", string(t.params))
	//fmt.Println("Serialized:", string(bs))
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
		// crypto.VerifySignature(h, t.Signature, pk)
		/*
			if !crypto.VerifySignature(h, t.Signature, pk) {
				// log.Println("FAIL Verifying signature")
				// log.Println("Signature", len(t.Signature), t.Signature)
				// return errors.New("signature verification fails")
				fmt.Println("FAIL VERIFYING SIGNATURE")
				fmt.Println("Transaction", string(t.params))
			}
		*/
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
	return nil
}

func (t *transactionV3) String() string {
	return string(t.params)
}

func (t *transactionV3) Bytes() ([]byte, error) {
	return t.params, nil
}

type TransactionV3 struct {
	*transactionV3
}

func (t *TransactionV3) ID() []byte {
	return t.GetID()
}

func (t *TransactionV3) Version() int {
	return int(t.transactionV3.Version.Value)
}

// TODO dummy just for compile
func (tx *TransactionV3) From() module.Address {
	return nil
}

func (tx *TransactionV3) To() module.Address {
	return nil
}

func (tx *TransactionV3) Value() int {
	return -1
}

func (tx *TransactionV3) StepLimit() int {
	return -1
}

func (tx *TransactionV3) TimeStamp() int64 {
	return -1
}

func (tx *TransactionV3) NID() int {
	return -1
}

func (tx *TransactionV3) Nonce() int64 {
	return -1
}

func (tx *TransactionV3) Hash() []byte {
	return nil
}

func (tx *TransactionV3) Signature() []byte {
	return nil
}

func NewTransactionV3(b []byte) (module.Transaction, error) {
	t := &transactionV3{Version: common.HexInt16{2}, params: b[:]}
	if err := json.Unmarshal(b, t); err != nil {
		return nil, err
	}
	return &TransactionV3{t}, nil
}
