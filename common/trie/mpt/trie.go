package mpt

import (
	"bytes"
	"errors"
	"log"
	"reflect"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

const maxNodeHeight = 65 // nibbles of 32bytes key(64) + root (1)

type (
	source struct {
		// prev is nil after Flush()
		prev          *source
		committedHash hash
		requestPool   map[string]trie.Object
	}

	mpt struct {
		root    node
		objType reflect.Type

		evaluated bool
		source    *source
		mutex     sync.Mutex
		bk        db.Bucket

		db db.Database
	}

	iteratorStack struct {
		n   node
		key []byte
	}

	iteratorImpl struct {
		key   []byte
		value trie.Object
		stack [maxNodeHeight]iteratorStack
		top   int
		end   bool

		m *mpt
	}
)

/*
 */
func newMpt(db db.Database, bk db.Bucket, initialHash hash, t reflect.Type) *mpt {
	return &mpt{root: nil,
		source: &source{requestPool: make(map[string]trie.Object),
			committedHash: hash(append([]byte(nil), []byte(initialHash)...))},
		bk: bk, objType: t, db: db}
}

func bytesToNibbles(k []byte) []byte {
	nibbles := make([]byte, len(k)*2)
	for i, v := range k {
		nibbles[i*2] = v >> 4 & 0x0F
		nibbles[i*2+1] = v & 0x0F
	}
	return nibbles
}

// have to shared mutex between snapshot and mutable
func (m *mpt) get(k []byte) (trie.Object, error) {
	k = bytesToNibbles(k)

	if m.evaluated == true {
		var value trie.Object
		var err error
		_, value, err = m.root.get(m, k)
		if err != nil || value == nil {
			return nil, err
		}
		return value, nil
	}
	if v, ok, lastCommittedHash := m.getFromSnapshot(k); ok {
		if v == nil {
			return nil, nil
		}

		m.source.committedHash = lastCommittedHash
		return v, nil
	} else if bytes.Compare(m.source.committedHash, lastCommittedHash) != 0 {
		m.source.committedHash = lastCommittedHash
		m.root = lastCommittedHash
	}

	var value trie.Object
	var err error
	if v, ok := m.root.(hash); ok {
		if len(v) == 0 {
			m.root = nil
		} else if bytes.Compare(v, m.source.committedHash) != 0 {
			m.root = m.source.committedHash
		}
	}

	if m.root == nil {
		if m.source.committedHash == nil {
			return nil, nil
		}
		m.root = m.source.committedHash
	}
	m.root, value, err = m.root.get(m, k)
	if err != nil {
		return nil, err
	} else if value == nil {
		return nil, nil
	}
	return value, nil
}

func (m *mpt) Get(k []byte) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, err := m.get(k)
	if err != nil || v == nil {
		return nil, err
	}
	return v.Bytes(), nil
}

func (m *mpt) evaluateTrie(requestPool map[string]trie.Object) error {
	var err error
	for k, v := range requestPool {
		if v == nil {
			if m.root, _, err = m.root.deleteChild(m, []byte(k)); err != nil {
				return err
			}
		} else {
			m.root, _ = m.set(m.root, []byte(k), v)
		}
	}
	return nil
}

/*
	Hash
*/
func (m *mpt) Hash() []byte {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.evaluated == true {
		return m.root.hash()
	}
	pool, lastCommittedHash := m.mergeSnapshot()
	if bytes.Compare(m.source.committedHash, lastCommittedHash) != 0 {
		m.source.committedHash = lastCommittedHash
		m.root = lastCommittedHash
	}

	m.source.prev = nil
	m.source.requestPool = pool

	if m.root == nil {
		m.root = m.source.committedHash
	}

	// That length of pool is zero means that this trie is already calculated
	if len(pool) == 0 {
		return m.root.hash()
	}
	if err := m.evaluateTrie(pool); err != nil {
		log.Printf("err : %s\n", err)
		return nil
	}
	h := m.root.hash()
	m.evaluated = true
	// Do not set nil to requestPool and prevSnapshot because next snapshot want data of previous snapshot
	//m.requestPool = nil
	//m.prevSnapshot = nil
	return h
}

// return true if current node or child node is changed
func (m *mpt) set(n node, k []byte, v trie.Object) (node, nodeState) {
	if n == nil {
		return &leaf{keyEnd: k[:], value: v, nodeBase: nodeBase{state: dirtyNode}}, dirtyNode
	}

	return n.addChild(m, k, v)
}

/*
Set inserts key and value into requestPool.
Hash, GetProof, Flush insert keys and values in requestPool into trie
*/
func (m *mpt) Set(k, v []byte) error {
	if k == nil || v == nil {
		return common.ErrIllegalArgument
	}
	k = bytesToNibbles(k)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.source.requestPool[string(k)] = byteValue(v)
	if m.source.requestPool[string(k)] == nil {
		panic("P")
	}

	return nil
}

func (m *mpt) Equal(immutable trie.Immutable, exact bool) bool {
	immutableTrie, ok := immutable.(*mpt)
	if ok == false {
		return false
	}
	passedMergedPool, passedCommittedHash := immutableTrie.mergeSnapshot()
	selfMergedPool, selfCommittedHash := m.mergeSnapshot()

	result := true
	if exact == false {
		if bytes.Compare(passedCommittedHash, selfCommittedHash) == 0 {
			if len(passedMergedPool) == len(selfMergedPool) {
				for k, v := range passedMergedPool {
					if selfMergedPool[k].Equal(v) == false {
						result = false
						break
					}
				}
			} else {
				result = false
			}
		}
	} else {
		if bytes.Compare(m.Hash(), immutable.Hash()) == 0 {
			result = true
		}
	}
	return result
}

func (m *mpt) Delete(k []byte) error {
	var err error
	k = bytesToNibbles(k)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.source.requestPool[string(k)] = nil
	return err
}

func (m *mpt) GetSnapshot() trie.Snapshot {
	mpt := newMpt(m.db, m.bk, m.source.committedHash, m.objType)
	mpt.mutex = m.mutex

	m.mutex.Lock()
	defer m.mutex.Unlock()
	mpt.source = m.source
	// Below means s1.Flush() was called after calling m.Reset(s1)

	committedHash := hash(append([]byte(nil), []byte(m.source.committedHash)...))
	if m.source.prev != nil && bytes.Compare(m.source.committedHash, m.source.prev.committedHash) != 0 {
		committedHash = hash(append([]byte(nil), []byte(m.source.prev.committedHash)...))
	}
	m.source = &source{committedHash: committedHash, prev: mpt.source, requestPool: make(map[string]trie.Object)}

	return mpt
}

func (m *mpt) getFromSnapshot(key []byte) (trie.Object, bool, hash) {
	var committedHash hash
	num := 0
	for snapshot := m.source; snapshot != nil; snapshot = snapshot.prev {
		num += len(snapshot.requestPool)
	}
	for snapshot := m.source; snapshot != nil; snapshot = snapshot.prev {
		committedHash = snapshot.committedHash
		if v, ok := snapshot.requestPool[string(key)]; ok {
			return v, true, committedHash
		}
	}

	return nil, false, committedHash
}

func (m *mpt) mergeSnapshot() (map[string]trie.Object, hash) {
	if m.source.prev == nil {
		return m.source.requestPool, m.source.committedHash
	}
	mergePool := make(map[string]trie.Object)
	var committedHash hash
	for snapshot := m.source; snapshot != nil; snapshot = snapshot.prev {
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

func traversalCommit(db db.Bucket, n node, cnt int) error {
	switch n := n.(type) {
	case *branch:
		if n.state == committedNode {
			return nil
		}
		for _, v := range n.nibbles {
			if v != nil {
				if err := traversalCommit(db, v, cnt+1); err != nil {
					return err
				}
			}
		}
		n.flush()

	case *extension:
		if n.state == committedNode {
			return nil
		}

		if err := traversalCommit(db, n.next, cnt+1); err != nil {
			return err
		}

	case *leaf:
		if n.state == committedNode {
			return nil
		}

		n.flush()

		if len(n.serialize()) < hashableSize && cnt != 0 { // root hash has to save hash
			return nil
		}
		err := db.Set(n.hash(), n.serialize())
		n.state = committedNode
		return err
	default:
		return nil
	}
	if len(n.serialize()) < hashableSize && cnt != 0 { // root hash has to save hash
		return nil
	}

	return db.Set(n.hash(), n.serialize())
}

func (m *mpt) Flush() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	pool, lastCommittedHash := m.mergeSnapshot()
	if bytes.Compare(m.source.committedHash, lastCommittedHash) != 0 {
		m.source.committedHash = lastCommittedHash
	}

	if len(pool) != 0 {
		if m.evaluated == false {
			if m.root == nil {
				m.root = m.source.committedHash
			}
			if err := m.evaluateTrie(pool); err != nil {
				return err
			}
			m.evaluated = true
		}
		if err := traversalCommit(m.bk, m.root, 0); err != nil {
			return err
		}
		m.source.committedHash = m.root.hash()
	} else {
		m.root = m.source.committedHash
	}

	m.source.requestPool = nil
	m.source.prev = nil
	return nil
}

func (m *mpt) GetProof(k []byte) [][]byte {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	pool, lastCommittedHash := m.mergeSnapshot()
	if bytes.Compare(m.source.committedHash, lastCommittedHash) != 0 {
		m.source.committedHash = lastCommittedHash
		m.root = lastCommittedHash
	}

	if m.root == nil {
		m.root = m.source.committedHash
	}

	// That length of pool is zero means that this trie is already calculated
	if len(pool) != 0 {
		if err := m.evaluateTrie(pool); err != nil {
			log.Printf("err : %s\n", err)
			return nil
		}
	}
	k = bytesToNibbles(k)
	var proofBuf [][]byte
	result := false
	//m.root, proofBuf, result = m.proof(m.root, k, 1)
	proofBuf, result = m.root.proof(m, k, 1)
	if result == false {
		return nil
	}
	proofBuf[0] = m.root.hash()
	return proofBuf
}

func (m *mpt) prove(k []byte, pb [][]byte, p []byte) (trie.Object, error) {
	buf := pb[0]
	keyLen := 1
	n := deserialize(buf, m.objType, m.db)
	if len(p) == hashableSize {
		if bytes.Compare(p, n.hash()) != 0 {
			return nil, errors.New("failed to prove")
		}
	} else {
		if bytes.Compare(p, buf) != 0 {
			return nil, errors.New("failed to prove")
		}
	}
	var next []byte
	switch nn := n.(type) {
	case *branch:
		if len(k) == 0 {
			return nn.value, nil
		}
		if h, ok := nn.nibbles[k[0]].(hash); ok {
			next = []byte(h)
		} else {
			next = nn.nibbles[k[0]].serialize()
		}
	case *extension:
		sharedLen := len(nn.sharedNibbles)
		if bytes.Compare(nn.sharedNibbles, k[:sharedLen]) != 0 {
			return nil, errors.New("failed to prove")
		}
		keyLen = sharedLen
		if h, ok := nn.next.(hash); ok {
			next = []byte(h)
		} else {
			next = nn.next.serialize()
		}
	case *leaf:
		if bytes.Compare(nn.keyEnd, k) != 0 {
			return nil, errors.New("failed to prove")
		}
		return nn.value, nil
	case hash:
		next = nn
	}
	return m.prove(k[keyLen:], pb[1:], next)
}

func (m *mpt) Prove(k []byte, p [][]byte) ([]byte, error) {
	// p[0] should be hash
	k = bytesToNibbles(k)
	v, err := m.prove(k, p[1:], p[0])
	if err != nil {
		return nil, err
	}
	return v.Bytes(), nil
}

func (m *mpt) Iterator() trie.Iterator {
	iter := newIterator(m)
	if err := iter.Next(); err != nil {
		return nil
	}
	return iter
}

func (m *mpt) Reset(immutable trie.Immutable) error {
	immutableTrie, ok := immutable.(*mpt)
	if ok == false {
		return common.ErrIllegalArgument
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	// Do not use reference.
	committedHash := make(hash, len(immutableTrie.source.committedHash))
	copy(committedHash, immutableTrie.source.committedHash)
	m.source = &source{prev: immutableTrie.source, requestPool: make(map[string]trie.Object), committedHash: committedHash}
	rootHash := make(hash, len(committedHash))
	copy(rootHash, committedHash)
	m.root = hash(rootHash)
	m.db = immutableTrie.db
	return nil
}

func (m *mpt) Empty() bool {
	var pool map[string]trie.Object
	var commitedHash hash
	if pool, commitedHash = m.mergeSnapshot(); commitedHash == nil {
		nilCnt := 0
		for _, v := range pool {
			if v == nil {
				nilCnt++
			}
		}
		return nilCnt == len(pool)
	}
	return len(pool) == 0 && m.root == nil
}

func newMptFromImmutable(immutable trie.Immutable) *mpt {
	if m, ok := immutable.(*mpt); ok {
		mpt := newMpt(m.db, m.bk, m.source.committedHash, m.objType)
		mpt.source = m.source
		// Below means s1.Flush() was called after calling m.Reset(s1)
		if m.source.prev != nil && bytes.Compare(m.source.committedHash, m.source.prev.committedHash) != 0 {
			m.source.committedHash = hash(append([]byte(nil), []byte(m.source.prev.committedHash)...))
		}
		mpt.source = &source{committedHash: m.source.committedHash, prev: m.source, requestPool: make(map[string]trie.Object)}
		return mpt
	}

	return nil
}
