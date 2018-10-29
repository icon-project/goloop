package mpt

import (
	"bytes"
	"reflect"
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
	trieValue interface {
		Bytes() []byte
		// Serialize() byte
		Compare(v trieValue) bool
		//Clone() bool
		//Flush() bool
	}
	node interface {
		hash() []byte
		serialize() []byte
		// TODO: test hashable // if seriazlied data size is bigger than 32, serialize() returns hash(serialize)
		//serialize(hashable bool) []byte
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

func (v byteValue) Bytes() []byte {
	return v
}
func (v byteValue) Compare(t trieValue) bool {
	if b, ok := t.(byteValue); ok {
		return bytes.Compare(b, v) == 0
	}
	return false

}

//func decodeLeaf(buf []byte) node {
//	tagSize, _, _ := getContentSize(buf)
//	// get key
//	keyTagSize, keyContentSize, _ := getContentSize(buf[tagSize:])
//	keyBuf := buf[tagSize+keyTagSize : tagSize+keyTagSize+keyContentSize]
//	keyBuf, _, _ = decodeKey(keyBuf)
//	offset := tagSize + keyTagSize + keyContentSize
//	valTagSize, valContentSize, _ := getContentSize(buf[offset:])
//	valBuf := buf[offset+valTagSize : offset+valTagSize+valContentSize]
//	return &leaf{keyEnd: keyBuf, value: valBuf}
//}

func decodeExtension(buf []byte) node {
	// serialized extension has sharedNibbles & hash of branch
	// get list tagSize and content size
	// tagSize, contentSize, _ := getContentSize(buf)
	tagSize, _, _ := getContentSize(buf)
	// get key tag size and key length
	keyTagSize, keyContentSize, _ := getContentSize(buf[tagSize:])
	// get value tag size and value length
	valTagSize, valContentSize, _ := getContentSize(buf[tagSize+keyTagSize+keyContentSize:])
	valOffset := tagSize + keyTagSize + keyContentSize + valTagSize
	key := buf[tagSize+keyTagSize : tagSize+keyTagSize+keyContentSize]
	key, _, _ = decodeKey(key)
	// TODO: if length of decodded data is bigger than hashable, the data is set to hash
	// but shorter than hashable, the data is set to seriazlie
	return &extension{sharedNibbles: key, next: hash(buf[valOffset : valOffset+valContentSize]), serializedValue: buf}
}

func decodeValue(buf []byte, t reflect.Type) trieValue {
	var result trieValue
	// TODO: check how to compare type
	if t == nil {
		panic("PANIC")
	}
	if t == reflect.TypeOf([]byte{}) {
		result = byteValue(buf)
	}
	return result
}

func decodeLeafExt(buf []byte, t reflect.Type) node {
	tagSize, _, _ := getContentSize(buf)
	keyTagSize, keyContentSize, _ := getContentSize(buf[tagSize:])
	// get value tag size and value length
	valTagSize, valContentSize, _ := getContentSize(buf[tagSize+keyTagSize+keyContentSize:])
	valOffset := tagSize + keyTagSize + keyContentSize + valTagSize
	key := buf[tagSize+keyTagSize : tagSize+keyTagSize+keyContentSize]
	var nodeType int
	key, nodeType, _ = decodeKey(key)
	if nodeType == 0 { //extension
		return &extension{sharedNibbles: key, next: hash(buf[valOffset : valOffset+valContentSize]), serializedValue: buf}
	}
	l := &leaf{keyEnd: key, value: decodeValue(buf[valOffset:valOffset+valContentSize], t), serializedValue: buf}
	return l

}
func decodeBranch(buf []byte, t reflect.Type) node {
	// serialized branch can have list which is another branch(sharednibbles/value) or a leaf(keyEnd/value) or  hexa(serialized(rlp))
	tagSize, contentSize, _ := getContentSize(buf)
	// child is leaf, hash or nil(128)
	newBranch := &branch{}
	for i, valueIndex := tagSize, 0; i < tagSize+contentSize; valueIndex++ {
		// if list, call decoderLear
		// if single byte
		b := buf[i]
		if b < 0x80 { // hash or value if valueIndex is 16
			newBranch.nibbles[valueIndex] = nil
			i++
		} else if b < 0xb8 {
			tagSize, contentSize, _ := getContentSize(buf[i:])
			buf := buf[i:]
			if valueIndex == 16 {
				newBranch.value = decodeValue(buf[tagSize:tagSize+contentSize], t)
			} else {
				// hash node
				if contentSize == 0 {
					newBranch.nibbles[valueIndex] = nil
				} else {
					newBranch.nibbles[valueIndex] = hash(buf[tagSize : tagSize+contentSize])
				}
			}

			i += tagSize + contentSize
		} else if 0xC0 < b && b < 0xf7 {
			tagSize, contentSize, _ := getContentSize(buf[i:])
			newBranch.nibbles[valueIndex] = decodeLeafExt(buf[i:i+tagSize+contentSize], t)
			i += tagSize + contentSize
		}
	}
	return newBranch
}

// even : 00 or 20 bit sequence
// odd : 1X or 3X bit sequence

//0        0000    |       extension              even
//1        0001    |       extension              odd
//2        0010    |   terminating (leaf)         even
//3        0011    |   terminating (leaf)         odd

// get first nibble and check if 0x2 | nibble is true, leaf. if not, extension
//2nd bit is 1, leaf
// if nodeType is 0, extension. leaf is 1
func decodeKey(buf []byte) (keyBuf []byte, nodeType int, err error) {
	firstNib := buf[0] >> 4
	index := 0

	nodeType = 0
	if firstNib&0x2 == 0x2 {
		nodeType = 1
	}
	if firstNib%2 == 0 { // even. first byte is just padding byte
		keyBuf = make([]byte, (len(buf)-1)*2)
	} else { // odd
		keyBuf = make([]byte, (len(buf)*2 - 1))
		keyBuf[0] = buf[0] & 0x0F
		index = 1
	}

	buf = buf[1:]
	for i := 0; i < len(buf); i++ {
		keyBuf[i*2+index] = buf[i] >> 4
		keyBuf[i*2+1+index] = buf[i] & 0x0F
	}
	return keyBuf, nodeType, nil
}

func deserialize(buf []byte, t reflect.Type) node {
	switch c, _ := countListMember(buf); c {
	case 2:
		n := decodeLeafExt(buf, t)
		return n
	case 17:
		n := decodeBranch(buf, t)
		return n
	default:
		return nil
	}
	return nil
}
