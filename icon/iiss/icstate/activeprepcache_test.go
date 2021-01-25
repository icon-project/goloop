package icstate

import (
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/stretchr/testify/assert"
)

func TestActivePrepCache(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	tree := trie_manager.NewMutableForObject(database, nil, icobject.ObjectType)
	oss := icobject.NewObjectStoreState(tree)
	activePRepCache := newActivePRepCache(oss)

	// add new active prep
	addr1 := common.NewAddressFromString("hx1")
	activePRepCache.Add(addr1)

	// size
	assert.Equal(t, 1, activePRepCache.Size())

	// there's no in arraydb, because it didn't flush
	val := activePRepCache.arraydb.Get(0)
	assert.Nil(t, val)

	activePRepCache.Flush()

	// there should be in arraydb after it flushed
	val = activePRepCache.arraydb.Get(0)
	assert.NotNil(t, val)


	addr2 := common.NewAddressFromString("hx2")
	activePRepCache.Add(addr2)
	addr3 := common.NewAddressFromString("hx3")
	activePRepCache.Add(addr3)
	// will write addr2, addr3 on DB
	activePRepCache.Flush()

	// remove all items in Map
	activePRepCache.Remove(addr2)
	activePRepCache.Remove(addr3)
	assert.Equal(t, 1, activePRepCache.Size())
	activePRepCache.Remove(addr1)
	assert.Equal(t, 0, activePRepCache.Size())

	// reset cannot recover although it reset
	// TODO fix me, Reset() cannot recover after explicit Remove()
	activePRepCache.Reset()
	assert.Equal(t, 3, activePRepCache.Size())

	// DB should be affected by Clear()
	activePRepCache.Clear()
	val = activePRepCache.arraydb.Get(0)
	assert.NotNil(t, val)

	// items in DB should be removed after Remove() Flush()
	activePRepCache.Flush()
	val = activePRepCache.arraydb.Get(0)
	assert.Nil(t, val)
}
