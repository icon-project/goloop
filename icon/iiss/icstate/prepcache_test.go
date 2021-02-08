package icstate

import (
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/stretchr/testify/assert"
)

func TestPrepBaseCache(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	s := NewStateFromSnapshot(NewSnapshot(database, nil), false)

	addr := common.NewAddressFromString("hx1")

	// cache added
	base := s.prepBaseCache.Get(addr, false)
	assert.Nil(t, base)
	base = s.prepBaseCache.Get(addr, true)

	addr = common.NewAddressFromString("hx2")

	// cache added
	base = s.prepBaseCache.Get(addr, true)
	base.SetPRep("name", "emal" , "web" , "country", "city","deatil", "end", addr)

	key := icutils.ToKey(addr)
	val := s.prepBaseCache.dict.Get(key)

	assert.Nil(t,val)

	// DB write
	s.prepBaseCache.Flush()
	key = icutils.ToKey(addr)
	val = s.prepBaseCache.dict.Get(key)
	assert.NotNil(t,val)

	// Reset() reverts Clear(), should get after reset()
	base = s.prepBaseCache.Get(addr, true)
	base.Clear()

	s.prepBaseCache.Reset()
	base = s.prepBaseCache.Get(addr, true)
	assert.False(t, base.IsEmpty())
	assert.Equal(t, "name" , base.name)

	// item is removed in the map,
	// after it flush to DB, it is removed in DB
	base = s.prepBaseCache.Get(addr, true)
	base.Clear()
	s.prepBaseCache.Flush()
	key = icutils.ToKey(addr)
	val = s.prepBaseCache.dict.Get(key)
	assert.Nil(t,val)

	// Reset cannot get items from DB after clear()
	s.prepBaseCache.Clear()
	s.prepBaseCache.Reset()

	assert.Equal(t, 0, len(s.prepBaseCache.bases))

	// but it can get item, using Get() specifically
	addr = common.NewAddressFromString("hx1")
	base = s.prepBaseCache.Get(addr, true)

	assert.Equal(t, 1, len(s.prepBaseCache.bases))
}

func TestPrepStatusCache(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	s := NewStateFromSnapshot(NewSnapshot(database, nil), false)

	addr := common.NewAddressFromString("hx1")

	// cache added
	status := s.prepStatusCache.Get(addr, false)
	assert.Nil(t, status)
	status = s.prepStatusCache.Get(addr, true)

	addr = common.NewAddressFromString("hx2")
	status = s.prepStatusCache.Get(addr, true)
	status.SetVTotal(100)

	// cache added
	key := icutils.ToKey(addr)
	val := s.prepStatusCache.dict.Get(key)

	assert.Nil(t,val)

	// DB write
	s.prepStatusCache.Flush()
	key = icutils.ToKey(addr)
	val = s.prepStatusCache.dict.Get(key)
	assert.NotNil(t,val)

	// Reset() reverts Clear(), should get after reset()
	status = s.prepStatusCache.Get(addr, true)
	status.Clear()
	s.prepStatusCache.Reset()
	status = s.prepStatusCache.Get(addr, true)
	assert.False(t, status.IsEmpty())
	assert.Equal(t, 100, status.vTotal)

	// item is removed in the map,
	// after it flush to DB, it is removed in DB
	status = s.prepStatusCache.Get(addr, true)
	status.Clear()
	s.prepStatusCache.Flush()
	key = icutils.ToKey(addr)
	val = s.prepStatusCache.dict.Get(key)
	assert.Nil(t,val)

	// Reset cannot get items from DB after clear()
	s.prepStatusCache.Clear()
	s.prepStatusCache.Reset()

	assert.Equal(t, 0, len(s.prepStatusCache.statuses))

	// but it can get item, using Get() specifically
	addr = common.NewAddressFromString("hx1")
	status = s.prepStatusCache.Get(addr, true)

	assert.Equal(t, 1, len(s.prepStatusCache.statuses))
}
