package ompt

import (
	"fmt"

	"github.com/icon-project/goloop/common/trie"
	"golang.org/x/crypto/sha3"
)

type (
	leaf struct {
		keyEnd []byte
		// value  []byte
		value trie.Object

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

	if printSerializedValue {
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
	// TODO: have to change below sha function.
	sha := sha3.NewLegacyKeccak256()
	sha.Write(serialized)
	digest := sha.Sum(serialized[:0])

	l.hashedValue = digest
	l.serializedValue = serialized

	if printHash {
		fmt.Printf("hash leaf : <%x>\n", digest)
	}
	return digest
}
