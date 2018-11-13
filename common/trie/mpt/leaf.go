package mpt

import (
	"bytes"
	"fmt"
	"github.com/icon-project/goloop/common/trie"
)

type (
	leaf struct {
		nodeBase
		keyEnd []byte
		value  trie.Object
	}
)

func (l *leaf) serialize() []byte {
	if l.state == dirtyNode {
		l.serializedValue = nil
		l.hashedValue = nil
	} else if l.serializedValue != nil {
		if printSerializedValue {
			fmt.Println("leaf serialize cached. serialized = ", l.serializedValue)
			fmt.Println("leaf serialize cached. val = ", string(l.value.Bytes()))
		}
		return l.serializedValue
	}

	keyLen := len(l.keyEnd)
	keyArray := make([]byte, keyLen/2+1)
	keyIndex := 0
	if keyLen%2 == 1 {
		keyArray[0] = 0x3<<4 | l.keyEnd[0]
		keyIndex++
	} else {
		keyArray[0] = 0x20
	}

	for i := 0; i < keyLen/2; i++ {
		keyArray[i+1] = l.keyEnd[i*2+keyIndex]<<4 | l.keyEnd[i*2+1+keyIndex]
	}

	result := encodeList(encodeByte(keyArray), encodeByte(l.value.Bytes()))
	l.serializedValue = make([]byte, len(result))
	copy(l.serializedValue, result)
	// if this node is reserealized, hashed value has to be reset
	if l.hashedValue != nil {
		l.hashedValue = nil
	}
	l.state = serializedNode

	if printSerializedValue {
		fmt.Println("leaf val = ", string(l.value.Bytes()))
		fmt.Println("serialize leaf : ", result)
	}
	return result
}

func (l *leaf) hash() []byte {
	if l.state == dirtyNode {
		l.serializedValue = nil
		l.hashedValue = nil
	} else if l.hashedValue != nil {
		return l.hashedValue
	}

	serialized := l.serialize()
	serializeCopied := make([]byte, len(serialized))
	copy(serializeCopied, serialized)
	digest := calcHash(serializeCopied)

	l.hashedValue = make([]byte, len(digest))
	copy(l.hashedValue, digest)
	l.state = serializedNode

	if printHash {
		fmt.Printf("hash leaf : <%x>\n", digest)
	}
	return digest
}

func (l *leaf) addChild(m *mpt, k []byte, v trie.Object) (node, nodeState) {
	//fmt.Println("leaf addChild : k ", k, ", v : ", v)
	match, same := compareHex(k, l.keyEnd)
	// case 1 : match = 0 -> new branch
	switch {
	case same == true:
		if l.value.Equal(v) {
			return l, l.state
		}
		l.value = v
		l.state = dirtyNode
	case match == 0:
		newBranch := &branch{nodeBase: nodeBase{state: dirtyNode}}
		newBranch.addChild(m, k, v)
		newBranch.addChild(m, l.keyEnd, l.value)
		return newBranch, newBranch.state
	// case 2 : 0 < match < len(n,value) -> new extension
	default:
		newBranch := &branch{nodeBase: nodeBase{state: dirtyNode}}
		newExt := &extension{sharedNibbles: k[:match], next: newBranch, nodeBase: nodeBase{state: dirtyNode}}
		newBranch.addChild(m, k[match:], v)
		newBranch.addChild(m, l.keyEnd[match:], l.value)
		return newExt, newExt.state
	}
	return l, l.state
}

func (l *leaf) deleteChild(m *mpt, k []byte) (node, nodeState, error) {
	// not same key
	if bytes.Compare(l.keyEnd, k) != 0 {
		return l, l.state, nil
	}
	return nil, dirtyNode, nil
}

func (l *leaf) flush() {
	l.value.Flush()
}

func (l *leaf) get(m *mpt, k []byte) (node, trie.Object, error) {
	if bytes.Compare(k, l.keyEnd) != 0 {
		return l, nil, nil
	}
	return l, l.value, nil
}
