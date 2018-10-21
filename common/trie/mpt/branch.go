package mpt

import (
	"golang.org/x/crypto/sha3"
)

type (
	branch struct {
		nibbles         [17]node
		hashedValue     []byte
		serializedValue []byte
		dirty           bool
	}
)

// TODO: optimize serialize
func (br *branch) serialize() []byte {
	var serializedNodes []byte
	listLen := 0
	var serialized []byte
	for i := 0; i < 16; i++ {
		switch br.nibbles[i].(type) {
		case *leaf:
			serialized = br.nibbles[i].serialize()
		case nil:
			serialized = encodeByte(nil)
		default:
			serialized = encodeByte(br.nibbles[i].hash())
		}
		listLen += len(serialized)
		serializedNodes = append(serializedNodes, serialized...)
	}

	if br.nibbles[16] != nil {
		v := br.nibbles[16].(*leaf)
		encodedLeaf := encodeByte(v.value)
		listLen += len(encodedLeaf)
		serializedNodes = append(serializedNodes, encodedLeaf...)
	} else {
		serializedNodes = append(serializedNodes, encodeByte(nil)...)
		listLen++
	}
	serialized = append(makePrefix(listLen, 0xc0), serializedNodes...)
	br.dirty = false
	br.serializedValue = make([]byte, len(serialized))
	copy(br.serializedValue, serialized)
	//	fmt.Println("branch : serialized : ", serialized)
	return serialized
}

func (br *branch) hash() []byte {
	if br.dirty == false && br.hashedValue != nil {
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

	return digest
}
