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
		Amount: big.NewInt(a1),
		ExpireHeight: eh1,
	}
	u2 := Unstake{
		Amount: big.NewInt(a2),
		ExpireHeight: eh2,
	}
	u3 := Unstake{
		Amount: big.NewInt(a3),
		ExpireHeight: eh3,
	}

	var us1 = Unstakes {
		&u1, &u2, &u3,
	}

	us2 := us1.Clone()

	assert.True(t, us1.Equal(us2))
	assert.True(t, us1.Has())
	assert.Equal(t, a1 + a2 + a3, us1.GetUnstakeAmount().Int64())


	t.Run("increase Unstakes", func(t *testing.T) {
		unstakes := Unstakes{}

		// add unstake u1
		err := unstakes.increaseUnstake(u1.Amount, u1.ExpireHeight)
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 1, len(unstakes))
		assert.True(t, u1.Equal(unstakes[0]))
		assert.Equal(t, a1, unstakes.GetUnstakeAmount().Int64())

		// add unstake u2
		err = unstakes.increaseUnstake(u2.Amount, u2.ExpireHeight)
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 2, len(unstakes))
		assert.True(t, u2.Equal(unstakes[1]))
		assert.Equal(t, a1 + a2, unstakes.GetUnstakeAmount().Int64())

		// update last unstake
		setMaxUnstakeCount(2)
		err = unstakes.increaseUnstake(u3.Amount, u3.ExpireHeight)
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 2, len(unstakes))
		assert.Equal(t, a2 + a3, unstakes[1].Amount.Int64())
		assert.Equal(t, eh3, unstakes[1].ExpireHeight)
		assert.Equal(t, a1 + a2 + a3, unstakes.GetUnstakeAmount().Int64())
		setMaxUnstakeCount(0)
	})

	t.Run("decrease Unstakes", func(t *testing.T) {
		unstakes := Unstakes{}
		err := unstakes.increaseUnstake(u1.Amount, u1.ExpireHeight)
		assert.NoError(t, err)
		err = unstakes.increaseUnstake(u2.Amount, u2.ExpireHeight)
		assert.NoError(t, err)
		err = unstakes.increaseUnstake(u3.Amount, u3.ExpireHeight)
		assert.NoError(t, err)

		total := a1 + a2 + a3
		// decrease amount of slot
		err = unstakes.decreaseUnstake(bigOne)
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 3, len(unstakes))
		assert.Equal(t, total - bigOne.Int64(), unstakes.GetUnstakeAmount().Int64())

		// delete 1 slot
		err = unstakes.decreaseUnstake(new(big.Int).Sub(u3.Amount, bigOne))
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 2, len(unstakes))
		assert.Equal(t, total - a3, unstakes.GetUnstakeAmount().Int64())

		// delete 1 slot and decrease 1 slot
		err = unstakes.decreaseUnstake(new(big.Int).Add(u2.Amount, bigOne))
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 1, len(unstakes))
		assert.Equal(t, a1 - bigOne.Int64(), unstakes.GetUnstakeAmount().Int64())

		// > total unstake. delete all
		err = unstakes.decreaseUnstake(u1.Amount)
		assert.NoError(t, err)
		assert.False(t, unstakes.Has())
		assert.Equal(t, 0, len(unstakes))
		assert.Equal(t, int64(0), unstakes.GetUnstakeAmount().Int64())
	})
}