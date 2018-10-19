package mpt

import (
	"golang.org/x/crypto/sha3"
)

type (
	extension struct {
		sharedNibbles   []byte
		next            node
		hashedValue     []byte
		serializedValue []byte
		dirty           bool
	}
)

func (ex *extension) serialize() []byte {
	keyLen := len(ex.sharedNibbles)
	keyArray := make([]byte, keyLen/2+1)
	keyIndex := 0
	if keyLen%2 == 1 {
		keyArray[0] = 0x1<<4 | ex.sharedNibbles[0]
		keyIndex++
	} else {
		keyArray[0] = 0x00
	}

	for i := 0; i < keyLen/2; i++ {
		keyArray[i+1] = ex.sharedNibbles[i*2+keyIndex]<<4 | ex.sharedNibbles[i*2+1+keyIndex]
	}
	serialized := encodeList(keyArray, ex.next.hash())
	ex.serializedValue = make([]byte, len(serialized))
	copy(ex.serializedValue, serialized)
	//	fmt.Println("extension : serialized : ", serialized)
	return serialized
}

func (ex *extension) hash() []byte {
	if ex.dirty == false && ex.hashedValue != nil {
		return ex.hashedValue
	}
	serialized := ex.serialize()
	// TODO: have to change below sha function.
	sha := sha3.NewLegacyKeccak256()
	sha.Write(serialized)
	digest := sha.Sum(serialized[:0])

	ex.hashedValue = make([]byte, len(digest))
	copy(ex.hashedValue, digest)
	ex.dirty = false

	return digest
}
