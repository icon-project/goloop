package transaction

import (
	"bytes"
	"encoding/json"
	"math/big"
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/pkg/errors"
)

const (
	ConfigTXTimestampBackwardMargin = int64(5 * time.Minute / time.Microsecond)
	ConfigTXTimestampForwardMargin  = int64(5 * time.Minute / time.Microsecond)
	ConfigTXTimestampForwardLimit   = int64(10 * time.Minute / time.Microsecond)
	ConfigOnCheckingTimestamp       = false
)

const (
	LimitTypeInvoke = "invoke"
	LimitTypeCall   = "query"
)

var AllLimitTypes = []string{
	LimitTypeInvoke,
	LimitTypeCall,
}

// TODO It assumes normal transaction. When supporting patch, add skipping
// timestamp checking for it at PreValidate().
type Transaction interface {
	module.Transaction
	PreValidate(wc state.WorldContext, update bool) error
	GetHandler(cm contract.ContractManager) (TransactionHandler, error)
	Timestamp() int64
	Nonce() *big.Int
}

type transaction struct {
	Transaction
}

func (t *transaction) Reset(s db.Database, k []byte) error {
	tx, err := newTransaction(k)
	if err != nil {
		return err
	}
	t.Transaction = tx
	return nil
}

func (t *transaction) Flush() error {
	return nil
}

func (t *transaction) Equal(obj trie.Object) bool {
	if tx, ok := obj.(*transaction); ok {
		return bytes.Equal(tx.Transaction.ID(), t.Transaction.ID())
	}
	return false
}

func (t *transaction) Bytes() []byte {
	return t.Transaction.Bytes()
}

func (t *transaction) MarshalBinary() (data []byte, err error) {
	return t.Bytes(), nil
}

func (t *transaction) UnmarshalBinary(data []byte) error {
	if tx, err := newTransaction(data); err != nil {
		return err
	} else {
		t.Transaction = tx
		return nil
	}
}

func (t *transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Transaction)
}

func (t *transaction) UnmarshalJSON(data []byte) error {
	if tx, err := newTransactionFromJSON(data); err != nil {
		return err
	} else {
		t.Transaction = tx
		return nil
	}
}

func (t *transaction) Resolve(builder merkle.Builder) error {
	return nil
}

func NewTransaction(b []byte) (Transaction, error) {
	if tx, err := newTransaction(b); err != nil {
		return nil, err
	} else {
		return &transaction{tx}, nil
	}
}

func newTransaction(b []byte) (Transaction, error) {
	if len(b) < 1 {
		return nil, errors.New("IllegalTransactionData")
	}
	if b[0] == '{' {
		if tx, err := newTransactionFromJSON(b); err == nil {
			return tx, nil
		}
	}
	return newTransactionV3FromBytes(b)
}

func NewTransactionFromJSON(b []byte) (Transaction, error) {
	if tx, err := newTransactionFromJSON(b); err != nil {
		return nil, err
	} else {
		return &transaction{tx}, nil
	}
}

func newTransactionFromJSON(b []byte) (Transaction, error) {
	tx, err := newTransactionV2V3FromJSON(b)
	if err != nil {
		return nil, err
	}
	return tx, nil
}
