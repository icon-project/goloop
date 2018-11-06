package ompt

import (
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

func NewImmutableForObject(db db.Database, h []byte, t reflect.Type) trie.ImmutableForObject {
	return NewMPT(db, h, t)
}

func NewMutableForObject(db db.Database, h []byte, t reflect.Type) trie.MutableForObject {
	return NewMPT(db, h, t)
}

func NewImmutable(db db.Database, h []byte) trie.Immutable {
	return NewMPTForBytes(db, h)
}

func NewMutable(database db.Database, h []byte) trie.Mutable {
	return NewMPTForBytes(database, h)
}

type manager struct {
	db db.Database
}

func (m *manager) NewImmutableForObject(h []byte, t reflect.Type) trie.ImmutableForObject {
	return NewImmutableForObject(m.db, h, t)
}

func (m *manager) NewMutableForObject(h []byte, t reflect.Type) trie.MutableForObject {
	return NewMutableForObject(m.db, h, t)
}

func (m *manager) NewImmutable(h []byte) trie.Immutable {
	return NewImmutable(m.db, h)
}

func (m *manager) NewMutable(h []byte) trie.Mutable {
	return NewMutable(m.db, h)
}

func NewManager(db db.Database) trie.Manager {
	return &manager{db}
}
