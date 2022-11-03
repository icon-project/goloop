/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package icstate

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
)

func getRandomVoteState() VoteState {
	return []VoteState{None, Success, Failure}[rand.Intn(3)]
}

func newDummyAddress(value int) module.Address {
	bs := make([]byte, common.AddressBytes)
	for i := 0; value != 0 && i < 8; i++ {
		bs[common.AddressBytes-1-i] = byte(value & 0xFF)
		value >>= 8
	}
	return common.MustNewAddress(bs)
}

func newDummyPRepBase(i int) *PRepBaseState {
	info := newDummyPRepInfo(i)
	pb := NewPRepBaseState()
	pb.UpdateInfo(info)
	return pb
}

func newDummyPRepStatus(value int) *PRepStatusState {
	ps := NewPRepStatus()
	_ = ps.Activate()
	ps.SetDelegated(big.NewInt(rand.Int63n(1000) + 1))
	ps.SetBonded(big.NewInt(rand.Int63n(1000) + 1))
	return ps
}

func newDummyPRep(i int) *PRep {
	owner := newDummyAddress(i)
	pb := newDummyPRepBase(i)
	ps := newDummyPRepStatus(i)
	return &PRep{
		owner:           owner,
		pb:              pb,
		PRepStatusState: ps,
	}
}

type dummyPRepSetEntry struct {
	prep      *PRep
	status    Status
	grade     Grade
	power     *big.Int
	delegated *big.Int
	bonded    *big.Int
	owner     module.Address
	pubKey    bool
}

func (d *dummyPRepSetEntry) PRep() *PRep {
	return d.prep
}

func (d *dummyPRepSetEntry) Status() Status {
	return Active
}

func (d *dummyPRepSetEntry) Grade() Grade {
	return d.grade
}

func (d *dummyPRepSetEntry) Power(_ int64) *big.Int {
	return d.power
}

func (d *dummyPRepSetEntry) Delegated() *big.Int {
	return d.delegated
}

func (d *dummyPRepSetEntry) Bonded() *big.Int {
	return d.bonded
}

func (d *dummyPRepSetEntry) Owner() module.Address {
	return d.prep.Owner()
}

func (d *dummyPRepSetEntry) HasPubKey() bool {
	return d.pubKey
}

func newDummyPRepSetEntry(
	prep *PRep, grade Grade, power, delegated, bonded int, pubKey bool,
) *dummyPRepSetEntry {
	return &dummyPRepSetEntry{
		prep:      prep,
		grade:     grade,
		power:     big.NewInt(int64(power)),
		delegated: big.NewInt(int64(delegated)),
		bonded:    big.NewInt(int64(bonded)),
		pubKey:    pubKey,
	}
}

func newDummyPRepSet(size int) PRepSet {
	prepSetEntries := make([]PRepSetEntry, size)
	for i := 0; i < size; i++ {
		prepSetEntries[i] = NewPRepSetEntry(newDummyPRep(i), false)
	}
	prepSet := NewPRepSet(prepSetEntries)
	return prepSet
}

func TestPRepSet_Sort_OnTermEnd(t *testing.T) {
	br := int64(5)
	prep1 := newDummyPRep(1)
	prep1.lastState = None
	prep1.vPenaltyMask = (rand.Uint32() & uint32(0x3FFFFFFF)) | uint32(1)
	prep2 := newDummyPRep(2)
	prep2.lastState = None
	prep3 := newDummyPRep(3)
	prep3.lastState = None
	prep3.lastHeight = 1
	prep4 := newDummyPRep(4)
	prep4.lastState = Success
	prep5 := newDummyPRep(5)
	prep5.lastState = None
	prep5.lastHeight = 2
	prep6 := newDummyPRep(6)
	prepSetEntries := []PRepSetEntry{
		&dummyPRepSetEntry{
			prep:      prep1,
			grade:     GradeMain,
			power:     big.NewInt(1),
			delegated: big.NewInt(1),
			bonded:    big.NewInt(1),
			pubKey:    true,
		},
		&dummyPRepSetEntry{
			prep:      prep2,
			grade:     GradeSub,
			power:     big.NewInt(2),
			delegated: big.NewInt(2),
			bonded:    big.NewInt(2),
			pubKey:    false,
		},
		&dummyPRepSetEntry{
			prep:      prep3,
			grade:     GradeSub,
			power:     big.NewInt(3),
			delegated: big.NewInt(3),
			bonded:    big.NewInt(3),
			pubKey:    false,
		},
		&dummyPRepSetEntry{
			prep:      prep4,
			grade:     GradeMain,
			power:     big.NewInt(4),
			delegated: big.NewInt(4),
			bonded:    big.NewInt(4),
			pubKey:    false,
		},
		&dummyPRepSetEntry{
			prep:      prep5,
			grade:     GradeCandidate,
			power:     big.NewInt(3),
			delegated: big.NewInt(3),
			bonded:    big.NewInt(3),
			pubKey:    true,
		},
		&dummyPRepSetEntry{
			prep:      prep6,
			grade:     GradeCandidate,
			power:     big.NewInt(0),
			delegated: big.NewInt(0),
			bonded:    big.NewInt(0),
			pubKey:    true,
		},
	}

	prepSet := NewPRepSet(prepSetEntries)
	assert.Equal(t, 2, prepSet.GetPRepSize(GradeMain))
	assert.Equal(t, 2, prepSet.GetPRepSize(GradeSub))
	assert.Equal(t, 2, prepSet.GetPRepSize(GradeCandidate))
	assert.Equal(t, 6, prepSet.Size())
	assert.Equal(t, big.NewInt(13), prepSet.TotalBonded())
	assert.Equal(t, big.NewInt(13), prepSet.TotalDelegated())
	assert.Equal(t, big.NewInt(13), prepSet.GetTotalPower(br))

	tests := []struct {
		name       string
		rev        int
		main       int
		sub        int
		extra      int
		expect     []*PRep
		expectMain int
		expectSub  int
	}{
		{
			"Sort by power",
			icmodule.RevisionResetPenaltyMask,
			1, 2, 0,
			[]*PRep{prep4, prep5, prep3, prep2, prep1, prep6},
			1, 2,
		},
		{
			"Sort by power",
			icmodule.RevisionEnableIISS3,
			1, 2, 0,
			[]*PRep{prep4, prep5, prep3, prep2, prep1, prep6},
			1, 2,
		},
		{
			"Sort by power + extra main prep",
			icmodule.RevisionExtraMainPReps,
			1, 2, 1,
			[]*PRep{prep4, prep3, prep5, prep2, prep1, prep6},
			2, 1,
		},
		{
			"Sort by power + extra main prep with zero count",
			icmodule.RevisionExtraMainPReps,
			1, 2, 0,
			[]*PRep{prep4, prep5, prep3, prep2, prep1, prep6},
			1, 2,
		},
		{
			"Sort by power + pubKey + extra main prep",
			icmodule.RevisionBTP2,
			1, 2, 1,
			[]*PRep{prep5, prep1, prep6, prep4, prep3, prep2},
			2, 0,
		},
		{
			"Sort by power + pubKey + extra main prep with zero count",
			icmodule.RevisionBTP2,
			1, 2, 0,
			[]*PRep{prep5, prep1, prep6, prep4, prep3, prep2},
			1, 1,
		},
		{
			"Too big sub prep, extra main prep",
			icmodule.RevisionBTP2,
			1, 6, 10,
			[]*PRep{prep5, prep1, prep6, prep4, prep3, prep2},
			2, 0,
		},
		{
			"Too big sub prep, extra main prep with zero main prep",
			icmodule.RevisionBTP2,
			0, 6, 10,
			[]*PRep{prep1, prep5, prep6, prep4, prep3, prep2},
			2, 0,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s rev=%d", tt.name, tt.rev), func(t *testing.T) {
			prepSet.Sort(tt.main, tt.sub, tt.extra, br, tt.rev)
			err := prepSet.OnTermEnd(tt.rev, tt.main, tt.sub, tt.extra, 0, br)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectMain, prepSet.GetPRepSize(GradeMain))
			assert.Equal(t, tt.expectSub, prepSet.GetPRepSize(GradeSub))

			for j := 0; j < prepSet.Size(); j++ {
				// check sort order
				aEntry := prepSet.GetByIndex(j)
				ePRep := tt.expect[j]
				ao := aEntry.Owner()
				eo := ePRep.Owner()
				assert.True(t, ao.Equal(eo), fmt.Sprintf("e:%s a:%s", eo, ao))

				// check grade of P-Rep
				switch {
				case j < tt.expectMain:
					if tt.rev >= icmodule.RevisionBTP2 &&
						(aEntry.HasPubKey() == false || aEntry.Power(br).Sign() == 0) {
						assert.Equal(t, GradeCandidate, aEntry.PRep().Grade())
					} else {
						assert.Equal(t, GradeMain, aEntry.PRep().Grade())
					}
				case j < tt.expectMain+tt.expectSub:
					if tt.rev >= icmodule.RevisionBTP2 &&
						(aEntry.HasPubKey() == false || aEntry.Power(br).Sign() == 0) {
						assert.Equal(t, GradeCandidate, aEntry.PRep().Grade())
					} else {
						assert.Equal(t, GradeSub, aEntry.PRep().Grade())
					}
				default:
					assert.Equal(t, GradeCandidate, aEntry.PRep().Grade())
				}

				if tt.rev == icmodule.RevisionResetPenaltyMask {
					assert.Zero(t, aEntry.PRep().GetVPenaltyCount())
				}
			}
		})
	}
}

func TestPRepSet_SortByGrade(t *testing.T) {
	br := int64(5)
	prep1 := newDummyPRep(1)
	prep2 := newDummyPRep(2)
	prep3 := newDummyPRep(3)
	prep4 := newDummyPRep(4)
	prep5 := newDummyPRep(5)
	prepSetEntries := []PRepSetEntry{
		&dummyPRepSetEntry{
			prep:      prep1,
			grade:     GradeMain,
			power:     big.NewInt(1),
			delegated: big.NewInt(1),
			bonded:    big.NewInt(1),
		},
		&dummyPRepSetEntry{
			prep:      prep2,
			grade:     GradeSub,
			power:     big.NewInt(2),
			delegated: big.NewInt(2),
			bonded:    big.NewInt(2),
		},
		&dummyPRepSetEntry{
			prep:      prep3,
			grade:     GradeSub,
			power:     big.NewInt(3),
			delegated: big.NewInt(3),
			bonded:    big.NewInt(3),
		},
		&dummyPRepSetEntry{
			prep:      prep4,
			grade:     GradeMain,
			power:     big.NewInt(4),
			delegated: big.NewInt(4),
			bonded:    big.NewInt(4),
		},
		&dummyPRepSetEntry{
			prep:      prep5,
			grade:     GradeCandidate,
			power:     big.NewInt(3),
			delegated: big.NewInt(3),
			bonded:    big.NewInt(3),
		},
	}
	expect := []*PRep{prep4, prep1, prep3, prep2, prep5}

	prepSet := NewPRepSet(prepSetEntries)

	prepSet.SortByGrade(br)
	for j := 0; j < prepSet.Size(); j++ {
		rEntry := prepSet.GetByIndex(j)
		ePRep := expect[j]
		ro := rEntry.Owner()
		eo := ePRep.Owner()
		assert.True(t, ro.Equal(eo), fmt.Sprintf("e:%s r:%s", eo, ro))
	}
}
