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
	"bytes"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
)

func newDummyAddress(value int) module.Address {
	bs := make([]byte, common.AddressBytes)
	for i := 0; value != 0 && i < 8; i++ {
		bs[common.AddressBytes-1-i] = byte(value & 0xFF)
		value >>= 8
	}
	return common.MustNewAddress(bs)
}

func newDummyAddresses(size int) []module.Address {
	addrs := make([]module.Address, size)
	for i := 0; i < size; i++ {
		addrs[i] = newDummyAddress(i + 1)
	}
	return addrs
}

func newDummyPRepBase(i int) *PRepBaseState {
	info := newDummyPRepInfo(i)
	pb := NewPRepBaseState()
	pb.UpdateInfo(info)
	return pb
}

func newDummyPRepStatus(value int) *PRepStatusState {
	owner := newDummyAddress(value)
	ps := NewPRepStatus(owner)
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

func newDummyPReps(size int) []*PRep {
	preps := make([]*PRep, size)
	for i := 0; i < size; i++ {
		preps[i] = newDummyPRep(i + 1)
	}
	return preps
}

func newDummyPRepSet(size int) PRepSet {
	preps := make([]*PRep, size)
	for i := 0; i < size; i++ {
		preps[i] = newDummyPRep(i)
	}
	return NewPRepSet(preps)
}

func TestPRep_IsElectable(t *testing.T) {
	br := icmodule.ToRate(5)
	activeDSAMask := int64(3)
	sc := newMockStateContext(map[string]interface{}{
		"blockHeight": int64(1000),
		"revision": icmodule.RevisionIISS4,
	})

	args := []struct {
		status      Status
		bonded      *big.Int
		dsaMask     int64
		pt          icmodule.PenaltyType
		isElectable bool
	}{
		{Active, icmodule.BigIntZero, int64(3), icmodule.PenaltyNone, false},
		{Active, icmodule.BigIntZero, int64(3), icmodule.PenaltyNone,false},
		{Active, icmodule.BigIntZero, int64(3), icmodule.PenaltyNone, false},
		{Active, big.NewInt(100), int64(0), icmodule.PenaltyNone, false},
		{Active, big.NewInt(100), int64(1), icmodule.PenaltyNone, false},
		{Active, big.NewInt(100), int64(3), icmodule.PenaltyNone, true},
		{Active, big.NewInt(100), int64(3), icmodule.PenaltyAccumulatedValidationFailure, false},
		{Active, big.NewInt(100), int64(3), icmodule.PenaltyValidationFailure, false},
		{Active, big.NewInt(100), int64(3), icmodule.PenaltyDoubleVote, false},
		{Unregistered, big.NewInt(100), int64(3), icmodule.PenaltyNone, false},
		{Disqualified, big.NewInt(100), int64(3), icmodule.PenaltyNone, false},
	}

	for i, arg := range args {
		prep := newDummyPRep(1)
		if arg.status == Unregistered || arg.status == Disqualified {
			_, err := prep.DisableAs(arg.status)
			assert.NoError(t, err)
		}
		prep.SetBonded(arg.bonded)
		prep.SetDSAMask(arg.dsaMask)
		name := fmt.Sprintf("name-%02d", i)

		if arg.pt != icmodule.PenaltyNone {
			err := prep.NotifyEvent(sc, icmodule.PRepEventImposePenalty, arg.pt)
			assert.NoError(t, err)
		}
		assert.Zero(t, arg.bonded.Cmp(prep.Bonded()))
		assert.Equal(t, arg.dsaMask, prep.GetDSAMask())

		t.Run(name, func(t *testing.T) {
			assert.Equal(t, arg.isElectable, prep.IsElectable(br, activeDSAMask))
		})
	}
}

/*
func TestPRepSet_Sort_OnTermEnd(t *testing.T) {
	br := icmodule.ToRate(5)
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
	prepSetEntries := []*PRep{
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
		sc         icmodule.StateContext
		main       int
		sub        int
		extra      int
		expect     []*PRep
		expectMain int
		expectSub  int
	}{
		{
			"Sort by power",
			NewStateContext(1000, icmodule.RevisionResetPenaltyMask, icmodule.RevisionResetPenaltyMask),
			1, 2, 0,
			[]*PRep{prep4, prep5, prep3, prep2, prep1, prep6},
			1, 2,
		},
		{
			"Sort by power",
			NewStateContext(1000, icmodule.RevisionEnableIISS3, icmodule.RevisionEnableIISS3),
			1, 2, 0,
			[]*PRep{prep4, prep5, prep3, prep2, prep1, prep6},
			1, 2,
		},
		{
			"Sort by power + extra main prep",
			NewStateContext(1000, icmodule.RevisionExtraMainPReps, icmodule.RevisionExtraMainPReps),
			1, 2, 1,
			[]*PRep{prep4, prep3, prep5, prep2, prep1, prep6},
			2, 1,
		},
		{
			"Sort by power + extra main prep with zero count",
			NewStateContext(1000, icmodule.RevisionExtraMainPReps, icmodule.RevisionExtraMainPReps),
			1, 2, 0,
			[]*PRep{prep4, prep5, prep3, prep2, prep1, prep6},
			1, 2,
		},
		{
			"Sort by power + pubKey + extra main prep",
			NewStateContext(1000, icmodule.RevisionBTP2, icmodule.RevisionBTP2),
			1, 2, 1,
			[]*PRep{prep5, prep1, prep6, prep4, prep3, prep2},
			2, 0,
		},
		{
			"Sort by power + pubKey + extra main prep with zero count",
			NewStateContext(1000, icmodule.RevisionBTP2, icmodule.RevisionBTP2),
			1, 2, 0,
			[]*PRep{prep5, prep1, prep6, prep4, prep3, prep2},
			1, 1,
		},
		{
			"Too big sub prep, extra main prep",
			NewStateContext(1000, icmodule.RevisionBTP2, icmodule.RevisionBTP2),
			1, 6, 10,
			[]*PRep{prep5, prep1, prep6, prep4, prep3, prep2},
			2, 0,
		},
		{
			"Too big sub prep, extra main prep with zero main prep",
			NewStateContext(1000, icmodule.RevisionBTP2, icmodule.RevisionBTP2),
			0, 6, 10,
			[]*PRep{prep1, prep5, prep6, prep4, prep3, prep2},
			2, 0,
		},
	}

	for _, tt := range tests {
		rev := tt.sc.Revision()
		t.Run(fmt.Sprintf("%s rev=%d", tt.name, rev), func(t *testing.T) {
			prepSet.Sort(tt.main, tt.sub, tt.extra, br, rev)
			err := prepSet.OnTermEnd(tt.sc, tt.main, tt.sub, tt.extra, 0, br)
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
					if rev >= icmodule.RevisionBTP2 &&
						(aEntry.HasPubKey() == false || aEntry.Power(br).Sign() == 0) {
						assert.Equal(t, GradeCandidate, aEntry.PRep().Grade())
					} else {
						assert.Equal(t, GradeMain, aEntry.PRep().Grade())
					}
				case j < tt.expectMain+tt.expectSub:
					if rev >= icmodule.RevisionBTP2 &&
						(aEntry.HasPubKey() == false || aEntry.Power(br).Sign() == 0) {
						assert.Equal(t, GradeCandidate, aEntry.PRep().Grade())
					} else {
						assert.Equal(t, GradeSub, aEntry.PRep().Grade())
					}
				default:
					assert.Equal(t, GradeCandidate, aEntry.PRep().Grade())
				}

				if rev == icmodule.RevisionResetPenaltyMask {
					assert.Zero(t, aEntry.PRep().GetVPenaltyCount())
				}
			}
		})
	}
}
*/

func TestPRepSet_SortForQuery(t *testing.T) {
	br := icmodule.ToRate(5)
	preps := newDummyPReps(6)
	prepSet := NewPRepSet(preps)

	for _, prep := range preps {
		dsaMask := rand.Int63n(4)
		prep.SetDSAMask(dsaMask)
	}

	args := []struct {
		rev           int
		activeDSAMask int64
	}{
		{icmodule.RevisionBTP2 - 1, 0},
		{icmodule.RevisionBTP2, 1},
		{icmodule.RevisionBTP2, 3},
	}

	for _, arg := range args {
		name := fmt.Sprintf("rev=%d", arg.rev)
		activeDSAMask := arg.activeDSAMask
		sc := newMockStateContext(map[string]interface{}{
			"blockHeight":     int64(1000),
			"revision":        arg.rev,
			"activeDSAMask":   activeDSAMask,
			"bondRequirement": br,
		})

		t.Run(name, func(t *testing.T) {
			prepSet.SortForQuery(sc)

			for i := 1; i < prepSet.Size(); i++ {
				p0 := prepSet.GetByIndex(i - 1)
				p1 := prepSet.GetByIndex(i)
				assert.True(t, checkPRepOrder(sc, p0, p1))
			}
		})
	}
}

func checkPRepOrder(sc icmodule.StateContext, p0, p1 *PRep) bool {
	rev := sc.Revision()
	br := sc.GetBondRequirement()
	activeDSAMask := sc.GetActiveDSAMask()

	if rev >= icmodule.RevisionBTP2 {
		if p0.HasPubKey(activeDSAMask) != p1.HasPubKey(activeDSAMask) {
			return p0.HasPubKey(activeDSAMask)
		}
		if p0.IsJailInfoElectable() != p1.IsJailInfoElectable() {
			return p0.IsJailInfoElectable()
		}
	}

	if cmp := p0.GetPower(br).Cmp(p1.GetPower(br)); cmp != 0 {
		return cmp > 0
	}
	if cmp := p0.Delegated().Cmp(p1.Delegated()); cmp != 0 {
		return cmp > 0
	}
	return bytes.Compare(p0.Owner().Bytes(), p1.Owner().Bytes()) > 0
}
