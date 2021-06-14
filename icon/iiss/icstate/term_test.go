package icstate

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

func newPRepSnapshot(owner module.Address, delegated int64, bond int64) *PRepSnapshot {
	status := NewPRepStatus()
	status.SetDelegated(big.NewInt(delegated))
	status.SetBonded(big.NewInt(bond))

	return NewPRepSnapshot(owner, status.GetBondedDelegation(5))
}

func newDummyAddress(value int) module.Address {
	bs := make([]byte, common.AddressBytes)
	for i := 0; value != 0 && i < 8; i++ {
		bs[common.AddressBytes-1-i] = byte(value & 0xFF)
		value >>= 8
	}
	return common.MustNewAddress(bs)
}

func newDummyPRepBase(i int) *PRepBase {
	ri := newDummyRegInfo(i)
	pb := NewPRepBase()
	_ = pb.SetRegInfo(ri)
	return pb
}

func newDummyPRepStatus() *PRepStatus {
	ps := NewPRepStatus()
	ps.SetStatus(Active)
	ps.SetDelegated(big.NewInt(rand.Int63n(1000)))
	ps.SetBonded(big.NewInt(rand.Int63n(1000)))
	return ps
}

func newDummyPRep(i int) *PRep {
	owner := newDummyAddress(i)
	pb := newDummyPRepBase(i)
	ps := newDummyPRepStatus()
	return &PRep{
		owner:      owner,
		PRepBase:   pb,
		PRepStatus: ps,
	}
}

func newDummyPReps(size int, br int64) *PReps {
	preps := make([]*PRep, size)
	for i := 0; i < size; i++ {
		preps[i] = newDummyPRep(i)
	}
	return newPReps(preps, br)
}

func newDummyPRepSnapshots(size int) *PRepSnapshots {
	ret := NewEmptyPRepSnapshots()
	tbd := new(big.Int)
	for i := 0; i < size; i++ {
		owner := newDummyAddress(i)
		bd := big.NewInt(int64(size - i))
		ret.append(i, owner, bd)
		tbd.Add(tbd, bd)
	}
	ret.totalBondedDelegation = tbd
	return ret
}

func TestPRepSnapshot_Equal(t *testing.T) {
	owner0 := newDummyAddress(0)
	owner1 := newDummyAddress(1)
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
	owner := newDummyAddress(1)
	delegated := int64(1000)
	bond := delegated / 2
	snapshot := newPRepSnapshot(owner, delegated, bond)
	bs, err := codec.BC.MarshalToBytes(snapshot)
	assert.NoError(t, err)

	var ps2 PRepSnapshot
	_, err = codec.BC.UnmarshalFromBytes(bs, &ps2)
	assert.NoError(t, err)

	assert.True(t, snapshot.Equal(&ps2))
}

func TestPRepSnapshots_Equal(t *testing.T) {
	size := 150
	electedPRepCount := 100
	br := int64(5)
	preps := newDummyPReps(size, br)
	snapshots := NewPRepSnapshots(preps, electedPRepCount, br)

	cases := []struct {
		p0, p1 *PRepSnapshots
		result bool
	}{
		{nil, nil, true},
		{nil, snapshots, false},
		{snapshots, nil, false},
		{snapshots, snapshots, true},
		{snapshots, snapshots.Clone(), true},
	}

	for _, c := range cases {
		assert.Equal(t, c.result, c.p0.Equal(c.p1))
	}
}

func TestPRepSnapshots_NewPRepSnapshots(t *testing.T) {
	br := int64(5)

	type args struct {
		size             int
		electedPRepCount int
	}

	tests := []struct {
		name string
		in   args
	}{
		{
			"size == electedPRepCount",
			args{10, 10},
		},
		{
			"size > electedPRepCount",
			args{11, 10},
		},
		{
			"size < electedPRepCount",
			args{5, 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in

			preps := newDummyPReps(in.size, br)
			snapshots := NewPRepSnapshots(preps, in.electedPRepCount, br)
			count := icutils.Min(in.size, in.electedPRepCount)
			assert.Equal(t, count, snapshots.Len())

			tbd := new(big.Int)
			for i := 0; i < count; i++ {
				tbd.Add(tbd, preps.GetPRepByIndex(i).GetBondedDelegation(br))
			}
			assert.Zero(t, tbd.Cmp(snapshots.TotalBondedDelegation()))
		})
	}
}

func TestPRepSnapshot_RLP(t *testing.T) {
	br := int64(5)
	size := 10
	electedPRepCount := size
	var pss0, pss1 *PRepSnapshots

	preps := newDummyPReps(size, br)
	pss0 = NewPRepSnapshots(preps, electedPRepCount, br)

	bs, err := codec.BC.MarshalToBytes(pss0)
	assert.NoError(t, err)
	assert.True(t, len(bs) > 0)

	_, err = codec.BC.UnmarshalFromBytes(bs, &pss1)
	assert.NoError(t, err)

	assert.True(t, pss0.Equal(pss1))
	assert.Equal(t, size, pss0.Len())
	assert.Equal(t, size, pss1.Len())

	pss0 = NewEmptyPRepSnapshots()
	bs, err = codec.BC.MarshalToBytes(pss0)
	_, err = codec.BC.UnmarshalFromBytes(bs, &pss1)
	assert.True(t, pss0.Equal(pss1))
}

// ============================================================

func TestTerm_Equal(t *testing.T) {
	t0 := NewTerm(0, 10)
	tSequence := t0.Clone()
	tSequence.sequence = t0.sequence + 1
	tSet := tSequence.Clone()
	tSet.Set(t0)
	tSH := t0.Clone()
	tSH.startHeight = t0.startHeight + 1
	tPeriod := t0.Clone()
	tPeriod.period = t0.period + 1
	tIrep := t0.Clone()
	tIrep.irep = new(big.Int).SetInt64(t0.irep.Int64() + 1)
	tRrep := t0.Clone()
	tRrep.rrep = new(big.Int).SetInt64(t0.rrep.Int64() + 1)
	tTS := t0.Clone()
	tTS.totalSupply = new(big.Int).SetInt64(t0.totalSupply.Int64() + 1)
	tTD := t0.Clone()
	tTD.totalDelegated = new(big.Int).SetInt64(t0.totalDelegated.Int64() + 1)
	tSnapshots := t0.Clone()
	tSnapshots.SetPRepSnapshots(newDummyPRepSnapshots(100))

	cases := []struct {
		name   string
		t0, t1 *Term
		result bool
	}{
		{"nil comp", nil, nil, true},
		{"same instance", t0, t0, true},
		{"clone", t0, t0.Clone(), true},
		{"NewTerm() with same param", t0, NewTerm(0, 10), true},
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
			assert.Equal(t, c.result, c.t0.Equal(c.t1), "%v\n%v", c.t0, c.t1)
		})
	}
}

func TestTerm_Bytes(t *testing.T) {
	term := NewTerm(0, 10)

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
	term := NewTerm(0, 43120)
	term2 := term.Clone()
	assert.True(t, term.Equal(term2))

	size := 100
	prepSnapshots := newDummyPRepSnapshots(size)
	term.SetPRepSnapshots(prepSnapshots)
	term2 = term.Clone()
	assert.True(t, term.Equal(term2))
	assert.True(t, term.prepSnapshots.Equal(term2.prepSnapshots))
}

func TestTerm_TotalBondedDelegation(t *testing.T) {
	size := 100
	term := NewTerm(0, 43120)
	prepSnapshots := newDummyPRepSnapshots(size)
	term.SetPRepSnapshots(prepSnapshots.Clone())
	assert.Equal(t, term.GetElectedPRepCount(), prepSnapshots.Len())
	assert.Zero(t, prepSnapshots.TotalBondedDelegation().Cmp(term.TotalBondedDelegation()))

	// GetPRepSnapshot...()
	for i := 0; i < size; i++ {
		ps := term.GetPRepSnapshotByIndex(i)
		assert.Equal(t, ps, prepSnapshots.Get(i))
		assert.Equal(t, ps, term.GetPRepSnapshotByOwner(ps.Owner()))
	}
}
