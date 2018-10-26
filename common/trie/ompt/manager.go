package ompt

import (
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

// TODO: DB should be passed as parameter.
func NewImmutable(s db.Store, t reflect.Type, rootHash []byte) trie.ImmutableForObject {
	return newMpt(s, t, rootHash)
}

func NewMutable(s db.Store, t reflect.Type, rootHash []byte) trie.MutableForObject {
	return newMpt(s, t, rootHash)
}
