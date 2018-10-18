package mpt

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

type (
	mpt struct {
		root node
		// committedHash is root hash in database
		committedHash hash
		// Set() inserts key & value into requestPool
		requestPool map[string][]byte
		// keys and values in requestPool are inserted into trie then they move to appliedPool
		appliedPool map[string][]byte
		mutex       sync.Mutex
		db          db.DB
	}
)

/*
 */
func newMpt(initialHash hash) *mpt {
	return &mpt{committedHash: initialHash, requestPool: make(map[string][]byte), appliedPool: make(map[string][]byte)}
}

func bytesToNibbles(k []byte) []byte {
	nibbles := make([]byte, len(k)*2)
	for i, v := range k {
		nibbles[i*2] = v >> 4 & 0x0F
		nibbles[i*2+1] = v & 0x0F
	}
	return nibbles
}

func (m *mpt) get(n node, k []byte) (node, []byte, error) {
	var result []byte
	var err error
	switch n := n.(type) {
	case *branch:
		return m.get(n.nibbles[k[0]], k[1:])
	case *extension:
		match := compareHex(n.sharedNibbles, k)
		n.next, result, err = m.get(n.next, k[match:])
		if err != nil {
			return nil, nil, err
		}
	case *leaf:
		return n, n.val, nil
	// if node is hash get serialized value with hash from db then deserialize it.
	case hash:
		serializedValue, err := m.db.Get(n)
		if err != nil {
			return nil, nil, err
		}
		deserializedNode := deserialize(serializedValue)
		_, result, err = m.get(deserializedNode, k)
		return deserializedNode, result, err
	}
	return nil, result, err
}

func (m *mpt) Get(k []byte) ([]byte, error) {
	k = bytesToNibbles(k)
	if v, ok := m.requestPool[string(k)]; ok {
		return v, nil
	}
	var val []byte
	var err error
	m.root, val, err = m.get(m.root, k)
	if err != nil {
		fmt.Println("Get error : ", err)
		return nil, err
	}
	return val, nil
}

func (m *mpt) RootHash() []byte {
	for k, v := range m.requestPool {
		m.root, _ = m.set(m.root, []byte(k), v)
		delete(m.requestPool, k)
		m.appliedPool[k] = v
	}
	h := m.root.hash()
	return h
}

// return true if current node or child node is changed
func (m *mpt) set(n node, k, v []byte) (node, bool) {
	switch n := n.(type) {
	case *branch:
		n.nibbles[k[0]], n.dirty = m.set(n.nibbles[k[0]], k[1:], v)
	case *extension:
		match := compareHex(k, n.sharedNibbles)
		// case 1 : match = 0 -> new branch
		switch {
		case match == 0:
			newBranch := &branch{}
			newBranch.nibbles[k[0]], _ = m.set(nil, k[1:], v)
			newBranch.nibbles[n.sharedNibbles[0]] = n
			n.sharedNibbles = n.sharedNibbles[1:]
			return newBranch, true

		// case 2 : 0 < match < len(sharedNibbles) -> new extension
		case match < len(n.sharedNibbles):
			newBranch := &branch{}
			newExt := &extension{}
			newExt.sharedNibbles = k[:match]
			newExt.next = newBranch
			if match+1 == len(n.sharedNibbles) {
				newBranch.nibbles[n.sharedNibbles[match]] = n.next
			} else {
				newBranch.nibbles[n.sharedNibbles[match]] = n
				n.sharedNibbles = n.sharedNibbles[match+1:]
			}
			newBranch.nibbles[k[match]], _ = m.set(nil, k[match+1:], v)
			return newExt, true
		// case 3 : match == len(sharedNibbles) -> go to next
		case match < len(k):
			n.next, n.dirty = m.set(n.next, k[match:], v)
		//case match == len(n.sharedNibbles):
		default:
			nextBranch := n.next.(*branch)
			nextBranch.nibbles[16] = &leaf{keyEnd: v, val: v}
		}
	case *leaf:
		match := compareHex(k, n.keyEnd)
		// case 1 : match = 0 -> new branch
		switch {
		case match == 0:
			if bytes.Compare(v, n.val) == 0 { // same key, same value
				return n, false
			}
			newBranch := &branch{}
			if len(k) == 0 {
				newBranch.nibbles[16], _ = m.set(nil, nil, v)
			} else {
				newBranch.nibbles[k[0]], _ = m.set(nil, k[1:], v)
			}

			if len(n.keyEnd) == 0 {
				newBranch.nibbles[16], _ = m.set(nil, nil, n.val)
			} else {
				newBranch.nibbles[n.keyEnd[0]], _ = m.set(nil, n.keyEnd[1:], n.val)
			}

			return newBranch, true
		// case 2 : 0 < match < len(n,val) -> new extension
		case match < len(n.keyEnd):
			newExt := &extension{}
			newExt.sharedNibbles = k[:match]
			newBranch := &branch{}
			newExt.next = newBranch
			if match == len(k) {
				newBranch.nibbles[16], _ = m.set(nil, nil, v)
			} else {
				newBranch.nibbles[k[match]], _ = m.set(nil, k[match+1:], v)
			}
			newBranch.nibbles[n.keyEnd[match]], _ = m.set(nil, n.keyEnd[match+1:], n.val)
			return newExt, true
		// case match == len(n.keyEnd)
		case match < len(k):
			newExt := &extension{}
			newExt.sharedNibbles = k[:match]
			newBranch := &branch{}
			newExt.next = newBranch
			newBranch.nibbles[16], _ = m.set(nil, nil, n.val)
			newBranch.nibbles[k[match]], _ = m.set(nil, k[match+1:], v)
			return newExt, true
		// case 3 : match == len(n.val) -> update value
		default:
			n.val = v
		}
	case hash:
		// TODO: have to check error.
		serializedValue, _ := m.db.Get(k)
		decodeingNode := deserialize(serializedValue)
		return m.set(decodeingNode, k, v)

	default:
		// return new leaf
		return &leaf{keyEnd: k[:], val: v}, true
	}
	return n, false
}

/*
Set inserts key and value into requestPool.
RootHash, Proof, Flush insert keys and values in requestPool into trie
*/
func (m *mpt) Set(k, v []byte) error {
	// TODO: if k or v is nil, return error for invalid param
	if k == nil || v == nil {
		return nil // TODO: proper error
	}
	k = bytesToNibbles(k)
	m.mutex.Lock()
	m.requestPool[string(k)] = v
	m.mutex.Unlock()
	//tr.root, _ = set(tr.root, k, v)
	return nil
}

// TODO: implement delete
func del(n node, k []byte) (node, error) {
	var nextNode node
	switch n := n.(type) {
	case *branch:
		nextNode, _ = del(n.nibbles[k[0]], k[1:])
		n.nibbles[k[0]] = nextNode
		// check remaining nibbles on n(current node)
		// if n has only 1 remaining node, node must be changed to extension.
		// if n has only value with no remaining node, node must be changed to leaf
		index := -1
		for i, nn := range n.nibbles {
			if nn != nil {
				if index != -1 {
					index = i
					break
				}
				index = i
			}
		}

		if index != -1 {
			if index == 16 { // above nextNode would be nil
				return n.nibbles[16], nil
			} else {
				// check nextNode.
				// if nextNode is short or branch, n must be short
				switch nn := n.nibbles[index].(type) {
				case *extension:
					return &extension{sharedNibbles: append([]byte{byte(index)}, nn.sharedNibbles...), next: nn.next}, nil
				case *branch:
					return &extension{sharedNibbles: []byte{byte(index)}, next: nn}, nil
				case *leaf:
					return &leaf{keyEnd: append([]byte{byte(index)}, nn.keyEnd...), val: nn.val}, nil
				}
			}
		}
		n.nibbles[k[0]] = nextNode

	case *extension:
		// if child node is extension node, merge current node.
		// It can not be possible to link extension from extension directly.
		// extension has only branch as next node.

		// if child node is leaf after deleting, this extension must merge child node and be changed to leaf.
		// if child node is leaf, new leaf(keyEnd = extension.key + child.keyEnd, val = child.val)
		switch nn := nextNode.(type) {
		case *extension:
			n.sharedNibbles = k[:len(nn.sharedNibbles)+1]
			n.next = nn.next
		case *leaf: // make new leaf and return it
			return &leaf{keyEnd: k, val: nn.val}, nil
		}

	case *leaf:
		return nil, nil
	}
	return n, nil
}

func (m *mpt) Delete(k []byte) error {
	var err error
	k = bytesToNibbles(k)
	m.root, err = del(m.root, k)
	return err
}

func (m *mpt) GetSnapshot() trie.Snapshot {
	mpt := newMpt(m.committedHash)
	m.mutex.Lock()
	for k, v := range m.requestPool {
		mpt.requestPool[k] = v
	}
	for k, v := range m.appliedPool {
		mpt.requestPool[k] = v
	}
	m.mutex.Unlock()

	return mpt
}

func traversalCommit(n node, db db.DB) error {
	switch n := n.(type) {
	case *branch:
		for _, v := range n.nibbles {
			if err := traversalCommit(v, db); err != nil {
				return err
			}
		}
	case *extension:
		if err := traversalCommit(n.next, db); err != nil {
			return err
		}
	default:
		return nil
	}
	return n.commit(db)
}

/*
 */
func (m *mpt) Flush(db db.DB) error {
	for k, v := range m.requestPool {
		m.root, _ = m.set(m.root, []byte(k), v)
		delete(m.requestPool, string(k))
	}

	if err := traversalCommit(m.root, db); err != nil {
		return err
	}
	m.committedHash = m.root.hash()
	m.db = db
	return nil
}

func addProof(buf [][]byte, index int, hash []byte) {
	if len(buf) == index {
		buf = make([][]byte, len(buf)+10)
	}
	copy(buf[index], hash)
}

func (m *mpt) proof(n node, k []byte) ([][]byte, int) {
	var buf [][]byte
	var i int
	switch n := n.(type) {
	case *branch:
		buf, i = m.proof(n.nibbles[k[0]], k[1:])
		if n.hashedValue == nil {
			addProof(buf, i, n.serialize())
		} else {
			addProof(buf, i, n.hashedValue)
		}
	case *extension:
		match := compareHex(n.sharedNibbles, k)
		buf, i = m.proof(n.next, k[match:])
		if n.hashedValue == nil {
			addProof(buf, i, n.serialize())
		} else {
			addProof(buf, i, n.hashedValue)
		}
	case *leaf:
		return nil, 0
	case hash:
		// TODO: have to check error
		serializedValued, _ := m.db.Get(k)
		decodeingNode := deserialize(serializedValued)
		return m.proof(decodeingNode, k)
	}
	return buf, i + 1
}

// TODO: Implement Proof
func (m *mpt) Proof(k []byte) [][]byte {
	m.root.serialize()
	k = bytesToNibbles(k)
	buf, _ := m.proof(m.root, k)
	return buf
}

// TODO: proper error
func (m *mpt) Reset(immutable trie.Immutable) error {
	in, ok := immutable.(*mpt)
	if ok == false {
		return nil
	}

	kv := make(map[string][]byte)
	for k, v := range in.requestPool {
		kv[k] = v
	}
	for k, v := range in.appliedPool {
		kv[k] = v
	}

	m.requestPool = kv
	m.committedHash = make([]byte, len(in.committedHash))
	copy(m.committedHash, in.committedHash)
	rootHash := make([]byte, len(in.committedHash))
	copy(rootHash, in.committedHash)
	m.root = hash(rootHash)
	m.db = in.db
	return nil
}
