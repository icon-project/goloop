package transaction

import (
	"bytes"
	"encoding/json"
	"math/big"
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
)

var TransactionType = reflect.TypeOf((*transaction)(nil))

// TODO It assumes normal transaction. When supporting patch, add skipping
// timestamp checking for it at PreValidate().
type Transaction interface {
	module.Transaction
	PreValidate(wc state.WorldContext, update bool) error
	GetHandler(cm contract.ContractManager) (Handler, error)
	Timestamp() int64
	Nonce() *big.Int
	To() module.Address
	IsSkippable() bool
}

type GenesisTransaction interface {
	Transaction
	CID() int
	NID() int
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
	if tx, err := newTransactionFromJSON(data, false); err != nil {
		return err
	} else {
		t.Transaction = tx
		return nil
	}
}

func (t *transaction) Resolve(builder merkle.Builder) error {
	return nil
}

func (t *transaction) NID() int {
	return t.Transaction.(GenesisTransaction).NID()
}

func (t *transaction) CID() int {
	return t.Transaction.(GenesisTransaction).CID()
}

func (t *transaction) ClearCache() {
	// nothing to do
}

func NewTransaction(b []byte) (Transaction, error) {
	if tx, err := newTransaction(b); err != nil {
		return nil, err
	} else {
		return &transaction{tx}, nil
	}
}

func NewGenesisTransaction(b []byte) (GenesisTransaction, error) {
	if js, err := jsonCompact(b); err == nil {
		b = js
	}
	if tx, err := parseV3Genesis(b, false); err != nil {
		return nil, err
	} else {
		return &transaction{tx}, nil
	}
}

func NewTransactionFromJSON(b []byte) (Transaction, error) {
	if tx, err := newTransactionFromJSON(b, false); err != nil {
		return nil, err
	} else {
		return &transaction{tx}, nil
	}
}

func Wrap(t Transaction) Transaction {
	if _, ok := t.(*transaction); ok {
		return t
	} else {
		return &transaction{t}
	}
}

func Unwrap(t module.Transaction) module.Transaction {
	if tp, ok := t.(*transaction); ok {
		return tp.Transaction
	} else {
		return t
	}
}
