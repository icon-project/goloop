package mpt

import (
	"bytes"
	"fmt"
	"reflect"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

type (
	source struct {
		// prevSource is nil after Flush()
		prevSource    *source
		committedHash hash
		requestPool   map[string]trie.Object
	}

	mpt struct {
		root    node
		objType reflect.Type

		rootHashed bool
		curSource  *source
		mutex      sync.Mutex
		db         db.Bucket
	}
)

/*
 */
func newMpt(db db.Bucket, initialHash hash, t reflect.Type) *mpt {
	return &mpt{root: hash(append([]byte(nil), []byte(initialHash)...)),
		curSource: &source{requestPool: make(map[string]trie.Object), committedHash: hash(append([]byte(nil), []byte(initialHash)...))},
		db:        db, objType: t}
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
		match, _ := compareHex(n.sharedNibbles, k)
		if len(n.sharedNibbles) != match {
			return n, nil, err
		}
		n.next, result, err = m.get(n.next, k[match:])
		if err != nil {
			return n, nil, err
		}
	case *leaf:
		if bytes.Compare(k, n.keyEnd) != 0 {
			return n, nil, nil
		}
		return n, n.value, nil
	// if node is hash, get serialized value with hash from db then deserialize it.
	case hash:
		if len(n) == 0 {
			return n, nil, err
		}
		serializedValue, err := m.db.Get(n)
		if err != nil {
			return n, nil, err
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

// TODO: check committed hash in previous source
// TODO: If not same between previous and current committed hash, update current committed hash
func (m *mpt) Get(k []byte) ([]byte, error) {
	k = bytesToNibbles(k)
	pool, lastCommitedHash := m.mergeSnapshot()
	if len(lastCommitedHash) != 0 {
		m.root = lastCommitedHash
	}
	if v, ok := pool[string(k)]; ok {
		if v == nil {
			return nil, nil
		}
		return v.(byteValue), nil
	}

	var value trie.Object
	var err error
	m.root, value, err = m.get(m.root, k)
	if err != nil || value == nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
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
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.rootHashed == true {
		return m.root.hash()
	}
	pool, lastCommitedHash := m.mergeSnapshot()
	if len(lastCommitedHash) != 0 {
		m.root = lastCommitedHash
	}
	// That length of pool is zero means that this trie is already calculated
	if len(pool) == 0 {
		return m.root.hash()
	}
	m.evaluateTrie(pool)
	h := m.root.hash()
	m.rootHashed = true
	fmt.Printf("RootHash : %x\n", h)
	// Do not set nil to requestPool and prevSnapshot because next snapshot want data in previous snapshot
	//m.requestPool = nil
	//m.prevSnapshot = nil
	return h
}

// return true if current node or child node is changed
func (m *mpt) set(n node, k []byte, v trie.Object) (node, bool) {
	//fmt.Println("set n ", n,", k ", k, ", v : ", v)

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
	defer m.mutex.Unlock()
	copied := make([]byte, len(v))
	copy(copied, v)
	m.curSource.requestPool[string(k)] = byteValue(append([]byte(nil), v...))
	return nil
}

func (m *mpt) delete(n node, k []byte) (node, bool, error) {
	//fmt.Println("delete n ", n,", k ", k, ", v : ", string(k))
	if n == nil {
		return n, false, nil
	}

	return n.deleteChild(m, k)
}

func (m *mpt) Delete(k []byte) error {
	var err error
	k = bytesToNibbles(k)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.curSource.requestPool[string(k)] = nil
	return err
}

func (m *mpt) GetSnapshot() trie.Snapshot {
	mpt := newMpt(m.db, m.curSource.committedHash, m.objType)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	mpt.curSource = m.curSource
	// Below means s1.Flush() was called after calling m.Reset(s1)
	if m.curSource.prevSource != nil && bytes.Compare(m.curSource.committedHash, m.curSource.prevSource.committedHash) != 0 {
		m.curSource.committedHash = hash(append([]byte(nil), []byte(m.curSource.prevSource.committedHash)...))
	}
	m.curSource = &source{committedHash: mpt.curSource.committedHash, prevSource: mpt.curSource, requestPool: make(map[string]trie.Object)}

	return mpt
}

func (m *mpt) mergeSnapshot() (map[string]trie.Object, hash) {
	mergePool := make(map[string]trie.Object)
	var committedHash hash
	for snapshot := m.curSource; snapshot != nil; snapshot = snapshot.prevSource {
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
func traversalCommit(db db.Bucket, n node, cnt int) error {
	switch n := n.(type) {
	case *branch:
		for _, v := range n.nibbles {
			if err := traversalCommit(db, v, cnt+1); err != nil {
				return err
			}
		}
	case *extension:
		if err := traversalCommit(db, n.next, cnt+1); err != nil {
			return err
		}

	case *leaf:
		//serialized := n.serialize()
		//// if length of serialized leaf is smaller hashable(32), parent node (branch) must have serialized data of this
		//if len(serialized) < hashableSize {
		//	return nil
		//}
	default:
		return nil
	}
	if len(n.serialize()) < hashableSize && cnt != 0 { // root hash has to save hash
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
	m.mutex.Lock()
	defer m.mutex.Unlock()
	pool, lastCommitedHash := m.mergeSnapshot()
	m.curSource.committedHash = lastCommitedHash

	m.curSource.requestPool = pool
	if len(pool) != 0 {
		if m.rootHashed == false {
			if len(lastCommitedHash) != 0 {
				m.root = lastCommitedHash
			}
			m.evaluateTrie(pool)
			m.rootHashed = true
		}
		//fmt.Printf("Flush : m.root.hash() : %x\n", m.root.hash())
		if err := traversalCommit(m.db, m.root, 0); err != nil {
			return err
		}
		m.curSource.committedHash = m.root.hash()
	} else {
		m.root = m.curSource.committedHash
	}

	m.curSource.requestPool = nil
	m.curSource.prevSource = nil
	return nil
}

func addProof(buf [][]byte, index int, hash []byte) {
	if len(buf) == index {
		buf = make([][]byte, len(buf)+10)
	}
	copy(buf[index], hash)
}

// bool : if find k, true else false
// [][]byte : stored seiazlied child. If child is smaller than hashableSize, this is nil
// depth starts 0
func (m *mpt) proof(n node, k []byte, depth int) (node, [][]byte, bool) {
	var proofBuf [][]byte
	var result bool
	switch n := n.(type) {
	case *branch:
		if len(k) != 0 {
			n.nibbles[k[0]], proofBuf, result = m.proof(n.nibbles[k[0]], k[1:], depth+1)
			if result == false {
				return n, nil, false
			}
			buf := n.serialize()
			if len(buf) < hashableSize && depth != 0 {
				return n, nil, true
			}

			if proofBuf == nil {
				proofBuf = make([][]byte, depth+1)
			}
			proofBuf[depth] = buf
		} else {
			// find k
			buf := n.serialize()
			if len(buf) < hashableSize && depth != 0 {
				return n, nil, true
			}
			proofBuf = make([][]byte, depth+1)
			proofBuf[depth] = buf
			return n, proofBuf, result
		}
	case *extension:
		match, same := compareHex(n.sharedNibbles, k[:len(n.sharedNibbles)])
		if same == false {
			return n, nil, false
		}
		n.next, proofBuf, result = m.proof(n.next, k[match:], depth+1)
		if result == false {
			return n, nil, false
		}
		buf := n.serialize()
		if len(buf) < hashableSize && depth != 0 {
			return n, nil, true
		}
		if proofBuf == nil {
			proofBuf = make([][]byte, depth+1)
		}
		proofBuf[depth] = buf
	case *leaf:
		if bytes.Compare(k, n.keyEnd) != 0 {
			return n, nil, false
		}
		buf := n.serialize()
		if len(buf) < hashableSize && depth != 0 {
			return n, nil, true
		}
		proofBuf = make([][]byte, depth+1)
		proofBuf[depth] = buf
	// if node is hash, get serialized value with hash from db then deserialize it.
	case hash:
		serializedValue, err := m.db.Get(n)
		if err != nil {
			return nil, nil, false
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
		return m.proof(deserializedNode, k, depth+1)
	}
	return n, proofBuf, result
}

//func (m *mpt) proof1(n node, k []byte) ([][]byte, int) {
//	var buf [][]byte
//	var i int
//	switch n := n.(type) {
//	case *branch:
//		buf, i = m.proof(n.nibbles[k[0]], k[1:])
//		if n.hashedValue == nil {
//			addProof(buf, i, n.serialize())
//		} else {
//			addProof(buf, i, n.hashedValue)
//		}
//	case *extension:
//		match, _ := compareHex(n.sharedNibbles, k)
//		buf, i = m.proof(n.next, k[match:])
//		if n.hashedValue == nil {
//			addProof(buf, i, n.serialize())
//		} else {
//			addProof(buf, i, n.hashedValue)
//		}
//	case *leaf:
//		return nil, 0
//	case hash:
//		// TODO: have to check error
//		if len(n) == 0 {
//			return nil, 0
//		}
//		serializedValued, _ := m.db.Get(k)
//		decodeingNode := deserialize(serializedValued, m.objType)
//		return m.proof(decodeingNode, k)
//	}
//	return buf, i + 1
//}

// ethereum uses k, v DB as parameter to Prove()
// Key / Value = hash(encoding node) / encoding
// then verify key with roothash and DB passed to Prove()
// TODO: Implement Verify function and verify Proof
func (m *mpt) Proof(k []byte) [][]byte {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	pool, lastCommitedHash := m.mergeSnapshot()
	if len(lastCommitedHash) != 0 {
		m.root = lastCommitedHash
	}
	// That length of pool is zero means that this trie is already calculated
	if len(pool) != 0 {
		fmt.Println("pool : ", pool)
		m.evaluateTrie(pool)
	}
	k = bytesToNibbles(k)
	var proofBuf [][]byte
	fmt.Println("Proof ", m.root)
	m.root, proofBuf, _ = m.proof(m.root, k, 0)
	return proofBuf
}

// Not used
//func (m *mpt) Load(db db.Bucket, root []byte) error {
//	// use db to check validation
//	if _, err := db.Get(root); err != nil {
//		return err
//	}
//
//	m.curSource.committedHash = root
//	m.root = hash(root)
//	m.db = db
//	return nil
//}

func (m *mpt) Reset(immutable trie.Immutable) error {
	immutableTrie, ok := immutable.(*mpt)
	if ok == false {
		return common.ErrIllegalArgument
	}

	// Do not use reference.
	committedHash := make(hash, len(immutableTrie.curSource.committedHash))
	copy(committedHash, immutableTrie.curSource.committedHash)
	m.curSource = &source{prevSource: immutableTrie.curSource, requestPool: make(map[string]trie.Object), committedHash: committedHash}
	rootHash := make(hash, len(committedHash))
	copy(rootHash, committedHash)
	m.root = hash(rootHash)
	m.db = immutableTrie.db
	return nil
}

type mptForObj struct {
	*mpt
}

func newMptForObj(db db.Bucket, initialHash hash, t reflect.Type) *mptForObj {
	return &mptForObj{
		mpt: &mpt{root: hash(append([]byte(nil), []byte(initialHash)...)),
			curSource: &source{requestPool: make(map[string]trie.Object), committedHash: hash(append([]byte(nil), []byte(initialHash)...))},
			db:        db, objType: t},
	}
}

// TODO: refactorying
func (m *mptForObj) Get(k []byte) (trie.Object, error) {
	k = bytesToNibbles(k)
	if v, ok := m.curSource.requestPool[string(k)]; ok {
		return v.(byteValue), nil
	}
	var value trie.Object
	var err error
	m.root, value, err = m.get(m.root, k)
	if err != nil || value == nil {
		return nil, err
	}
	return value, nil
}

// TODO: check v is immutable???
func (m *mptForObj) Set(k []byte, v trie.Object) error {
	if k == nil || v == nil {
		return common.ErrIllegalArgument
	}
	k = bytesToNibbles(k)
	m.mutex.Lock()
	//copied := make(reflect.TypeOf(v), len(v))
	//copy(copied, v)
	//m.curSource.requestPool[string(k)] = byteValue(append([]byte(nil), v...))
	// have to check v is guaranteed as immutable
	m.curSource.requestPool[string(k)] = v
	m.mutex.Unlock()
	return nil
}

func (m *mptForObj) GetSnapshot() trie.SnapshotForObject {
	mptSnapshot := m.mpt.GetSnapshot()
	mpt, ok := mptSnapshot.(*mpt)
	if ok == false {
		panic("illegal vairable")
	}
	return &mptForObj{mpt: mpt}
}

func (m *mptForObj) Reset(s trie.ImmutableForObject) {
}

func (m *mptForObj) Hash() []byte {
	return nil
}
