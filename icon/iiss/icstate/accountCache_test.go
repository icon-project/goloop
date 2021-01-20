package icstate

import (
	"fmt"
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
	account = s.accountCache.Get(addr2)
	assert.Equal(t, 0,account.stake.Cmp(big.NewInt(0)))
	assert.True(t, account.IsEmpty())

	account.SetStake(big.NewInt(int64(100)))

	// flush without add
	s.accountCache.Flush()

	// cannot affect DB, it didn't add addr2
	o = s.accountCache.dict.Get(addr2)
	assert.Nil(t, o)

	// add new addr2
	s.accountCache.Add(newAccount(addr2))
	account = s.accountCache.Get(addr2)
	account.SetStake(big.NewInt(int64(100)))

	// flush
	s.accountCache.Flush()

	// exist in dict
	o = s.accountCache.dict.Get(addr2)
	assert.NotEqual(t, nil, o)

	// remove
	s.accountCache.Remove(addr1)
	account = s.accountCache.Get(addr1)
	assert.Equal(t, true, account.IsEmpty())

	// cannot get even if reset
	s.accountCache.Reset()
	account = s.accountCache.Get(addr1)
	assert.Equal(t, true, account.IsEmpty())

	// clear
	s.accountCache.Clear()
	// nothing to flush, cannot affect dictDB
	s.accountCache.Flush()
	// Get() gets data directly from dictDB, if there's no in Map
	account = s.accountCache.Get(addr2)
	fmt.Println(account.address)
	assert.Equal(t, false, account.IsEmpty())
}
