package trie_manager

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"reflect"
)

type trieManager struct {
	database db.Database
}

func (m *trieManager) NewImmutable(rootHash []byte) trie.Immutable {
	return NewImmutable(m.database, rootHash)
}

func (m *trieManager) NewImmutableForObject(h []byte, t reflect.Type) trie.ImmutableForObject {
	return NewImmutableForObject(m.database, h, t)
}

func (m *trieManager) NewMutable(rootHash []byte) trie.Mutable {
	return NewMutable(m.database, rootHash)
}

func (m *trieManager) NewMutableForObject(h []byte, t reflect.Type) trie.MutableForObject {
	return NewMutableForObject(m.database, h, t)
}

func New(database db.Database) trie.Manager {
	return &trieManager{database}
}
