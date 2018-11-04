package mpt

import (
	"fmt"

	"github.com/icon-project/goloop/common/trie"
	"golang.org/x/crypto/sha3"
)

type (
	extension struct {
		nodeBase
		sharedNibbles []byte
		next          node
	}
)

// changeState change state from passed state which is returned state by nibbles
func (ex *extension) changeState(s nodeState) {
	if s == dirtyNode && ex.state != dirtyNode {
		ex.state = dirtyNode
	}
}

func (ex *extension) serialize() []byte {
	if ex.state == dirtyNode {
		ex.serializedValue = nil
		ex.hashedValue = nil
	} else if ex.serializedValue != nil { // not dirty & has serialized value
		if printSerializedValue {
			fmt.Println("extension : serialized : ", ex.state)
			fmt.Println("cached serialize extension : ", ex.serializedValue)
		}
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
	ex.state = serializedNode
	if printSerializedValue {
		fmt.Println("serialize extension : ", serialized)
	}
	return serialized
}

func (ex *extension) hash() []byte {
	if ex.state == dirtyNode {
		ex.serializedValue = nil
		ex.hashedValue = nil
	} else if ex.hashedValue != nil { // not diry & has hashed value
		if printHash {
			fmt.Printf("cached hash extension <%x>\n", ex.hashedValue)
		}
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
	ex.state = serializedNode

	if printHash {
		fmt.Printf("hash extension <%x>\n", digest)
	}
	return digest
}

func (ex *extension) addChild(m *mpt, k []byte, v trie.Object) (node, nodeState) {
	//fmt.Println("extension addChild : k ", k, ", v : ", v)
	match, same := compareHex(k, ex.sharedNibbles)
	switch {
	case same == true:
		dirty := dirtyNode
		ex.next, dirty = ex.next.addChild(m, k[match:], v)
		ex.changeState(dirty)
		return ex, dirty
	case match == 0:
		newBranch := &branch{nodeBase: nodeBase{state: dirtyNode}}
		//newBranch.nibbles[k[0]], _ = m.set(nil, k[1:], v)
		newBranch.addChild(m, k, v)
		if len(ex.sharedNibbles) == 1 {
			newBranch.nibbles[ex.sharedNibbles[0]] = ex.next
		} else {
			newBranch.nibbles[ex.sharedNibbles[0]] = ex
			ex.sharedNibbles = ex.sharedNibbles[1:]
		}
		return newBranch, dirtyNode

	// case 2 : 0 < match < len(sharedNibbles) -> new extension
	case match < len(ex.sharedNibbles):
		newBranch := &branch{nodeBase: nodeBase{state: dirtyNode}}
		newExt := &extension{sharedNibbles: k[:match], next: newBranch, nodeBase: nodeBase{state: dirtyNode}}
		if match+1 == len(ex.sharedNibbles) {
			newBranch.nibbles[ex.sharedNibbles[match]] = ex.next
		} else {
			newBranch.nibbles[ex.sharedNibbles[match]] = ex
			ex.sharedNibbles = ex.sharedNibbles[match+1:]
		}
		newBranch.addChild(m, k[match:], v)
		return newExt, dirtyNode
	// case 3 : match < len(k) && len(ex.sharedNibbles) < len(k) -> go to next
	case match < len(k):
		dirty := dirtyNode
		ex.next, dirty = ex.next.addChild(m, k[match:], v)
		ex.changeState(dirty)
		return ex, ex.state
	default:
		panic("Not consider")
	}
	return ex, dirtyNode
}

func (ex *extension) deleteChild(m *mpt, k []byte) (node, nodeState, error) {
	var nextNode node
	// cannot find data. Not exist
	match, _ := compareHex(ex.sharedNibbles, k)
	if len(ex.sharedNibbles) != match {
		return ex, noneNode, nil
	}
	dirty := dirtyNode
	if ex.next == nil {
		return ex, ex.state, nil
	}
	if nextNode, dirty, _ = ex.next.deleteChild(m, k[len(ex.sharedNibbles):]); dirty != dirtyNode {
		return ex, ex.state, nil
	}
	ex.changeState(dirty)
	switch nn := nextNode.(type) {
	// if child node is extension node, merge current node.
	// It can not be possible to link extension from extension directly.
	// extension has only branch as next node.
	case *extension:
		ex.sharedNibbles = append(ex.sharedNibbles, nn.sharedNibbles...)
		ex.next = nn.next
	// if child node is leaf after deleting, this extension must merge next node and be changed to leaf.
	// if child node is leaf, new leaf(keyEnd = extension.key + child.keyEnd, val = child.val)
	case *leaf: // make new leaf and return it
		return &leaf{keyEnd: append(ex.sharedNibbles, nn.keyEnd...), value: nn.value,
			nodeBase: nodeBase{state: dirtyNode}}, dirtyNode, nil
	}
	ex.next = nextNode
	return ex, dirtyNode, nil
}
