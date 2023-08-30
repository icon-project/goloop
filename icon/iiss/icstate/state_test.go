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

func newNodeOnlyRegInfo(node module.Address) *PRepInfo {
	return &PRepInfo{
		Node: node,
	}
}

func TestState_RegisterPRep(t *testing.T) {
	var err error
	size := 10
	irep := icmodule.BigIntInitialIRep
	state := newDummyState(false)

	for i := 0; i < size; i++ {
		owner := newDummyAddress(i)
		ri := newDummyPRepInfo(i)
		err = state.RegisterPRep(owner, ri, irep, 0)
		assert.NoError(t, err)
		err = state.Flush()
		assert.NoError(t, err)

		prep := state.GetPRepByOwner(owner)
		assert.NotNil(t, prep)
		assert.True(t, prep.Owner().Equal(owner))

		pb := state.GetPRepBaseByOwner(owner, false)
		assert.NotNil(t, pb)
		info := pb.info()
		assert.Truef(t, info.equal(ri), "DifferentInfo exp=%+v real=%+v", ri, info)

		ps := state.GetPRepStatusByOwner(owner, false)
		assert.NotNil(t, ps)
		assert.Equal(t, GradeCandidate, ps.Grade())
		assert.Equal(t, Active, ps.Status())
		assert.Zero(t, ps.Delegated().Int64())
		assert.Zero(t, ps.Bonded().Int64())
		assert.Equal(t, None, ps.LastState())
		assert.Zero(t, ps.LastHeight())
		assert.Zero(t, ps.VTotal())
		assert.Zero(t, ps.VFail())
	}
}

func TestState_SetPRep(t *testing.T) {
	var err error
	size := 10
	irep := icmodule.BigIntInitialIRep
	bh := int64(100)
	state := newDummyState(false)

	for i := 0; i < size; i++ {
		owner := newDummyAddress(i)
		ri := newDummyPRepInfo(i)
		err = state.RegisterPRep(owner, ri, irep, 0)
		assert.NoError(t, err)

		err = state.Flush()
		assert.NoError(t, err)

		node := newDummyAddress(i + 100)
		assert.False(t, node.Equal(owner))
		ri = newNodeOnlyRegInfo(node)
		_, err = state.SetPRep(bh, owner, ri)
		assert.NoError(t, err)

		err = state.Flush()
		assert.NoError(t, err)

		node2 := state.GetNodeByOwner(owner)
		assert.True(t, node2.Equal(node))
	}
}

func TestState_GetNodeByOwner(t *testing.T) {
	owner := newDummyAddress(1)
	state := newDummyState(false)

	node := state.GetNodeByOwner(nil)
	assert.Nil(t, node)

	node = state.GetNodeByOwner(owner)
	assert.Nil(t, node)
}

func TestState_Reset(t *testing.T) {
	state := newDummyState(false)
	ss := state.GetSnapshot()
	assert.NoError(t, state.Reset(ss))
}

func TestState_GetSnapshot(t *testing.T) {
	state := newDummyState(false)
	snapshot := state.GetSnapshot()
	assert.NotNil(t, snapshot)
}

func TestState_GetAccountState(t *testing.T) {
	var err error
	stake := int64(100)
	addr := common.MustNewAddressFromString("hx1")
	state := newDummyState(false)
	as := state.GetAccountState(addr)
	assert.NotNil(t, as)

	err = as.SetStake(big.NewInt(stake))
	assert.NoError(t, err)

	state.ClearCache()

	as2 := state.GetAccountState(addr)
	assert.True(t, as2.equal(&as.accountData))

	ass := state.GetAccountSnapshot(addr)
	assert.True(t, ass.equal(&as2.accountData))
}

func TestState_GetUnstakingTimerState(t *testing.T) {
	height := int64(100)
	addrs := newDummyAddresses(3)
	state := newDummyState(false)
	ts := state.GetUnstakingTimerState(height)
	assert.True(t, ts.IsEmpty())

	for _, addr := range addrs {
		ts.Add(addr)
	}

	err := state.Flush()
	assert.NoError(t, err)
	state.ClearCache()

	tss := state.GetUnstakingTimerSnapshot(height)
	assert.False(t, tss.IsEmpty())
	assert.True(t, tss.Equal(ts.GetSnapshot()))
}

func TestState_GetUnbondingTimerState(t *testing.T) {
	height := int64(100)
	addrs := newDummyAddresses(3)
	state := newDummyState(false)
	ts := state.GetUnbondingTimerState(height)
	assert.True(t, ts.IsEmpty())

	for _, addr := range addrs {
		ts.Add(addr)
	}

	err := state.Flush()
	assert.NoError(t, err)
	state.ClearCache()

	tss := state.GetUnbondingTimerSnapshot(height)
	assert.False(t, tss.IsEmpty())
	assert.True(t, tss.Equal(ts.GetSnapshot()))
}

func TestState_GetNetworkScoreTimerState(t *testing.T) {
	height := int64(100)
	addrs := newDummyAddresses(3)
	state := newDummyState(false)
	ts := state.GetNetworkScoreTimerState(height)
	assert.True(t, ts.IsEmpty())

	for _, addr := range addrs {
		ts.Add(addr)
	}

	err := state.Flush()
	assert.NoError(t, err)
	state.ClearCache()

	tss := state.GetNetworkScoreTimerSnapshot(height)
	assert.False(t, tss.IsEmpty())
	assert.True(t, tss.Equal(ts.GetSnapshot()))
}

func TestState_GetPRepByOwner(t *testing.T) {
	addr := newDummyAddress(1234)
	state := newDummyState(false)
	prep := state.GetPRepByOwner(addr)
	assert.Nil(t, prep)
}

func TestState_SetIssue(t *testing.T) {
	var err error
	prevBlockFee := int64(1)
	overIssuedIScore := int64(2)
	totalReward := int64(3)

	issue := NewIssue()
	issue.SetPrevBlockFee(big.NewInt(prevBlockFee))
	issue.SetOverIssuedIScore(big.NewInt(overIssuedIScore))
	issue.SetTotalReward(big.NewInt(totalReward))

	state := newDummyState(false)
	err = state.SetIssue(issue)
	assert.NoError(t, err)

	state.Flush()
	state.ClearCache()

	issue2, err := state.GetIssue()
	assert.NoError(t, err)
	assert.True(t, issue.Equal(issue2))

	assert.Equal(t, prevBlockFee, issue2.PrevBlockFee().Int64())
	assert.Equal(t, overIssuedIScore, issue2.OverIssuedIScore().Int64())
	assert.Equal(t, totalReward, issue2.TotalReward().Int64())
}

func TestState_SetTermSnapshot(t *testing.T) {
	seq := 1
	period := int64(43120)
	term := newTermState(termVersion1, seq, period).GetSnapshot()

	state := newDummyState(false)
	err := state.SetTermSnapshot(term)
	assert.NoError(t, err)

	state.Flush()
	state.ClearCache()

	term2 := state.GetTermSnapshot()
	assert.True(t, term.Equal(term2))
	assert.Equal(t, seq, term2.Sequence())
	assert.Equal(t, period, term2.Period())
}

func TestState_SetRewardCalcInfo(t *testing.T) {
	prevBlockHeight := int64(1000)
	prevCalcReward := int64(100)
	startBlockHeight := int64(2000)
	prevHash := make([]byte, 32)
	prevHash[0] = 1

	rc := NewRewardCalcInfo()

	state := newDummyState(false)
	orc, err := state.GetRewardCalcInfo()
	assert.NoError(t, err)
	assert.True(t, orc.Equal(rc))

	rc.SetPrevHeight(prevBlockHeight)
	rc.SetPrevCalcReward(big.NewInt(prevCalcReward))
	rc.SetStartHeight(startBlockHeight)
	rc.SetPrevHash(prevHash)

	err = state.SetRewardCalcInfo(rc)
	assert.NoError(t, err)

	state.Flush()
	state.ClearCache()

	orc, err = state.GetRewardCalcInfo()
	assert.NoError(t, err)
	assert.Equal(t, prevBlockHeight, orc.PrevHeight())
	assert.Equal(t, prevCalcReward, orc.PrevCalcReward().Int64())
	assert.Equal(t, startBlockHeight, orc.StartHeight())
	assert.Equal(t, prevHash, orc.PrevHash())
}

func TestState_SetUnstakeSlotMax(t *testing.T) {
	state := newDummyState(false)

	slots := state.GetUnstakeSlotMax()
	assert.Zero(t, slots)

	err := state.SetUnstakeSlotMax(100)
	assert.NoError(t, err)

	slots = state.GetUnstakeSlotMax()
	assert.Equal(t, int64(100), slots)
}

func TestState_SetTotalDelegation(t *testing.T) {
	state := newDummyState(false)

	value := state.GetTotalDelegation()
	assert.Equal(t, int64(0), value.Int64())

	value = big.NewInt(100)
	err := state.SetTotalDelegation(value)
	assert.NoError(t, err)
	state.Flush()
	state.ClearCache()

	value2 := state.GetTotalDelegation()
	assert.Zero(t, value.Cmp(value2))
}

func TestState_SetTotalBond(t *testing.T) {
	state := newDummyState(false)

	value := state.GetTotalBond()
	assert.Equal(t, int64(0), value.Int64())

	value = big.NewInt(100)
	err := state.SetTotalBond(value)
	assert.NoError(t, err)
	state.Flush()
	state.ClearCache()

	value2 := state.GetTotalBond()
	assert.Zero(t, value.Cmp(value2))
}

func TestState_GetOwnerByNode(t *testing.T) {
	var err error
	var address module.Address
	irep := big.NewInt(100)
	owner := newDummyAddress(1)
	node := newDummyAddress(101)

	state := newDummyState(false)

	ri := newDummyPRepInfo(1)
	err = state.RegisterPRep(owner, ri, irep, 1234)
	assert.NoError(t, err)

	state.Flush()
	state.ClearCache()

	address = state.GetOwnerByNode(owner)
	assert.True(t, address.Equal(owner))
	address = state.GetOwnerByNode(node)
	assert.True(t, address.Equal(node))

	blockHeight := int64(1000)
	ri = newNodeOnlyRegInfo(node)
	update, err := state.SetPRep(blockHeight, owner, ri)
	assert.True(t, update)
	assert.NoError(t, err)

	state.Flush()
	state.ClearCache()

	address = state.GetOwnerByNode(node)
	assert.True(t, address.Equal(owner))
}

func TestState_OnBlockVote(t *testing.T) {
	var err error
	irep := big.NewInt(100)
	owner := newDummyAddress(1)
	state := newDummyState(false)

	err = state.OnBlockVote(owner, true, 1000)
	assert.Error(t, err)

	ri := newDummyPRepInfo(1)
	err = state.RegisterPRep(owner, ri, irep, 1234)
	assert.NoError(t, err)
	state.Flush()
	state.ClearCache()

	blockHeight := int64(1000)
	for i := 0; i < 5; i++ {
		err = state.OnBlockVote(owner, true, blockHeight)
		assert.NoError(t, err)
		state.Flush()
		state.ClearCache()

		ps := state.GetPRepStatusByOwner(owner, false)
		assert.Equal(t, Success, ps.LastState())
		assert.Equal(t, int64(i+1), ps.GetVTotal(blockHeight))
		assert.Zero(t, ps.GetVFail(blockHeight))
		assert.Zero(t, ps.GetVFailCont(blockHeight))

		blockHeight++
	}

	for i := 0; i < 5; i++ {
		err = state.OnBlockVote(owner, false, blockHeight)
		assert.NoError(t, err)
		state.Flush()
		state.ClearCache()

		ps := state.GetPRepStatusByOwner(owner, false)
		assert.Equal(t, Failure, ps.LastState())
		assert.Equal(t, int64(i+6), ps.GetVTotal(blockHeight))
		assert.Equal(t, int64(i+1), ps.GetVFail(blockHeight))
		assert.Equal(t, int64(i+1), ps.GetVFailCont(blockHeight))

		blockHeight++
	}

	state.OnBlockVote(owner, true, blockHeight)
	assert.NoError(t, err)
	state.Flush()
	state.ClearCache()

	ps := state.GetPRepStatusByOwner(owner, false)
	assert.Equal(t, Success, ps.LastState())
	assert.Equal(t, int64(11), ps.GetVTotal(blockHeight))
	assert.Equal(t, int64(5), ps.GetVFail(blockHeight))
	assert.Equal(t, int64(0), ps.GetVFailCont(blockHeight))
}

func TestState_OnMainPRepReplaced(t *testing.T) {
	var err error
	var sc icmodule.StateContext
	limit := 30

	type input struct {
		rev     int
		termRev int
	}
	args := []struct {
		in input
	}{
		{input{icmodule.RevisionIISS4, icmodule.RevisionIISS4 - 1}},
		{input{icmodule.RevisionIISS4, icmodule.RevisionIISS4}},
		{input{icmodule.RevisionIISS4 + 1, icmodule.RevisionIISS4}},
		{input{icmodule.RevisionIISS4 + 1, icmodule.RevisionIISS4 + 1}},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			state := newDummyState(false)
			owners := newDummyAddresses(2)

			for i := 0; i < len(owners); i++ {
				ri := newDummyPRepInfo(1)
				err = state.RegisterPRep(owners[i], ri, new(big.Int), 1234)
				assert.NoError(t, err)
			}

			sc = NewStateContext(1000, arg.in.rev, arg.in.termRev, nil)

			state.Flush()
			state.ClearCache()

			ps := state.GetPRepStatusByOwner(owners[1], false)
			assert.Equal(t, GradeCandidate, ps.Grade())

			err = state.OnMainPRepReplaced(sc, owners[0], owners[1])
			assert.Error(t, err) // Invalid: C -> M

			err = ps.OnTermEnd(sc, GradeSub, limit)
			assert.NoError(t, err)
			state.Flush()
			state.ClearCache()

			ps = state.GetPRepStatusByOwner(owners[1], false)
			assert.Equal(t, GradeSub, ps.Grade())

			termRev := sc.TermRevision()
			if sc.Revision() < termRev {
				termRev = sc.Revision()
			}
			sc = NewStateContext(sc.BlockHeight()+1, sc.Revision(), termRev, nil)

			err = state.OnMainPRepReplaced(sc, owners[0], owners[1])
			assert.NoError(t, err)

			state.Flush()
			state.ClearCache()

			ps = state.GetPRepStatusByOwner(owners[1], false)
			assert.Equal(t, GradeMain, ps.Grade())
		})
	}
}

func TestState_ImposePenalty(t *testing.T) {
	var err error
	owner := newDummyAddress(1)
	ri := newDummyPRepInfo(1)

	type input struct {
		rev     int
		termRev int
		pt      icmodule.PenaltyType
	}
	type output struct {
		jailFlags int
	}
	args := []struct {
		in  input
		out output
	}{
		{
			input{
				icmodule.RevisionIISS4 - 1,
				icmodule.RevisionIISS4 - 1,
				icmodule.PenaltyValidationFailure,
			},
			output{0},
		},
		{
			input{
				icmodule.RevisionIISS4,
				icmodule.RevisionIISS4 - 1,
				icmodule.PenaltyValidationFailure,
			},
			output{0},
		},
		{
			input{
				icmodule.RevisionIISS4,
				icmodule.RevisionIISS4,
				icmodule.PenaltyValidationFailure,
			},
			output{JFlagInJail},
		},
		{
			input{
				icmodule.RevisionIISS4 + 1,
				icmodule.RevisionIISS4,
				icmodule.PenaltyValidationFailure,
			},
			output{JFlagInJail},
		},
		{
			input{
				icmodule.RevisionIISS4 + 1,
				icmodule.RevisionIISS4 + 1,
				icmodule.PenaltyValidationFailure,
			},
			output{JFlagInJail},
		},
		{
			input{
				icmodule.RevisionIISS4 - 1,
				icmodule.RevisionIISS4 - 1,
				icmodule.PenaltyDoubleVote,
			},
			output{0},
		},
		{
			input{
				icmodule.RevisionIISS4,
				icmodule.RevisionIISS4 - 1,
				icmodule.PenaltyDoubleVote,
			},
			output{0},
		},
		{
			input{
				icmodule.RevisionIISS4,
				icmodule.RevisionIISS4,
				icmodule.PenaltyDoubleVote,
			},
			output{JFlagInJail | JFlagDoubleVote},
		},
		{
			input{
				icmodule.RevisionIISS4 + 1,
				icmodule.RevisionIISS4,
				icmodule.PenaltyDoubleVote,
			},
			output{JFlagInJail | JFlagDoubleVote},
		},
		{
			input{
				icmodule.RevisionIISS4 + 1,
				icmodule.RevisionIISS4 + 1,
				icmodule.PenaltyDoubleVote,
			},
			output{JFlagInJail | JFlagDoubleVote},
		},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			state := newDummyState(false)
			pt := arg.in.pt

			err = state.RegisterPRep(owner, ri, icmodule.BigIntZero, 1234)
			assert.NoError(t, err)
			state.Flush()
			state.ClearCache()

			sc := NewStateContext(10000, arg.in.rev, arg.in.termRev, nil)
			ps := state.GetPRepStatusByOwner(owner, false)
			err = state.ImposePenalty(sc, pt, owner, ps)
			assert.NoError(t, err)
			state.Flush()
			state.ClearCache()

			ps = state.GetPRepStatusByOwner(owner, false)
			if pt.IsTypeOfValidationFailurePenalty() {
				assert.Equal(t, 1, ps.GetVPenaltyCount())
				assert.True(t, ps.IsAlreadyPenalized())
			}

			assert.Equal(t, arg.out.jailFlags, ps.JailFlags())
			assert.Zero(t, ps.UnjailRequestHeight())
			assert.Zero(t, ps.MinDoubleVoteHeight())
		})
	}
}

func TestState_ReducePRepBonded(t *testing.T) {
	var err error
	totalBond := int64(100)
	owner := newDummyAddress(1)
	state := newDummyState(false)

	ri := newDummyPRepInfo(1)
	err = state.RegisterPRep(owner, ri, new(big.Int), 1234)
	assert.NoError(t, err)
	state.Flush()
	state.ClearCache()

	err = state.SetTotalBond(big.NewInt(totalBond))
	assert.NoError(t, err)
	state.Flush()
	state.ClearCache()

	ps := state.GetPRepStatusByOwner(owner, false)
	ps.SetBonded(big.NewInt(totalBond))
	state.Flush()
	state.ClearCache()

	amount := int64(10)
	err = state.ReducePRepBonded(owner, big.NewInt(amount))
	assert.NoError(t, err)
	state.Flush()
	state.ClearCache()
	assert.Equal(t, totalBond-amount, state.GetTotalBond().Int64())

	ps = state.GetPRepStatusByOwner(owner, false)
	assert.Equal(t, totalBond-amount, ps.Bonded().Int64())
}

func TestState_DisablePRep(t *testing.T) {
	var err error
	totalDelegation := int64(100)
	owner := newDummyAddress(1)
	state := newDummyState(false)

	ri := newDummyPRepInfo(1)
	err = state.RegisterPRep(owner, ri, new(big.Int), 1234)
	assert.NoError(t, err)
	state.Flush()
	state.ClearCache()

	err = state.SetTotalDelegation(big.NewInt(totalDelegation))
	assert.NoError(t, err)
	state.Flush()
	state.ClearCache()

	assert.Equal(t, totalDelegation, state.GetTotalDelegation().Int64())

	delegation := totalDelegation
	ps := state.GetPRepStatusByOwner(owner, false)
	ps.SetDelegated(big.NewInt(delegation))
	state.Flush()
	state.ClearCache()

	ps = state.GetPRepStatusByOwner(owner, false)
	assert.Equal(t, delegation, ps.Delegated().Int64())
	assert.Equal(t, Active, ps.Status())

	sc := NewStateContext(1000, icmodule.RevisionPreIISS4, icmodule.RevisionPreIISS4, nil)
	err = state.DisablePRep(sc, owner, Unregistered)
	assert.NoError(t, err)
	state.Flush()
	state.ClearCache()

	assert.Zero(t, state.GetTotalDelegation().Int64())

	ps = state.GetPRepStatusByOwner(owner, false)
	assert.Equal(t, delegation, ps.Delegated().Int64())
	assert.Equal(t, Unregistered, ps.Status())
}

func TestState_CheckValidationPenalty(t *testing.T) {
	var err error
	condition := 10
	owner := newDummyAddress(1)
	state := newDummyState(false)
	err = state.SetValidationPenaltyCondition(condition)
	assert.NoError(t, err)

	ri := newDummyPRepInfo(1)
	err = state.RegisterPRep(owner, ri, new(big.Int), 1234)
	assert.NoError(t, err)
	state.Flush()
	state.ClearCache()

	blockHeight := int64(1000)
	ps := state.GetPRepStatusByOwner(owner, false)
	for i := 0; i < condition; i++ {
		err = ps.OnBlockVote(blockHeight, false)
		assert.NoError(t, err)
		state.Flush()
		state.ClearCache()

		isPenalized := state.CheckValidationPenalty(ps, blockHeight)
		if i < 9 {
			assert.False(t, isPenalized)
		} else {
			assert.True(t, isPenalized)
		}
		blockHeight++
	}
}

func TestState_CheckConsistentValidationPenalty(t *testing.T) {
	var err error
	owner := newDummyAddress(1)
	state := newDummyState(false)
	err = state.SetConsistentValidationPenaltyCondition(icmodule.DefaultConsistentValidationPenaltyCondition)
	assert.NoError(t, err)

	ri := newDummyPRepInfo(1)
	err = state.RegisterPRep(owner, ri, new(big.Int), 1234)
	assert.NoError(t, err)
	state.Flush()
	state.ClearCache()

	ps := state.GetPRepStatusByOwner(owner, false)
	for rev := 0; rev <= icmodule.LatestRevision; rev++ {
		isPenalty := state.CheckConsistentValidationPenalty(rev, ps)
		assert.False(t, isPenalty)
	}
}

func TestState_GetUnstakeLockPeriod(t *testing.T) {
	var err error
	termPeriod := int64(43120)
	lMin := big.NewInt(5)
	lMax := big.NewInt(20)
	minLockPeriod := lMin.Int64() * termPeriod
	maxLockPeriod := lMax.Int64() * termPeriod
	totalSupply := big.NewInt(1000)

	state := newDummyState(false)
	err = state.setLockMinMultiplier(lMin)
	assert.NoError(t, err)
	err = state.setLockMaxMultiplier(lMax)
	assert.NoError(t, err)
	err = state.SetTermPeriod(termPeriod)
	assert.NoError(t, err)

	state.Flush()
	state.ClearCache()

	prevPeriod := int64(0)
	rev := icmodule.LatestRevision
	for i := 0; i <= 10; i++ {
		totalStake := int64(i * 100)
		if i > 0 && i < 10 {
			totalStake += rand.Int63n(50)
		}
		err = state.SetTotalStake(big.NewInt(totalStake))
		assert.NoError(t, err)

		state.Flush()
		state.ClearCache()

		periodInBlock := state.GetUnstakeLockPeriod(rev, totalSupply)
		assert.True(t, periodInBlock >= minLockPeriod)
		assert.True(t, periodInBlock <= maxLockPeriod)

		if i == 0 {
			assert.True(t, periodInBlock == maxLockPeriod)
		} else if i < 8 {
			assert.True(t, periodInBlock < prevPeriod)
		} else {
			assert.True(t, periodInBlock == minLockPeriod)
		}

		prevPeriod = periodInBlock
	}
}

func TestState_SetIllegalDelegation(t *testing.T) {
	addr := newDummyAddress(1)
	state := newDummyState(false)

	o := state.GetIllegalDelegation(addr)
	assert.Nil(t, o)

	var ds Delegations = make([]*Delegation, 3)
	for i := 0; i < 3; i++ {
		addr := newDummyAddress(100 + i)
		ds[i] = NewDelegation(addr.(*common.Address), big.NewInt(int64(i+1)))
	}

	o = NewIllegalDelegation(addr, ds)
	err := state.SetIllegalDelegation(o)
	assert.NoError(t, err)

	state.Flush()
	state.ClearCache()

	o = state.GetIllegalDelegation(addr)
	assert.True(t, addr.Equal(o.Address()))
	assert.True(t, ds.Equal(o.Delegations()))

	err = state.DeleteIllegalDelegation(addr)
	assert.NoError(t, err)

	o = state.GetIllegalDelegation(addr)
	assert.Nil(t, o)
}

func TestState_SetPRepIllegalDelegated(t *testing.T) {
	state := newDummyState(false)
	addrs := newDummyAddresses(3)
	values := []int64{-100, 0, 100}

	for i, v := range values {
		err := state.SetPRepIllegalDelegated(addrs[i], big.NewInt(v))
		assert.NoError(t, err)

		state.Flush()
		state.ClearCache()

		v2 := state.GetPRepIllegalDelegated(addrs[i])
		assert.Equal(t, v, v2.Int64())
	}
}

func TestState_SetLastBlockVotersSnapshot(t *testing.T) {
	voters := newDummyAddresses(7)
	state := newDummyState(false)

	bvs := state.GetLastBlockVotersSnapshot()
	assert.Nil(t, bvs)

	bvs = NewBlockVotersSnapshot(voters)
	err := state.SetLastBlockVotersSnapshot(bvs)
	assert.NoError(t, err)
	state.Flush()
	state.ClearCache()

	bvs2 := state.GetLastBlockVotersSnapshot()
	assert.True(t, bvs.Equal(bvs2))
}

func TestState_OnValidatorOut(t *testing.T) {
	type arg struct {
		votes    []bool
		fails    int64
		failCont int64
	}

	args := []arg{
		{votes: []bool{true, true, true}, fails: 0, failCont: 0},
		{votes: []bool{false, true, true}, fails: 1, failCont: 0},
		{votes: []bool{true, false, true}, fails: 1, failCont: 0},
		{votes: []bool{true, true, false}, fails: 1, failCont: 1},
		{votes: []bool{true, false, false}, fails: 2, failCont: 2},
		{votes: []bool{false, true, false}, fails: 2, failCont: 1},
		{votes: []bool{false, false, true}, fails: 2, failCont: 0},
		{votes: []bool{false, false, false}, fails: 3, failCont: 3},
	}

	var err error

	for i, a := range args {
		name := fmt.Sprintf("%d-%v-%d", i, a.votes, a.fails)
		t.Run(name, func(t *testing.T) {
			blockHeight := int64(1000)
			irep := big.NewInt(100)
			owner := newDummyAddress(1)
			state := newDummyState(false)

			err = state.OnValidatorOut(blockHeight, owner)
			assert.Error(t, err)

			ri := newDummyPRepInfo(1)
			err = state.RegisterPRep(owner, ri, irep, 1234)
			assert.NoError(t, err)
			state.Flush()
			state.ClearCache()

			for _, vote := range a.votes {
				ps := state.GetPRepStatusByOwner(owner, false)
				ps.OnBlockVote(blockHeight, vote)
				state.Flush()
				state.ClearCache()
				blockHeight++
			}

			err = state.OnValidatorOut(blockHeight-1, owner)
			assert.NoError(t, err)
			state.Flush()
			state.ClearCache()

			ps := state.GetPRepStatusByOwner(owner, false)
			assert.Equal(t, None, ps.LastState())
			assert.Equal(t, int64(len(a.votes)), ps.GetVTotal(blockHeight))
			assert.Equal(t, a.fails, ps.GetVFail(blockHeight))
			assert.Equal(t, a.failCont, ps.GetVFailCont(blockHeight))
		})
	}
}

func TestState_InitCommissionInfo(t *testing.T) {
	rate := icmodule.ToRate(10)
	maxRate := icmodule.ToRate(30)
	maxChangeRate := icmodule.ToRate(1)
	owner := newDummyAddress(1)

	state := newDummyState(false)

	ci, err := NewCommissionInfo(rate, maxRate, maxChangeRate)
	assert.NoError(t, err)
	assert.NotNil(t, ci)

	err = state.InitCommissionInfo(owner, ci)
	assert.Error(t, err)

	ri := newDummyPRepInfo(0)
	err = state.RegisterPRep(owner, ri, nil, 0)
	assert.NoError(t, err)

	pb := state.GetPRepBaseByOwner(owner, false)
	assert.NotNil(t, pb)
	assert.Equal(t, icmodule.Rate(0), pb.CommissionRate())
	assert.Equal(t, icmodule.Rate(0), pb.MaxCommissionRate())
	assert.Equal(t, icmodule.Rate(0), pb.MaxCommissionChangeRate())

	jso := pb.ToJSON(owner)
	assert.Nil(t, jso["commissionRate"])
	assert.Nil(t, jso["maxCommissionRate"])
	assert.Nil(t, jso["maxCommissionChangeRate"])

	err = state.InitCommissionInfo(owner, ci)
	assert.NoError(t, err)

	pb = state.GetPRepBaseByOwner(owner, false)
	assert.NotNil(t, pb)
	assert.Equal(t, rate, pb.CommissionRate())
	assert.Equal(t, maxRate, pb.MaxCommissionRate())
	assert.Equal(t, maxChangeRate, pb.MaxCommissionChangeRate())

	jso = pb.ToJSON(owner)
	assert.Equal(t, int64(rate), jso["commissionRate"])
	assert.Equal(t, int64(maxRate), jso["maxCommissionRate"])
	assert.Equal(t, int64(maxChangeRate), jso["maxCommissionChangeRate"])
}
