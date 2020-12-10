package ompt

import (
	"reflect"

	"github.com/icon-project/goloop/common/merkle"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

type mptForBytes struct {
	*mpt
}

func (m *mptForBytes) Get(k []byte) ([]byte, error) {
	obj, err := m.mpt.Get(k)
	if err != nil || obj == nil {
		return nil, err
	}
	return obj.Bytes(), nil
}

func (m *mptForBytes) Set(k, v []byte) ([]byte, error) {
	obj := bytesObject(v)
	old, err := m.mpt.Set(k, obj)
	if old == nil {
		return nil, err
	}
	ob := old.(bytesObject)
	return ob.Bytes(), err
}

func (m *mptForBytes) Delete(k []byte) ([]byte, error) {
	old, err := m.mpt.Delete(k)
	if old == nil {
		return nil, err
	}
	ob := old.(bytesObject)
	return ob.Bytes(), err
}

func (m *mptForBytes) RootHash() []byte {
	return m.mpt.Hash()
}

func (m *mptForBytes) GetSnapshot() trie.Snapshot {
	s := m.mpt.GetSnapshot()
	s2, _ := s.(*mpt)
	return &mptForBytes{mpt: s2}
}

func (m *mptForBytes) Reset(s trie.Immutable) error {
	s2, _ := s.(*mptForBytes)
	m.mpt.Reset(s2.mpt)
	return nil
}

func (m *mptForBytes) Prove(k []byte, proof [][]byte) ([]byte, error) {
	obj, err := m.mpt.Prove(k, proof)
	if err != nil {
		return nil, err
	}
	return obj.Bytes(), nil
}

type iteratorForBytes struct {
	trie.IteratorForObject
}

func (i *iteratorForBytes) Get() ([]byte, []byte, error) {
	o, k, err := i.IteratorForObject.Get()
	if o != nil {
		return o.Bytes(), k, err
	}
	return nil, nil, err
}

func (m *mptForBytes) Iterator() trie.Iterator {
	return m.Filter(nil)
}

func (m *mptForBytes) Filter(prefix []byte) trie.Iterator {
	i := m.mpt.Filter(prefix)
	if i == nil {
		return nil
	}
	return &iteratorForBytes{i}
}

func (m *mptForBytes) Equal(object trie.Immutable, exact bool) bool {
	if m2, ok := object.(*mptForBytes); ok {
		return m.mpt.Equal(m2.mpt, exact)
	} else {
		panic("Equal with invalid object")
	}
	return false
}

func (m *mptForBytes) Resolve(bd merkle.Builder) {
	m.mpt.Resolve(bd)
}

func NewMPTForBytes(db db.Database, h []byte) *mptForBytes {
	return &mptForBytes{
		NewMPT(db, h, reflect.TypeOf(bytesObject(nil))),
	}
}

func MPTFromImmutableForBytes(immutable trie.Immutable) *mptForBytes {
	if m, ok := immutable.(*mptForBytes); ok {
		if m2 := MPTFromImmutable(m.mpt); m2 != nil {
			return &mptForBytes{m2}
		}
	}
	return nil
}
