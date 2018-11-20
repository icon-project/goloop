package service

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
	"reflect"
)

type receiptList struct {
	snapshot trie.ImmutableForObject
}

type receiptIterator struct {
	trie.IteratorForObject
}

func (i *receiptIterator) Get() (module.Receipt, error) {
	obj, _, err := i.IteratorForObject.Get()
	if err != nil {
		return nil, err
	}
	rct, ok := obj.(module.Receipt)
	if ok {
		return rct, nil
	} else {
		return nil, errors.Errorf("InvalidReceiptObject:%T", obj)
	}
}

func (l *receiptList) Iterator() module.ReceiptIterator {
	// TODO Implement
	panic("implement me")
}

func (l *receiptList) Get(n int) (module.Receipt, error) {
	// TODO make key for index, and retreive object and cast it to the type.
	return nil, nil
}

func (l *receiptList) Hash() []byte {
	return l.snapshot.Hash()
}

func (l *receiptList) Flush() error {
	if s, ok := l.snapshot.(trie.SnapshotForObject); ok {
		return s.Flush()
	}
	return nil
}

var receiptType = reflect.TypeOf((*receipt)(nil)).Elem()

func NewReceiptListFromSlice(database db.Database, rl []module.Receipt) module.ReceiptList {
	// tree := trie_manager.NewMutableForObject(database, nil, receiptType)
	// TODO Add receipt with key from index.
	snapshot := trie.SnapshotForObject(nil)
	return &receiptList{snapshot}
}

func NewReceiptListFromHash(database db.Database, h []byte) module.ReceiptList {
	immutable := trie_manager.NewImmutableForObject(database, h, receiptType)
	return &receiptList{immutable}
}
