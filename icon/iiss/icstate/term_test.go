package icstate

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/icmodule"
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

func newTestRewardFund() *RewardFund {
	return &RewardFund{
		Iglobal: icutils.ToLoop(3_000_000),
		Icps:    icmodule.ToRate(10),
		Iprep:   icmodule.ToRate(13),
		Ivoter:  icmodule.ToRate(77),
		Irelay:  icmodule.ToRate(0),
	}
}

func newTestRewardFund2() *RewardFund2 {
	rf := NewRewardFund2()
	rf.SetIGlobal(big.NewInt(1_000_000))
	allocation := map[RFundKey]icmodule.Rate{
		KeyIprep:  icmodule.ToRate(77),
		KeyIwage:  icmodule.ToRate(10),
		KeyIcps:   icmodule.ToRate(10),
		KeyIrelay: icmodule.ToRate(3),
	}
	rf.SetAllocation(allocation)
	return rf
}

func newTermState(version, sequence int, period int64) *TermState {
	rf := NewRewardFund()
	rf2 := NewRewardFund2()
	if version == termVersion1 {
		rf = newTestRewardFund()
	} else {
		rf2 = newTestRewardFund2()
	}
	return &TermState{
		termData: termData{
			version:     version,
			sequence:    sequence,
			period:      period,
			rewardFund:  rf,
			rewardFund2: rf2,
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
	br := icmodule.ToRate(5)
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
	br := icmodule.ToRate(5)

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
	br := icmodule.ToRate(5)
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
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)

	tests := []struct {
		name      string
		termState *TermState
	}{
		{
			"Version1",
			newTermState(termVersion1, 0, 10),
		},
		{
			"Version2",
			newTermState(termVersion2, 0, 10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o1 := icobject.New(TypeTerm, tt.termState.GetSnapshot())
			serialized := o1.Bytes()

			o2 := new(icobject.Object)
			if err := o2.Reset(database, serialized); err != nil {
				t.Errorf("Failed to get object from bytes")
				return
			}

			assert.Equal(t, serialized, o2.Bytes())
			assert.True(t, o1.Equal(o2))
		})
	}
}

func TestTerm_TotalBondedDelegation(t *testing.T) {
	size := 100
	term := newTermState(termVersion1, 0, 43120)
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

func TestTermSnapshot_RLPDecodeFields(t *testing.T) {
	const (
		sequence        = 1
		startHeight     = int64(100)
		termPeriod      = icmodule.DecentralizedTermPeriod
		br              = icmodule.Rate(1000)
		revision        = icmodule.RevisionBTP2
		isDecentralized = true
	)

	totalSupply := icutils.ToLoop(10_000_000)
	totalDelegated := icutils.ToLoop(1_000_000)
	rf := newTestRewardFund()
	rf2 := newTestRewardFund2()
	prepSnapshots := newDummyPRepSnapshots(100)

	termState := &TermState{
		termData: termData{
			sequence:        sequence,
			startHeight:     startHeight,
			period:          termPeriod,
			irep:            icmodule.BigIntZero,
			rrep:            icmodule.BigIntZero,
			totalSupply:     totalSupply,
			totalDelegated:  totalDelegated,
			rewardFund:      rf.Clone(),
			rewardFund2:     rf2,
			bondRequirement: br,
			revision:        revision,
			prepSnapshots:   prepSnapshots.Clone(),
			isDecentralized: isDecentralized,
		},
	}

	termSnapshot := termState.GetSnapshot()
	termObject := icobject.New(TypeTerm, termSnapshot)

	buf := bytes.NewBuffer(nil)
	e := codec.BC.NewEncoder(buf)

	assert.NoError(t, e.Encode(termObject))
	assert.NoError(t, e.Close())

	bs := buf.Bytes()
	termObject2 := &icobject.Object{}
	d := codec.BC.NewDecoder(bytes.NewReader(bs))
	assert.NoError(t, termObject2.RLPDecodeSelf(d, NewObjectImpl))

	termSnapshot2 := ToTerm(termObject2)
	assert.True(t, termObject.Equal(termObject2))
	assert.True(t, termSnapshot.Equal(termSnapshot2))
	assert.Equal(t, br, termSnapshot2.BondRequirement())
}

func TestTermData_Iglobal(t *testing.T) {
	rf := newTestRewardFund()
	rf2 := newTestRewardFund2()
	tests := []struct {
		version int
		want    *big.Int
	}{
		{
			termVersion1,
			rf.Iglobal,
		},
		{
			termVersion2,
			rf2.IGlobal(),
		},
	}

	term := termData{
		rewardFund:  newTestRewardFund(),
		rewardFund2: newTestRewardFund2(),
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("Version : %d", tt.version), func(t *testing.T) {
			term.version = tt.version
			assert.Equal(t, tt.want, term.Iglobal())
			assert.Equal(t, tt.version, term.Version())
		})
	}
}

func TestTermData_GetIISSVersion(t *testing.T) {
	tests := []struct {
		revision int
		want     int
	}{
		{
			icmodule.RevisionIISS,
			IISSVersion2,
		},
		{
			icmodule.Revision13,
			IISSVersion2,
		},
		{
			icmodule.RevisionEnableIISS3,
			IISSVersion3,
		},
		{
			icmodule.RevisionPreIISS4,
			IISSVersion3,
		},
		{
			icmodule.RevisionIISS4,
			IISSVersion4,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Revision%d", tt.revision), func(t *testing.T) {
			term := termData{revision: tt.revision}
			assert.Equal(t, tt.want, term.GetIISSVersion())
			assert.Equal(t, tt.revision, term.Revision())
		})
	}
}
