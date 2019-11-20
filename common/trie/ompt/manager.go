package ompt

import (
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/cache"
)

func NewImmutableForObject(db db.Database, h []byte, t reflect.Type) trie.ImmutableForObject {
	return NewMPT(db, h, t)
}

func NewMutableForObject(db db.Database, h []byte, t reflect.Type) trie.MutableForObject {
	return NewMPT(db, h, t)
}

func NewMutableFromImmutableForObject(immutable trie.ImmutableForObject) trie.MutableForObject {
	return MPTFromImmutable(immutable)
}

func NewImmutable(db db.Database, h []byte) trie.Immutable {
	return NewMPTForBytes(db, h)
}

func NewMutable(database db.Database, h []byte) trie.Mutable {
	return NewMPTForBytes(database, h)
}

func NewMutableFromImmutable(immutable trie.Immutable) trie.Mutable {
	return MPTFromImmutableForBytes(immutable)
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

func SetCacheOfMutable(mutable trie.Mutable, cache *cache.NodeCache) {
	m := mutable.(*mptForBytes)
	m.cache = cache
}

func SetCacheOfMutableForObject(mutable trie.MutableForObject, cache *cache.NodeCache) {
	m := mutable.(*mpt)
	m.cache = cache
}
