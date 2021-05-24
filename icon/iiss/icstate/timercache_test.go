package icstate

import (
	"testing"

	"github.com/icon-project/goloop/service/scoredb"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

var testTimerDictPrefix = containerdb.ToKey(
	containerdb.HashBuilder, scoredb.DictDBPrefix, "timer_test",
)

func TestTimerCache(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	tree := trie_manager.NewMutableForObject(database, nil, icobject.ObjectType)
	oss := icobject.NewObjectStoreState(tree)

	tc := newTimerCache(oss, testTimerDictPrefix)

	tss1 := tc.GetSnapshot(100)
	assert.Nil(t, tss1)
	timer := tc.Get(100)
	addr1 := common.MustNewAddressFromString("hx1")
	// add address to timer 100
	timer.Add(addr1)
	// add timer 100 to tc

	// should get 100
	tss1 = tc.GetSnapshot(100)
	assert.NotNil(t, tss1)
	assert.True(t, tss1.Contains(addr1))

	tc.Flush()

	timer = tc.Get(110)
	addr2 := common.MustNewAddressFromString("hx2")
	timer.Add(addr2)

	tss2 := tc.GetSnapshot(110)
	assert.NotNil(t, tss2)
	assert.False(t, tss2.IsEmpty())

	// revert changes and check.
	tc.Reset()
	assert.True(t, timer.IsEmpty())

	ss1 := tree.GetSnapshot()

	// item 110 added and flushed, DB will have both 100, 110
	timer.Add(addr2)
	tc.Flush()

	ss2 := tree.GetSnapshot()

	// switch back to ss1 and check
	tree.Reset(ss1)
	tc.Reset()

	timer = tc.Get(110)
	assert.True(t, timer.IsEmpty())

	// switch forward to ss2 and check
	tree.Reset(ss2)
	tc.Reset()

	timer = tc.Get(110)
	assert.False(t, timer.IsEmpty())

	// test Clear() whether it flushes the change
	addr3 := common.MustNewAddressFromString("hx3")
	timer.Add(addr3)
	tc.Clear()

	// original tc may reuse the object, so use new one.
	tc2 := newTimerCache(oss, testTimerDictPrefix)
	tss3 := tc2.GetSnapshot(110)
	assert.True(t, tss3.Contains(addr3))
	assert.True(t, tss3.Contains(addr2))
	tss4 := tc2.GetSnapshot(100)
	assert.True(t, tss4.Contains(addr1))
}
