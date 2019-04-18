package transaction

import (
	"bytes"
	"github.com/icon-project/goloop/common/errors"
	"log"
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie/trie_manager"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
)

type transactionList struct {
	trie trie.ImmutableForObject
}

func intToKey(i int) []byte {
	b, err := codec.MP.MarshalToBytes(uint(i))
	if err != nil {
		log.Panicf("Fail to marshal int i=%d", i)
	}
	return b
}

func (l *transactionList) Get(i int) (module.Transaction, error) {
	obj, err := l.trie.Get(intToKey(i))
	if err != nil {
		return nil, errors.WithCode(err, errors.NotFoundError)
	}
	if tx, ok := obj.(module.Transaction); ok {
		return tx, nil
	}
	return nil, errors.InvalidStateError.Errorf("IllegalObjectType(%T)", obj)
}

type transactionIterator struct {
	trie.IteratorForObject
}

func (i *transactionIterator) Get() (module.Transaction, int, error) {
	obj, key, err := i.IteratorForObject.Get()
	if err != nil {
		return nil, 0, errors.WithCode(err, errors.NotFoundError)
	}
	if obj == nil {
		return nil, 0, nil
	}
	var idx uint
	if _, err := codec.MP.UnmarshalFromBytes(key, &idx); err != nil {
		return nil, 0, err
	}
	if tx, ok := obj.(module.Transaction); ok {
		return tx, int(idx), nil
	}
	return nil, 0, errors.InvalidStateError.Errorf("IllegalObjectType(%T)", obj)
}

func (l *transactionList) Iterator() module.TransactionIterator {
	return &transactionIterator{l.trie.Iterator()}
}

func (l *transactionList) Hash() []byte {
	return l.trie.Hash()
}

func (l *transactionList) Equal(t module.TransactionList) bool {
	return bytes.Equal(l.trie.Hash(), t.Hash())
}

func (l *transactionList) Flush() error {
	if ss, ok := l.trie.(trie.SnapshotForObject); ok {
		return ss.Flush()
	}
	return nil
}

func NewTransactionListFromHash(d db.Database, h []byte) module.TransactionList {
	t := trie_manager.NewImmutableForObject(d, h, reflect.TypeOf((*transaction)(nil)))
	return &transactionList{t}
}

func NewTransactionListFromSlice(dbase db.Database, list []module.Transaction) module.TransactionList {
	mt := trie_manager.NewMutableForObject(dbase, nil, reflect.TypeOf((*transaction)(nil)))
	for idx, tx := range list {
		mt.Set(intToKey(idx), tx.(trie.Object))
	}
	return &transactionList{mt.GetSnapshot()}
}
