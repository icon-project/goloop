package trie_manager

import (
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/cache"
	"github.com/icon-project/goloop/common/trie/ompt"
)

func NewImmutableForObject(db db.Database, h []byte, t reflect.Type) trie.ImmutableForObject {
	return ompt.NewImmutableForObject(db, h, t)
}

func NewMutableForObject(db db.Database, h []byte, t reflect.Type) trie.MutableForObject {
	return ompt.NewMutableForObject(db, h, t)
}

func NewMutableFromImmutableForObject(object trie.ImmutableForObject) trie.MutableForObject {
	return ompt.NewMutableFromImmutableForObject(object)
}

func NewImmutable(db db.Database, h []byte) trie.Immutable {
	return ompt.NewImmutable(db, h)
}

func NewMutable(database db.Database, h []byte) trie.Mutable {
	return ompt.NewMutable(database, h)
}

func NewMutableFromImmutable(object trie.Immutable) trie.Mutable {
	return ompt.NewMutableFromImmutable(object)
}

func SetCacheOfMutable(mutable trie.Mutable, cache *cache.NodeCache) {
	ompt.SetCacheOfMutable(mutable, cache)
}

func SetCacheOfMutableForObject(mutable trie.MutableForObject, cache *cache.NodeCache) {
	ompt.SetCacheOfMutableForObject(mutable, cache)
}
