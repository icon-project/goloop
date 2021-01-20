package icstate

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
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

	o := s.accountCache.dict.Get(addr1)
	account = ToAccount(o.Object(), addr1)
	assert.Equal(t, 0,account.stake.Cmp(big.NewInt(40)))

	// dict deleted
	s.accountCache.dict.Delete(addr2)
	account = s.accountCache.Get(addr2)
	assert.Equal(t, 0,account.stake.Cmp(big.NewInt(100)))

	// reset
	s.accountCache.Reset()

	account = s.accountCache.Get(addr2)
	assert.Equal(t, 0,account.stake.Cmp(big.NewInt(0)))

	account.SetStake(big.NewInt(int64(100)))

	// flush without add
	s.accountCache.Flush()

	o = s.accountCache.dict.Get(addr2)
	assert.Equal(t, nil, o)
	//account = ToAccount(o.Object(), addr2)

	// add new
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
