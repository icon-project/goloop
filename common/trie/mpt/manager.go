package mpt

import (
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
	db db.Bucket
}

func NewManager(db db.Bucket) trie.Manager {
	return &manager{db: db}
}

func (m *manager) NewImmutable(rootHash []byte) trie.Immutable {
	mpt := newMpt(m.db, rootHash)
	return mpt
}

func (m *manager) NewMutable(rootHash []byte) trie.Mutable {
	return newMpt(m.db, rootHash)
}
