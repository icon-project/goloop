package mpt

import (
	"fmt"
	"github.com/icon-project/goloop/common/trie"
	"golang.org/x/crypto/sha3"
)

type (
	branch struct {
		nodeBase
		nibbles [16]node
		value   trie.Object
	}
)

// changeState change state from passed state which is returned state by nibbles
func (br *branch) changeState(s nodeState) {
	if s == dirtyNode && br.state != dirtyNode {
		br.state = dirtyNode
	}
}

func (br *branch) serialize() []byte {
	if br.state == dirtyNode {
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
	br.state = serializedNode

	if printSerializedValue {
		fmt.Println("serialize branch : ", serialized)
	}
	return serialized
}

func (br *branch) hash() []byte {
	if br.state == dirtyNode {
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
	br.state = serializedNode

	if printHash {
		fmt.Printf("hash branch : <%x>\n", digest)
	}

	return digest
}

func (br *branch) addChild(m *mpt, k []byte, v trie.Object) (node, nodeState) {
	//fmt.Println("branch addChild : k ", k, ", v : ", v)
	if len(k) == 0 {
		if v.Equal(br.value) == true {
			return br, br.state
		}
		br.value = v
		br.state = dirtyNode
		return br, dirtyNode
	}
	dirty := dirtyNode
	if br.nibbles[k[0]] == nil {
		br.nibbles[k[0]], dirty = m.set(br.nibbles[k[0]], k[1:], v)
	} else {
		br.nibbles[k[0]], dirty = br.nibbles[k[0]].addChild(m, k[1:], v)
	}
	br.changeState(dirty)
	return br, br.state
}

func (br *branch) deleteChild(m *mpt, k []byte) (node, nodeState, error) {
	//fmt.Println("branch deleteChild : k ", k)
	var nextNode node
	if len(k) == 0 {
		br.value = nil
		br.state = dirtyNode
	} else {
		dirty := dirtyNode
		if br.nibbles[k[0]] == nil {
			return br, br.state, nil
		}

		if nextNode, dirty, _ = br.nibbles[k[0]].deleteChild(m, k[1:]); dirty != dirtyNode {
			return br, br.state, nil
		}
		br.nibbles[k[0]] = nextNode
		br.changeState(dirty)
	}

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
		if br.value == nil {
			// check nextNode.
			// if nextNode is extension or branch, n must be extension
			switch nn := br.nibbles[remainingNibble].(type) {
			case *extension:
				return &extension{sharedNibbles: append([]byte{byte(remainingNibble)}, nn.sharedNibbles...),
					next: nn.next, nodeBase: nodeBase{state: dirtyNode}}, dirtyNode, nil
			case *branch:
				return &extension{sharedNibbles: []byte{byte(remainingNibble)}, next: nn,
					nodeBase: nodeBase{state: dirtyNode}}, dirtyNode, nil
			case *leaf:
				return &leaf{keyEnd: append([]byte{byte(remainingNibble)}, nn.keyEnd...), value: nn.value,
					nodeBase: nodeBase{state: dirtyNode}}, dirtyNode, nil
			}
		} else if remainingNibble == 16 {
			return &leaf{value: br.value, nodeBase: nodeBase{state: dirtyNode}}, dirtyNode, nil
		}
	}
	return br, dirtyNode, nil
}
