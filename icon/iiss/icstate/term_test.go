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
	status := NewPRepStatus(owner)
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

func newTestRewardFundV1() *RewardFund {
	rf := NewRewardFund(RFVersion1)
	rf.SetIGlobal(big.NewInt(3_000_000))
	allocation := map[RFundKey]icmodule.Rate{
		KeyIprep:  icmodule.ToRate(13),
		KeyIcps:   icmodule.ToRate(10),
		KeyIrelay: icmodule.ToRate(0),
		KeyIvoter: icmodule.ToRate(77),
	}
	rf.SetAllocation(allocation)
	return rf
}

func newTestRewardFundV2() *RewardFund {
	rf := NewRewardFund(RFVersion2)
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
	var rf *RewardFund
	var mb *big.Int
	if version == termVersion1 {
		rf = newTestRewardFundV1()
	} else {
		rf = newTestRewardFundV2()
		mb = icmodule.BigIntZero
	}
	return &TermState{
		termData: termData{
			version:     version,
			sequence:    sequence,
			period:      period,
			rewardFund:  rf,
			minimumBond: mb,
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
	br := icmodule.ToRate(5)
	cfg := NewPRepCountConfig(22, 78, 3)
	sc := newMockStateContext(map[string]interface{}{
		"bh": int64(1000),
		"rev": icmodule.RevisionBTP2-1,
	})

	preps := newDummyPReps(size)
	prepSet := NewPRepSet(sc, preps, cfg)
	snapshots := prepSet.ToPRepSnapshots(br)

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
	sc := newMockStateContext(nil)

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
			cfg := NewPRepCountConfig(4, in.electedPRepCount-4, 3)

			preps := newDummyPReps(in.size)
			prepSet := NewPRepSet(sc, preps, cfg)
			snapshots := prepSet.ToPRepSnapshots(br)
			count := icutils.Min(in.size, in.electedPRepCount)
			assert.Equal(t, count, len(snapshots))
		})
	}
}

func TestPRepSnapshots_RLP(t *testing.T) {
	br := icmodule.ToRate(5)
	size := 10
	var pss0, pss1 PRepSnapshots

	preps := newDummyPReps(size)
	pss0 = make(PRepSnapshots, size)
	for i, prep := range preps {
		pss0[i] = NewPRepSnapshot(prep.Owner(), prep.GetPower(br))
	}

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
	rf1 := newTestRewardFundV1()
	rf2 := newTestRewardFundV2()
	prepSnapshots := newDummyPRepSnapshots(100)

	for version := termVersion1; version < termVersionReserved; version++ {
		rf := rf1
		var irep, rrep, mb *big.Int
		if version == termVersion1 {
			irep = icmodule.BigIntZero
			rrep = icmodule.BigIntZero
		} else {
			rf = rf2
			mb = icmodule.BigIntZero
		}
		termState := &TermState{
			termData: termData{
				version:         version,
				sequence:        sequence,
				startHeight:     startHeight,
				period:          termPeriod,
				irep:            irep,
				rrep:            rrep,
				totalSupply:     totalSupply,
				totalDelegated:  totalDelegated,
				rewardFund:      rf,
				bondRequirement: br,
				revision:        revision,
				prepSnapshots:   prepSnapshots.Clone(),
				isDecentralized: isDecentralized,
				minimumBond:     mb,
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
			icmodule.RevisionIISS4R0,
			IISSVersion3,
		},
		{
			icmodule.RevisionIISS4R1,
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
