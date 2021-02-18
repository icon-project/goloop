/*
 * Copyright 2020 ICON Foundation
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package icstate

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func testFactory(tag icobject.Tag) (icobject.Impl, error) {
	return nil, errors.New("Unsupported")
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
}

func setTermPeriodTest(t *testing.T, s *State) {
	actual := s.GetTermPeriod()
	assert.Equal(t, int64(0), actual)

	tp := int64(10)
	s.SetTermPeriod(tp)
	actual = s.GetTermPeriod()
	assert.Equal(t, tp, actual)

}

func setIRepTest(t *testing.T, s *State) {
	actual := s.GetIRep()
	assert.Nil(t, actual)

	irep := big.NewInt(10)
	s.SetIRep(irep)
	actual = s.GetIRep()
	assert.Equal(t, 0, actual.Cmp(irep))
}

func setRRepTest(t *testing.T, s *State) {
	actual := s.GetRRep()
	assert.Nil(t, actual)

	rrep := big.NewInt(10)
	s.SetIRep(rrep)
	actual = s.GetIRep()
	assert.Equal(t, 0, actual.Cmp(rrep))
}

func setMainPRepCountTest(t *testing.T, s *State) {
	count := int64(0)
	actual := s.GetMainPRepCount()
	sCount := s.GetMainPRepCount()
	assert.Equal(t, count, actual)
	assert.Equal(t, count, sCount)

	count = int64(10)
	s.SetMainPRepCount(count)
	actual = s.GetMainPRepCount()
	sCount = s.GetMainPRepCount()
	assert.Equal(t, count, actual)
	assert.Equal(t, count, sCount)
}

func setSubPRepCountTest(t *testing.T, s *State) {
	count := int64(0)
	actual := s.GetSubPRepCount()
	sCount := s.GetSubPRepCount()
	assert.Equal(t, count, actual)
	assert.Equal(t, count, sCount)

	count = int64(20)
	s.SetSubPRepCount(count)
	actual = s.GetSubPRepCount()
	sCount = s.GetSubPRepCount()
	assert.Equal(t, count, actual)
	assert.Equal(t, count, sCount)
}

func setTotalStakeTest(t *testing.T, s *State) {
	ts := new(big.Int)
	actual := s.GetTotalStake()
	assert.Equal(t, 0, actual.Cmp(ts))

	ts = big.NewInt(20)
	s.SetTotalStake(ts)
	actual = s.GetTotalStake()
	assert.Equal(t, 0, actual.Cmp(ts))
}

func setBondRequirementTest(t *testing.T, s *State) {
	br := int64(0)
	actual := s.GetBondRequirement()
	assert.Equal(t, br, actual)

	br = 5
	s.SetBondRequirement(br)
	actual = s.GetBondRequirement()
	assert.Equal(t, br, actual)

	br = 0
	err := s.SetBondRequirement(br)
	assert.NoError(t, err)
	actual = s.GetBondRequirement()
	assert.Equal(t, br, actual)
}

func setLockVariablesTest(t *testing.T, s *State) {
	actualMin := s.GetLockMin()
	actualMax := s.GetLockMax()
	assert.Nil(t, actualMin)
	assert.Nil(t, actualMax)

	min := big.NewInt(10)
	max := big.NewInt(1)
	err := s.SetLockVariables(min, max)
	assert.Error(t, err)
	actualMin = s.GetLockMin()
	actualMax = s.GetLockMax()
	assert.Nil(t, actualMin)
	assert.Nil(t, actualMax)

	min = big.NewInt(1)
	max = big.NewInt(10)
	err = s.SetLockVariables(min, max)
	assert.NoError(t, err)
	actualMin = s.GetLockMin()
	actualMax = s.GetLockMax()
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