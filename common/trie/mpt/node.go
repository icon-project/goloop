package mpt

import (
	"bytes"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
	"golang.org/x/crypto/sha3"
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

type nodeState int

const (
	noneNode nodeState = iota
	dirtyNode
	serializedNode
	committedNode
)

type (
	nodeBase struct {
		hashedValue     []byte
		serializedValue []byte
		state           nodeState
	}
	node interface {
		hash() []byte
		serialize() []byte
		addChild(m *mpt, k []byte, v trie.Object) (node, nodeState)
		deleteChild(m *mpt, k []byte) (node, nodeState, error)
		get(m *mpt, k []byte) (node, trie.Object, error)
		proof(m *mpt, k []byte, depth int) ([][]byte, bool)
	}
	byteValue []byte
	hash      []byte
)

const printHash = false
const printSerializedValue = false

func (h hash) serialize() []byte {
	return h
}

func (h hash) hash() []byte {
	return h
}

func (h hash) addChild(m *mpt, k []byte, v trie.Object) (node, nodeState) {
	if len(h) == 0 {
		return &leaf{keyEnd: k[:], value: v}, dirtyNode
	}
	serializedValue, err := m.bk.Get(h)
	if serializedValue == nil || err != nil {
		return h, dirtyNode
	}
	return m.set(deserialize(serializedValue, m.objType, m.db), k, v)
}

func (h hash) deleteChild(m *mpt, k []byte) (node, nodeState, error) {
	if len(h) == 0 {
		return h, noneNode, nil // TODO: proper error
	}
	serializedValue, err := m.bk.Get(h)
	if serializedValue == nil || err != nil {
		return h, noneNode, err
	}
	deserializedNode := deserialize(serializedValue, m.objType, m.db)
	if deserializedNode == nil {
		return h, noneNode, nil
	}
	return deserializedNode.deleteChild(m, k)
}

func (h hash) get(m *mpt, k []byte) (node, trie.Object, error) {
	serializedValue, err := m.bk.Get(h)
	if err != nil || serializedValue == nil {
		return h, nil, err
	}
	deserializedNode := deserialize(serializedValue, m.objType, m.db)
	switch m := deserializedNode.(type) {
	case *branch:
		m.hashedValue = h
	case *extension:
		m.hashedValue = h
	case *leaf:
		m.hashedValue = h
	default:
		log.Panicf("Not considered!! serializedValue : %x, deserializedValue : %x, key : %x\n",
			serializedValue, deserializedNode, h)
	}
	return deserializedNode.get(m, k)
}

func (h hash) proof(m *mpt, k []byte, depth int) ([][]byte, bool) {
	serializedValue, err := m.bk.Get(h)
	if err != nil || serializedValue == nil {
		return nil, false
	}
	deserializedNode := deserialize(serializedValue, m.objType, m.db)
	switch m := deserializedNode.(type) {
	case *branch:
		m.hashedValue = h
	case *extension:
		m.hashedValue = h
	case *leaf:
		m.hashedValue = h
	}
	return deserializedNode.proof(m, k, depth+1)
}

func (v byteValue) Bytes() []byte {
	return v
}

func (v byteValue) Reset(s db.Database, k []byte) error {
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

func (v byteValue) Resolve(builder merkle.Builder) error {
	panic("Imeplement me")
}

func (v byteValue) ClearCache() {
	// do nothing
}

func calcHash(data ...[]byte) []byte {
	if trie.ConfigUseKeccak256 {
		sha := sha3.NewLegacyKeccak256()
		for _, d := range data {
			sha.Write(d)
		}
		sum := sha.Sum([]byte{})
		return sum[:]
	} else {
		sha := sha3.New256()
		for _, d := range data {
			sha.Write(d)
		}
		sum := sha.Sum([]byte{})
		return sum[:]
	}
}
