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

package iiss

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
)

var bigOne = big.NewInt(1)

func TestAccount(t *testing.T) {
	t1 := Account{
		version: 100,
		staked: big.NewInt(60),
		unstakes: []unstake{
			{
				amount: big.NewInt(5),
				expireHeight: 10,
			},
			{
				amount: big.NewInt(10),
				expireHeight: 20,
			},
		},
		delegated: big.NewInt(20),
		delegations: []delegation{
			{
				target: common.NewAddressFromString("hx1"),
				amount: big.NewInt(10),
			},
			{
				target: common.NewAddressFromString("hx2"),
				amount: big.NewInt(10),
			},
		},
		bonds: []bond{
			{
				target: common.NewAddressFromString("hx3"),
				amount: big.NewInt(10),
			},
			{
				target: common.NewAddressFromString("hx4"),
				amount: big.NewInt(10),
			},
		},
		unbondings: []unbonding{
			{
				target: common.NewAddressFromString("hx5"),
				amount: big.NewInt(10),
				expireHeight: 20,
			},
			{
				target: common.NewAddressFromString("hx6"),
				amount: big.NewInt(10),
				expireHeight: 30,
			},
		},
	}

	bs := t1.Bytes()

	t2 := new(Account)
	err := t2.SetBytes(bs)
	assert.NoError(t, err, "error %+v", err)

	assert.True(t, t1.Equal(t2))
}

func TestUnstake(t *testing.T) {
	u1 := unstake{
		amount: big.NewInt(5),
		expireHeight: 10,
	}
	u2 := unstake{
		amount: big.NewInt(10),
		expireHeight: 20,
	}
	u3 := unstake{
		amount: big.NewInt(10),
		expireHeight: 30,
	}

	t.Run("increase Unstake", func(t *testing.T) {
		unstakes := unstakeList{}

		// add unstake
		err := unstakes.increaseUnstake(u1.amount, u1.expireHeight)
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 1, len(unstakes))
		assert.True(t, u1.equal(unstakes[0]))
		assert.Equal(t, u1.amount.Int64(), unstakes.getUnstakeAmount().Int64())

		// add unstake
		err = unstakes.increaseUnstake(u2.amount, u2.expireHeight)
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 2, len(unstakes))
		assert.True(t, u2.equal(unstakes[1]))
		assert.Equal(t, u1.amount.Int64()+u2.amount.Int64(), unstakes.getUnstakeAmount().Int64())

		setMaxUnstakeCount(2)

		// update last unstake
		err = unstakes.increaseUnstake(u3.amount, u3.expireHeight)
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 2, len(unstakes))
		assert.Equal(t, u2.amount.Int64()+u3.amount.Int64(), unstakes[1].amount.Int64())
		assert.Equal(t, u3.expireHeight, unstakes[1].expireHeight)
		assert.Equal(t, u1.amount.Int64()+u2.amount.Int64()+u3.amount.Int64(), unstakes.getUnstakeAmount().Int64())

		setMaxUnstakeCount(0)
	})

	t.Run("decrease Unstake", func(t *testing.T) {
		unstakes := unstakeList{}
		err := unstakes.increaseUnstake(u1.amount, u1.expireHeight)
		assert.NoError(t, err)
		err = unstakes.increaseUnstake(u2.amount, u2.expireHeight)
		assert.NoError(t, err)
		err = unstakes.increaseUnstake(u3.amount, u3.expireHeight)
		assert.NoError(t, err)

		total := u1.amount.Int64() + u2.amount.Int64() + u3.amount.Int64()
		// decrease 1 slot
		err = unstakes.decreaseUnstake(bigOne)
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 3, len(unstakes))
		assert.Equal(t, total - bigOne.Int64(), unstakes.getUnstakeAmount().Int64())

		// delete 1 slot
		err = unstakes.decreaseUnstake(new(big.Int).Sub(u3.amount, bigOne))
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 2, len(unstakes))
		assert.Equal(t, total - u3.amount.Int64(), unstakes.getUnstakeAmount().Int64())

		// delete 1 slot and decrease 1 slot
		err = unstakes.decreaseUnstake(new(big.Int).Add(u2.amount, bigOne))
		assert.NoError(t, err)
		assert.True(t, unstakes.Has())
		assert.Equal(t, 1, len(unstakes))
		assert.Equal(t, u1.amount.Int64() - bigOne.Int64(), unstakes.getUnstakeAmount().Int64())

		// > total unstake. delete all
		err = unstakes.decreaseUnstake(u1.amount)
		assert.NoError(t, err)
		assert.False(t, unstakes.Has())
		assert.Equal(t, 0, len(unstakes))
		assert.Equal(t, int64(0), unstakes.getUnstakeAmount().Int64())
	})
}
