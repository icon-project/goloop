package icstate

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

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

func newDummyPRepSnapshots(size int) PRepSnapshots {
	if size == 0 {
		return nil
	}
	ret := make(PRepSnapshots, size)
	for i := 0; i < size; i++ {
		owner := newDummyAddress(i)
		bd := big.NewInt(int64(size - i))
		ret[i] = NewPRepSnapshot(owner, bd)
	}
	return ret
}

func newTermState(sequence int, period int64) *TermState {
	return &TermState{
		termData: termData{
			sequence:   sequence,
			period:     period,
			rewardFund: NewRewardFund(),
		},
	}
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
	br := icutils.PercentToRate(5)
	preps := newDummyPRepSet(size)
	snapshots := preps.ToPRepSnapshots(electedPRepCount, br)

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
		assert.Equal(t, c.result, c.p0.Equal(c.p1))
	}
}

func TestPRepSnapshots_NewPRepSnapshots(t *testing.T) {
	br := icutils.PercentToRate(5)

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

			preps := newDummyPRepSet(in.size)
			snapshots := preps.ToPRepSnapshots(in.electedPRepCount, br)
			count := icutils.Min(in.size, in.electedPRepCount)
			assert.Equal(t, count, len(snapshots))
		})
	}
}

func TestPRepSnapshot_RLP(t *testing.T) {
	br := icutils.PercentToRate(5)
	size := 10
	electedPRepCount := size
	var pss0, pss1 PRepSnapshots

	preps := newDummyPRepSet(size)
	pss0 = preps.ToPRepSnapshots(electedPRepCount, br)

	bs, err := codec.BC.MarshalToBytes(pss0)
	assert.NoError(t, err)
	assert.True(t, len(bs) > 0)

	_, err = codec.BC.UnmarshalFromBytes(bs, &pss1)
	assert.NoError(t, err)

	assert.True(t, pss0.Equal(pss1))
	assert.Equal(t, size, len(pss0))
	assert.Equal(t, size, len(pss1))

	pss0 = make(PRepSnapshots, 0)
	bs, err = codec.BC.MarshalToBytes(pss0)
	_, err = codec.BC.UnmarshalFromBytes(bs, &pss1)
	assert.True(t, pss0.Equal(pss1))
}

func TestTerm_Bytes(t *testing.T) {
	term := newTermState(0, 10)

	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	o1 := icobject.New(TypeTerm, term.GetSnapshot())
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())
	assert.True(t, o1.Equal(o2))
}

func TestTerm_TotalBondedDelegation(t *testing.T) {
	size := 100
	term := newTermState(0, 43120)
	prepSnapshots := newDummyPRepSnapshots(size)
	term.SetPRepSnapshots(prepSnapshots.Clone())
	assert.Equal(t, term.GetElectedPRepCount(), len(prepSnapshots))

	totalPower := new(big.Int)
	for _, snapshot := range prepSnapshots {
		totalPower.Add(totalPower, snapshot.BondedDelegation())
	}
	assert.Zero(t, totalPower.Cmp(term.getTotalPower()))

	for i := 0; i < size; i++ {
		ps := term.GetPRepSnapshotByIndex(i)
		assert.Equal(t, ps, prepSnapshots[i])
	}
}
