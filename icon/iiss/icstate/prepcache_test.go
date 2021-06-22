package icstate

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

func newDummyState(readonly bool) *State {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	return NewStateFromSnapshot(NewSnapshot(database, nil), readonly, nil)
}

func flushAndNewState(s *State, readonly bool) *State {
	ss := s.GetSnapshot()
	ss.Flush()
	return NewStateFromSnapshot(NewSnapshot(ss.store.Database(), ss.Bytes()), false, nil)
}

func newDummyRegInfo(i int) *RegInfo {
	city := fmt.Sprintf("Seoul%d", i)
	country := "KOR"
	name := fmt.Sprintf("node%d", i)
	email := fmt.Sprintf("%s@email.com", name)
	website := fmt.Sprintf("https://%s.example.com/", name)
	details := fmt.Sprintf("%sdetails/", website)
	endpoint := fmt.Sprintf("%s.example.com:9080", name)
	node := module.Address(nil)

	return NewRegInfo(city, country, details, email, name, endpoint, website, node)
}

func TestPRepBaseCache(t *testing.T) {
	var err error
	var created bool
	var base, base1, base2 *PRepBaseState
	var addr1, addr2 module.Address

	s := newDummyState(false)

	addr1 = common.MustNewAddressFromString("hx1")
	ri1 := newDummyRegInfo(1)

	// cache added
	base, created = s.prepBaseCache.Get(addr1, false)
	assert.Nil(t, base)
	assert.False(t, created)
	base, created = s.prepBaseCache.Get(addr1, true)
	assert.True(t, created)
	assert.True(t, base.IsEmpty())
	err = base.SetRegInfo(ri1)
	assert.NoError(t, err)

	addr2 = common.MustNewAddressFromString("hx2")
	ri2 := newDummyRegInfo(2)

	// cache added
	base, created = s.prepBaseCache.Get(addr2, true)
	assert.NotNil(t, base)
	assert.True(t, created)
	err = base.SetRegInfo(ri2)
	assert.NoError(t, err)

	s = flushAndNewState(s, false)

	base, created = s.prepBaseCache.Get(addr2, false)
	assert.NotNil(t, base)
	assert.False(t, created)

	// Reset() reverts Clear(), should get after reset()
	base, created = s.prepBaseCache.Get(addr2, true)
	assert.False(t, created)
	base.Clear()
	assert.True(t, base.IsEmpty())

	base, created = s.prepBaseCache.Get(addr2, false)
	assert.True(t, base.IsEmpty())
	assert.False(t, created)

	s.prepBaseCache.Reset()
	base, created = s.prepBaseCache.Get(addr2, true)
	assert.False(t, base.IsEmpty())
	assert.False(t, created)
	assert.Equal(t, "node2", base.info.name)

	// item is removed from the map,
	// after it flush to DB, it is removed in DB
	base, created = s.prepBaseCache.Get(addr2, true)
	base.Clear()
	s.prepBaseCache.Flush()
	base, created = s.prepBaseCache.Get(addr2, false)
	assert.Nil(t, base)
	assert.False(t, created)

	// Reset cannot get items from DB after clear()
	s.prepBaseCache.Clear()
	s.prepBaseCache.Reset()

	// but it can get item, using Get() specifically
	base1, created = s.prepBaseCache.Get(addr1, false)
	assert.NotNil(t, base1)
	assert.False(t, created)

	base2, created = s.prepBaseCache.Get(addr2, false)
	assert.Nil(t, base2)
	assert.False(t, created)
}

func TestPRepStatusCache(t *testing.T) {
	var created bool
	var status *PRepStatusState
	var addr1, addr2 module.Address

	s := newDummyState(true)

	addr1 = common.MustNewAddressFromString("hx1")
	vTotal := int64(100)

	// check if item is not present
	status, created = s.prepStatusCache.Get(addr1, false)
	assert.Nil(t, status)
	assert.False(t, created)

	// cache added
	status, created = s.prepStatusCache.Get(addr1, true)
	assert.NotNil(t, status)
	assert.True(t, created)
	err := status.Activate()
	assert.NoError(t, err)
	ss1 := status.GetSnapshot()

	addr2 = common.MustNewAddressFromString("hx2")
	status, created = s.prepStatusCache.Get(addr2, true)
	assert.NotNil(t, status)
	assert.True(t, created)
	status.SetVTotal(vTotal)
	assert.Equal(t, vTotal, status.VTotal())

	s = flushAndNewState(s, false)

	// Reset() reverts Clear(), should get after reset()
	status, created = s.prepStatusCache.Get(addr2, false)
	status.Clear()
	s.prepStatusCache.Reset()

	status, created = s.prepStatusCache.Get(addr2, false)
	assert.False(t, status.IsEmpty())
	assert.False(t, created)
	assert.Equal(t, vTotal, status.VTotal())

	// item is removed in the map,
	// after it flush to DB, it is removed in DB
	status, created = s.prepStatusCache.Get(addr2, false)
	status.Clear()
	s.prepStatusCache.Flush()

	status, created = s.prepStatusCache.Get(addr2, false)
	assert.Nil(t, status)
	assert.False(t, created)

	// Reset cannot get items from DB after clear()
	s.prepStatusCache.Clear()
	s.prepStatusCache.Reset()

	// but it can get item, using Get() specifically
	status, created = s.prepStatusCache.Get(addr1, false)
	assert.NotNil(t, status)
	assert.True(t, ss1.Equal(status.GetSnapshot()))
	assert.False(t, created)
}
