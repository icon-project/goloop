package mpt

import (
	"bytes"
	"errors"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"log"
	"reflect"
	"sync"
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
		db        db.Database
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

func (m *mpt) get(k []byte) (trie.Object, error) {
	k = bytesToNibbles(k)
	m.mutex.Lock()
	defer m.mutex.Unlock()

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
			// TODO : check if m.root is nil
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
	m.evaluateTrie(pool)
	h := m.root.hash()
	m.evaluated = true
	// Do not set nil to requestPool and prevSnapshot because next snapshot want data of previous snapshot
	//m.requestPool = nil
	//m.prevSnapshot = nil
	return h
}

// return true if current node or child node is changed
func (m *mpt) set(n node, k []byte, v trie.Object) (node, nodeState) {
	//fmt.Println("set n ", n,", k ", k, ", v : ", v)

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
	copied := make([]byte, len(v))
	copy(copied, v)
	m.source.requestPool[string(k)] = byteValue(append([]byte(nil), v...))
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
	m.mutex.Lock()
	defer m.mutex.Unlock()
	mpt.source = m.source
	// Below means s1.Flush() was called after calling m.Reset(s1)
	if m.source.prev != nil && bytes.Compare(m.source.committedHash, m.source.prev.committedHash) != 0 {
		m.source.committedHash = hash(append([]byte(nil), []byte(m.source.prev.committedHash)...))
	}
	m.source = &source{committedHash: mpt.source.committedHash, prev: mpt.source, requestPool: make(map[string]trie.Object)}

	return mpt
}

func (m *mpt) getFromSnapshot(key []byte) (trie.Object, bool, hash) {
	var committedHash hash
	for snapshot := m.source; snapshot != nil; snapshot = snapshot.prev {
		if v, ok := snapshot.requestPool[string(key)]; ok {
			return v, true, snapshot.committedHash
		}
		committedHash = snapshot.committedHash
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

// TODO: Optimize
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
			m.evaluateTrie(pool)
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
		serializedValue, err := m.bk.Get(n)
		if err != nil || serializedValue == nil {
			return n, nil, false
		}
		deserializedNode := deserialize(serializedValue, m.objType, m.db)
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

// ethereum uses k, v DB as parameter to Prove()
// Key / Value = hash(encoding node) / encoding
// then verify key with roothash and DB passed to Prove()
// TODO: Implement Verify function and verify Proof
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
		m.evaluateTrie(pool)
	}
	k = bytesToNibbles(k)
	var proofBuf [][]byte
	m.root, proofBuf, _ = m.proof(m.root, k, 0)
	return proofBuf
}

func (m *mpt) Iterator() trie.Iterator {
	iter := newIterator(m)
	iter.Next()
	return iter
}

func (m *mpt) Reset(immutable trie.Immutable) error {
	immutableTrie, ok := immutable.(*mpt)
	if ok == false {
		return common.ErrIllegalArgument
	}

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

func newMpFromImmutable(immutable trie.Immutable) *mpt {
	if m, ok := immutable.(*mpt); ok {
		mpt := newMpt(m.db, m.bk, m.source.committedHash, m.objType)
		mpt.source = m.source
		// Below means s1.Flush() was called after calling m.Reset(s1)
		if m.source.prev != nil && bytes.Compare(m.source.committedHash, m.source.prev.committedHash) != 0 {
			m.source.committedHash = hash(append([]byte(nil), []byte(m.source.prev.committedHash)...))
		}
		mpt.source = &source{committedHash: m.source.committedHash, prev: m.source, requestPool: make(map[string]trie.Object)}
		return m
	}

	return nil
}

func (m *mpt) initIterator(iter *iteratorImpl) {
	var data []byte
	if n, ok := m.root.(hash); ok {
		var err error
		data, err = m.bk.Get(n)
		if err != nil {
			log.Fatalln("Failed to get value. key : %x", n)
			return
		} else if len(data) == 0 {
			return
		}
		iter.stack[0].n = deserialize(data, m.objType, m.db)
	} else {
		iter.stack[0].n = m.root
	}
}

func newIterator(m *mpt) *iteratorImpl {
	iter := &iteratorImpl{key: nil, value: nil, top: -1, m: m}
	m.Hash()
	m.initIterator(iter)

	return iter
}

func (iter *iteratorImpl) nextChildNode(m *mpt, n node, key []byte) ([]byte, trie.Object) {
	switch nn := n.(type) {
	case *branch:
		iter.top++
		iter.stack[iter.top].n = n
		iter.stack[iter.top].key = key
		if nn.value != nil {
			return key, nn.value
		}
		for i, nibbleNode := range nn.nibbles {
			if nibbleNode != nil {
				newKey := make([]byte, len(key)+1)
				if len(key) > 0 {
					copy(newKey, key)
				}
				newKey[len(key)] = byte(i)
				return iter.nextChildNode(m, nibbleNode, newKey)
			}
		}
	case *extension:
		newKey := make([]byte, len(key)+len(nn.sharedNibbles))
		if len(key) > 0 {
			copy(newKey, key)
		}
		copy(newKey[len(key):], nn.sharedNibbles)
		return iter.nextChildNode(m, nn.next, newKey)
	case *leaf:
		newKey := make([]byte, len(key)+len(nn.keyEnd))
		if len(key) > 0 {
			copy(newKey, key)
		}
		if len(nn.keyEnd) > 0 {
			copy(newKey[len(key):], nn.keyEnd)
		}
		iter.top++
		iter.stack[iter.top].key = newKey
		iter.stack[iter.top].n = n
		return newKey, nn.value
	case hash:
		serializedValue, err := m.bk.Get(nn)
		if err != nil {
			return nil, nil
		}
		if serializedValue == nil {
			return nil, nil
		}
		return iter.nextChildNode(m, deserialize(serializedValue, m.objType, m.db), key)
	}
	panic("Not considered!!!")
}

func (iter *iteratorImpl) Next() error {
	if iter.end == true {
		return errors.New("NoMoreItem")
	}
	if iter.top == -1 && len(iter.key) == 0 {
		iter.key, iter.value = iter.nextChildNode(iter.m, iter.stack[0].n, nil)
	} else {
		n := iter.stack[iter.top]
		switch nn := n.n.(type) {
		case *branch:
			for _, nibbleNode := range nn.nibbles {
				if nibbleNode != nil {
					iter.key, iter.value = iter.nextChildNode(iter.m, nibbleNode, iter.key)
				}
			}
		case *leaf:
			findNext := false
			prevKey := n.key
			for iter.top != 0 && findNext == false {
				iter.top--
				stackNode := iter.stack[iter.top]
				startNibble := byte(0)
				keyIndex := len(stackNode.key)
				startNibble = prevKey[keyIndex] + 1
				branchNode := stackNode.n.(*branch)
				branchNode.nibbles[prevKey[keyIndex]] = nil
				prevKey = stackNode.key
				for i := startNibble; i < 16; i++ {
					if branchNode.nibbles[i] != nil {
						findNext = true
						newKey := make([]byte, len(stackNode.key)+1)
						copy(newKey, prevKey)
						newKey[len(stackNode.key)] = i
						iter.key, iter.value = iter.nextChildNode(iter.m, branchNode.nibbles[i], newKey)
						break
					}
				}
			}
			if findNext == false {
				iter.key = nil
				iter.value = nil
				iter.end = true
			}
		}
	}
	return nil
}

func (iter *iteratorImpl) Has() bool {
	if iter.end {
		return false
	}
	return iter.value != nil
}

func (iter *iteratorImpl) get() (value trie.Object, key []byte, err error) {
	k := iter.key
	remainder := len(k) % 2
	returnKey := make([]byte, len(k)/2+remainder)
	if remainder > 0 {
		returnKey[0] = k[0]
	}
	for i := remainder; i < len(k)/2+remainder; i++ {
		returnKey[i] = k[i*2-remainder]<<4 | k[i*2+1-remainder]
	}
	return iter.value, returnKey, nil
}

func (iter *iteratorImpl) Get() (value []byte, key []byte, err error) {
	v, k, err := iter.get()
	if err != nil && v == nil {
		return nil, nil, err
	}
	return v.Bytes(), k, nil
}
