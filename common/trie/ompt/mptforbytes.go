package ompt

import (
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

type mptForBytes struct {
	*mpt
}

func (m *mptForBytes) Get(k []byte) ([]byte, error) {
	obj, err := m.mpt.Get(k)
	if err != nil {
		return nil, err
	}
	return obj.Bytes(), nil
}

func (m *mptForBytes) Set(k, v []byte) error {
	obj := BytesObject(v)
	return m.mpt.Set(k, obj)
}

func (m *mptForBytes) Proof(k []byte) [][]byte {
	// TODO
	return nil
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

func (m *mptForBytes) Dump() {
	m.mpt.Dump()
}

func NewMPTForBytes(db db.Database, h []byte) *mptForBytes {
	return &mptForBytes{
		NewMPT(db, h, reflect.TypeOf(BytesObject(nil))),
	}
}
