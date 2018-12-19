package scoredb

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"testing"
)

func TestNewArrayDB(t *testing.T) {
	mdb := db.NewMapDB()
	tree := trie_manager.NewMutable(mdb, nil)
	store := &TestStore{tree}

	arraydb := NewArrayDB(store, "Test")
	arraydb.Put("Value1")
	arraydb.Put("Value2")
	arraydb.Set(1, "Value3")

	if err := arraydb.Set(2, "Value4"); err == nil {
		t.Errorf("It should fail on Set(2,Value4)")
		return
	}

	if s := arraydb.Size(); s != 2 {
		t.Errorf("Size must be 2, but s=%d", s)
		return
	}

	if v := arraydb.Get(0).String(); v != "Value1" {
		t.Errorf("Fail to verify array exp=%s value=%s", "Value1", v)
		return
	}

	if v := arraydb.Pop().String(); v != "Value3" {
		t.Errorf("Poped value=%s is different from Value3", v)
		return
	}
	if v := arraydb.Pop(); v == nil {
		t.Errorf("Poped value must not be nil")
		return
	}
	if v := arraydb.Pop(); v != nil {
		t.Errorf("Poping on empty array should return nil")
		return
	}
}
