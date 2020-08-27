package state

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
)

func TestAccountSnapshot_Equal(t *testing.T) {
	database := db.NewMapDB()
	as := newAccountState(database, nil, nil, false)

	s1 := as.GetSnapshot()
	if !s1.Equal(s1) {
		t.Errorf("Fail to check equality with same snapshot")
		return
	}

	s2 := as.GetSnapshot()
	if !s1.Equal(s2) {
		t.Errorf("Fail to check equality with another snapshot without change")
		return
	}

	v1 := s1.GetBalance()
	as.SetBalance(new(big.Int).Add(v1, big.NewInt(30)))

	s3 := as.GetSnapshot()
	if s1.Equal(s3) {
		t.Errorf("Fail to compare snapshot after SetBalance()")
	}

	kv := []byte("Test")
	as.SetValue(kv, kv)

	s4 := as.GetSnapshot()
	if s3.Equal(s4) {
		t.Errorf("Fail to compare snapshot after SetValue()")
	}

	as.DeleteValue(kv)

	s5 := as.GetSnapshot()
	if !s3.Equal(s5) {
		t.Errorf("Fail to compare snapshot after DeleteValue()")
	}
}

func TestAccountSnapshot_Bytes(t *testing.T) {
	database := db.NewMapDB()
	as := newAccountState(database, nil, nil, false)
	v1 := big.NewInt(3000)
	as.SetBalance(v1)
	tv := []byte("Puha")
	as.SetValue(tv, tv)
	s1 := as.GetSnapshot()

	serialized := s1.Bytes()
	s1.Flush()

	t.Logf("Serialized:% X", serialized)

	s2 := new(accountSnapshotImpl)
	s2.Reset(database, serialized)

	assert.Equal(t, serialized, s2.Bytes())

	v2 := s2.GetBalance()
	if v1.Cmp(v2) != 0 {
		t.Errorf("Fail to get same balance")
	}

	tv2, _ := s2.GetValue(tv)
	assert.Equal(t, tv, tv2)
}
