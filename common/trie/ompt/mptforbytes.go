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
	if err != nil || obj == nil {
		return nil, err
	}
	return obj.Bytes(), nil
}

func (m *mptForBytes) Set(k, v []byte) error {
	obj := bytesObject(v)
	return m.mpt.Set(k, obj)
}

func (m *mptForBytes) Proof(k []byte) [][]byte {
	return m.mpt.Proof(k)
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

func (m *mptForBytes) Prove(k []byte, proof [][]byte) ([]byte, error) {
	obj, err := m.mpt.Prove(k, proof)
	if err != nil {
		return nil, err
	}
	return obj.Bytes(), nil
}

func NewMPTForBytes(db db.Database, h []byte) *mptForBytes {
	return &mptForBytes{
		NewMPT(db, h, reflect.TypeOf(bytesObject(nil))),
	}
}
