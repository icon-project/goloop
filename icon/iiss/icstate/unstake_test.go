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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

var bigOne = big.NewInt(1)

func TestUnstake(t *testing.T) {
	amount := int64(12)
	eh := int64(100)

	u1 := newUnstake()
	u1.Amount.SetInt64(amount)
	u1.ExpireHeight = eh

	u2 := u1.Clone()

	assert.True(t, u1.Equal(u2))
}

func TestUnstakes(t *testing.T) {
	a1 := int64(5)
	a2 := int64(10)
	a3 := int64(20)
	eh1 := int64(10)
	eh2 := int64(20)
	eh3 := int64(30)

	u1 := Unstake{
		Amount:       big.NewInt(a1),
		ExpireHeight: eh1,
	}
	u2 := Unstake{
		Amount:       big.NewInt(a2),
		ExpireHeight: eh2,
	}
	u3 := Unstake{
		Amount:       big.NewInt(a3),
		ExpireHeight: eh3,
	}

	var us1 = Unstakes{
		&u1, &u2, &u3,
	}

	us2 := us1.Clone()

	assert.True(t, us1.Equal(us2))
	assert.True(t, us1.Has())
	assert.Equal(t, a1+a2+a3, us1.GetUnstakeAmount().Int64())

	t.Run("increase Unstakes", func(t *testing.T) {
		unstakes := Unstakes{}
		unstakeSlotMax := int64(2)

		// add unstake u1
		err := unstakes.increaseUnstake(u1.Amount, u1.ExpireHeight, unstakeSlotMax)
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 1, len(unstakes))
		assert.True(t, u1.Equal(unstakes[0]))
		assert.Equal(t, a1, unstakes.GetUnstakeAmount().Int64())

		// add unstake u2
		err = unstakes.increaseUnstake(u2.Amount, u2.ExpireHeight, unstakeSlotMax)
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 2, len(unstakes))
		assert.True(t, u2.Equal(unstakes[1]))
		assert.Equal(t, a1+a2, unstakes.GetUnstakeAmount().Int64())

		// update last unstake
		err = unstakes.increaseUnstake(u3.Amount, u3.ExpireHeight, unstakeSlotMax)
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 2, len(unstakes))
		assert.Equal(t, a2+a3, unstakes[1].Amount.Int64())
		assert.Equal(t, eh3, unstakes[1].ExpireHeight)
		assert.Equal(t, a1+a2+a3, unstakes.GetUnstakeAmount().Int64())
	})

	t.Run("decrease Unstakes", func(t *testing.T) {
		unstakes := Unstakes{}
		unstakeSlotMax := int64(3)
		err := unstakes.increaseUnstake(u1.Amount, u1.ExpireHeight, unstakeSlotMax)
		assert.NoError(t, err)
		err = unstakes.increaseUnstake(u2.Amount, u2.ExpireHeight, unstakeSlotMax)
		assert.NoError(t, err)
		err = unstakes.increaseUnstake(u3.Amount, u3.ExpireHeight, unstakeSlotMax)
		assert.NoError(t, err)

		total := a1 + a2 + a3
		// decrease Value of slot
		_, err = unstakes.decreaseUnstake(bigOne)
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 3, len(unstakes))
		assert.Equal(t, total-bigOne.Int64(), unstakes.GetUnstakeAmount().Int64())

		// delete 1 slot
		_, err = unstakes.decreaseUnstake(new(big.Int).Sub(u3.Amount, bigOne))
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 2, len(unstakes))
		assert.Equal(t, total-a3, unstakes.GetUnstakeAmount().Int64())

		// delete 1 slot and decrease 1 slot
		_, err = unstakes.decreaseUnstake(new(big.Int).Add(u2.Amount, bigOne))
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 1, len(unstakes))
		assert.Equal(t, a1-bigOne.Int64(), unstakes.GetUnstakeAmount().Int64())

		// > total unstake. delete all
		_, err = unstakes.decreaseUnstake(u1.Amount)
		assert.NoError(t, err)
		assert.False(t, unstakes.Has())
		assert.Equal(t, 0, len(unstakes))
		assert.Equal(t, int64(0), unstakes.GetUnstakeAmount().Int64())
	})
}

func TestIncreaseUnstake(t *testing.T) {
	unstakeSlotMax := int64(3)
	a0 := int64(5)
	a1 := int64(10)
	a2 := int64(20)
	a3 := int64(30)
	eh0 := int64(10)
	eh1 := int64(20)
	eh2 := int64(30)
	eh3 := int64(40)

	u0 := Unstake{Amount: big.NewInt(a0), ExpireHeight: eh0}
	u1 := Unstake{Amount: big.NewInt(a1), ExpireHeight: eh1}
	u2 := Unstake{Amount: big.NewInt(a2), ExpireHeight: eh2}
	u3 := Unstake{Amount: big.NewInt(a3), ExpireHeight: eh3}

	us := Unstakes{&u1}

	//u0 will place in 0 index(front of u1)
	err := us.increaseUnstake(big.NewInt(a0), eh0, unstakeSlotMax)
	assert.NoError(t, err)
	assert.True(t, u0.Equal(us[0]))
	assert.True(t, u1.Equal(us[1]))

	err = us.increaseUnstake(big.NewInt(a2), eh2, unstakeSlotMax)
	assert.NoError(t, err)
	assert.True(t, u0.Equal(us[0]))
	assert.True(t, u1.Equal(us[1]))
	assert.True(t, u2.Equal(us[2]))

	//unstake of last index will be updated
	err = us.increaseUnstake(big.NewInt(a3-a2), eh3, unstakeSlotMax)
	assert.NoError(t, err)
	assert.True(t, u0.Equal(us[0]))
	assert.True(t, u1.Equal(us[1]))
	assert.True(t, u3.Equal(us[2]))
}

func TestDecreaseUnstake(t *testing.T) {
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

	u0 := Unstake{Amount: big.NewInt(a0), ExpireHeight: eh0}
	u1 := Unstake{Amount: big.NewInt(a1), ExpireHeight: eh1}
	u2 := Unstake{Amount: big.NewInt(a2), ExpireHeight: eh2}
	u3 := Unstake{Amount: big.NewInt(a3), ExpireHeight: eh3}
	u4 := Unstake{Amount: big.NewInt(a4), ExpireHeight: eh4}

	us := Unstakes{&u0, &u1, &u2, &u3, &u4}
	assert.Equal(t, len(us), 5)

	//remove last unstake
	j, err := us.decreaseUnstake(u4.Amount)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(us))
	assert.True(t, us[0].Equal(&u0))
	assert.Equal(t, 1, len(j))
	assert.Equal(t, eh4, j[0].Height)

	//remove 2 unstakes
	j, err = us.decreaseUnstake(new(big.Int).Add(u2.Amount, u3.Amount))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(us))
	assert.True(t, us[0].Equal(&u0))
	assert.True(t, us[1].Equal(&u1))
	assert.Equal(t, 2, len(j))
	assert.Equal(t, eh3, j[0].Height)
	assert.Equal(t, eh2, j[1].Height)

	//remove last unstake and decrease first unstake
	v := big.NewInt(a1 + 1)
	j, err = us.decreaseUnstake(v)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(us))
	expectedUnstake := &Unstake{Amount: big.NewInt(a0 - 1), ExpireHeight: eh0}
	assert.True(t, us[0].Equal(expectedUnstake))
	assert.Equal(t, 1, len(j))
	assert.Equal(t, eh1, j[0].Height)
}
