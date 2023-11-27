package icstate

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestAccountCache(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	mutable := trie_manager.NewMutableForObject(database, nil, icobject.ObjectType)
	oss := icobject.NewObjectStoreState(mutable)
	cache := newAccountCache(oss)

	addr1 := common.MustNewAddressFromString("hx1")
	addr2 := common.MustNewAddressFromString("hx2")

	account := cache.Get(addr1, false)
	assert.Nil(t, account)

	account = cache.Get(addr1, true)
	assert.NoError(t, account.SetStake(big.NewInt(int64(40))))

	account = cache.Get(addr2, true)
	assert.NoError(t, account.SetStake(big.NewInt(int64(100))))

	// flush to the database
	cache.Flush()
	ss1 := mutable.GetSnapshot()
	err := ss1.Flush()
	assert.NoError(t, err)

	// check stored value with new cache instance
	mutable = trie_manager.NewMutableForObject(database, ss1.Hash(), icobject.ObjectType)
	oss = icobject.NewObjectStoreState(mutable)
	cache = newAccountCache(oss)
	ass1 := cache.GetSnapshot(addr1)
	assert.NotNil(t, ass1)
	assert.Equal(t, 0, ass1.Stake().Cmp(big.NewInt(40)))

	ass2 := cache.GetSnapshot(addr2)
	assert.NotNil(t, ass2)
	assert.Equal(t, 0, ass2.Stake().Cmp(big.NewInt(100)))

	// take snapshot for the state ( addr1 -> 40, addr2 -> 100 )
	ss1 = mutable.GetSnapshot()

	ac1 := cache.Get(addr1, true)
	ac2 := cache.Get(addr2, true)
	assert.NotNil(t, ac1)
	assert.NoError(t, ac1.SetStake(big.NewInt(50)))
	assert.Equal(t, 0, ac1.Stake().Cmp(big.NewInt(50)))
	ac2.Clear()
	assert.True(t, ac2.IsEmpty())

	// take snapshot for the state ( addr1 -> 50, addr2 -> 0 )
	cache.Flush()
	ss2 := mutable.GetSnapshot()

	// reset to ss1 and check values
	mutable.Reset(ss1)
	cache.Reset()
	assert.Equal(t, 0, ac1.Stake().Cmp(big.NewInt(40)))
	assert.Equal(t, 0, ac2.Stake().Cmp(big.NewInt(100)))

	// reset to ss2 and check values
	mutable.Reset(ss2)
	cache.Reset()
	assert.Equal(t, 0, ac1.Stake().Cmp(big.NewInt(50)))
	assert.True(t, ac2.IsEmpty())
}
