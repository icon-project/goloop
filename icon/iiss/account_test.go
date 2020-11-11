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
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
)

var bigOne = big.NewInt(1)
var t1 = Account{
	version: 100,
	staked: big.NewInt(100),
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
			Address: common.NewAddressFromString("hx1"),
			Value:   common.NewHexInt(10),
		},
		{
			Address: common.NewAddressFromString("hx2"),
			Value:   common.NewHexInt(10),
		},
	},
	bonded: big.NewInt(20),
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

func TestAccount(t *testing.T) {
	bs := t1.Bytes()

	t2 := new(Account)
	err := t2.SetBytes(bs)
	assert.NoError(t, err, "error %+v", err)

	assert.True(t, t1.Equal(t2))
}

func TestAccount_GetUnstakeAmount(t *testing.T) {
	assert.Equal(t, int64(15), t1.GetUnstakeAmount().Int64())
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

var d1 = []interface{} {
	map[string]interface{}{
		"address": "hx1",
		"value": "0x1",
	},
	map[string]interface{}{
		"address": "hx2",
		"value": "0x2",
	},
}

func TestAccount_Delegation(t *testing.T) {
	v1 := 1
	v2 := 2
	tests := []struct {
		name string
		param []interface{}
		err bool
		totalDelegate int
	}{
		{"Nil param", nil, false, 0},
		{"Empty param", []interface{} {}, false, 0},
		{
			"Success",
			[]interface{} {
				map[string]interface{}{
					"address": "hx1",
					"value": fmt.Sprintf("0x%x", v1),
				},
				map[string]interface{}{
					"address": "hx2",
					"value": fmt.Sprintf("0x%x", v2),
				},
			},
			false,
			v1 + v2,
		},
		{
			"Not enough voting power",
			[]interface{} {
				map[string]interface{}{
					"address": "hx10000",
					"value": "0x10000000000000000000",
				},
				map[string]interface{}{
					"address": "hx20000",
					"value": "0x20000000000000000000",
				},
			},
			true,
			0,
		},
		{
			"Duplicated target address",
			[]interface{} {
				map[string]interface{}{
					"address": "hx1",
					"value": fmt.Sprintf("0x%x", v1),
				},
				map[string]interface{}{
					"address": "hx1",
					"value": fmt.Sprintf("0x%x", v2),
				},
			},
			true,
			0,
		},
		{
			"Too many delegations",
			[]interface{} {
				map[string]interface{}{
					"address": "hx1",
					"value": fmt.Sprintf("0x%x", v1),
				},
				map[string]interface{}{
					"address": "hx2",
					"value": fmt.Sprintf("0x%x", v2),
				},
				map[string]interface{}{
					"address": "hx3",
					"value": fmt.Sprintf("0x%x", v2),
				},
			},
			true,
			0,
		},
	}

	setMaxDelegationCount(2)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t2 := t1.Clone()
			err := t2.SetDelegation(tt.param)
			if tt.err {
				assert.Error(t, err, "SetDelegation() was not failed for %v.", tt.param)
			} else {
				assert.NoError(t, err, "SetDelegation() was failed for %v. err=%v", tt.param, err)

				got, err := t2.GetDelegationInfo()
				assert.NoError(t, err, "GetDelegationInfo() was failed for %v. err=%v", tt.param, err)

				_, ok := got["delegations"]
				if tt.totalDelegate == 0 && ok {
					t.Errorf("GetDelegationIfo() = %v, want %v", got["delegations"], tt.param)
				}
				if !reflect.DeepEqual(got["totalDelegated"], big.NewInt(int64(tt.totalDelegate))) {
					t.Errorf("GetDelegationIfo() = %v, want %v", got["totalDelegated"], tt.param)
				}
			}
		})
	}

	setMaxDelegationCount(0)
}
