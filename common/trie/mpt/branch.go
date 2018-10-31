package mpt

import (
	"fmt"
	"github.com/icon-project/goloop/common/trie"
	"golang.org/x/crypto/sha3"
)

type (
	branch struct {
		nibbles [16]node
		value   trie.Object

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
		case nil:
			serialized = encodeByte(nil)
		default:
			if serialized = br.nibbles[i].serialize(); hashableSize <= len(serialized) {
				serialized = encodeByte(br.nibbles[i].hash())
			}
		}
		serializedNodes = append(serializedNodes, serialized...)
	}

	if br.value == nil {
		serialized = encodeList(serializedNodes, encodeByte(nil))
	} else {
		// value of branch does not use hash
		serialized = encodeList(serializedNodes, encodeByte(br.value.Bytes()))
	}
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
	serializedCopy := make([]byte, len(serialized))
	copy(serializedCopy, serialized)
	// TODO: have to change below sha function.
	sha := sha3.NewLegacyKeccak256()
	sha.Write(serializedCopy)
	digest := sha.Sum(serializedCopy[:0])

	br.hashedValue = make([]byte, len(digest))
	copy(br.hashedValue, digest)
	br.dirty = false

	if printHash {
		fmt.Printf("hash branch : <%x>\n", digest)
	}

	return digest
}

func (br *branch) addChild(m *mpt, k []byte, v trie.Object) (node, bool) {
	if len(k) == 0 {
		br.value = v
		return br, true
	}
	br.nibbles[k[0]], br.dirty = m.set(br.nibbles[k[0]], k[1:], v)
	return br, true
}

func (br *branch) deleteChild(m *mpt, k []byte) (node, bool, error) {
	var nextNode node
	if nextNode, br.dirty, _ = m.delete(br.nibbles[k[0]], k[1:]); br.dirty == false {
		return br, false, nil
	}
	br.nibbles[k[0]] = nextNode

	// check remaining nibbles on n(current node)
	// 1. if n has only 1 remaining node after deleting, n will be removed and the remaining node will be changed to extension.
	// 2. if n has only value with no remaining node after deleting, node must be changed to leaf
	// Branch has least 2 nibbles before deleting so branch cannot be empty after deleting
	remainingNibble := 16
	for i, nn := range br.nibbles {
		if nn != nil {
			if remainingNibble != 16 { // already met another nibble
				remainingNibble = -1
				break
			}
			remainingNibble = i
		}
	}

	//If remainingNibble is -1, branch has 2 more nibbles.
	if remainingNibble != -1 {
		if remainingNibble == 16 {
			return &leaf{value: br.value}, true, nil
		} else {
			// check nextNode.
			// if nextNode is extension or branch, n must be extension
			switch nn := br.nibbles[remainingNibble].(type) {
			case *extension:
				return &extension{sharedNibbles: append([]byte{byte(remainingNibble)}, nn.sharedNibbles...),
					next: nn.next, dirty: true}, true, nil
			case *branch:
				return &extension{sharedNibbles: []byte{byte(remainingNibble)}, next: nn, dirty: true}, true, nil
			case *leaf:
				return &leaf{keyEnd: append([]byte{byte(remainingNibble)}, nn.keyEnd...), value: nn.value, dirty: true}, true, nil
			}
		}
	}
	return br, true, nil
}
