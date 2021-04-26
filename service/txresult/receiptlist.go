package txresult

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/log"

	"github.com/icon-project/goloop/common/merkle"

	"github.com/icon-project/goloop/common/codec"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
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
	return obj.(module.Receipt), nil
}

func (l *receiptList) Iterator() module.ReceiptIterator {
	return &receiptIterator{l.immutableTrie.Iterator()}
}

func (l *receiptList) Get(n int) (module.Receipt, error) {
	b, err := codec.BC.MarshalToBytes(uint(n))
	if err != nil {
		return nil, err
	}
	obj, err := l.immutableTrie.Get(b)
	if err != nil {
		return nil, err
	}
	if rct, ok := obj.(module.Receipt); !ok {
		return nil, common.ErrNotFound
	} else {
		return rct, nil
	}
}

func (l *receiptList) GetProof(n int) ([][]byte, error) {
	b, err := codec.BC.MarshalToBytes(uint(n))
	if err != nil {
		return nil, err
	}
	proof := l.immutableTrie.GetProof(b)
	if proof == nil {
		return nil, errors.ErrNotFound
	}
	return proof, nil
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
	mt := trie_manager.NewMutableForObject(database, nil, ReceiptType)
	for idx, r := range list {
		k, _ := codec.BC.MarshalToBytes(uint(idx))
		_, err := mt.Set(k, r.(*receipt))
		if err != nil {
			log.Panicf("NewTransactionListFromSlice FAILs err=%+v", err)
			return nil
		}
	}
	return &receiptList{mt.GetSnapshot()}
}

func NewReceiptListFromHash(database db.Database, h []byte) module.ReceiptList {
	immutable := trie_manager.NewImmutableForObject(database, h, ReceiptType)
	return &receiptList{immutable}
}

func NewReceiptListWithBuilder(builder merkle.Builder, h []byte) module.ReceiptList {
	database := builder.Database()
	snapshot := trie_manager.NewImmutableForObject(database, h, ReceiptType)
	snapshot.Resolve(builder)
	return &receiptList{snapshot}
}
