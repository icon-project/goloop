/*
 * Copyright 2020 ICON Foundation
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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

func testFactory(_ icobject.Tag) (icobject.Impl, error) {
	return nil, errors.New("Unsupported")
}

type mockCallContext struct {
	icmodule.CallContext
	gov module.Address
}

func (cc *mockCallContext) Governance() module.Address {
	return cc.gov
}

func newMockCallContext() icmodule.CallContext {
	return &mockCallContext{
		gov: common.MustNewAddressFromString("cx1"),
	}
}

func Test_networkValue(t *testing.T) {
	database := db.NewMapDB()
	database = icobject.AttachObjectFactory(database, testFactory)
	tree := trie_manager.NewMutableForObject(database, nil, icobject.ObjectType)

	oss := icobject.NewObjectStoreState(tree)

	s := &State{
		readonly: false,
		store:    oss,
	}

	// test for SetTermPeriod
	t.Run("SetTermPeriod", func(t *testing.T) { setTermPeriodTest(t, s) })

	// test for SetIRep
	t.Run("SetIRep", func(t *testing.T) { setIRepTest(t, s) })

	// test for SetRRep
	t.Run("SetRRep", func(t *testing.T) { setRRepTest(t, s) })

	// test for SetMainPRepCount
	t.Run("SetMainPRepCount", func(t *testing.T) { setMainPRepCountTest(t, s) })

	// test for SetSubPRepCount
	t.Run("SetSubPRepCount", func(t *testing.T) { setSubPRepCountTest(t, s) })

	// test for SetTotalStake
	t.Run("SetTotalStake", func(t *testing.T) { setTotalStakeTest(t, s) })

	// test for SetBondRequirement
	t.Run("SetBondRequirement", func(t *testing.T) { setBondRequirementTest(t, s) })

	// test for SetLockVariables
	t.Run("SetLockVariables", func(t *testing.T) { setLockVariablesTest(t, s) })

	// test for SetRewardFund
	t.Run("SetRewardFund-version1", func(t *testing.T) { setRewardFundV1Test(t, s) })
	t.Run("SetRewardFund-version2", func(t *testing.T) { setRewardFundV2Test(t, s) })

	// test for SetUnbondingPeriodMultiplier
	t.Run("SetUnbondingPeriodMultiplier", func(t *testing.T) { setUnbondingPeriodMultiplier(t, s) })
}

func setTermPeriodTest(t *testing.T, s *State) {
	actual := s.GetTermPeriod()
	assert.Equal(t, int64(0), actual)

	tp := int64(10)
	assert.NoError(t, s.SetTermPeriod(tp))
	actual = s.GetTermPeriod()
	assert.Equal(t, tp, actual)

}

func setIRepTest(t *testing.T, s *State) {
	actual := s.GetIRep()
	assert.Zero(t, actual.Sign())

	irep := big.NewInt(10)
	assert.NoError(t, s.SetIRep(irep))
	actual = s.GetIRep()
	assert.Equal(t, 0, actual.Cmp(irep))
}

func setRRepTest(t *testing.T, s *State) {
	actual := s.GetRRep()
	assert.Zero(t, actual.Sign())

	rrep := big.NewInt(10)
	assert.NoError(t, s.SetIRep(rrep))
	actual = s.GetIRep()
	assert.Equal(t, 0, actual.Cmp(rrep))
}

func setMainPRepCountTest(t *testing.T, s *State) {
	actual := s.GetMainPRepCount()
	sCount := s.GetMainPRepCount()
	assert.Zero(t, actual)
	assert.Zero(t, sCount)

	count := int64(10)
	assert.NoError(t, s.SetMainPRepCount(count))
	actual = s.GetMainPRepCount()
	sCount = s.GetMainPRepCount()
	assert.Equal(t, count, actual)
	assert.Equal(t, count, sCount)

	err := s.SetMainPRepCount(-5)
	assert.Error(t, err)
	assert.Equal(t, count, s.GetMainPRepCount())
}

func setSubPRepCountTest(t *testing.T, s *State) {
	for i := 0; i < 2; i++ {
		actual := s.GetSubPRepCount()
		assert.Zero(t, actual)
	}

	count := int64(20)
	assert.NoError(t, s.SetSubPRepCount(count))
	for i := 0; i < 2; i++ {
		actual := s.GetSubPRepCount()
		assert.Equal(t, count, actual)
	}

	assert.Error(t, s.SetSubPRepCount(-10))
	assert.Equal(t, count, s.GetSubPRepCount())
}

func setTotalStakeTest(t *testing.T, s *State) {
	ts := new(big.Int)
	actual := s.GetTotalStake()
	assert.Equal(t, 0, actual.Cmp(ts))

	ts = big.NewInt(20)
	assert.NoError(t, s.SetTotalStake(ts))
	actual = s.GetTotalStake()
	assert.Equal(t, 0, actual.Cmp(ts))
}

func setBondRequirementTest(t *testing.T, s *State) {
	br := icmodule.ToRate(0)
	actual := s.GetBondRequirement()
	assert.Equal(t, br, actual)

	br = icmodule.ToRate(5)
	assert.NoError(t, s.SetBondRequirement(br))
	actual = s.GetBondRequirement()
	assert.Equal(t, br, actual)

	br = icmodule.ToRate(0)
	err := s.SetBondRequirement(br)
	assert.NoError(t, err)
	actual = s.GetBondRequirement()
	assert.Equal(t, br, actual)

	err = s.SetBondRequirement(icmodule.ToRate(101))
	assert.Error(t, err)
	actual = s.GetBondRequirement()
	assert.Equal(t, br, actual)

	err = s.SetBondRequirement(icmodule.ToRate(-1))
	assert.Error(t, err)
	actual = s.GetBondRequirement()
	assert.Equal(t, br, actual)
}

func setLockVariablesTest(t *testing.T, s *State) {
	actualMin := s.GetLockMinMultiplier()
	actualMax := s.GetLockMaxMultiplier()
	assert.Nil(t, actualMin)
	assert.Nil(t, actualMax)

	min := big.NewInt(10)
	max := big.NewInt(1)
	err := s.SetLockVariables(min, max)
	assert.Error(t, err)
	actualMin = s.GetLockMinMultiplier()
	actualMax = s.GetLockMaxMultiplier()
	assert.Nil(t, actualMin)
	assert.Nil(t, actualMax)

	min = big.NewInt(1)
	max = big.NewInt(10)
	err = s.SetLockVariables(min, max)
	assert.NoError(t, err)
	actualMin = s.GetLockMinMultiplier()
	actualMax = s.GetLockMaxMultiplier()
	assert.Equal(t, 0, actualMin.Cmp(min))
	assert.Equal(t, 0, actualMax.Cmp(max))

	invalidMin := big.NewInt(-1)
	err = s.SetLockVariables(invalidMin, max)
	assert.Error(t, err)
	actualMin = s.GetLockMinMultiplier()
	actualMax = s.GetLockMaxMultiplier()
	assert.Equal(t, 0, actualMin.Cmp(min))
	assert.Equal(t, 0, actualMax.Cmp(max))

	invalidMin = big.NewInt(-10)
	invalidMax := big.NewInt(-1)
	err = s.SetLockVariables(invalidMin, invalidMax)
	assert.Error(t, err)
	actualMin = s.GetLockMinMultiplier()
	actualMax = s.GetLockMaxMultiplier()
	assert.Equal(t, 0, actualMin.Cmp(min))
	assert.Equal(t, 0, actualMax.Cmp(max))
}

func setRewardFundV1Test(t *testing.T, s *State) {
	rf, err := NewSafeRewardFundV1(
		new(big.Int).SetInt64(100000),
		icmodule.ToRate(50),
		icmodule.ToRate(50),
		icmodule.ToRate(0),
		icmodule.ToRate(0),
	)
	assert.NoError(t, err)

	err = s.SetRewardFund(rf)
	assert.NoError(t, err)

	actual := s.GetRewardFundV1()
	assert.True(t, rf.Equal(actual))
}

func setRewardFundV2Test(t *testing.T, s *State) {
	rf, err := NewSafeRewardFundV2(
		new(big.Int).SetInt64(100000),
		icmodule.ToRate(50),
		icmodule.ToRate(50),
		icmodule.ToRate(0),
		icmodule.ToRate(0),
	)
	assert.NoError(t, err)

	err = s.SetRewardFund(rf)
	assert.NoError(t, err)

	actual := s.GetRewardFundV2()
	assert.True(t, rf.Equal(actual))
}

func setUnbondingPeriodMultiplier(t *testing.T, s *State) {
	p := int64(0)
	actual := s.GetUnbondingPeriodMultiplier()
	assert.Equal(t, p, actual)

	p = -1
	err := s.SetUnbondingPeriodMultiplier(p)
	assert.Error(t, err)
	actual = s.GetUnbondingPeriodMultiplier()
	assert.Equal(t, int64(0), actual) // not changed

	p = 10
	err = s.SetUnbondingPeriodMultiplier(p)
	assert.NoError(t, err)
	actual = s.GetUnbondingPeriodMultiplier()
	assert.Equal(t, p, actual)
}

func TestState_SetNetworkScore(t *testing.T) {
	cc := newMockCallContext()
	state := newDummyState(false)

	scores := state.GetNetworkScores(cc)
	assert.Equal(t, 1, len(scores))
	assert.True(t, scores[GovernanceKey].Equal(cc.Governance()))

	cps := common.MustNewAddressFromString("cx2")
	err := state.SetNetworkScore("cps", cps)
	assert.NoError(t, err)
	assert.NoError(t, state.Flush())
	state.ClearCache()

	scores = state.GetNetworkScores(cc)
	assert.Equal(t, 2, len(scores))
	assert.True(t, scores[GovernanceKey].Equal(cc.Governance()))
	assert.True(t, scores["cps"].Equal(cps))

	gov := common.MustNewAddressFromString("cx123")
	err = state.SetNetworkScore(GovernanceKey, gov)
	assert.Error(t, err)
	scores = state.GetNetworkScores(cc)
	assert.False(t, scores[GovernanceKey].Equal(gov))

	assert.NoError(t, state.Flush())
	state.ClearCache()
	scores = state.GetNetworkScores(cc)
	assert.False(t, scores[GovernanceKey].Equal(gov))

	invalidRole := "invalidRole"
	score := common.MustNewAddressFromString("cx5678")
	err = state.SetNetworkScore(invalidRole, score)
	assert.Error(t, err)
	scores = state.GetNetworkScores(cc)
	assert.Nil(t, scores[invalidRole])

	assert.NoError(t, state.Flush())
	state.ClearCache()
	scores = state.GetNetworkScores(cc)
	assert.Nil(t, scores[invalidRole])
}

func TestState_SetExtraMainPRepCount(t *testing.T) {
	var err error
	state := newDummyState(false)
	count := state.GetExtraMainPRepCount()
	assert.Equal(t, int64(icmodule.DefaultExtraMainPRepCount), count)

	err = state.SetExtraMainPRepCount(int64(-1))
	assert.Error(t, err)
	count = state.GetExtraMainPRepCount()
	assert.Equal(t, int64(icmodule.DefaultExtraMainPRepCount), count)

	newCount := int64(5)
	err = state.SetExtraMainPRepCount(newCount)
	assert.NoError(t, err)
	assert.NoError(t, state.Flush())
	state.ClearCache()

	count = state.GetExtraMainPRepCount()
	assert.Equal(t, newCount, count)
}

func TestState_SetUnbondingMax(t *testing.T) {
	state := newDummyState(false)

	value := state.GetUnbondingMax()
	assert.Equal(t, int64(0), value)

	value = int64(100)
	err := state.SetUnbondingMax(value)
	assert.NoError(t, err)
	assert.Equal(t, value, state.GetUnbondingMax())

	assert.NoError(t, state.Flush())
	state.ClearCache()
	assert.Equal(t, value, state.GetUnbondingMax())

	for _, v := range []int64{-1, 0} {
		err = state.SetUnbondingMax(v)
		assert.Error(t, err)
		assert.Equal(t, value, state.GetUnbondingMax())

		assert.NoError(t, state.Flush())
		state.ClearCache()
		assert.Equal(t, value, state.GetUnbondingMax())
	}
}

func TestState_SetValidationPenaltyCondition(t *testing.T) {
	state := newDummyState(false)

	err := state.SetValidationPenaltyCondition(10)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), state.GetValidationPenaltyCondition())

	assert.NoError(t, state.Flush())
	state.ClearCache()
	assert.Equal(t, int64(10), state.GetValidationPenaltyCondition())

	for _, v := range []int{-1, 0} {
		err = state.SetValidationPenaltyCondition(v)
		assert.Error(t, err)
		assert.Equal(t, int64(10), state.GetValidationPenaltyCondition())

		assert.NoError(t, state.Flush())
		state.ClearCache()
		assert.Equal(t, int64(10), state.GetValidationPenaltyCondition())
	}
}

func TestState_SetConsistentValidationPenaltyCondition(t *testing.T) {
	state := newDummyState(false)
	assert.Equal(t, int64(0), state.GetConsistentValidationPenaltyCondition())

	for _, value := range []int64{1, 15, 30} {
		err := state.SetConsistentValidationPenaltyCondition(value)
		assert.NoError(t, err)
		assert.Equal(t, value, state.GetConsistentValidationPenaltyCondition())

		assert.NoError(t, state.Flush())
		state.ClearCache()
		assert.Equal(t, value, state.GetConsistentValidationPenaltyCondition())
	}

	expValue := int64(30)
	for _, value := range []int64{-1, 0} {
		err := state.SetConsistentValidationPenaltyCondition(value)
		assert.Error(t, err)
		assert.Equal(t, expValue, state.GetConsistentValidationPenaltyCondition())

		assert.NoError(t, state.Flush())
		state.ClearCache()
		assert.Equal(t, expValue, state.GetConsistentValidationPenaltyCondition())
	}
}

func TestState_SetConsistentValidationPenaltyMask(t *testing.T) {
	state := newDummyState(false)
	assert.Equal(t, 0, state.GetConsistentValidationPenaltyMask())

	for _, mask := range []int64{1, 15, 30} {
		err := state.SetConsistentValidationPenaltyMask(mask)
		assert.NoError(t, err)
		assert.Equal(t, int(mask), state.GetConsistentValidationPenaltyMask())

		assert.NoError(t, state.Flush())
		state.ClearCache()
		assert.Equal(t, int(mask), state.GetConsistentValidationPenaltyMask())
	}

	expValue := 30
	for _, value := range []int64{-1, 0, 31} {
		err := state.SetConsistentValidationPenaltyMask(value)
		assert.Error(t, err)
		assert.Equal(t, expValue, state.GetConsistentValidationPenaltyMask())

		assert.NoError(t, state.Flush())
		state.ClearCache()
		assert.Equal(t, expValue, state.GetConsistentValidationPenaltyMask())
	}
}

func TestState_SetConsistentValidationPenaltySlashRate(t *testing.T) {
	var slashingRate icmodule.Rate
	rev := icmodule.RevisionIISS4R0 - 1
	pt := icmodule.PenaltyAccumulatedValidationFailure

	state := newDummyState(false)

	assert.Equal(t, icmodule.Rate(0), state.getConsistentValidationPenaltySlashRate())
	slashingRate, _ = state.GetSlashingRate(rev, pt)
	assert.Equal(t, icmodule.Rate(0), slashingRate)

	rates := []icmodule.Rate{
		icmodule.ToRate(0),
		icmodule.ToRate(50),
		icmodule.ToRate(100),
	}
	for _, rate := range rates {
		err := state.SetSlashingRate(rev, pt, rate)
		assert.NoError(t, err)
		assert.Equal(t, rate, state.getConsistentValidationPenaltySlashRate())
		slashingRate, _ = state.GetSlashingRate(rev, pt)
		assert.Equal(t, rate, slashingRate)

		assert.NoError(t, state.Flush())
		state.ClearCache()
		assert.Equal(t, rate, state.getConsistentValidationPenaltySlashRate())
		slashingRate, _ = state.GetSlashingRate(rev, pt)
		assert.Equal(t, rate, slashingRate)
	}

	expRate := rates[2]
	for _, rate := range []icmodule.Rate{
		icmodule.ToRate(-10),
		icmodule.ToRate(101),
	} {
		err := state.SetSlashingRate(rev, pt, rate)
		assert.Error(t, err)
		assert.Equal(t, expRate, state.getConsistentValidationPenaltySlashRate())
		slashingRate, _ = state.GetSlashingRate(rev, pt)
		assert.Equal(t, expRate, slashingRate)

		assert.NoError(t, state.Flush())
		state.ClearCache()
		assert.Equal(t, expRate, state.getConsistentValidationPenaltySlashRate())
		slashingRate, _ = state.GetSlashingRate(rev, pt)
		assert.Equal(t, expRate, slashingRate)
	}
}

func TestState_SetDelegationSlotMax(t *testing.T) {
	state := newDummyState(false)
	assert.Equal(t, 0, state.GetDelegationSlotMax())

	for _, slot := range []int64{10, 20, 100} {
		err := state.SetDelegationSlotMax(slot)
		assert.NoError(t, err)
		assert.Equal(t, int(slot), state.GetDelegationSlotMax())

		assert.NoError(t, state.Flush())
		state.ClearCache()
		assert.Equal(t, int(slot), state.GetDelegationSlotMax())
	}
}

func TestState_SetNonVotePenaltySlashRate(t *testing.T) {
	var slashingRate icmodule.Rate
	state := newDummyState(false)

	for _, rev := range []int{icmodule.RevisionIISS4R0 - 2, icmodule.RevisionIISS4R0 - 1} {
		assert.Equal(t, icmodule.Rate(0), state.getNonVotePenaltySlashRate())

		for _, rate := range []icmodule.Rate{
			icmodule.ToRate(-1),
			icmodule.ToRate(101),
		} {
			err := state.setNonVotePenaltySlashRate(rate)
			assert.Error(t, err)
			assert.Equal(t, icmodule.Rate(0), state.getNonVotePenaltySlashRate())
			slashingRate, _ = state.GetSlashingRate(rev, icmodule.PenaltyMissedNetworkProposalVote)
			assert.Equal(t, icmodule.Rate(0), slashingRate)

			assert.NoError(t, state.Flush())
			state.ClearCache()
			assert.Equal(t, icmodule.Rate(0), state.getNonVotePenaltySlashRate())
			slashingRate, _ = state.GetSlashingRate(rev, icmodule.PenaltyMissedNetworkProposalVote)
			assert.Equal(t, icmodule.Rate(0), slashingRate)
		}

		for _, rate := range []icmodule.Rate{
			icmodule.ToRate(100),
			icmodule.ToRate(50),
			icmodule.ToRate(0),
		} {
			err := state.SetSlashingRate(rev, icmodule.PenaltyMissedNetworkProposalVote, rate)
			assert.NoError(t, err)

			assert.Equal(t, rate, state.getNonVotePenaltySlashRate())
			slashingRate, _ = state.GetSlashingRate(rev, icmodule.PenaltyMissedNetworkProposalVote)
			assert.Equal(t, rate, slashingRate)

			assert.NoError(t, state.Flush())
			state.ClearCache()
			assert.Equal(t, rate, state.getNonVotePenaltySlashRate())
			slashingRate, _ = state.GetSlashingRate(rev, icmodule.PenaltyMissedNetworkProposalVote)
			assert.Equal(t, rate, slashingRate)
		}
	}
}

func TestState_SetSlashingRate(t *testing.T) {
	args := []struct {
		penaltyType icmodule.PenaltyType
		initRate    icmodule.Rate
		rate        icmodule.Rate
	}{
		{
			icmodule.PenaltyPRepDisqualification,
			icmodule.ToRate(100), icmodule.ToRate(100),
		},
		{
			icmodule.PenaltyAccumulatedValidationFailure,
			icmodule.ToRate(0), icmodule.Rate(1),
		},
		{
			icmodule.PenaltyValidationFailure,
			icmodule.ToRate(0), icmodule.Rate(0),
		},
		{
			icmodule.PenaltyMissedNetworkProposalVote,
			icmodule.ToRate(0), icmodule.Rate(1),
		},
		{
			icmodule.PenaltyDoubleSign,
			icmodule.ToRate(0), icmodule.Rate(1000),
		},
	}

	// Not exists -> Rate(1)
	for _, rev := range []int{icmodule.RevisionIISS4R0, icmodule.RevisionIISS4R1} {
		state := newDummyState(false)

		for i, in := range args {
			name := fmt.Sprintf("case1-%02d-%s", i, in.penaltyType)
			t.Run(name, func(t *testing.T) {
				rate, err := state.GetSlashingRate(rev, in.penaltyType)
				assert.NoError(t, err)
				assert.Equal(t, in.initRate, rate)

				err = state.SetSlashingRate(rev, in.penaltyType, in.rate)
				assert.NoError(t, err)

				rate, err = state.GetSlashingRate(rev, in.penaltyType)
				assert.NoError(t, err)
				assert.Equal(t, in.rate, rate)
			})
		}

		for i, in := range args {
			name := fmt.Sprintf("case2-%02d-%s", i, in.penaltyType)
			t.Run(name, func(t *testing.T) {
				oldRate := in.rate
				in.rate = icmodule.Rate(70)
				rate, err := state.GetSlashingRate(rev, in.penaltyType)
				assert.NoError(t, err)
				assert.Equal(t, oldRate, rate)

				err = state.SetSlashingRate(rev, in.penaltyType, in.rate)
				assert.NoError(t, err)

				rate, err = state.GetSlashingRate(rev, in.penaltyType)
				assert.NoError(t, err)
				assert.Equal(t, in.rate, rate)
			})
		}
	}
}

func TestState_SetMinimumBond(t *testing.T) {
	s := newDummyState(false)

	bond := s.GetMinimumBond()
	assert.Zero(t, bond.Sign())

	args := []struct {
		bond    *big.Int
		success bool
	}{
		{nil, false},
		{big.NewInt(-1), false},
		{big.NewInt(-1000), false},
		{icmodule.BigIntZero, true},
		{big.NewInt(1000), true},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name_%02d_%s", i, arg.bond)
		prevBond := s.GetMinimumBond()

		t.Run(name, func(t *testing.T) {
			err := s.SetMinimumBond(arg.bond)
			if arg.success {
				assert.NoError(t, err)
				assert.Zero(t, arg.bond.Cmp(s.GetMinimumBond()))
			} else {
				assert.Error(t, err)
				assert.Zero(t, prevBond.Cmp(s.GetMinimumBond()))
			}
		})
	}
}

func TestState_SetIISSVersion(t *testing.T) {
	state := newDummyState(false)

	for _, ver := range []int{IISSVersion2, IISSVersion3, IISSVersion4} {
		assert.NoError(t, state.SetIISSVersion(ver))
		assert.Equal(t, ver, state.GetIISSVersion())
	}
}

func TestState_SetRRep(t *testing.T) {
	rrep := big.NewInt(1000)
	state := newDummyState(false)
	assert.NoError(t, state.SetRRep(rrep))
	assert.Zero(t, rrep.Cmp(state.GetRRep()))
}

func TestState_GetPRepCountConfig(t *testing.T) {
	const (
		main  = 22
		sub   = 78
		extra = 3
	)
	state := newDummyState(false)
	assert.NoError(t, state.SetMainPRepCount(main))
	assert.NoError(t, state.SetSubPRepCount(sub))
	assert.NoError(t, state.SetExtraMainPRepCount(extra))

	for _, rev := range []int{icmodule.RevisionExtraMainPReps - 1, icmodule.RevisionExtraMainPReps} {
		cfg := state.GetPRepCountConfig(rev)
		assert.Equal(t, main, cfg.MainPReps())
		assert.Equal(t, sub, cfg.SubPReps())
		assert.Equal(t, main+sub, cfg.ElectedPReps())
		if rev < icmodule.RevisionExtraMainPReps {
			assert.Zero(t, cfg.ExtraMainPReps())
		} else {
			assert.Equal(t, extra, cfg.ExtraMainPReps())
		}
	}
}

func TestState_GetNetworkInfoInJSON(t *testing.T) {
	irep := big.NewInt(100)
	rrep := big.NewInt(200)
	minBond := big.NewInt(1234)
	rates := []icmodule.Rate{icmodule.ToRate(1), icmodule.ToRate(5)}

	state := newDummyState(false)
	assert.NoError(t, state.SetIRep(irep))
	assert.NoError(t, state.SetRRep(rrep))
	assert.NoError(t, state.SetMinimumBond(minBond))
	assert.NoError(t, state.SetSlashingRate(
		icmodule.RevisionIISS4R0-1, icmodule.PenaltyAccumulatedValidationFailure, rates[0]))
	assert.NoError(t, state.SetSlashingRate(
		icmodule.RevisionIISS4R0-1, icmodule.PenaltyMissedNetworkProposalVote, rates[1]))

	for _, rev := range []int{icmodule.RevisionIISS4R0 - 1, icmodule.RevisionIISS4R0, icmodule.RevisionIISS4R1} {
		jso, err := state.GetNetworkInfoInJSON(rev)
		assert.NoError(t, err)
		if rev < icmodule.RevisionIISS4R0 {
			irep2 := jso["irep"].(*big.Int)
			assert.Zero(t, irep.Cmp(irep2))
			rrep2 := jso["rrep"].(*big.Int)
			assert.Zero(t, rrep.Cmp(rrep2))
			assert.Equal(t, rates[0].Percent(), jso["consistentValidationPenaltySlashRatio"])
			assert.Equal(t, rates[1].Percent(), jso["proposalNonVotePenaltySlashRatio"])

			_, ok := jso["minimumBond"]
			assert.False(t, ok)
		} else {
			keys := []string{
				"irep",
				"rrep",
				"consistentValidationPenaltySlashRatio",
				"proposalNonVotePenaltySlashRatio",
			}
			for _, key := range keys {
				_, ok := jso[key]
				assert.False(t, ok)
			}

			mb := jso["minimumBond"].(*big.Int)
			assert.Zero(t, mb.Cmp(minBond))
		}
	}
}
