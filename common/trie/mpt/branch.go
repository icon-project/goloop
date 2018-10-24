package mpt

import (
	"fmt"
	"golang.org/x/crypto/sha3"
)

type (
	branch struct {
		nibbles         [16]node
		value           []byte
		hashedValue     []byte
		serializedValue []byte
		dirty           bool // if dirty is true, must retry getting hashedValue & serializedValue
	}
)

func (br *branch) serialize() []byte {
	if br.dirty == true {
		br.serializedValue = nil
		br.hashedValue = nil
	} else if br.serializedValue != nil { // not dirty & has serialized value
		return br.serializedValue
	}

	var serializedNodes []byte
	var serialized []byte
	for i := 0; i < 16; i++ {
		switch br.nibbles[i].(type) {
		case *leaf:
			serialized = br.nibbles[i].serialize()
		case nil:
			serialized = encodeByte(nil)
		default:
			if serialized = br.nibbles[i].serialize(); hashableSize <= len(serialized) {
				serialized = encodeByte(br.nibbles[i].hash())
			}
		}
		serializedNodes = append(serializedNodes, serialized...)
	}

	serialized = encodeList(serializedNodes, encodeByte(br.value))
	br.serializedValue = make([]byte, len(serialized))
	copy(br.serializedValue, serialized)
	br.hashedValue = nil
	br.dirty = false

	if printSerializedValue {
		fmt.Println("serialize branch : ", serialized)
	}
	return serialized
}

func (br *branch) hash() []byte {
	if br.dirty == true {
		br.serializedValue = nil
		br.hashedValue = nil
	} else if br.hashedValue != nil { // not diry & has hashed value
		return br.hashedValue
	}

	serialized := br.serialize()
	// TODO: have to change below sha function.
	sha := sha3.NewLegacyKeccak256()
	sha.Write(serialized)
	digest := sha.Sum(serialized[:0])

	br.hashedValue = make([]byte, len(digest))
	copy(br.hashedValue, digest)
	br.dirty = false

	if printHash {
		fmt.Printf("hash branch : <%x>\n", digest)
	}

	return digest
}
