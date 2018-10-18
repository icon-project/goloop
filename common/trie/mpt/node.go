package mpt

import (
	"github.com/icon-project/goloop/common/db"
	"golang.org/x/crypto/sha3"
)

/*
	A node in a Merkle Patricia trie is one of the following:
	1. NULL (represented as the empty string)
	2. branch A 17-item node [ v0 ... v15, vt ]
	3. leaf A 2-item node [ encodedPath, value ]
	4. extension A 2-item node [ encodedPath, key ]
*/
type (
	node interface {
		hash() []byte
		serialize() []byte
		commit(db db.DB) error
	}

	branch struct {
		nibbles         [17]node
		hashedValue     []byte
		serializedValue []byte
		dirty           bool
	}

	extension struct {
		sharedNibbles   []byte
		next            node
		hashedValue     []byte
		serializedValue []byte
		dirty           bool
	}

	leaf struct {
		keyEnd          []byte
		val             []byte
		hashedValue     []byte
		serializedValue []byte
		dirty           bool
	}

	hash []byte
)

func (br *branch) value() []byte {
	return nil
}

func (br *branch) serialize() []byte {
	var serializedNodes []byte
	listLen := 0
	var serialized []byte
	for i := 0; i < 16; i++ {
		if br.nibbles[i] != nil {
			switch br.nibbles[i].(type) {
			case *leaf:
				serialized = br.nibbles[i].serialize()
			default:
				serialized = encodeByte(br.nibbles[i].hash())
			}
		} else {
			serialized = []byte{0x80}
		}
		listLen += len(serialized)
		serializedNodes = append(serializedNodes, serialized...)
	}

	if br.nibbles[16] != nil {
		v := br.nibbles[16].(*leaf)
		encodedLeaf := encodeByte(v.val)
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

func (br *branch) commit(db db.DB) error {
	if br.dirty == true || br.serializedValue == nil || br.hashedValue == nil {
		br.hash()
	}

	db.Set(br.hashedValue, br.serializedValue)
	return nil
}

func nilbbesToHex(nibbles []byte) []byte {
	keyLen := len(nibbles)
	keyArray := make([]byte, keyLen/2)

	for i := 0; i < keyLen/2; i++ {
		keyArray[i] = nibbles[i*2]<<4 | nibbles[i*2+1]
	}
	return keyArray
}

func (ex *extension) serialize() []byte {
	keyLen := len(ex.sharedNibbles)
	keyArray := make([]byte, 1)
	index := 0
	if keyLen%2 == 1 {
		keyArray[0] = 1<<4 | ex.sharedNibbles[0]
		index = 1
	} else {
		keyArray[0] = 0
	}
	keyArray = append(keyArray, nilbbesToHex(ex.sharedNibbles[index:])...)
	serialized := encodeList(encodeByte(keyArray), encodeByte(ex.next.hash()))
	ex.serializedValue = make([]byte, len(serialized))
	copy(ex.serializedValue, serialized)
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

func (ex *extension) commit(db db.DB) error {
	if ex.dirty == true || ex.serializedValue == nil || ex.hashedValue == nil {
		ex.hash()
	}

	db.Set(ex.hashedValue, ex.serializedValue)
	return nil
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

	return encodeList(encodeByte(keyArray), encodeByte(l.val))
}

func (l *leaf) hash() []byte {
	if l.dirty == false && l.hashedValue != nil {
		return l.hashedValue
	}
	serialized := l.serialize()
	// TODO: have to change below sha function.
	sha := sha3.NewLegacyKeccak256()
	sha.Write(serialized)
	digest := sha.Sum(serialized[:0])

	l.hashedValue = digest
	l.serializedValue = serialized
	l.dirty = false

	return digest
}

func (l *leaf) commit(db db.DB) error {
	return nil
}

func (h hash) serialize() []byte {
	return h
}

func (h hash) hash() []byte {
	return h
}

func (h hash) commit(db db.DB) error {
	return nil
}

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
	key, _ = decodeKey(key)
	return &extension{sharedNibbles: key, next: hash(buf[valOffset : valOffset+valContentSize])}
}

func decodeBranch(buf []byte) node {
	// serialized branch can have list which is another branch(sharednibbles/value) or a leaf(keyEnd/value) or  hexa(serialized(rlp))
	tagSize, contentSize, _ := getContentSize(buf)
	// child is leaf, hash or nil(128)
	valueIndex := 0
	newBranch := &branch{}
	for i := tagSize; i < tagSize+contentSize; {
		// if list, call decoderLear
		// if single byte
		b := buf[i]
		if b < 0x80 { // hash or value if valueIndex is 16
			newBranch.nibbles[valueIndex] = nil
			i++
		} else if b < 0xb8 {
			tagSize, contentSize, _ := getContentSize(buf[i:])
			buf := buf[i:]
			newBranch.nibbles[valueIndex] = hash(buf[tagSize : tagSize+contentSize])
			i += tagSize + contentSize
		} else if 0xC0 < b && b < 0xf7 {
			tagSize, contentSize, _ := getContentSize(buf[i:])
			newBranch.nibbles[valueIndex] = decodeLeaf(buf[i : i+tagSize+contentSize])
			i += tagSize + contentSize
		}
		valueIndex++
		// TODO: have to check last index. last index has only encoded value
	}
	return newBranch
}

func decodeKey(buf []byte) ([]byte, error) {
	firstNib := buf[0] >> 4
	var newBuf []byte
	index := 0
	if firstNib%2 == 0 { // even. first byte is just padding byte
		newBuf = make([]byte, (len(buf)-1)*2)
	} else { // odd
		newBuf = make([]byte, (len(buf)*2 - 1))
		newBuf[0] = buf[0] & 0x0F
		index = 1
	}

	buf = buf[1:]
	for i := 0; i < len(buf); i++ {
		newBuf[i*2+index] = buf[i] >> 4
		newBuf[i*2+1+index] = buf[i] & 0x0F
	}
	return newBuf, nil
}

func decodeLeaf(buf []byte) node {
	tagSize, _, _ := getContentSize(buf)
	// get key
	keyTagSize, keyContentSize, _ := getContentSize(buf[tagSize:])
	keyBuf := buf[tagSize+keyTagSize : tagSize+keyTagSize+keyContentSize]
	keyBuf, _ = decodeKey(keyBuf)
	offset := tagSize + keyTagSize + keyContentSize
	valTagSize, valContentSize, _ := getContentSize(buf[offset:])
	valBuf := buf[offset+valTagSize : offset+valTagSize+valContentSize]
	return &leaf{keyEnd: keyBuf, val: valBuf}
}

// TODO: have to modify. ethereum code
func deserialize(buf []byte) node {
	switch c, _ := countListMember(buf); c {
	case 2:
		n := decodeExtension(buf)
		return n
	case 17:
		n := decodeBranch(buf)
		return n
	default:
		return nil
	}
	return nil
}
