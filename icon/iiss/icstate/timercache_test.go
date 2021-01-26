package icstate

import (
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/stretchr/testify/assert"
)

var testTimerDictPrefix = containerdb.ToKey(containerdb.RawBuilder, "timer_test")

func TestTimerCache(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	tree := trie_manager.NewMutableForObject(database, nil, icobject.ObjectType)
	oss := icobject.NewObjectStoreState(tree)

	tc := newTimerCache(oss, testTimerDictPrefix)
	timer := newTimer(100)
	addr := common.NewAddressFromString("hx1")
	// add address to timer 100
	timer.Add(addr)
	// add timer 100 to tc
	tc.Add(timer)

	// should get 100
	res := tc.Get(100)
	assert.NotNil(t, res)

	// should not get 100 from DB, because it didn't flush
	o := tc.dict.Get(100)
	assert.Nil(t, o)

	// flushed(100)
	tc.Flush()

	// should not be nil
	o = tc.dict.Get(100)
	assert.NotNil(t, o)

	timer = newTimer(110)
	addr = common.NewAddressFromString("hx2")
	timer.Add(addr)
	// new timer 110 added
	tc.Add(timer)

	// 110 should not be empty
	timer = tc.Get(110)
	assert.False(t, timer.IsEmpty())

	// the item 110 in map will be removed after reset(), because there is no in DB
	tc.Reset()
	timer = tc.Get(110)
	assert.Nil(t, timer)

	timer = newTimer(110)
	addr = common.NewAddressFromString("hx2")

	// item 110 added and flushed, DB will have both 100, 110
	timer.Add(addr)
	tc.Add(timer)
	tc.Flush()

	// Reset cannot recover the data after it is explicitly removed
	// remove item 100 in the map, not DB
	tc.Remove(100)
	tc.Reset()
	timer = tc.Get(100)
	// should be still empty
	assert.False(t, timer.IsEmpty())

	// after Clear(), it cannot recover any data from DB by Reset()
	tc.Clear()
	tc.Reset()
	assert.Equal(t, 0, len(tc.timers))

	// but, it can recover specific item, using Get()
	timer = tc.Get(110)
	assert.NotNil(t, timer)
}
