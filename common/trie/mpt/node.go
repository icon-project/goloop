package mpt

import (
	"bytes"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

/*
	A node in a Merkle Patricia trie is one of the following:
	1. NULL (represented as the empty string)
	2. branch A 17-item node [ v0 ... v15, vt ]
	3. leaf A 2-item node [ encodedPath, value ]
	4. extension A 2-item node [ encodedPath, key ]

	and hash node.
	hash node is just byte array having hash of the node.
*/
const hashableSize = 32

type (
	node interface {
		hash() []byte
		serialize() []byte
		addChild(m *mpt, k []byte, v trie.Object) (node, bool)
		deleteChild(m *mpt, k []byte) (node, bool, error)
	}
	byteValue []byte
	hash      []byte
)

const printHash = false
const printSerializedValue = false

func (h hash) serialize() []byte {
	// Not valid
	return nil
}

func (h hash) hash() []byte {
	return h
}

func (h hash) addChild(m *mpt, k []byte, v trie.Object) (node, bool) {
	if len(h) == 0 {
		return &leaf{keyEnd: k[:], value: v}, true
	}
	serializedValue, _ := m.db.Get(h)
	return m.set(deserialize(serializedValue, m.objType), k, v)
}

func (h hash) deleteChild(m *mpt, k []byte) (node, bool, error) {
	if m.db == nil {
		return h, true, nil // TODO: proper error
	}
	serializedValue, err := m.db.Get(h)
	if err != nil {
		return h, true, err
	}
	return m.delete(deserialize(serializedValue, m.objType), k)
}

func (v byteValue) Bytes() []byte {
	return v
}

func (v byteValue) Reset(s db.Store, k []byte) error {
	return nil
}

func (v byteValue) Flush() error {
	return nil
}

func (v byteValue) Equal(o trie.Object) bool {
	if b, ok := o.(byteValue); ok {
		return bytes.Compare(b, v) == 0
	}
	return false

}
