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
	"github.com/icon-project/goloop/icon/iiss/icutils"
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

func TestPRep_IRep(t *testing.T) {
	prep := newDummyPRep(1)
	assert.Zero(t, prep.IRep().Sign())
}

func TestPRep_NodeAddress(t *testing.T) {
	owner := newDummyAddress(1)
	prep := newDummyPRep(1)
	assert.True(t, owner.Equal(prep.NodeAddress()))

	newOwner := newDummyAddress(100)
	assert.False(t, owner.Equal(newOwner))

	prepInfo := &PRepInfo{
		Node: newOwner,
	}
	prep.getPRepBaseState().UpdateInfo(prepInfo)
	assert.True(t, newOwner.Equal(prep.NodeAddress()))
}

func TestPRep_ToJSON(t *testing.T) {
	bh := int64(1000)
	sc := newMockStateContext(map[string]interface{}{
		"blockHeight":   bh,
		"revision":      icmodule.RevisionIISS4R1,
		"activeDSAMask": int64(1),
	})
	br := sc.GetBondRequirement()

	newOwner := newDummyAddress(100)
	prep := newDummyPRep(1)
	prepInfo := &PRepInfo{
		Node: newOwner,
	}
	pb := prep.getPRepBaseState()
	pb.UpdateInfo(prepInfo)
	info := prep.Info()

	jso := prep.ToJSON(sc)
	assert.True(t, prep.Owner().Equal(jso["address"].(module.Address)))
	assert.True(t, prep.NodeAddress().Equal(jso["nodeAddress"].(module.Address)))
	assert.Equal(t, *info.City, jso["city"])
	assert.Equal(t, *info.Country, jso["country"])
	assert.Equal(t, *info.Details, jso["details"])
	assert.Equal(t, *info.Email, jso["email"])
	assert.Equal(t, *info.Name, jso["name"])
	assert.Equal(t, *info.WebSite, jso["website"])
	assert.Equal(t, prep.LastHeight(), jso["lastHeight"])
	assert.Equal(t, int(prep.Grade()), jso["grade"])
	assert.Equal(t, int(prep.Status()), jso["status"])
	assert.Equal(t, int(prep.getPenaltyType(sc)), jso["penalty"])
	assert.Zero(t, prep.Bonded().Cmp(jso["bonded"].(*big.Int)))
	assert.Zero(t, prep.Delegated().Cmp(jso["delegated"].(*big.Int)))
	assert.Zero(t, prep.GetPower(br).Cmp(jso["power"].(*big.Int)))
	assert.Equal(t, prep.GetVTotal(bh), jso["totalBlocks"])
	assert.Equal(t, prep.GetVTotal(bh)-prep.GetVFail(bh), jso["validatedBlocks"])
	assert.Equal(t, prep.HasPubKey(sc.GetActiveDSAMask()), jso["hasPublicKey"].(bool))
	assert.Equal(t, prep.JailFlags(), jso["jailFlags"].(int))
	assert.Equal(t, prep.UnjailRequestHeight(), jso["unjailRequestHeight"].(int64))
	assert.Equal(t, prep.MinDoubleSignHeight(), jso["minDoubleSignHeight"].(int64))
}

func TestPRep_IsElectable(t *testing.T) {
	br := icmodule.ToRate(5)
	activeDSAMask := int64(3)
	sc := newMockStateContext(map[string]interface{}{
		"blockHeight":     int64(1000),
		"revision":        icmodule.RevisionIISS4R1,
		"bondRequirement": br,
		"activeDSAMask":   activeDSAMask,
	})

	args := []struct {
		status      Status
		bonded      *big.Int
		dsaMask     int64
		pt          icmodule.PenaltyType
		isElectable bool
	}{
		{Active, icmodule.BigIntZero, int64(3), icmodule.PenaltyNone, false},
		{Active, icmodule.BigIntZero, int64(3), icmodule.PenaltyNone, false},
		{Active, icmodule.BigIntZero, int64(3), icmodule.PenaltyNone, false},
		{Active, big.NewInt(100), int64(0), icmodule.PenaltyNone, false},
		{Active, big.NewInt(100), int64(1), icmodule.PenaltyNone, false},
		{Active, big.NewInt(100), int64(3), icmodule.PenaltyNone, true},
		{Active, big.NewInt(100), int64(3), icmodule.PenaltyAccumulatedValidationFailure, false},
		{Active, big.NewInt(100), int64(3), icmodule.PenaltyValidationFailure, false},
		{Active, big.NewInt(100), int64(3), icmodule.PenaltyDoubleSign, false},
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
			err := prep.OnEvent(sc, icmodule.PRepEventImposePenalty, arg.pt)
			assert.NoError(t, err)
		}
		assert.Zero(t, arg.bonded.Cmp(prep.Bonded()))
		assert.Equal(t, arg.dsaMask, prep.GetDSAMask())

		t.Run(name, func(t *testing.T) {
			assert.Equal(t, arg.isElectable, prep.IsElectable(sc))
		})
	}
}

func TestPRepSet_Sort_OnTermEnd(t *testing.T) {
	const (
		mainPReps      = 22
		extraMainPReps = 3
		subPReps       = 78
		totalPReps     = 110
		activeDSAMask  = int64(3)
		limit          = 30
	)
	var err error
	cfg := NewPRepCountConfig(mainPReps, subPReps, extraMainPReps)

	args := []struct {
		rev int
	}{
		{icmodule.RevisionExtraMainPReps},
		{icmodule.RevisionBTP2},
		{icmodule.RevisionIISS4R0},
		{icmodule.RevisionIISS4R1},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			sc := newMockStateContext(map[string]interface{}{
				"blockHeight":     int64(1000),
				"revision":        arg.rev,
				"activeDSAMask":   activeDSAMask,
				"bondRequirement": icmodule.ToRate(5),
			})

			// Initialize PRepSet
			preps := newDummyPReps(totalPReps)
			for _, prep := range preps {
				dsaMask := rand.Int63n(activeDSAMask + 1)
				prep.SetDSAMask(dsaMask)
			}

			prepSet := NewPRepSet(sc, preps, cfg)
			err = prepSet.OnTermEnd(sc, limit)
			assert.NoError(t, err)

			sc.IncreaseBlockHeightBy(50)
			prep0 := prepSet.GetByIndex(0)
			err = prep0.OnEvent(sc, icmodule.PRepEventImposePenalty, icmodule.PenaltyValidationFailure)
			assert.NoError(t, err)

			prepWithNoPower := prepSet.GetByIndex(1)
			prepWithNoPower.SetBonded(icmodule.BigIntZero)
			prepWithNoPower.SetDelegated(big.NewInt(1000))

			sc.IncreaseBlockHeightBy(50)
			prepSet = NewPRepSet(sc, preps, cfg)
			err = prepSet.OnTermEnd(sc, limit)

			assert.NoError(t, err)
			mainPRepSize := prepSet.GetPRepSize(GradeMain)
			subPRepSize := prepSet.GetPRepSize(GradeSub)
			candidateSize := prepSet.GetPRepSize(GradeCandidate)
			assert.True(t, mainPRepSize <= mainPReps+extraMainPReps)
			assert.True(t, subPRepSize <= subPReps)
			assert.Equal(t, totalPReps, mainPRepSize+subPRepSize+candidateSize)

			var prevPower *big.Int
			electedPRepSize := mainPRepSize + subPRepSize
			for j := 0; j < totalPReps; j++ {
				prep := prepSet.GetByIndex(j)
				grade := prep.Grade()
				power := prep.GetPower(sc.GetBondRequirement())

				if j < electedPRepSize {
					if j < mainPRepSize {
						assert.Equal(t, GradeMain, grade)
					} else if j < electedPRepSize {
						assert.Equal(t, GradeSub, grade)
					}

					assert.True(t, prep.IsElectable(sc))
					if sc.Revision() >= icmodule.RevisionExtraMainPReps {
						assert.True(t, power.Sign() > 0)
						if j > 0 {
							assert.True(t, power.Cmp(prevPower) <= 0)
						}
					}
					if sc.Revision() >= icmodule.RevisionBTP2 {
						assert.True(t, prep.HasPubKey(activeDSAMask))
					}
					prevPower = power
				} else {
					assert.Equal(t, GradeCandidate, grade)
				}

				if prep.Owner().Equal(prep0.Owner()) {
					expGrade := GradeMain
					rev := sc.Revision()
					if rev >= icmodule.RevisionIISS4R1 {
						expGrade = GradeCandidate
					}
					assert.Equal(t, expGrade, grade)
					assert.Equal(t, expGrade == GradeMain, prep.IsElectable(sc))
				} else if prep.Owner().Equal(prepWithNoPower.Owner()) {
					assert.Equal(t, GradeCandidate, grade)
				}
			}
		})
	}
}

func TestPrepSetImpl_OnTermEnd(t *testing.T) {
	const (
		totalPReps    = 110
		limit         = 30
		activeDSAMask = int64(3)
	)

	noPubKeys := []int{0, 10}
	noBonds := []int{10, 20}
	inJails := []int{20, 30}

	var err error
	sc := newMockStateContext(map[string]interface{}{
		"blockHeight":     int64(1000),
		"revision":        icmodule.RevisionIISS4R1,
		"activeDSAMask":   activeDSAMask,
		"bondRequirement": icmodule.Rate(5),
	})
	cfg := NewPRepCountConfig(22, 78, 3)

	preps := newDummyPReps(totalPReps)
	for i, prep := range preps {
		if i >= noPubKeys[0] && i < noPubKeys[1] {
			prep.SetDSAMask(0)
		} else {
			prep.SetDSAMask(activeDSAMask)
		}

		if i >= noBonds[0] && i < noBonds[1] {
			prep.SetBonded(icmodule.BigIntZero)
		}

		if i >= inJails[0] && i < inJails[1] {
			err = prep.OnEvent(sc, icmodule.PRepEventImposePenalty, icmodule.PenaltyValidationFailure)
			assert.NoError(t, err)
			assert.True(t, prep.IsInJail())
		}
	}

	electables := 0
	for _, prep := range preps {
		if prep.IsElectable(sc) {
			electables++
		}
	}

	prepSet := NewPRepSet(sc, preps, cfg)
	err = prepSet.OnTermEnd(sc, limit)
	assert.NoError(t, err)
	assert.Equal(t, len(preps), prepSet.Size())

	electables2 := 0
	for _, prep := range preps {
		if prep.IsElectable(sc) {
			electables2++
		}
	}
	assert.Equal(t, electables, electables2)
	assert.Equal(t, cfg.MainPReps()+cfg.ExtraMainPReps(), prepSet.GetPRepSize(GradeMain))
	assert.Equal(t, electables, prepSet.GetPRepSize(GradeMain)+prepSet.GetPRepSize(GradeSub))
	assert.Equal(t, totalPReps-electables, prepSet.GetPRepSize(GradeCandidate))

	var expGrade Grade
	for i, prep := range preps {
		grade := prep.Grade()
		if i < prepSet.GetPRepSize(GradeMain) {
			expGrade = GradeMain
		} else if i < electables {
			expGrade = GradeSub
		} else {
			expGrade = GradeCandidate
			assert.False(t, prep.IsElectable(sc))
		}
		assert.Equal(t, expGrade, grade)
	}
}

func TestSortByPower(t *testing.T) {
	br := icmodule.ToRate(5)
	preps := newDummyPReps(6)

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
			SortByPower(sc, preps)
			size := len(preps)

			for i := 1; i < size; i++ {
				assert.True(t, checkPRepOrder(sc, preps[i-1], preps[i]))
			}
		})
	}
}

func TestChooseExtraMainPReps(t *testing.T) {
	const (
		size = 10
	)
	br := icmodule.ToRate(5)

	args := []struct {
		eligible           int
		extraMainPRepCount int
		exp                int
	}{
		{0, 0, 0},
		{0, 3, 0},
		{1, 3, 1},
		{2, 3, 2},
		{3, 3, 3},
		{5, 2, 2},
		{size, 3, 3},
	}

	for i, arg := range args {
		eligible := arg.eligible
		extraMainPRepCount := arg.extraMainPRepCount
		exp := arg.exp

		name := fmt.Sprintf("case-%02d", i)
		preps := newDummyPReps(size)
		for j := 0; j < size-arg.eligible; j++ {
			preps[j].SetBonded(icmodule.BigIntZero)
		}

		t.Run(name, func(t *testing.T) {
			extras := chooseExtraMainPReps(preps, extraMainPRepCount, func(prep *PRep) bool {
				return prep.GetPower(br).Sign() > 0
			})
			assert.Equal(t, exp, len(extras))
			for j := 0; j < exp; j++ {
				assert.Equal(t, preps[size-eligible+j], extras[j])
			}
		})
	}
}

func TestCopyPReps(t *testing.T) {
	srcPReps := newDummyPReps(10)

	args := []struct {
		excludes []int
	}{
		{nil},
		{[]int{0, 5}},
		{[]int{7}},
	}

	for i, arg := range args {
		name := fmt.Sprintf("case-%02d", i)
		dstPReps := make([]*PRep, len(srcPReps))
		excludeMap := make(map[string]bool)
		for _, idx := range arg.excludes {
			prep := srcPReps[idx]
			excludeMap[icutils.ToKey(prep.Owner())] = true
		}

		t.Run(name, func(t *testing.T) {
			copyPReps(srcPReps, dstPReps, excludeMap)
			nils := 0
			for _, prep := range dstPReps {
				if prep == nil {
					nils++
				} else {
					assert.False(t, excludeMap[icutils.ToKey(prep.Owner())])
				}
			}
			assert.Equal(t, nils, len(excludeMap))
		})
	}
}

func TestClassifyPReps(t *testing.T) {
	preps := newDummyPReps(10)
	args := []struct {
		main    int
		elected int
	}{
		{10, 10},
		{3, 7},
		{3, 5},
		{7, 7},
	}

	for i, arg := range args {
		name := fmt.Sprintf("case-%02d", i)
		main := arg.main
		elected := arg.elected

		t.Run(name, func(t *testing.T) {
			mainPReps, subPReps := classifyPReps(preps, main, elected)
			assert.Equal(t, main, len(mainPReps))
			assert.Equal(t, elected-main, len(subPReps))
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
