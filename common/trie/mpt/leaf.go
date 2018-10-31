package mpt

import (
	"bytes"
	"fmt"
	"github.com/icon-project/goloop/common/trie"
	"golang.org/x/crypto/sha3"
)

type (
	leaf struct {
		keyEnd []byte
		value  trie.Object

		hashedValue     []byte
		serializedValue []byte
		dirty           bool // if dirty is true, must retry getting hashedValue & serializedValue
	}
)

func (l *leaf) serialize() []byte {
	if l.dirty == true {
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
	l.dirty = false

	if printSerializedValue {
		fmt.Println("leaf val = ", string(l.value.Bytes()))
		fmt.Println("serialize leaf : ", result)
	}
	return result
}

func (l *leaf) hash() []byte {
	if l.dirty == true {
		l.serializedValue = nil
		l.hashedValue = nil
	} else if l.hashedValue != nil {
		return l.hashedValue
	}

	serialized := l.serialize()
	serializeCopied := make([]byte, len(serialized))
	copy(serializeCopied, serialized)
	// TODO: have to change below sha function.
	sha := sha3.NewLegacyKeccak256()
	sha.Write(serializeCopied)
	digest := sha.Sum(serializeCopied[:0])

	l.hashedValue = make([]byte, len(digest))
	copy(l.hashedValue, digest)
	l.dirty = false

	if printHash {
		fmt.Printf("hash leaf : <%x>\n", digest)
	}
	return digest
}

func (l *leaf) addChild(m *mpt, k []byte, v trie.Object) (node, bool) {
	match, same := compareHex(k, l.keyEnd)
	// case 1 : match = 0 -> new branch
	switch {
	case same == true:
		if l.value.Equal(v) {
			return l, false
		}
		l.value = v
	case match == 0:
		newBranch := &branch{}
		newBranch.addChild(m, k, v)
		newBranch.addChild(m, l.keyEnd, l.value)
		return newBranch, true
	// case 2 : 0 < match < len(n,value) -> new extension
	default:
		newBranch := &branch{}
		newExt := &extension{sharedNibbles: k[:match], next: newBranch}
		newBranch.addChild(m, k[match:], v)
		newBranch.addChild(m, l.keyEnd[match:], l.value)
		return newExt, true
	}
	return l, true
}

func (l *leaf) deleteChild(m *mpt, k []byte) (node, bool, error) {
	// not same key
	if bytes.Compare(l.keyEnd, k) != 0 {
		return l, false, nil
	}
	return nil, true, nil
}
