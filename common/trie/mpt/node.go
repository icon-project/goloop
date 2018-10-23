package mpt

/*
	A node in a Merkle Patricia trie is one of the following:
	1. NULL (represented as the empty string)
	2. branch A 17-item node [ v0 ... v15, vt ]
	3. leaf A 2-item node [ encodedPath, value ]
	4. extension A 2-item node [ encodedPath, key ]

	and hash node.
	hash node is just byte array having hash of the node.
*/
type (
	node interface {
		hash() []byte
		serialize() []byte
	}
	hash []byte
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
				newBranch.value = buf[tagSize : tagSize+contentSize]
			} else {
				// hash node
				newBranch.nibbles[valueIndex] = hash(buf[tagSize : tagSize+contentSize])
			}

			i += tagSize + contentSize
		} else if 0xC0 < b && b < 0xf7 {
			tagSize, contentSize, _ := getContentSize(buf[i:])
			newBranch.nibbles[valueIndex] = decodeLeaf(buf[i : i+tagSize+contentSize])
			i += tagSize + contentSize
		}
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
	return &leaf{keyEnd: keyBuf, value: valBuf}
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
