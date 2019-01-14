package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

type transactionV3Data struct {
	Version   common.HexUint16 `json:"version"`
	From      common.Address   `json:"from"`
	To        common.Address   `json:"to"`
	Value     *common.HexInt   `json:"value"`
	StepLimit common.HexInt    `json:"stepLimit"`
	TimeStamp common.HexInt64  `json:"timestamp"`
	NID       *common.HexInt16 `json:"nid,omitempty"`
	Nonce     *common.HexInt   `json:"nonce,omitempty"`
	Signature common.Signature `json:"signature"`
	DataType  *string          `json:"dataType,omitempty"`
	Data      json.RawMessage  `json:"data,omitempty"`
}

func (tx *transactionV3Data) calcHash() ([]byte, error) {
	// sha := sha3.New256()
	sha := bytes.NewBuffer(nil)
	sha.Write([]byte("icx_sendTransaction"))

	// data
	if tx.Data != nil {
		sha.Write([]byte(".data."))
		if len(tx.Data) > 0 {
			var obj interface{}
			if err := json.Unmarshal(tx.Data, &obj); err != nil {
				return nil, err
			}
			if bs, err := serializeValue(obj); err != nil {
				return nil, err
			} else {
				sha.Write(bs)
			}
		}
	}

	// dataType
	if tx.DataType != nil {
		sha.Write([]byte(".dataType."))
		sha.Write([]byte(*tx.DataType))
	}

	// from
	sha.Write([]byte(".from."))
	sha.Write([]byte(tx.From.String()))

	// nid
	if tx.NID != nil {
		sha.Write([]byte(".nid."))
		sha.Write([]byte(tx.NID.String()))
	}

	// nonce
	if tx.Nonce != nil {
		sha.Write([]byte(".nonce."))
		sha.Write([]byte(tx.Nonce.String()))
	}

	// stepLimit
	sha.Write([]byte(".stepLimit."))
	sha.Write([]byte(tx.StepLimit.String()))

	// timestamp
	sha.Write([]byte(".timestamp."))
	sha.Write([]byte(tx.TimeStamp.String()))

	// to
	sha.Write([]byte(".to."))
	sha.Write([]byte(tx.To.String()))

	// value
	if tx.Value != nil {
		sha.Write([]byte(".value."))
		sha.Write([]byte(tx.Value.String()))
	}

	// version
	sha.Write([]byte(".version."))
	sha.Write([]byte(tx.Version.String()))

	return crypto.SHA3Sum256(sha.Bytes()), nil
}

type transactionV3 struct {
	transactionV3Data
	txHash []byte
	hash   []byte
}

func (tx *transactionV3) Timestamp() int64 {
	return tx.TimeStamp.Value
}

func (tx *transactionV3) verifySignature() error {
	pk, err := tx.Signature.RecoverPublicKey(tx.TxHash())
	if err != nil {
		return err
	}
	addr := common.NewAccountAddressFromPublicKey(pk)
	if addr.Equal(&tx.From) {
		return nil
	}
	return errors.New("InvalidSignature")
}

func (tx *transactionV3) TxHash() []byte {
	if tx.txHash == nil {
		h, err := tx.calcHash()
		if err != nil {
			tx.txHash = []byte{}
		} else {
			tx.txHash = h
		}
	}
	return tx.txHash
}

var stepsForTransfer = big.NewInt(100000)

func (tx *transactionV3) ID() []byte {
	return tx.TxHash()
}

func (tx *transactionV3) Version() int {
	return module.TransactionVersion3
}

func (tx *transactionV3) Verify() error {
	if tx.Data != nil && (*tx.DataType) == "call" && !tx.To.IsContract() {
		return ErrContractIsRequired
	}

	if err := tx.verifySignature(); err != nil {
		return err
	}

	return nil
}

func (tx *transactionV3) PreValidate(wc WorldContext, update bool) error {
	stepPrice := wc.StepPrice()

	trans := new(big.Int)
	trans.Set(&tx.StepLimit.Int)
	trans.Mul(trans, stepPrice)
	if tx.Value != nil {
		trans.Add(trans, &tx.Value.Int)
	}

	as1 := wc.GetAccountState(tx.From.ID())
	balance1 := as1.GetBalance()
	if balance1.Cmp(trans) < 0 {
		return ErrNotEnoughBalance
	}

	if configOnCheckingTimestamp {
		tsdiff := wc.BlockTimeStamp() - tx.TimeStamp.Value
		if tsdiff < int64(-5*time.Minute/time.Microsecond) ||
			tsdiff > int64(5*time.Minute/time.Microsecond) {
			return ErrTimeOut
		}
	}

	if update {
		as2 := wc.GetAccountState(tx.To.ID())
		balance2 := as2.GetBalance()
		if tx.Value != nil {
			balance2.Add(balance2, &tx.Value.Int)
		}
		balance1.Sub(balance1, trans)
		as1.SetBalance(balance1)
		as2.SetBalance(balance2)
	}
	return nil
}

func (tx *transactionV3) GetHandler(cm ContractManager) (TransactionHandler, error) {
	var value *big.Int
	if tx.Value != nil {
		value = &tx.Value.Int
	} else {
		value = big.NewInt(0)
	}
	return NewTransactionHandler(cm,
		&tx.From,
		&tx.To,
		value,
		&tx.StepLimit.Int,
		tx.DataType,
		tx.Data)
}

func (tx *transactionV3) Group() module.TransactionGroup {
	return module.TransactionGroupNormal
}

func (tx *transactionV3) Bytes() []byte {
	bs, err := codec.MarshalToBytes(&tx.transactionV3Data)
	if err != nil {
		log.Panicf("Fail to marshal transaction=%+v", tx)
		return nil
	}
	return bs
}

func (tx *transactionV3) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &tx.transactionV3Data)
	if err != nil {
		return err
	}
	if tx.transactionV3Data.Version.Value != module.TransactionVersion3 {
		return errors.New("InvalidTransactionVersion")
	}
	return nil
}

func (tx *transactionV3) Hash() []byte {
	if tx.hash == nil {
		tx.hash = crypto.SHA3Sum256(tx.Bytes())
	}
	return tx.hash
}

func (tx *transactionV3) Nonce() *big.Int {
	if nonce := tx.transactionV3Data.Nonce; nonce != nil {
		return &nonce.Int
	}
	return nil
}

func (tx *transactionV3) ToJSON(version int) (interface{}, error) {
	if version == module.TransactionVersion3 {
		jso := map[string]interface{}{
			"version":   &tx.transactionV3Data.Version,
			"from":      &tx.transactionV3Data.From,
			"to":        &tx.transactionV3Data.To,
			"stepLimit": &tx.transactionV3Data.StepLimit,
			"timestamp": &tx.transactionV3Data.TimeStamp,
			"signature": &tx.transactionV3Data.Signature,
		}
		if tx.transactionV3Data.Value != nil {
			jso["value"] = tx.transactionV3Data.Value
		}
		if tx.transactionV3Data.NID != nil {
			jso["nid"] = tx.transactionV3Data.NID
		}
		if tx.transactionV3Data.Nonce != nil {
			jso["nonce"] = tx.transactionV3Data.Nonce
		}
		if tx.transactionV3Data.DataType != nil {
			jso["dataType"] = *tx.transactionV3Data.DataType
		}
		if tx.transactionV3Data.Data != nil {
			jso["data"] = json.RawMessage(tx.transactionV3Data.Data)
		}
		jso["txHash"] = common.HexBytes(tx.ID())

		return jso, nil
	} else {
		return nil, errors.New("InvalidVersion:" + strconv.Itoa(version))
	}
}

func (tx *transactionV3) MarshalJSON() ([]byte, error) {
	if obj, err := tx.ToJSON(module.TransactionVersion3); err != nil {
		return nil, err
	} else {
		return json.Marshal(obj)
	}
}

func newTransactionV3FromJSON(jso *transactionV3JSON) (*transactionV3, error) {
	tx := new(transactionV3)
	tx.transactionV3Data = jso.transactionV3Data
	return tx, nil
}

func newTransactionV3FromBytes(bs []byte) (Transaction, error) {
	tx := new(transactionV3)
	if err := tx.SetBytes(bs); err != nil {
		return nil, err
	} else {
		return tx, nil
	}
}
