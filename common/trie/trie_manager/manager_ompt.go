// +build ompt

package trie_manager

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/ompt"
	"reflect"
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

func MutableFromImmutable(object trie.Immutable) trie.Mutable {
	return ompt.MutableFromImmutable(object)
}
