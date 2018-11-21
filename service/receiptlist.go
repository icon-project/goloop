package service

import (
	"fmt"
	"log"
	"reflect"

	"github.com/icon-project/goloop/common/codec"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

type receiptList struct {
	immutableTrie trie.ImmutableForObject
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
	return &receiptIterator{l.immutableTrie.Iterator()}
}

func (l *receiptList) Get(n int) (module.Receipt, error) {
	b, err := codec.MP.MarshalToBytes(uint(n))
	if err != nil {
		return nil, err
	}
	obj, err := l.immutableTrie.Get(b)
	if err != nil {
		return nil, err
	}
	if tx, ok := obj.(module.Receipt); ok {
		return tx, nil
	}
	return nil, fmt.Errorf("IllegalObjectType(%T)", obj)
}

func (l *receiptList) Hash() []byte {
	return l.immutableTrie.Hash()
}

func (l *receiptList) Flush() error {
	if s, ok := l.immutableTrie.(trie.SnapshotForObject); ok {
		return s.Flush()
	}
	return nil
}

func NewReceiptListFromSlice(database db.Database, list []Receipt) module.ReceiptList {
	mt := trie_manager.NewMutableForObject(database, nil, reflect.TypeOf(&receipt{}))
	for idx, r := range list {
		k, _ := codec.MP.MarshalToBytes(uint(idx))
		if rp, ok := r.(*receipt); ok {
			err := mt.Set(k, rp)
			if err != nil {
				log.Fatalf("NewTransanctionListFromSlice FAILs err=%+v", err)
				return nil
			}
		} else {
			log.Panicf("Failed to assert receipt")
		}

	}
	return &receiptList{mt.GetSnapshot()}
}

func NewReceiptListFromHash(database db.Database, h []byte) module.ReceiptList {
	immutable := trie_manager.NewImmutableForObject(database, h, reflect.TypeOf(&receipt{}))
	return &receiptList{immutable}
}
