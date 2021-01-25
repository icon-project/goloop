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

	// test for SetCalculatePeriod
	t.Run("SetCalculatePeriod", func(t *testing.T) { setCalculatePeriodTest(t, s) })

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
}

func setTermPeriodTest(t *testing.T, s *State) {
	actual := GetTermPeriod(s)
	assert.Equal(t, int64(0), actual)

	tp := int64(10)
	SetTermPeriod(s, tp)
	actual = GetTermPeriod(s)
	assert.Equal(t, tp, actual)

}

func setCalculatePeriodTest(t *testing.T, s *State) {
	cp := int64(0)
	actual := GetCalculatePeriod(s)
	assert.Equal(t, actual, cp)

	cp = int64(10)
	SetCalculatePeriod(s, cp)
	actual = GetCalculatePeriod(s)
	assert.Equal(t, actual, cp)
}

func setIRepTest(t *testing.T, s *State) {
	actual := GetIRep(s)
	assert.Nil(t, actual)

	irep := big.NewInt(10)
	SetIRep(s, irep)
	actual = GetIRep(s)
	assert.Equal(t, 0, actual.Cmp(irep))
}

func setRRepTest(t *testing.T, s *State) {
	actual := GetRRep(s)
	assert.Nil(t, actual)

	rrep := big.NewInt(10)
	SetIRep(s, rrep)
	actual = GetIRep(s)
	assert.Equal(t, 0, actual.Cmp(rrep))
}

func setMainPRepCountTest(t *testing.T, s *State) {
	count := int64(0)
	actual := GetMainPRepCount(s)
	sCount := s.GetMainPRepCount()
	assert.Equal(t, count, actual)
	assert.Equal(t, int(count), sCount)

	count = int64(10)
	SetMainPRepCount(s, count)
	actual = GetMainPRepCount(s)
	sCount = s.GetMainPRepCount()
	assert.Equal(t, count, actual)
	assert.Equal(t, int(count), sCount)
}

func setSubPRepCountTest(t *testing.T, s *State) {
	count := int64(0)
	actual := GetSubPRepCount(s)
	sCount := s.GetSubPRepCount()
	assert.Equal(t, count, actual)
	assert.Equal(t, int(count), sCount)

	count = int64(20)
	SetSubPRepCount(s, count)
	actual = GetSubPRepCount(s)
	sCount = s.GetSubPRepCount()
	assert.Equal(t, count, actual)
	assert.Equal(t, int(count), sCount)
}

func setTotalStakeTest(t *testing.T, s *State) {
	ts := new(big.Int)
	actual := GetTotalStake(s)
	assert.Equal(t, 0, actual.Cmp(ts))

	ts = big.NewInt(20)
	SetTotalStake(s, ts)
	actual = GetTotalStake(s)
	assert.Equal(t, 0, actual.Cmp(ts))
}

func setBondRequirementTest(t *testing.T, s *State) {
	br := int64(0)
	actual := GetBondRequirement(s)
	assert.Equal(t, br, actual)

	br = 5
	SetBondRequirement(s, br)
	actual = GetBondRequirement(s)
	assert.Equal(t, br, actual)

	err := SetBondRequirement(s, 0)
	assert.Error(t, err)
	actual = GetBondRequirement(s)
	assert.Equal(t, br, actual)
}

func setLockVariablesTest(t *testing.T, s *State) {
	actualMin := GetLockMin(s)
	actualMax := GetLockMax(s)
	assert.Nil(t, actualMin)
	assert.Nil(t, actualMax)

	min := big.NewInt(10)
	max := big.NewInt(1)
	err := SetLockVariables(s, min, max)
	assert.Error(t, err)
	actualMin = GetLockMin(s)
	actualMax = GetLockMax(s)
	assert.Nil(t, actualMin)
	assert.Nil(t, actualMax)

	min = big.NewInt(1)
	max = big.NewInt(10)
	err = SetLockVariables(s, min, max)
	assert.NoError(t, err)
	actualMin = GetLockMin(s)
	actualMax = GetLockMax(s)
	assert.Equal(t, 0, actualMin.Cmp(min))
	assert.Equal(t, 0, actualMax.Cmp(max))
}
