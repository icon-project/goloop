// +build mapTrie

package trie_manager

import (
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/mpt"
)

func NewImmutableForObject(db db.Database, h []byte, t reflect.Type) trie.ImmutableForObject {
	return mpt.NewImmutableForObject(db, h, t)
}

func NewMutableForObject(db db.Database, h []byte, t reflect.Type) trie.MutableForObject {
	return mpt.NewMutableForObject(db, h, t)
}

func NewMutableFromImmutableForObject(object trie.ImmutableForObject) trie.MutableForObject {
	return mpt.NewMutableFromImmutableForObject(object)
}

func NewImmutable(db db.Database, h []byte) trie.Immutable {
	return mpt.NewImmutable(db, h)
}

func NewMutable(database db.Database, h []byte) trie.Mutable {
	return mpt.NewMutable(database, h)
}

func MutableFromImmutable(object trie.Immutable) trie.Mutable {
	return mpt.NewMutableFromImmutable(object)
}
