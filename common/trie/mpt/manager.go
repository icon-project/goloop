package mpt

import (
	"log"
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

func NewImmutable(database db.Database, rootHash []byte) trie.Immutable {
	bk, err := database.GetBucket(db.MerkleTrie)
	if err != nil {
		log.Panicln("FAIL to get Bucket", err)
	}

	return newMpt(bk, rootHash, reflect.TypeOf([]byte{}))
}

func NewMutable(database db.Database, rootHash []byte) trie.Mutable {
	bk, err := database.GetBucket(db.MerkleTrie)
	if err != nil {
		log.Panicln("FAIL to get Bucket", err)
	}
	return newMpt(bk, rootHash, reflect.TypeOf([]byte{}))
}

func NewImmutableForObject(database db.Database, h []byte, t reflect.Type) trie.ImmutableForObject {
	bk, err := database.GetBucket(db.MerkleTrie)
	if err != nil {
		log.Panicln("FAIL to get Bucket", err)
	}
	return newMptForObj(bk, h, t)

}

func NewMutableForObject(database db.Database, h []byte, t reflect.Type) trie.MutableForObject {
	bk, err := database.GetBucket(db.MerkleTrie)
	if err != nil {
		log.Panicln("FAIL to get Bucket", err)
	}
	return newMptForObj(bk, h, t)
}

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
	return NewImmutable(m.db, rootHash)
}

func (m *manager) NewMutable(rootHash []byte) trie.Mutable {
	return NewMutable(m.db, rootHash)
}

func (m *manager) NewImmutableForObject(h []byte, t reflect.Type) trie.ImmutableForObject {
	return NewImmutableForObject(m.db, h, t)
}

func (m *manager) NewMutableForObject(h []byte, t reflect.Type) trie.MutableForObject {
	return NewMutableForObject(m.db, h, t)
}
