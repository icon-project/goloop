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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
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
	t.Run("SetRewardFund", func(t *testing.T) { setRewardFundTest(t, s) })

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
	assert.Nil(t, actual)

	irep := big.NewInt(10)
	assert.NoError(t, s.SetIRep(irep))
	actual = s.GetIRep()
	assert.Equal(t, 0, actual.Cmp(irep))
}

func setRRepTest(t *testing.T, s *State) {
	actual := s.GetRRep()
	assert.Nil(t, actual)

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
	br := icutils.PercentToRate(0)
	actual := s.GetBondRequirement()
	assert.Equal(t, br, actual)

	br = icutils.PercentToRate(5)
	assert.NoError(t, s.SetBondRequirement(br))
	actual = s.GetBondRequirement()
	assert.Equal(t, br, actual)

	br = icutils.PercentToRate(0)
	err := s.SetBondRequirement(br)
	assert.NoError(t, err)
	actual = s.GetBondRequirement()
	assert.Equal(t, br, actual)

	err = s.SetBondRequirement(icutils.PercentToRate(101))
	assert.Error(t, err)
	actual = s.GetBondRequirement()
	assert.Equal(t, br, actual)

	err = s.SetBondRequirement(icutils.PercentToRate(-1))
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

func setRewardFundTest(t *testing.T, s *State) {
	rf := NewRewardFund()
	actual := s.GetRewardFund()
	assert.Equal(t, rf, actual)

	rf.Iglobal.SetInt64(100000)
	rf.Iprep.SetInt64(50)
	rf.Ivoter.SetInt64(50)
	err := s.SetRewardFund(rf)
	assert.NoError(t, err)
	actual = s.GetRewardFund()
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
	state.Flush()
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

	state.Flush()
	state.ClearCache()
	scores = state.GetNetworkScores(cc)
	assert.False(t, scores[GovernanceKey].Equal(gov))

	invalidRole := "invalidRole"
	score := common.MustNewAddressFromString("cx5678")
	err = state.SetNetworkScore(invalidRole, score)
	assert.Error(t, err)
	scores = state.GetNetworkScores(cc)
	assert.Nil(t, scores[invalidRole])

	state.Flush()
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
	state.Flush()
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

	state.Flush()
	state.ClearCache()
	assert.Equal(t, value, state.GetUnbondingMax())

	for _, v := range []int64{-1, 0} {
		err = state.SetUnbondingMax(v)
		assert.Error(t, err)
		assert.Equal(t, value, state.GetUnbondingMax())

		state.Flush()
		state.ClearCache()
		assert.Equal(t, value, state.GetUnbondingMax())
	}
}

func TestState_SetValidationPenaltyCondition(t *testing.T) {
	state := newDummyState(false)

	err := state.SetValidationPenaltyCondition(10)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), state.GetValidationPenaltyCondition())

	state.Flush()
	state.ClearCache()
	assert.Equal(t, int64(10), state.GetValidationPenaltyCondition())

	for _, v := range []int{-1, 0} {
		err = state.SetValidationPenaltyCondition(v)
		assert.Error(t, err)
		assert.Equal(t, int64(10), state.GetValidationPenaltyCondition())

		state.Flush()
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

		state.Flush()
		state.ClearCache()
		assert.Equal(t, value, state.GetConsistentValidationPenaltyCondition())
	}

	expValue := int64(30)
	for _, value := range []int64{-1, 0} {
		err := state.SetConsistentValidationPenaltyCondition(value)
		assert.Error(t, err)
		assert.Equal(t, expValue, state.GetConsistentValidationPenaltyCondition())

		state.Flush()
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

		state.Flush()
		state.ClearCache()
		assert.Equal(t, int(mask), state.GetConsistentValidationPenaltyMask())
	}

	expValue := 30
	for _, value := range []int64{-1, 0, 31} {
		err := state.SetConsistentValidationPenaltyMask(value)
		assert.Error(t, err)
		assert.Equal(t, expValue, state.GetConsistentValidationPenaltyMask())

		state.Flush()
		state.ClearCache()
		assert.Equal(t, expValue, state.GetConsistentValidationPenaltyMask())
	}
}

func TestState_SetConsistentValidationPenaltySlashRatio(t *testing.T) {
	state := newDummyState(false)
	assert.Equal(t, icmodule.Rate(0), state.GetConsistentValidationPenaltySlashRatio())

	ratios := []icmodule.Rate{
		icutils.PercentToRate(0),
		icutils.PercentToRate(50),
		icutils.PercentToRate(100),
	}
	for _, ratio := range ratios {
		err := state.SetConsistentValidationPenaltySlashRatio(ratio)
		assert.NoError(t, err)
		assert.Equal(t, ratio, state.GetConsistentValidationPenaltySlashRatio())

		state.Flush()
		state.ClearCache()
		assert.Equal(t, ratio, state.GetConsistentValidationPenaltySlashRatio())
	}

	expRatio := ratios[2]
	for _, ratio := range []icmodule.Rate{
		icutils.PercentToRate(-10),
		icutils.PercentToRate(101),
	} {
		err := state.SetConsistentValidationPenaltySlashRatio(ratio)
		assert.Error(t, err)
		assert.Equal(t, expRatio, state.GetConsistentValidationPenaltySlashRatio())

		state.Flush()
		state.ClearCache()
		assert.Equal(t, expRatio, state.GetConsistentValidationPenaltySlashRatio())
	}
}

func TestState_SetDelegationSlotMax(t *testing.T) {
	state := newDummyState(false)
	assert.Equal(t, 0, state.GetDelegationSlotMax())

	for _, slot := range []int64{10, 20, 100} {
		err := state.SetDelegationSlotMax(slot)
		assert.NoError(t, err)
		assert.Equal(t, int(slot), state.GetDelegationSlotMax())

		state.Flush()
		state.ClearCache()
		assert.Equal(t, int(slot), state.GetDelegationSlotMax())
	}
}

func TestState_SetNonVotePenaltySlashRatio(t *testing.T) {
	state := newDummyState(false)
	assert.Equal(t, icmodule.Rate(0), state.GetNonVotePenaltySlashRatio())

	for _, ratio := range []icmodule.Rate{
		icutils.PercentToRate(-1),
		icutils.PercentToRate(101),
	} {
		err := state.SetNonVotePenaltySlashRatio(ratio)
		assert.Error(t, err)
		assert.Equal(t, icmodule.Rate(0), state.GetNonVotePenaltySlashRatio())

		state.Flush()
		state.ClearCache()
		assert.Equal(t, icmodule.Rate(0), state.GetNonVotePenaltySlashRatio())
	}

	for _, ratio := range []icmodule.Rate{
		icutils.PercentToRate(100),
		icutils.PercentToRate(50),
		icutils.PercentToRate(0),
	} {
		err := state.SetNonVotePenaltySlashRatio(ratio)
		assert.NoError(t, err)
		assert.Equal(t, ratio, state.GetNonVotePenaltySlashRatio())

		state.Flush()
		state.ClearCache()
		assert.Equal(t, ratio, state.GetNonVotePenaltySlashRatio())
	}
}
