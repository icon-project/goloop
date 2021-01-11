package icstate

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/stretchr/testify/assert"
	"math/big"
	"math/rand"
	"testing"
)

func newAddress(value byte) module.Address {
	bs := make([]byte, common.AddressBytes)
	bs[common.AddressBytes-1] = value
	address, _ := common.NewAddress(bs)
	return address
}

func newPRepSnapshot(owner module.Address, delegated int64) *PRepSnapshot {
	return &PRepSnapshot{owner, big.NewInt(delegated)}
}

func newPRepSnapshots(seed int, size int) []*PRepSnapshot {
	ret := make([]*PRepSnapshot, size, size)
	for i := 0; i < size; i++ {
		owner := newAddress(byte(seed + i))
		ret[i] = newPRepSnapshot(owner, int64(size-i))
	}
	return ret
}

func TestPRepSnapshot_Equal(t *testing.T) {
	owner0 := newAddress(0)
	owner1 := newAddress(1)
	delegated := int64(1000)
	p0 := newPRepSnapshot(owner0, delegated)

	cases := []struct {
		p0, p1 *PRepSnapshot
		result bool
	}{
		{nil, nil, true},
		{p0, p0, true},
		{p0, newPRepSnapshot(owner0, delegated), true},
		{nil, p0, false},
		{p0, nil, false},
		{p0, newPRepSnapshot(owner1, delegated), false},
		{p0, newPRepSnapshot(owner0, 2000), false},
	}

	for _, c := range cases {
		assert.True(t, c.p0.Equal(c.p1) == c.result)
	}
}

func TestPRepSnapshots_Equal(t *testing.T) {
	size := 3
	snapshots := make(PRepSnapshots, size, size)

	for i := 0; i < size; i++ {
		snapshots[i] = newPRepSnapshot(newAddress(byte(i)), rand.Int63())
	}

	cases := []struct {
		p0, p1 PRepSnapshots
		result bool
	}{
		{nil, nil, true},
		{nil, snapshots, false},
		{snapshots, nil, false},
		{snapshots, snapshots, true},
	}

	for _, c := range cases {
		assert.True(t, c.p0.Equal(c.p1) == c.result)
	}
}

func TestTerm_GetPRepSnapshot(t *testing.T) {
	size := 100
	term := newTerm()
	prepSnapshots := newPRepSnapshots(0, size)
	term.SetPRepSnapshots(prepSnapshots)

	for i := 0; i < size; i++ {
		ps := term.GetPRepSnapshot(i)
		assert.Equal(t, ps, prepSnapshots[i])
		assert.Equal(t, ps, term.GetPRepSnapshotByOwner(ps.owner))
	}
}

func TestTerm_SetPRepSnapshots(t *testing.T) {
	term := newTerm()

	size := 30
	prepSnapshots := newPRepSnapshots(0, size)

	term.SetPRepSnapshots(prepSnapshots)
	assert.True(t, len(term.prepSnapshots) == len(prepSnapshots))
	assert.True(t, len(term.snapshotMap) == len(prepSnapshots))

	for i := 0; i < size; i++ {
		owner := newAddress(byte(i))
		key := icutils.ToKey(owner)

		ps := term.snapshotMap[key]
		assert.True(t, ps != nil)
	}

	for i := 0; i < size; i++ {
		owner := newAddress(byte(i))
		err := term.RemovePRepSnapshot(owner)
		assert.Nil(t, err)

		key := icutils.ToKey(owner)
		ps := term.snapshotMap[key]
		assert.True(t, ps == nil)
	}

	assert.Equal(t, 0, len(term.prepSnapshots))
	assert.Equal(t, 0, len(term.snapshotMap))
}

func TestTerm_Clone(t *testing.T) {
	term := newTerm()
	term2 := term.Clone()
	assert.True(t, term.Equal(term2))

	size := 100
	prepSnapshots := newPRepSnapshots(100, size)
	term.SetPRepSnapshots(prepSnapshots)
	term2 = term.Clone()
	assert.True(t, term.Equal(term2))

	for i := 0; i < size; i++ {
		assert.True(t, term.prepSnapshots[i] != term2.prepSnapshots[i])
	}
}
