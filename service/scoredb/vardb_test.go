package scoredb

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"log"
	"testing"
)

type TestStore struct {
	mutable trie.Mutable
}

func (s *TestStore) GetValue(k []byte) ([]byte, error) {
	v, err := s.mutable.Get(k)
	log.Printf("TestStore.GetValue(<%x>) -> <% x>, err=%+v", k, v, err)
	return v, err
}

func (s *TestStore) SetValue(k, v []byte) error {
	log.Printf("TestStore.SetValue(<%x>,<% x>)", k, v)
	return s.mutable.Set(k, v)
}

func (s *TestStore) DeleteValue(k []byte) error {
	log.Printf("TestStore.DeleteValue(<%x>)", k)
	return s.mutable.Delete(k)
}

func TestNewVarDB(t *testing.T) {
	mdb := db.NewMapDB()
	tree := trie_manager.NewMutable(mdb, nil)
	db := NewVarDB(&TestStore{tree}, 1)
	db.Set(int(1))

	if v := int(db.Int64()); v != 1 {
		log.Printf("Returned Bytes <% x>", db.Bytes())
		t.Errorf("Fail to retrieved value (%v) is different", v)
		return
	}
	db.Delete()

	if v := db.Int64(); v != 0 {
		t.Errorf("Delete value should be zero v=%d", v)
	}
}
