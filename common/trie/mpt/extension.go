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

func (ex *extension) addChild(m *mpt, k []byte, v trieValue) (node, bool) {
	match := compareHex(k, ex.sharedNibbles)
	switch {
	case match == 0:
		newBranch := &branch{}
		newBranch.nibbles[k[0]], _ = m.set(nil, k[1:], v)
		if len(ex.sharedNibbles) == 1 {
			newBranch.nibbles[ex.sharedNibbles[0]] = ex.next
		} else {
			newBranch.nibbles[ex.sharedNibbles[0]] = ex
			ex.sharedNibbles = ex.sharedNibbles[1:]
		}
		return newBranch, true

	// case 2 : 0 < match < len(sharedNibbles) -> new extension
	case match < len(ex.sharedNibbles):
		newBranch := &branch{}
		newExt := &extension{}
		newExt.sharedNibbles = k[:match]
		newExt.next = newBranch
		if match+1 == len(ex.sharedNibbles) {
			newBranch.nibbles[ex.sharedNibbles[match]] = ex.next
		} else {
			newBranch.nibbles[ex.sharedNibbles[match]] = ex
			ex.sharedNibbles = ex.sharedNibbles[match+1:]
		}
		if match == len(k) {
			newBranch.value = v
		} else {
			newBranch.nibbles[k[match]], _ = m.set(nil, k[match+1:], v)
		}
		return newExt, true
	// case 3 : match == len(sharedNibbles) -> go to next
	case match < len(k):
		ex.next, ex.dirty = m.set(ex.next, k[match:], v)
	//case match == len(n.sharedNibbles):
	default:
		nextBranch := ex.next.(*branch)
		nextBranch.value = v
	}
	return ex, true
}

func (ex *extension) deleteChild(m *mpt, k []byte) (node, bool, error) {
	var nextNode node
	// cannot find data. Not exist
	if nextNode, ex.dirty, _ = m.delete(ex.next, k[len(ex.sharedNibbles):]); ex.dirty == false {
		return ex, false, nil
	}
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
		return &leaf{keyEnd: append(ex.sharedNibbles, nn.keyEnd...), value: nn.value}, true, nil
	}
	ex.next = nextNode
	return ex, true, nil
}
