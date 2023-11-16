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

	"github.com/icon-project/goloop/icon/icmodule"
)

var bigOne = big.NewInt(1)

func TestUnstake(t *testing.T) {
	amount := big.NewInt(12)
	eh := int64(100)

	u1 := NewUnstake(amount, eh)
	u2 := u1.Clone()

	assert.True(t, u1.Equal(u2))
}

func TestUnstakes(t *testing.T) {
	revision := icmodule.RevisionMultipleUnstakes
	a1 := int64(5)
	a2 := int64(10)
	a3 := int64(20)
	eh1 := int64(10)
	eh2 := int64(20)
	eh3 := int64(30)

	u1 := NewUnstake(big.NewInt(a1), eh1)
	u2 := NewUnstake(big.NewInt(a2), eh2)
	u3 := NewUnstake(big.NewInt(a3), eh3)

	var us1 = Unstakes{u1, u2, u3}

	us2 := us1.Clone()

	assert.True(t, us1.Equal(us2))
	assert.True(t, !us1.IsEmpty())
	assert.Equal(t, a1+a2+a3, us1.GetUnstakeAmount().Int64())

	t.Run("increase Unstakes", func(t *testing.T) {
		unstakes := Unstakes{}
		unstakeSlotMax := 2

		// add unstake u1
		tl, err := unstakes.increaseUnstake(u1.GetValue(), u1.GetExpire(), unstakeSlotMax, revision)
		assert.NoError(t, err)
		assert.True(t, !unstakes.IsEmpty())
		assert.Equal(t, 1, len(unstakes))
		assert.True(t, u1.Equal(unstakes[0]))
		assert.Equal(t, a1, unstakes.GetUnstakeAmount().Int64())
		assert.Equal(t, len(tl), 1)
		assert.Equal(t, tl[0].Type, JobTypeAdd)
		assert.Equal(t, tl[0].Height, u1.GetExpire())

		// add unstake u2
		tl, err = unstakes.increaseUnstake(u2.GetValue(), u2.GetExpire(), unstakeSlotMax, revision)
		assert.NoError(t, err)
		assert.True(t, !unstakes.IsEmpty())
		assert.Equal(t, 2, len(unstakes))
		assert.True(t, u2.Equal(unstakes[1]))
		assert.Equal(t, a1+a2, unstakes.GetUnstakeAmount().Int64())
		assert.Equal(t, len(tl), 1)
		assert.Equal(t, tl[0].Type, JobTypeAdd)
		assert.Equal(t, tl[0].Height, u2.GetExpire())

		// update last unstake
		tl, err = unstakes.increaseUnstake(u3.GetValue(), u3.GetExpire(), unstakeSlotMax, revision)
		assert.NoError(t, err)
		assert.True(t, !unstakes.IsEmpty())
		assert.Equal(t, 2, len(unstakes))
		assert.Equal(t, a2+a3, unstakes[1].GetValue().Int64())
		assert.Equal(t, eh3, unstakes[1].GetExpire())
		assert.Equal(t, a1+a2+a3, unstakes.GetUnstakeAmount().Int64())
		assert.Equal(t, len(tl), 2)
		assert.Equal(t, tl[0].Type, JobTypeRemove)
		assert.Equal(t, tl[0].Height, u2.GetExpire())
		assert.Equal(t, tl[1].Type, JobTypeAdd)
		assert.Equal(t, tl[1].Height, u3.GetExpire())
	})

	t.Run("decrease Unstakes", func(t *testing.T) {
		noMeaning := int64(0)
		unstakes := Unstakes{}
		unstakeSlotMax := 3
		_, err := unstakes.increaseUnstake(u1.GetValue(), u1.GetExpire(), unstakeSlotMax, revision)
		assert.NoError(t, err)
		_, err = unstakes.increaseUnstake(u2.GetValue(), u2.GetExpire(), unstakeSlotMax, revision)
		assert.NoError(t, err)
		_, err = unstakes.increaseUnstake(u3.GetValue(), u3.GetExpire(), unstakeSlotMax, revision)
		assert.NoError(t, err)

		total := a1 + a2 + a3
		// decrease Value of slot
		_, err = unstakes.decreaseUnstake(bigOne, noMeaning, revision)
		assert.NoError(t, err)
		assert.True(t, !unstakes.IsEmpty())
		assert.Equal(t, 3, len(unstakes))
		assert.Equal(t, total-bigOne.Int64(), unstakes.GetUnstakeAmount().Int64())

		// delete 1 slot
		_, err = unstakes.decreaseUnstake(new(big.Int).Sub(u3.GetValue(), bigOne), noMeaning, revision)
		assert.NoError(t, err)
		assert.True(t, !unstakes.IsEmpty())
		assert.Equal(t, 2, len(unstakes))
		assert.Equal(t, total-a3, unstakes.GetUnstakeAmount().Int64())

		// delete 1 slot and decrease 1 slot
		_, err = unstakes.decreaseUnstake(new(big.Int).Add(u2.GetValue(), bigOne), noMeaning, revision)
		assert.NoError(t, err)
		assert.True(t, !unstakes.IsEmpty())
		assert.Equal(t, 1, len(unstakes))
		assert.Equal(t, a1-bigOne.Int64(), unstakes.GetUnstakeAmount().Int64())

		// > total unstake. delete all
		_, err = unstakes.decreaseUnstake(u1.GetValue(), noMeaning, revision)
		assert.NoError(t, err)
		assert.False(t, !unstakes.IsEmpty())
		assert.Equal(t, 0, len(unstakes))
		assert.Equal(t, int64(0), unstakes.GetUnstakeAmount().Int64())
	})
}

func TestIncreaseUnstake_multiple(t *testing.T) {
	revision := icmodule.RevisionMultipleUnstakes
	unstakeSlotMax := 3
	a0 := int64(5)
	a1 := int64(10)
	a2 := int64(20)
	a3 := int64(30)
	eh0 := int64(10)
	eh1 := int64(20)
	eh2 := int64(30)
	eh3 := int64(40)

	u0 := NewUnstake(big.NewInt(a0), eh0)
	u1 := NewUnstake(big.NewInt(a1), eh1)
	u2 := NewUnstake(big.NewInt(a2), eh2)
	u3 := NewUnstake(big.NewInt(a3), eh3)

	us := Unstakes{u1}

	// u0 will place in 0 index(front of u1)
	_, err := us.increaseUnstake(big.NewInt(a0), eh0, unstakeSlotMax, revision)
	assert.NoError(t, err)
	assert.True(t, u0.Equal(us[0]))
	assert.True(t, u1.Equal(us[1]))

	_, err = us.increaseUnstake(big.NewInt(a2), eh2, unstakeSlotMax, revision)
	assert.NoError(t, err)
	assert.True(t, u0.Equal(us[0]))
	assert.True(t, u1.Equal(us[1]))
	assert.True(t, u2.Equal(us[2]))

	// unstake of last index will be updated
	_, err = us.increaseUnstake(big.NewInt(a3-a2), eh3, unstakeSlotMax, revision)
	assert.NoError(t, err)
	assert.True(t, u0.Equal(us[0]))
	assert.True(t, u1.Equal(us[1]))
	assert.True(t, u3.Equal(us[2]))
}

func TestDecreaseUnstake_multiple(t *testing.T) {
	revision := icmodule.RevisionMultipleUnstakes
	noMeaning := int64(0)
	a0 := int64(5)
	a1 := int64(10)
	a2 := int64(20)
	a3 := int64(30)
	a4 := int64(40)
	eh0 := int64(10)
	eh1 := int64(20)
	eh2 := int64(30)
	eh3 := int64(40)
	eh4 := int64(50)

	u0 := NewUnstake(big.NewInt(a0), eh0)
	u1 := NewUnstake(big.NewInt(a1), eh1)
	u2 := NewUnstake(big.NewInt(a2), eh2)
	u3 := NewUnstake(big.NewInt(a3), eh3)
	u4 := NewUnstake(big.NewInt(a4), eh4)

	us := Unstakes{u0, u1, u2, u3, u4}
	assert.Equal(t, len(us), 5)

	// remove last unstake
	j, err := us.decreaseUnstake(u4.GetValue(), noMeaning, revision)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(us))
	assert.True(t, us[0].Equal(u0))
	assert.Equal(t, 1, len(j))
	assert.Equal(t, eh4, j[0].Height)

	// remove 2 unstakes
	j, err = us.decreaseUnstake(new(big.Int).Add(u2.GetValue(), u3.GetValue()), noMeaning, revision)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(us))
	assert.True(t, us[0].Equal(u0))
	assert.True(t, us[1].Equal(u1))
	assert.Equal(t, 2, len(j))
	assert.Equal(t, eh3, j[0].Height)
	assert.Equal(t, eh2, j[1].Height)

	// remove last unstake and decrease first unstake
	v := big.NewInt(a1 + 1)
	j, err = us.decreaseUnstake(v, noMeaning, revision)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(us))
	expectedUnstake := NewUnstake(big.NewInt(a0-1), eh0)
	assert.True(t, us[0].Equal(expectedUnstake))
	assert.Equal(t, 1, len(j))
	assert.Equal(t, eh1, j[0].Height)
}

func TestIncreaseUnstake_single(t *testing.T) {
	unstakeSlotMax := 1
	revision := icmodule.RevisionMultipleUnstakes - 1
	a0 := int64(5)
	a1 := int64(10)
	eh0 := int64(10)
	eh1 := int64(20)

	u0 := NewUnstake(big.NewInt(a0), eh0)
	u1 := NewUnstake(big.NewInt(a1), eh1)

	us := Unstakes{}

	// add unstakes
	_, err := us.increaseUnstake(u0.GetValue(), u0.GetExpire(), unstakeSlotMax, revision)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(us))
	assert.True(t, u0.Equal(us[0]))

	// update unstakes
	_, err = us.increaseUnstake(u1.GetValue(), u1.GetExpire(), unstakeSlotMax, revision)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(us))
	assert.Equal(t, u0.GetValue().Int64()+u1.GetValue().Int64(), us[0].GetValue().Int64())
	assert.Equal(t, u1.GetExpire(), us[0].GetExpire())
}

func TestDecreaseUnstake_single(t *testing.T) {
	revision := icmodule.RevisionMultipleUnstakes - 1
	a0 := int64(100)
	a1 := int64(50)
	eh0 := int64(10)
	eh1 := int64(20)

	u0 := NewUnstake(big.NewInt(a0), eh0)
	u1 := NewUnstake(big.NewInt(a1), eh1)

	us := Unstakes{u0.Clone()}
	assert.Equal(t, len(us), 1)
	assert.True(t, u0.Equal(us[0]))

	// update unstake
	_, err := us.decreaseUnstake(u1.GetValue(), u1.GetExpire(), revision)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(us))
	assert.Equal(t, u0.GetValue().Int64()-u1.GetValue().Int64(), us[0].GetValue().Int64())
	assert.Equal(t, u1.GetExpire(), us[0].GetExpire())

	// remove unstake
	_, err = us.decreaseUnstake(u0.GetValue(), u0.GetExpire(), revision)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(us))
}

func TestCalcUnstakeLockPeriod(t *testing.T) {
	termPeriod := int64(icmodule.DayBlock)
	lMin := big.NewInt(5 * termPeriod)
	lMax := big.NewInt(20 * termPeriod)
	totalSupply := big.NewInt(100)

	prevPeriod := int64(0)
	for i := 0; i <= 10; i++ {
		totalStake := big.NewInt(int64(i * 10))
		periodInBlock := CalcUnstakeLockPeriod(lMin, lMax, totalStake, totalSupply)
		assert.True(t, periodInBlock <= lMax.Int64())
		assert.True(t, periodInBlock >= lMin.Int64())

		if i == 0 {
			assert.True(t, periodInBlock == lMax.Int64())
		} else if i < 8 {
			assert.True(t, periodInBlock < prevPeriod)
		} else {
			assert.True(t, periodInBlock == lMin.Int64())
		}

		prevPeriod = periodInBlock
	}
}
