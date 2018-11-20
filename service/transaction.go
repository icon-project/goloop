package service

import (
	"bytes"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

type transaction struct {
	Transaction
}

func (t *transaction) Reset(s db.Database, k []byte) error {
	tx, err := newTransaction(k)
	if tx != nil {
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

func (t *transaction) execute(state *transitionState, database db.Database) error {
	// TODO: remove reference to here
	panic("Need to be removed")
	return nil
}
func (t *transaction) validate(mutable trie.Mutable, bucket db.Bucket) error {
	// TODO: remove reference to here.
	panic("Need to removed")
	return nil
}

func NewTransaction(b []byte) (module.Transaction, error) {
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
		return newTransactionFromJSON(b)
	}
	return nil, errors.New("UnknownFormat")
}

func NewTransactionFromJSON(b []byte) (module.Transaction, error) {
	if tx, err := newTransactionFromJSON(b); err != nil {
		return nil, err
	} else {
		return &transaction{tx}, nil
	}
}

func newTransactionFromJSON(b []byte) (Transaction, error) {
	tx, err := NewTransactionV2V3FromJSON(b)
	if err != nil {
		return nil, err
	}
	return tx, nil
}
