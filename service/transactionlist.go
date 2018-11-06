package service

import (
	"bytes"
	"fmt"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"log"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
)

type transactionList struct {
	trie trie.ImmutableForObject
}

func (l *transactionList) Get(i int) (module.Transaction, error) {
	b, err := codec.MP.MarshalToBytes(uint(i))
	if err != nil {
		return nil, err
	}
	obj, err := l.trie.Get(b)
	if err != nil {
		return nil, err
	}
	if tx, ok := obj.(module.Transaction); ok {
		return tx, nil
	}
	return nil, fmt.Errorf("IllegalObjectType(%T)", obj)
}

type transactionIterator struct {
	trie.IteratorForObject
}

func (i *transactionIterator) Get() (module.Transaction, int, error) {
	obj, key, err := i.IteratorForObject.Get()
	if err != nil {
		return nil, 0, err
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
	return nil, 0, fmt.Errorf("IllegalObjectType(%T)", obj)
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

func NewTransactionListFromTrie(t trie.ImmutableForObject) module.TransactionList {
	return &transactionList{t}
}

type transactionSlice struct {
	list []module.Transaction
	trie trie.Immutable
}

func (l *transactionSlice) Get(i int) (module.Transaction, error) {
	if i >= 0 && i < len(l.list) {
		return l.list[i], nil
	}
	return nil, common.ErrNotFound
}

type transactionSliceIterator struct {
	list []module.Transaction
	idx  int
}

func (i *transactionSliceIterator) Get() (module.Transaction, int, error) {
	if i.idx >= len(i.list) {
		return nil, 0, common.ErrInvalidState
	}
	return i.list[i.idx], i.idx, nil
}

func (i *transactionSliceIterator) Has() bool {
	return i.idx < len(i.list)
}

func (i *transactionSliceIterator) Next() error {
	if i.idx < len(i.list) {
		i.idx++
		return nil
	} else {
		return common.ErrInvalidState
	}
}

func (l *transactionSlice) Iterator() module.TransactionIterator {
	return &transactionSliceIterator{
		list: l.list,
		idx:  0,
	}
}

func (l *transactionSlice) Hash() []byte {
	return l.trie.RootHash()
}

func (l *transactionSlice) Equal(t module.TransactionList) bool {
	return bytes.Equal(l.trie.RootHash(), t.Hash())
}

func NewTransactionListFromSlice(list []module.Transaction) module.TransactionList {
	tm := trie_manager.New(nil)
	mt := tm.NewMutable(nil)
	for idx, tr := range list {
		k, _ := codec.MP.MarshalToBytes(uint(idx))
		v, err := tr.Bytes()
		if err != nil {
			log.Fatal("NewTrasactionListFromSlice FAILs", err)
			return nil
		}
		err = mt.Set(k, v)
		if err != nil {
			log.Fatalf("NewTransanctionListFromSlice FAILs", err)
			return nil
		}
	}
	s := mt.GetSnapshot()
	return &transactionSlice{list: list, trie: s}
}
