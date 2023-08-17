package transaction

import (
	"bytes"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie/trie_manager"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
)

type transactionList struct {
	trie   trie.ImmutableForObject
	writer db.Writer
}

func intToKey(i int) []byte {
	b, err := codec.BC.MarshalToBytes(uint(i))
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
	if _, err := codec.BC.UnmarshalFromBytes(key, &idx); err != nil {
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
	if l.writer != nil {
		defer l.writer.Prepare()
	}
	return l.trie.Hash()
}

func (l *transactionList) Equal(t module.TransactionList) bool {
	return bytes.Equal(l.trie.Hash(), t.Hash())
}

func (l *transactionList) Flush() error {
	if l.writer != nil {
		return l.writer.Flush()
	}
	if ss, ok := l.trie.(trie.SnapshotForObject); ok {
		return ss.Flush()
	}
	return nil
}

func NewTransactionListFromHash(d db.Database, h []byte) module.TransactionList {
	t := trie_manager.NewImmutableForObject(d, h, TransactionType)
	return &transactionList{t, nil}
}

func NewTransactionListFromSlice(dbase db.Database, list []module.Transaction) module.TransactionList {
	if len(list) == 0 {
		return NewTransactionListFromHash(dbase, nil)
	}
	writer := db.NewWriter(dbase)
	mt := trie_manager.NewMutableForObject(writer.Database(), nil, TransactionType)
	for idx, tx := range list {
		_, err := mt.Set(intToKey(idx), tx.(trie.Object))
		if err != nil {
			panic(err)
		}
	}
	snapshot := mt.GetSnapshot()
	writer.Add(snapshot)
	return &transactionList{snapshot, writer}
}

func NewTransactionListWithBuilder(builder merkle.Builder, h []byte) module.TransactionList {
	d := builder.Database()
	snapshot := trie_manager.NewImmutableForObject(d, h, TransactionType)
	snapshot.Resolve(builder)
	// log.Printf("NewTransactionListWithBuilder: hash=%x size=%d", h, builder.UnresolvedCount())
	return &transactionList{snapshot, nil}
}
