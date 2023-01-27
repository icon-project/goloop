package icstate

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

func newDummyState(readonly bool) *State {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	return NewStateFromSnapshot(NewSnapshot(database, nil), readonly, log.GlobalLogger())
}

func flushAndNewState(s *State, readonly bool) *State {
	ss := s.GetSnapshot()
	ss.Flush()
	return NewStateFromSnapshot(NewSnapshot(ss.store.Database(), ss.Bytes()), false, nil)
}

func newDummyPRepInfo(i int) *PRepInfo {
	city := fmt.Sprintf("Seoul%d", i)
	country := "KOR"
	name := fmt.Sprintf("node%d", i)
	email := fmt.Sprintf("%s@email.com", name)
	website := fmt.Sprintf("https://%s.example.com/", name)
	details := fmt.Sprintf("%sdetails/", website)
	endpoint := fmt.Sprintf("%s.example.com:9080", name)
	return &PRepInfo{
		City:        &city,
		Country:     &country,
		Name:        &name,
		Email:       &email,
		WebSite:     &website,
		Details:     &details,
		P2PEndpoint: &endpoint,
	}
}

func TestPRepBaseCache(t *testing.T) {
	var base, base1, base2 *PRepBaseState
	var addr1, addr2 module.Address

	s := newDummyState(false)

	addr1 = common.MustNewAddressFromString("hx1")
	ri1 := newDummyPRepInfo(1)

	// cache added
	base = s.prepBaseCache.Get(addr1, false)
	assert.Nil(t, base)
	base = s.prepBaseCache.Get(addr1, true)
	assert.NotNil(t, base)
	assert.True(t, base.IsEmpty())
	base.UpdateInfo(ri1)

	addr2 = common.MustNewAddressFromString("hx2")
	ri2 := newDummyPRepInfo(2)

	// cache added
	base = s.prepBaseCache.Get(addr2, true)
	assert.NotNil(t, base)
	assert.True(t, base.IsEmpty())
	base.UpdateInfo(ri2)

	s = flushAndNewState(s, false)

	base = s.prepBaseCache.Get(addr2, false)
	assert.NotNil(t, base)
	assert.False(t, base.IsEmpty())

	// Reset() reverts Clear(), should get after reset()
	base = s.prepBaseCache.Get(addr2, true)
	assert.False(t, base.IsEmpty())
	base.Clear()
	assert.True(t, base.IsEmpty())

	base = s.prepBaseCache.Get(addr2, false)
	assert.True(t, base.IsEmpty())

	s.prepBaseCache.Reset()
	base = s.prepBaseCache.Get(addr2, true)
	assert.False(t, base.IsEmpty())
	assert.True(t, base.info().equal(ri2))

	// item is removed from the map,
	// after it flush to DB, it is removed in DB
	base = s.prepBaseCache.Get(addr2, true)
	base.Clear()
	s.prepBaseCache.Flush()
	base = s.prepBaseCache.Get(addr2, false)
	assert.Nil(t, base)

	// Reset cannot get items from DB after clear()
	s.prepBaseCache.Clear()
	s.prepBaseCache.Reset()

	// but it can get item, using Get() specifically
	base1 = s.prepBaseCache.Get(addr1, false)
	assert.NotNil(t, base1)
	assert.False(t, base1.IsEmpty())
	assert.True(t, base1.info().equal(ri1))

	base2 = s.prepBaseCache.Get(addr2, false)
	assert.Nil(t, base2)
}

func TestPRepStatusCache(t *testing.T) {
	var status *PRepStatusState
	var addr1, addr2 module.Address

	s := newDummyState(true)

	addr1 = common.MustNewAddressFromString("hx1")
	vTotal := int64(100)

	// check if item is not present
	status = s.prepStatusCache.Get(addr1, false)
	assert.Nil(t, status)

	// cache added
	status = s.prepStatusCache.Get(addr1, true)
	assert.NotNil(t, status)
	assert.True(t, status.IsEmpty())
	err := status.Activate()
	assert.NoError(t, err)
	ss1 := status.GetSnapshot()

	addr2 = common.MustNewAddressFromString("hx2")
	status = s.prepStatusCache.Get(addr2, true)
	assert.NotNil(t, status)
	assert.True(t, status.IsEmpty())
	status.SetVTotal(vTotal)
	assert.Equal(t, vTotal, status.VTotal())

	s = flushAndNewState(s, false)

	// Reset() reverts Clear(), should get after reset()
	status = s.prepStatusCache.Get(addr2, false)
	status.Clear()
	s.prepStatusCache.Reset()

	status = s.prepStatusCache.Get(addr2, false)
	assert.NotNil(t, status)
	assert.False(t, status.IsEmpty())
	assert.Equal(t, vTotal, status.VTotal())

	// item is removed in the map,
	// after it flush to DB, it is removed in DB
	status = s.prepStatusCache.Get(addr2, false)
	status.Clear()
	s.prepStatusCache.Flush()

	status = s.prepStatusCache.Get(addr2, false)
	assert.Nil(t, status)

	// Reset cannot get items from DB after clear()
	s.prepStatusCache.Clear()
	s.prepStatusCache.Reset()

	// but it can get item, using Get() specifically
	status = s.prepStatusCache.Get(addr1, false)
	assert.NotNil(t, status)
	assert.True(t, ss1.Equal(status.GetSnapshot()))
	assert.False(t, status.IsEmpty())
}
