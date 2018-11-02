package mpt

import (
	"log"
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

//func NewImmutable(rootHash []byte) trie.Immutable {
//	return newMpt(rootHash)
//}

//func NewCache(rootHash []byte) trie.Cache {
//	return newMpt(rootHash)
//}

type manager struct {
	db db.Database
}

func NewManager(db db.Database) trie.Manager {
	if db == nil {
		log.Panic("Db is nil")
	}
	return &manager{db: db}
}

func (m *manager) NewImmutable(rootHash []byte) trie.Immutable {
	bk, err := m.db.GetBucket(db.MerkleTrie)
	if err != nil {
		log.Panicln("FAIL to get Bucket", err)
	}

	return newMpt(bk, rootHash, reflect.TypeOf([]byte{}))
}

func (m *manager) NewMutable(rootHash []byte) trie.Mutable {
	bk, err := m.db.GetBucket(db.MerkleTrie)
	if err != nil {
		log.Panicln("FAIL to get Bucket", err)
	}
	return newMpt(bk, rootHash, reflect.TypeOf([]byte{}))
}

// TODO: implement
func (m *manager) NewImmutableForObject(h []byte, t reflect.Type) trie.ImmutableForObject {
	bk, err := m.db.GetBucket(db.MerkleTrie)
	if err != nil {
		log.Panicln("FAIL to get Bucket", err)
	}
	return newMptForObj(bk, h, t)

}

// TODO: implement
func (m *manager) NewMutableForObject(h []byte, t reflect.Type) trie.MutableForObject {
	bk, err := m.db.GetBucket(db.MerkleTrie)
	if err != nil {
		log.Panicln("FAIL to get Bucket", err)
	}
	return newMptForObj(bk, h, t)
}
