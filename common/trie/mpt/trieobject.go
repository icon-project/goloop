package mpt

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"log"
	"reflect"
)

// struct for object trie
type mptForObj struct {
	*mpt
}

func newMptForObj(db db.Database, bk db.Bucket, initialHash hash, t reflect.Type) *mptForObj {
	return &mptForObj{
		mpt: &mpt{root: hash(append([]byte(nil), []byte(initialHash)...)),
			source: &source{requestPool: make(map[string]trie.Object), committedHash: hash(append([]byte(nil), []byte(initialHash)...))},
			bk:     bk, db: db, objType: t},
	}
}

func (m *mptForObj) Get(k []byte) (trie.Object, error) {
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
