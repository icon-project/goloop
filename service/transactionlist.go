package service

import (
	"fmt"

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
	return nil
}

func (l *transactionList) Equal(module.TransactionList) bool {
	return false
}

func NewTransactionListFromTrie(t trie.ImmutableForObject) module.TransactionList {
	return &transactionList{t}
}

type transactionSlice struct {
	list []module.Transaction
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
	return nil
}

func (l *transactionSlice) Equal(module.TransactionList) bool {
	return false
}

func NewTransactionListFromSlice(list []module.Transaction) module.TransactionList {
	return &transactionSlice{list: list}
}
