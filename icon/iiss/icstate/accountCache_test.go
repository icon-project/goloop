package icstate

import (
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"math/big"
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/stretchr/testify/assert"
)

func TestAccountCache(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	s := NewStateFromSnapshot(NewSnapshot(database, nil), false)

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")

	// add
	s.accountCache.Add(newAccount(addr1))
	s.accountCache.Add(newAccount(addr2))

	account := s.accountCache.Get(addr1)
	account.SetStake(big.NewInt(int64(40)))

	account = s.accountCache.Get(addr2)
	account.SetStake(big.NewInt(int64(100)))

	// flush
	s.accountCache.Flush()

	// there should be addr1 in DB after Flush()
	o := s.accountCache.dict.Get(addr1)
	account = ToAccount(o.Object(), addr1)
	assert.Equal(t, 0,account.stake.Cmp(big.NewInt(40)))

	// item(addr2) should be gotten from the map, although it is deleted in DB
	s.accountCache.dict.Delete(addr2)
	account = s.accountCache.Get(addr2)
	assert.Equal(t, 0,account.stake.Cmp(big.NewInt(100)))

	// reset
	s.accountCache.Reset()

	// Reset() will affect on items in map
	// Get() will return empty object, not nil, if there is no both in map and db
	account = s.accountCache.Get(addr2)
	assert.Equal(t, 0,account.stake.Cmp(big.NewInt(0)))

	assert.False(t, account.IsEmpty())

	account.SetStake(big.NewInt(int64(100)))


	// flush without add
	s.accountCache.Flush()

	// DB reflected after Flush()
	o = s.accountCache.dict.Get(addr2)
	account = ToAccount(o.Object(), addr2)
	assert.Equal(t, 0,account.stake.Cmp(big.NewInt(100)))


	// remove
	s.accountCache.Remove(addr1)
	account = s.accountCache.Get(addr1)
	assert.True(t, account.IsEmpty())

	// Should get after reset()
	s.accountCache.Reset()
	account = s.accountCache.Get(addr1)
	assert.False(t, account.IsEmpty())
	assert.Equal(t, 0,account.stake.Cmp(big.NewInt(40)))

	// clear
	s.accountCache.Clear()
	// nothing to flush, cannot affect dictDB
	s.accountCache.Flush()
	// Get() gets data directly from dictDB, if there's no in Map
	account = s.accountCache.Get(addr2)
	assert.Equal(t, false, account.IsEmpty())
	assert.Equal(t, 0,account.stake.Cmp(big.NewInt(100)))

	s.accountCache.Remove(addr2)
	s.accountCache.Flush()

	key := icutils.ToKey(addr2)
	account = s.accountCache.accounts[key]
	assert.Nil(t, account)
}
