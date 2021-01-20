package icstate

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
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

func newPRepSnapshot(owner module.Address, delegated int64, bond int64) *PRepSnapshot {
	status := NewPRepStatus(owner)
	status.SetDelegated(big.NewInt(delegated))
	status.SetBonded(big.NewInt(bond))

	return NewPRepSnapshotFromPRepStatus(status, 5)
}

func newPRepSnapshots(seed int, size int) PRepSnapshots {
	ret := make([]*PRepSnapshot, size, size)
	for i := 0; i < size; i++ {
		owner := newAddress(byte(seed + i))
		ret[i] = newPRepSnapshot(owner, int64(size-i), int64(size-i))
	}
	return ret
}

func TestPRepSnapshot_Equal(t *testing.T) {
	owner0 := newAddress(0)
	owner1 := newAddress(1)
	delegated := int64(1000)
	bond := delegated / 2
	p0 := newPRepSnapshot(owner0, delegated, bond)

	cases := []struct {
		p0, p1 *PRepSnapshot
		result bool
	}{
		{nil, nil, true},
		{p0, p0, true},
		{p0, p0.Clone(), true},
		{p0, newPRepSnapshot(owner0, delegated, bond), true},
		{nil, p0, false},
		{p0, nil, false},
		{p0, newPRepSnapshot(owner1, delegated, bond), false},
		{p0, newPRepSnapshot(owner0, delegated-1, bond), false},
		{p0, newPRepSnapshot(owner0, delegated, bond-1), false},
	}

	for _, c := range cases {
		assert.True(t, c.p0.Equal(c.p1) == c.result)
	}
}

func TestPRepSnapshot_Bytes(t *testing.T) {
	owner := newAddress(1)
	delegated := int64(1000)
	bond := delegated / 2
	ps := newPRepSnapshot(owner, delegated, bond)
	bs, err := codec.BC.MarshalToBytes(ps)
	assert.NoError(t, err)

	var ps2 PRepSnapshot
	_, err = codec.BC.UnmarshalFromBytes(bs, &ps2)
	assert.NoError(t, err)

	assert.True(t, ps.Equal(&ps2))
}

func TestPRepSnapshots_Equal(t *testing.T) {
	size := 3
	snapshots := make(PRepSnapshots, size, size)

	for i := 0; i < size; i++ {
		snapshots[i] = newPRepSnapshot(newAddress(byte(i)), rand.Int63(), rand.Int63())
	}

	cases := []struct {
		p0, p1 PRepSnapshots
		result bool
	}{
		{nil, nil, true},
		{nil, snapshots, false},
		{snapshots, nil, false},
		{snapshots, snapshots, true},
		{snapshots, snapshots.Clone(), true},
	}

	for _, c := range cases {
		assert.True(t, c.p0.Equal(c.p1) == c.result)
	}
}

func TestTerm_Equal(t *testing.T) {
	t0 := newTerm(0, 10)
	tSequence := t0.Clone()
	tSequence.sequence = t0.sequence + 1
	tSet := tSequence.Clone()
	tSet.Set(t0)
	tSH := t0.Clone()
	tSH.startHeight = t0.startHeight + 1
	tPeriod := t0.Clone()
	tPeriod.period = t0.period + 1
	tIrep := t0.Clone()
	tIrep.irep.SetInt64(t0.irep.Int64() + 1)
	tRrep := t0.Clone()
	tRrep.rrep.SetInt64(t0.rrep.Int64() + 1)
	tTS := t0.Clone()
	tTS.totalSupply.SetInt64(t0.totalSupply.Int64() + 1)
	tTD := t0.Clone()
	tTD.totalDelegated.SetInt64(t0.totalDelegated.Int64() + 1)
	tSnapshots := t0.Clone()
	tSnapshots.SetPRepSnapshots(newPRepSnapshots(1, 2))

	cases := []struct {
		name   string
		t0, t1 *Term
		result bool
	}{
		{"nil comp", nil, nil, true},
		{"same instance", t0, t0, true},
		{"clone", t0, t0.Clone(), true},
		{"newTerm() with same param", t0, newTerm(0, 10), true},
		{"nil to instance", nil, t0, false},
		{"instance to nil", t0, nil, false},
		{"Set()", t0, tSet, true},
		{"diff sequence", t0, tSequence, false},
		{"diff startHeight", t0, tSH, false},
		{"diff period", t0, tPeriod, false},
		{"diff IRep", t0, tIrep, false},
		{"diff RRep", t0, tRrep, false},
		{"diff totalSupply", t0, tTS, false},
		{"diff totalDelegated", t0, tTD, false},
		{"diff prepSnapshots", t0, tSnapshots, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.True(t, c.t0.Equal(c.t1) == c.result, "%v\n%v", c.t0, c.t1)
		})
	}
}

func TestTerm_Bytes(t *testing.T) {
	term := newTerm(0, 10)

	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	o1 := icobject.New(TypeTerm, term)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())
	assert.True(t, o1.Equal(o2))
}

func TestTerm_Clone(t *testing.T) {
	term := newTerm(0, 43120)
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

func TestTerm_PRepSnapshot(t *testing.T) {
	size := 100
	term := newTerm(0, 43120)
	prepSnapshots := newPRepSnapshots(0, size)
	term.SetPRepSnapshots(prepSnapshots.Clone())

	assert.True(t, len(term.prepSnapshots) == len(prepSnapshots))
	assert.True(t, len(term.snapshotMap) == len(prepSnapshots))

	// check snapshot values
	totalBondedDelegation := new(big.Int)
	for i, ps := range prepSnapshots {
		owner := ps.owner
		key := icutils.ToKey(owner)

		ps1 := term.prepSnapshots[i]
		assert.True(t, ps.Equal(ps1))

		ps2, ok := term.snapshotMap[key]
		assert.True(t, ok)
		assert.True(t, ps.Equal(ps2))

		totalBondedDelegation.Add(totalBondedDelegation, ps.BondedDelegation())
	}
	assert.Equal(t, 0, totalBondedDelegation.Cmp(term.GetTotalBondedDelegation()))

	// GetPRepSnapshot...()
	for i := 0; i < size; i++ {
		ps := term.GetPRepSnapshotByIndex(i)
		assert.Equal(t, ps, prepSnapshots[i])
		assert.Equal(t, ps, term.GetPRepSnapshotByOwner(ps.owner))
	}

	// RemovePRepSnapshot()
	for _, ps := range prepSnapshots {
		owner := ps.owner
		key := icutils.ToKey(owner)

		_, ok := term.snapshotMap[key]
		assert.True(t, ok)
		assert.NotEqual(t, -1, term.getPRepSnapshotIndex(owner))

		err := term.RemovePRepSnapshot(owner)
		assert.NoError(t, err)

		_, ok = term.snapshotMap[key]
		assert.False(t, ok)
		assert.Equal(t, -1, term.getPRepSnapshotIndex(owner))
	}

	assert.Equal(t, 0, len(term.prepSnapshots))
	assert.Equal(t, 0, len(term.snapshotMap))
}

func TestTerm_NewNextTerm(t *testing.T) {
	totalSupply := big.NewInt(1000)
	totalDelegated := big.NewInt(100)
	period := int64(100)
	irep := big.NewInt(2000)
	rrep := big.NewInt(1500)

	term := newTerm(0, 100)
	nTerm := NewNextTerm(term, period, irep, rrep, totalSupply, totalDelegated)

	assert.Equal(t, term.sequence+1, nTerm.sequence)
	assert.Equal(t, term.GetEndBlockHeight()+1, nTerm.startHeight)
	assert.Equal(t, period, nTerm.period)
	assert.Equal(t, irep.Int64(), nTerm.irep.Int64())
	assert.Equal(t, rrep.Int64(), nTerm.rrep.Int64())
	assert.Equal(t, totalSupply.Int64(), nTerm.totalSupply.Int64())
	assert.Equal(t, totalDelegated.Int64(), nTerm.totalDelegated.Int64())
	assert.Equal(t, FlagNextTerm, nTerm.flags & FlagNextTerm)
}
