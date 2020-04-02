package mpt

import (
	"bytes"
	"reflect"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie"
)

// struct for object trie
type mptForObj struct {
	*mpt
}

func (m *mptForObj) Prove(k []byte, p [][]byte) (trie.Object, error) {
	k = bytesToNibbles(k)
	return m.prove(k, p[1:], p[0])
}

func newMptForObj(db db.Database, bk db.Bucket, initialHash hash, t reflect.Type) *mptForObj {
	return &mptForObj{
		mpt: &mpt{root: hash(append([]byte(nil), []byte(initialHash)...)),
			source: &source{requestPool: make(map[string]trie.Object), committedHash: hash(append([]byte(nil), []byte(initialHash)...))},
			bk:     bk, db: db, objType: t},
	}
}

func newMptForObjFromImmutableForObject(immutable trie.ImmutableForObject) *mptForObj {
	if m, ok := immutable.(*mptForObj); ok {
		mpt := newMptForObj(m.db, m.bk, m.source.committedHash, m.objType)
		//mpt.source = m.source
		committedHash := m.source.committedHash
		// Below means s1.Flush() was called after calling m.Reset(s1)
		if m.source.prev != nil && bytes.Compare(m.source.committedHash, m.source.prev.committedHash) != 0 {
			committedHash = hash(append([]byte(nil), []byte(m.source.prev.committedHash)...))
		}
		mpt.source = &source{committedHash: committedHash, prev: m.source, requestPool: make(map[string]trie.Object)}
		return mpt
	}

	return nil
}

func (m *mptForObj) Get(k []byte) (trie.Object, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, err := m.get(k)
	if err != nil || v == nil {
		return nil, err
	}
	return v, nil
}

// TODO: check v is immutable???
func (m *mptForObj) Set(k []byte, v trie.Object) error {
	if k == nil || v == nil {
		return common.ErrIllegalArgument
	}
	k = bytesToNibbles(k)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	// have to check v is guaranteed as immutable
	m.source.requestPool[string(k)] = v
	return nil
}

func (m *mptForObj) Delete(k []byte) error {
	_, err := m.mpt.Delete(k)
	return err
}

func (m *mptForObj) GetSnapshot() trie.SnapshotForObject {
	mptSnapshot := m.mpt.GetSnapshot()
	mpt, ok := mptSnapshot.(*mpt)
	if ok == false {
		log.Fatalln("illegal variable")
	}
	return &mptForObj{mpt: mpt}
}

func (m *mptForObj) Reset(s trie.ImmutableForObject) {
	immutableTrie, ok := s.(*mptForObj)
	if ok == false {
		return
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
	return
}

func (m *mptForObj) Iterator() trie.IteratorForObject {
	iter := newIteratorObj(m)
	iter.Next()
	return iter
}

func (m *mptForObj) Equal(object trie.ImmutableForObject, exact bool) bool {
	immutableTrie, ok := object.(*mptForObj)
	if ok == false {
		return false
	}
	m.mpt.Equal(immutableTrie.mpt, exact)
	return false
}

func (m *mptForObj) Empty() bool {
	return m.mpt.Empty()
}

type iteratorObjImpl struct {
	*iteratorImpl
}

func newIteratorObj(m *mptForObj) *iteratorObjImpl {
	iter := &iteratorObjImpl{&iteratorImpl{key: nil, value: nil, top: -1, m: m.mpt}}
	m.Hash()
	m.initIterator(iter.iteratorImpl)
	return iter
}

func (iter *iteratorObjImpl) Get() (trie.Object, []byte, error) {
	return iter.get()
}
