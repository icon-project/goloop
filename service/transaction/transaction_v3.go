package transaction

import (
	"bytes"
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

const (
	txMaxDataSize                = 512 * 1024 // 512kB
	configCheckDataOnPreValidate = false
)

type transactionV3Data struct {
	Version   common.HexUint16 `json:"version"`
	From      common.Address   `json:"from"`
	To        common.Address   `json:"to"`
	Value     *common.HexInt   `json:"value"`
	StepLimit common.HexInt    `json:"stepLimit"`
	TimeStamp common.HexInt64  `json:"timestamp"`
	NID       *common.HexInt64 `json:"nid,omitempty"`
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
	bytes  []byte
}

func (tx *transactionV3) Timestamp() int64 {
	return tx.TimeStamp.Value
}

func (tx *transactionV3) verifySignature() error {
	pk, err := tx.Signature.RecoverPublicKey(tx.TxHash())
	if err != nil {
		return InvalidSignatureError.Wrap(err, "fail to recover public key")
	}
	addr := common.NewAccountAddressFromPublicKey(pk)
	if addr.Equal(tx.From()) {
		return nil
	}
	return InvalidSignatureError.New("fail to verify signature")
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

func (tx *transactionV3) From() module.Address {
	return &tx.transactionV3Data.From
}

func (tx *transactionV3) ID() []byte {
	return tx.TxHash()
}

func (tx *transactionV3) Version() int {
	return module.TransactionVersion3
}

func (tx *transactionV3) isDeployType(cType string) bool {
	if cType == state.CTAppZip || cType == state.CTAppJava {
		return true
	}
	return false
}

func (tx *transactionV3) Verify() error {
	// value >= 0
	if tx.Value != nil && tx.Value.Sign() < 0 {
		return InvalidTxValue.Errorf("InvalidTxValue(%s)", tx.Value.String())
	}

	// character level size of data element <= 512KB
	n, err := countBytesOfData(tx.Data)
	if err != nil {
		return InvalidTxValue.Wrapf(err, "InvalidData(%x)", tx.Data)
	} else if n > txMaxDataSize {
		return InvalidTxValue.Errorf("InvalidDataSize(%d)", n)
	}

	// Checkups by data types
	if tx.DataType != nil {
		switch *tx.DataType {
		case DataTypeCall:
			// element check
			if tx.Data == nil {
				return InvalidTxValue.Errorf("TxData for call is NIL")
			}
			if _, err := contract.ParseCallData(tx.Data); err != nil {
				return err
			}
		case DataTypeDeploy:
			// element check
			if tx.Data == nil {
				return InvalidTxValue.New("TxData for deploy is NIL")
			}
			if _, err := contract.ParseDeployData(tx.Data); err != nil {
				return InvalidTxValue.Wrap(err, "TxData is invalid")
			}
			if tx.Value != nil && tx.Value.Sign() != 0 {
				return InvalidTxValue.Errorf("InvalidTxValue(%s)", tx.Value.String())
			}
		case DataTypePatch:
			if tx.Data == nil {
				return InvalidTxValue.New("TxData for patch is NIL")
			}
			if _, err := contract.ParsePatchData(tx.Data); err != nil {
				return InvalidTxValue.Wrap(err, "TxData is invalid")
			}
		}
	}

	// signature verification
	if err := tx.verifySignature(); err != nil {
		return err
	}

	return nil
}

func (tx *transactionV3) ValidateNetwork(nid int) bool {
	if tx.NID == nil {
		return true
	}
	return int(tx.NID.Value) == nid
}

func (tx *transactionV3) PreValidate(wc state.WorldContext, update bool) error {
	// stepLimit >= default step + input steps
	cnt, err := MeasureBytesOfData(wc.Revision(), tx.Data)
	if err != nil {
		return err
	}
	minStep := big.NewInt(wc.StepsFor(state.StepTypeDefault, 1) + wc.StepsFor(state.StepTypeInput, cnt))
	if tx.StepLimit.Cmp(minStep) < 0 {
		return NotEnoughStepError.Errorf("NotEnoughStep(txStepLimit:%s, minStep:%s)", tx.StepLimit, minStep)
	}

	// balance >= (fee + value)
	stepPrice := wc.StepPrice()

	trans := new(big.Int)
	trans.Set(&tx.StepLimit.Int)
	trans.Mul(trans, stepPrice)
	if tx.Value != nil {
		trans.Add(trans, &tx.Value.Int)
	}

	as1 := wc.GetAccountState(tx.From().ID())
	balance1 := as1.GetBalance()
	if balance1.Cmp(trans) < 0 {
		return NotEnoughBalanceError.Errorf("OutOfBalance(balance:%s, value:%s)", balance1, trans)
	}

	// for cumulative balance check
	if update {
		as2 := wc.GetAccountState(tx.To().ID())
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

func (tx *transactionV3) GetHandler(cm contract.ContractManager) (Handler, error) {
	var value *big.Int
	if tx.Value != nil {
		value = &tx.Value.Int
	} else {
		value = big.NewInt(0)
	}
	return NewHandler(cm,
		tx.From(),
		tx.To(),
		value,
		&tx.StepLimit.Int,
		tx.DataType,
		tx.Data)
}

func (tx *transactionV3) Group() module.TransactionGroup {
	if tx.DataType != nil && *tx.DataType == DataTypePatch {
		return module.TransactionGroupPatch
	}
	return module.TransactionGroupNormal
}

func (tx *transactionV3) Bytes() []byte {
	if tx.bytes == nil {
		if bs, err := codec.MarshalToBytes(&tx.transactionV3Data); err != nil {
			log.Errorf("Fail to marshal transaction=%+v err=%+v", tx, err)
			return nil
		} else {
			tx.bytes = bs
		}
	}
	return tx.bytes
}

func (tx *transactionV3) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &tx.transactionV3Data)
	if err != nil {
		return InvalidFormat.Wrap(err, "fail to parse transaction bytes")
	}
	if tx.transactionV3Data.Version.Value != module.TransactionVersion3 {
		return InvalidVersion.Errorf("NotTxVersion3(%d)", tx.transactionV3Data.Version.Value)
	}
	nbs := make([]byte, len(bs))
	copy(nbs, bs)
	tx.bytes = nbs
	return nil
}

func (tx *transactionV3) Hash() []byte {
	return crypto.SHA3Sum256(tx.Bytes())
}

func (tx *transactionV3) Nonce() *big.Int {
	if nonce := tx.transactionV3Data.Nonce; nonce != nil {
		return &nonce.Int
	}
	return nil
}

func (tx *transactionV3) To() module.Address {
	return &tx.transactionV3Data.To
}

func (tx *transactionV3) ToJSON(version module.JSONVersion) (interface{}, error) {
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

}

func (tx *transactionV3) MarshalJSON() ([]byte, error) {
	if obj, err := tx.ToJSON(module.JSONVersionLast); err != nil {
		return nil, scoreresult.WithStatus(err, module.StatusIllegalFormat)
	} else {
		return json.Marshal(obj)
	}
}

func newTransactionV3FromJSONObject(jso *transactionV3JSON) (Transaction, error) {
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
