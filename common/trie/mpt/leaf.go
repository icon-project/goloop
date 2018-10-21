package mpt

import (
	"golang.org/x/crypto/sha3"
)

type (
	leaf struct {
		keyEnd          []byte
		value           []byte
		hashedValue     []byte
		serializedValue []byte
		dirty           bool
	}
)

func (l *leaf) hash() []byte {
	if l.dirty == false && l.hashedValue != nil {
		return l.hashedValue
	}
	serialized := l.serialize()
	//	fmt.Println("leaf hash : ", serialized)
	// TODO: have to change below sha function.
	sha := sha3.NewLegacyKeccak256()
	sha.Write(serialized)
	digest := sha.Sum(serialized[:0])

	l.hashedValue = digest
	l.serializedValue = serialized
	l.dirty = false

	//	fmt.Printf("leaf hash : <%x>", digest)
	return digest
}

func (l *leaf) serialize() []byte {
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

	return encodeList(keyArray, l.value)
}
