package mpt

import (
	"bytes"
	"reflect"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

type (
	source struct {
		// prevTrie is nil after Flush()
		prevTrie      *source
		committedHash hash
		requestPool   map[string]trie.Object
	}

	mpt struct {
		root    node
		objType reflect.Type

		rootHashed bool
		curTrie    *source
		mutex      sync.Mutex
		db         db.DB
	}
)

/*
 */
func newMpt(db db.DB, initialHash hash) *mpt {
	return &mpt{root: hash(append([]byte(nil), []byte(initialHash)...)),
		curTrie: &source{requestPool: make(map[string]trie.Object), committedHash: hash(append([]byte(nil), []byte(initialHash)...))},
		db:      db, objType: reflect.TypeOf([]byte{})}
}

func bytesToNibbles(k []byte) []byte {
	nibbles := make([]byte, len(k)*2)
	for i, v := range k {
		nibbles[i*2] = v >> 4 & 0x0F
		nibbles[i*2+1] = v & 0x0F
	}
	return nibbles
}

func (m *mpt) get(n node, k []byte) (node, trie.Object, error) {
	var result trie.Object
	var err error
	switch n := n.(type) {
	case *branch:
		if len(k) != 0 {
			n.nibbles[k[0]], result, err = m.get(n.nibbles[k[0]], k[1:])
		} else {
			result = n.value
		}
	case *extension:
		match := compareHex(n.sharedNibbles, k)
		n.next, result, err = m.get(n.next, k[match:])
		if err != nil {
			return nil, nil, err
		}
	case *leaf:
		return n, n.value, nil
	// if node is hash, get serialized value with hash from db then deserialize it.
	case hash:
		serializedValue, err := m.db.Get(n)
		if err != nil {
			return nil, nil, err
		}
		deserializedNode := deserialize(serializedValue, m.objType)
		switch m := deserializedNode.(type) {
		case *branch:
			m.hashedValue = n
		case *extension:
			m.hashedValue = n
		case *leaf:
			m.hashedValue = n
		}
		return m.get(deserializedNode, k)
	}
	return n, result, err
}

func (m *mpt) Get(k []byte) ([]byte, error) {
	k = bytesToNibbles(k)
	if v, ok := m.curTrie.requestPool[string(k)]; ok {
		return v.(byteValue), nil
	}
	var value trie.Object
	var err error
	m.root, value, err = m.get(m.root, k)
	if err != nil {
		return nil, err
	}
	return value.Bytes(), nil
}

func (m *mpt) evaluateTrie(requestPool map[string]trie.Object) {
	for k, v := range requestPool {
		if v == nil {
			m.root, _, _ = m.delete(m.root, []byte(k))
		} else {
			m.root, _ = m.set(m.root, []byte(k), v)
		}
	}
}

/*
	RootHash
*/
func (m *mpt) RootHash() []byte {
	if m.rootHashed == true {
		return m.root.hash()
	}
	pool, lastCommitedHash := m.mergeSnapshot()
	committedHash := lastCommitedHash
	if len(committedHash) != 0 {
		m.root = committedHash
	}
	// That length of pool is zero means that this trie is already calculated
	if len(pool) == 0 {
		return m.root.hash()
	}
	m.evaluateTrie(pool)
	h := m.root.hash()
	m.rootHashed = true
	// Do not set nil to requestPool and prevSnapshot because next snapshot want data in previous snapshot
	//m.requestPool = nil
	//m.prevSnapshot = nil
	return h
}

// return true if current node or child node is changed
func (m *mpt) set(n node, k []byte, v trie.Object) (node, bool) {
	//fmt.Println("set n ", n,", k ", k, ", v : ", string(v.(byteValue)))
	if n == nil {
		return &leaf{keyEnd: k[:], value: v}, true
	}

	return n.addChild(m, k, v)
}

/*
Set inserts key and value into requestPool.
RootHash, Proof, Flush insert keys and values in requestPool into trie
*/
func (m *mpt) Set(k, v []byte) error {
	if k == nil || v == nil {
		return common.ErrIllegalArgument
	}
	k = bytesToNibbles(k)
	m.mutex.Lock()
	copied := make([]byte, len(v))
	copy(copied, v)
	m.curTrie.requestPool[string(k)] = byteValue(append([]byte(nil), v...))
	m.mutex.Unlock()
	return nil
}

func (m *mpt) delete(n node, k []byte) (node, bool, error) {
	//fmt.Println("delete n = ", n, ", k = ", k)
	if n == nil {
		return n, false, nil
	}

	return n.deleteChild(m, k)
}

func (m *mpt) Delete(k []byte) error {
	var err error
	k = bytesToNibbles(k)
	m.mutex.Lock()
	m.curTrie.requestPool[string(k)] = nil
	m.mutex.Unlock()
	return err
}

func (m *mpt) GetSnapshot() trie.Snapshot {
	mpt := newMpt(m.db, m.curTrie.committedHash)
	m.mutex.Lock()
	mpt.curTrie = m.curTrie
	// Below means s1.Flush() was called after calling m.Reset(s1)
	if bytes.Compare(m.curTrie.committedHash, m.curTrie.prevTrie.committedHash) != 0 {
		m.curTrie.committedHash = hash(append([]byte(nil), []byte(m.curTrie.prevTrie.committedHash)...))
	}
	//fmt.Println("GetSnapshot. committedHash = ", m.curTrie.committedHash)
	m.curTrie = &source{committedHash: mpt.curTrie.committedHash, prevTrie: mpt.curTrie, requestPool: make(map[string]trie.Object)}
	m.mutex.Unlock()

	return mpt
}

func (m *mpt) mergeSnapshot() (map[string]trie.Object, hash) {
	mergePool := make(map[string]trie.Object)
	var committedHash hash
	for snapshot := m.curTrie; snapshot != nil; snapshot = snapshot.prevTrie {
		for k, v := range snapshot.requestPool {
			// add only not existing key
			if _, ok := mergePool[k]; ok == false {
				mergePool[k] = v
			}
		}
		committedHash = snapshot.committedHash
	}
	return mergePool, committedHash
}

// TODO: check whether this node is stored or not
func traversalCommit(db db.DB, n node) error {
	switch n := n.(type) {
	case *branch:
		for _, v := range n.nibbles {
			if err := traversalCommit(db, v); err != nil {
				return err
			}
		}
	case *extension:
		if err := traversalCommit(db, n.next); err != nil {
			return err
		}
	case *leaf:
		serialized := n.serialize()
		// if length of serialized leaf is smaller hashable(32), parent node (branch) must have serialized data of this
		if len(serialized) < hashableSize {
			return nil
		}
	default:
		return nil
	}
	return db.Set(n.hash(), n.serialize())
}

/*
	Flush saves all updated nodes to db.
	Requested data are inserted to db so the requested data in pool are cleared
	And preve
*/
func (m *mpt) Flush() error {
	pool, lastCommitedHash := m.mergeSnapshot()
	m.curTrie.committedHash = lastCommitedHash

	m.curTrie.requestPool = pool
	if len(pool) != 0 {
		if m.rootHashed == false {
			if len(lastCommitedHash) != 0 {
				m.root = lastCommitedHash
			}
			m.evaluateTrie(pool)
			m.rootHashed = true
		}
		if err := traversalCommit(m.db, m.root); err != nil {
			return err
		}
		m.curTrie.committedHash = m.root.hash()
	} else {
		m.root = m.curTrie.committedHash
	}

	m.curTrie.requestPool = nil
	m.curTrie.prevTrie = nil
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
		decodeingNode := deserialize(serializedValued, m.objType)
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

func (m *mpt) Load(db db.DB, root []byte) error {
	// use db to check validation
	if _, err := db.Get(root); err != nil {
		return err
	}

	m.curTrie.committedHash = root
	m.root = hash(root)
	m.db = db
	return nil
}

func (m *mpt) Reset(immutable trie.Immutable) error {
	immutableTrie, ok := immutable.(*mpt)
	if ok == false {
		return common.ErrIllegalArgument
	}

	// Do not use reference.
	committedHash := make(hash, len(immutableTrie.curTrie.committedHash))
	copy(committedHash, immutableTrie.curTrie.committedHash)
	m.curTrie = &source{prevTrie: immutableTrie.curTrie, requestPool: make(map[string]trie.Object), committedHash: committedHash}
	rootHash := make(hash, len(committedHash))
	copy(rootHash, committedHash)
	m.root = hash(rootHash)
	m.db = immutableTrie.db
	return nil
}
