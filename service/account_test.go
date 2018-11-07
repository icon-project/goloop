package service

import (
	"bytes"
	"github.com/icon-project/goloop/common/db"
	"math/big"
	"testing"
)

func TestAccountSnapshot_Equal(t *testing.T) {
	database := db.NewMapDB()
	as := newAccountState(database, nil)

	s1 := as.getSnapshot()
	if !s1.Equal(s1) {
		t.Errorf("Fail to check equality with same snapshot")
		return
	}

	s2 := as.getSnapshot()
	if !s1.Equal(s2) {
		t.Errorf("Fail to check equality with another snapshot without change")
		return
	}

	v1 := s1.getBalance()
	v1.Add(v1, big.NewInt(30))
	as.setBalance(v1)

	s3 := as.getSnapshot()
	if s1.Equal(s3) {
		t.Errorf("Fail to compare snapshot after setBalance()")
	}

	kv := []byte("Test")
	as.setValue(kv, kv)

	s4 := as.getSnapshot()
	if s3.Equal(s4) {
		t.Errorf("Fail to compare snapshot after setValue()")
	}

	as.deleteValue(kv)

	s5 := as.getSnapshot()
	if !s3.Equal(s5) {
		t.Errorf("Fail to compare snapshot after deleteValue()")
	}
}

func TestAccountSnapshot_Bytes(t *testing.T) {
	database := db.NewMapDB()
	as := newAccountState(database, nil)
	v1 := big.NewInt(3000)
	as.setBalance(v1)
	tv := []byte("Puha")
	as.setValue(tv, tv)
	s1 := as.getSnapshot()

	serialized := s1.Bytes()
	s1.Flush()

	s2 := new(accountSnapshotImpl)
	s2.Reset(database, serialized)

	v2 := s2.getBalance()
	if v1.Cmp(v2) != 0 {
		t.Errorf("Fail to get same balance")
	}

	tv2, _ := s2.getValue(tv)
	if !bytes.Equal(tv, tv2) {
		t.Errorf("Fail to get same value exp=%x returned=%x", tv, tv2)
	}
}
