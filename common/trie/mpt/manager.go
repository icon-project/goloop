package mpt

import (
	"github.com/icon-project/goloop/common/trie"
)

// TODO: DB should be passed as parameter.
func NewImmutable(rootHash []byte) trie.Immutable {
	return newMpt(rootHash)
}

func NewMutable(rootHash []byte) trie.Mutable {
	return newMpt(rootHash)
}
