package ompt

import (
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

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
	return nil
}

func (m *manager) NewMutable(h []byte) trie.Mutable {
	return nil
}

func NewManager(db db.Database) trie.Manager {
	return &manager{db}
}
