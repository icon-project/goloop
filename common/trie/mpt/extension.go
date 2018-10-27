package mpt

import (
	"fmt"
	"golang.org/x/crypto/sha3"
)

type (
	extension struct {
		sharedNibbles []byte
		next          node

		hashedValue     []byte
		serializedValue []byte
		dirty           bool // if dirty is true, must retry getting hashedValue & serializedValue
	}
)

func (ex *extension) serialize() []byte {
	if ex.dirty == true {
		ex.serializedValue = nil
		ex.hashedValue = nil
	} else if ex.serializedValue != nil { // not dirty & has serialized value
		return ex.serializedValue
	}

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

	var serialized []byte
	if serialized = ex.next.serialize(); hashableSize <= len(serialized) {
		serialized = encodeByte(ex.next.hash())
	}
	serialized = encodeList(encodeByte(keyArray), serialized)
	ex.serializedValue = make([]byte, len(serialized))
	copy(ex.serializedValue, serialized)
	ex.hashedValue = nil
	ex.dirty = false
	if printSerializedValue {
		fmt.Println("serialize extension : ", serialized)
	}
	return serialized
}

func (ex *extension) hash() []byte {
	if ex.dirty == true {
		ex.serializedValue = nil
		ex.hashedValue = nil
	} else if ex.hashedValue != nil { // not diry & has hashed value
		return ex.hashedValue
	}

	serialized := ex.serialize()
	serializeCopied := make([]byte, len(serialized))
	copy(serializeCopied, serialized)
	// TODO: have to change below sha function.
	sha := sha3.NewLegacyKeccak256()
	sha.Write(serializeCopied)
	digest := sha.Sum(serializeCopied[:0])

	ex.hashedValue = make([]byte, len(digest))
	copy(ex.hashedValue, digest)
	ex.dirty = false

	if printHash {
		fmt.Printf("hash extension <%x>\n", digest)
	}
	return digest
}
