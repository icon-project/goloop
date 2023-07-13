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

	"github.com/icon-project/goloop/icon/icmodule"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
)

func getTestAccount() *AccountState {
	assTest := &AccountState{
		accountData: accountData{
			stake: big.NewInt(100),
			unstakes: []*Unstake{
				NewUnstake(big.NewInt(5), 10),
				NewUnstake(big.NewInt(10), 20),
			},
			totalDelegation: big.NewInt(20),
			delegations: []*Delegation{
				NewDelegation(common.MustNewAddressFromString("hx1"), big.NewInt(10)),
				NewDelegation(common.MustNewAddressFromString("hx2"), big.NewInt(10)),
			},
			totalBond: big.NewInt(20),
			bonds: []*Bond{
				NewBond(common.MustNewAddressFromString("hx3"), big.NewInt(10)),
				NewBond(common.MustNewAddressFromString("hx4"), big.NewInt(10)),
			},
			totalUnbond: big.NewInt(20),
			unbonds: []*Unbond{
				NewUnbond(common.MustNewAddressFromString("hx5"), big.NewInt(10), 20),
				NewUnbond(common.MustNewAddressFromString("hx6"), big.NewInt(10), 30),
			},
		},
	}
	return assTest
}

func TestAccount_Bytes(t *testing.T) {
	assTest := getTestAccount()
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)

	o1 := icobject.New(TypeAccount, assTest.GetSnapshot())
	serialized := o1.Bytes()

	t.Logf("Serialized:%v", serialized)

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())

	ass2 := ToAccount(o2)
	assert.Equal(t, true, assTest.GetSnapshot().Equal(ass2))
}

func TestAccount_SetStake(t *testing.T) {
	account := newAccountStateWithSnapshot(nil)

	assert.Equal(t, 0, account.Stake().Cmp(new(big.Int)))

	e := account.SetStake(big.NewInt(-1))
	assert.Error(t, e)
	assert.Equal(t, 0, account.Stake().Cmp(new(big.Int)))

	s := big.NewInt(10)
	e = account.SetStake(s)
	assert.NoError(t, e)
	assert.Equal(t, 0, s.Cmp(account.Stake()))
}

func TestAccount_UpdateUnbonds(t *testing.T) {
	a := getTestAccount() // unbonds : [{address: hx5, value:10, bh: 20}, {hx6, 10, 30}]

	key5 := icutils.ToKey(common.MustNewAddressFromString("hx5"))
	key6 := icutils.ToKey(common.MustNewAddressFromString("hx6"))
	key7 := icutils.ToKey(common.MustNewAddressFromString("hx7"))
	key8 := icutils.ToKey(common.MustNewAddressFromString("hx8"))
	key9 := icutils.ToKey(common.MustNewAddressFromString("hx9"))

	// modify hx5 targeted unbonding, add a unbonding(hx7)
	// unbonds : [{address: hx5, value:40, bh: 50}, {hx6, 10, 30}, {hx7, 40, 50}]/
	delta := map[string]*big.Int{
		key5: big.NewInt(-30),
		key7: big.NewInt(-40),
	}
	expireHeight := int64(50)
	expectedUnbonds := Unbonds{
		NewUnbond(common.MustNewAddressFromString("hx5"), big.NewInt(40), 50),
		NewUnbond(common.MustNewAddressFromString("hx6"), big.NewInt(10), 30),
		NewUnbond(common.MustNewAddressFromString("hx7"), big.NewInt(40), 50),
	}
	expectedTL := []TimerJobInfo{{JobTypeAdd, expireHeight}, {JobTypeRemove, 20}}

	tl, err := a.UpdateUnbonds(delta, expireHeight)
	assert.NoError(t, err)
	assert.True(t, equalTimerJobSlice(expectedTL, tl))
	assert.True(t, a.unbonds.Equal(expectedUnbonds))
	assert.Equal(t, 0, a.Unbond().Cmp(expectedUnbonds.GetUnbondAmount()))

	// delete hx5 targeted unbonding, add 2 unbondings(hx8, hx9)
	// unbonds : [{hx6, 10, 30}, {hx7, 40, 50}, {hx8, 50, 100}, {hx9, 50, 100}]
	delta = map[string]*big.Int{
		key5: big.NewInt(50),
		key8: big.NewInt(-50),
		key9: big.NewInt(-50),
	}
	expireHeight = int64(100)
	expectedUnbonds = Unbonds{
		NewUnbond(common.MustNewAddressFromString("hx6"), big.NewInt(10), 30),
		NewUnbond(common.MustNewAddressFromString("hx7"), big.NewInt(40), 50),
		NewUnbond(common.MustNewAddressFromString("hx8"), big.NewInt(50), 100),
		NewUnbond(common.MustNewAddressFromString("hx9"), big.NewInt(50), 100),
	}
	expectedTL = []TimerJobInfo{
		{JobTypeAdd, expireHeight},
	}

	tl, err = a.UpdateUnbonds(delta, expireHeight)
	assert.NoError(t, err)
	assert.True(t, equalTimerJobSlice(expectedTL, tl))
	assert.True(t, a.unbonds.Equal(expectedUnbonds))
	assert.Equal(t, 0, a.Unbond().Cmp(expectedUnbonds.GetUnbondAmount()))

	//remove all unbondings
	delta = map[string]*big.Int{
		key6: big.NewInt(10),
		key7: big.NewInt(40),
		key8: big.NewInt(50),
		key9: big.NewInt(50),
	}
	expireHeight = int64(150)
	expectedUnbonds = Unbonds{}
	expectedTL = []TimerJobInfo{
		{JobTypeRemove, 30},
		{JobTypeRemove, 50},
		{JobTypeRemove, 100},
	}

	tl, err = a.UpdateUnbonds(delta, expireHeight)
	assert.NoError(t, err)
	assert.True(t, equalTimerJobSlice(expectedTL, tl))
	assert.True(t, a.unbonds.Equal(expectedUnbonds))
	assert.Equal(t, 0, a.Unbond().Cmp(expectedUnbonds.GetUnbondAmount()))

}

func TestAccount_RemoveUnbonding(t *testing.T) {
	a := getTestAccount() // unbonds : [{address: hx5, value:10, bh: 20}, {hx6, 10, 30}]
	ub1 := NewUnbond(common.MustNewAddressFromString("hx5"), big.NewInt(10), 20)
	ub2 := NewUnbond(common.MustNewAddressFromString("hx6"), big.NewInt(10), 30)
	assert.Contains(t, a.unbonds, ub1)
	assert.Contains(t, a.unbonds, ub2)
	expected := big.NewInt(20)
	assert.Equal(t, 0, expected.Cmp(a.unbonds.GetUnbondAmount()))
	assert.Equal(t, 0, a.totalUnbond.Cmp(a.unbonds.GetUnbondAmount()))

	//invalid height
	err := a.RemoveUnbond(1)
	assert.Error(t, err)
	assert.Contains(t, a.unbonds, ub1)
	assert.Contains(t, a.unbonds, ub2)
	assert.Equal(t, 0, expected.Cmp(a.unbonds.GetUnbondAmount()))
	assert.Equal(t, 0, a.totalUnbond.Cmp(a.unbonds.GetUnbondAmount()))

	err = a.RemoveUnbond(30)
	assert.NoError(t, err)
	assert.Contains(t, a.unbonds, ub1)
	assert.NotContains(t, a.unbonds, ub2)
	expected = big.NewInt(10)
	assert.Equal(t, 0, expected.Cmp(a.unbonds.GetUnbondAmount()))
	assert.Equal(t, 0, a.totalUnbond.Cmp(a.unbonds.GetUnbondAmount()))

	err = a.RemoveUnbond(20)
	assert.NoError(t, err)
	assert.NotContains(t, a.unbonds, ub1)
	assert.NotContains(t, a.unbonds, ub2)
	expected = big.NewInt(0)
	assert.Equal(t, 0, expected.Cmp(a.unbonds.GetUnbondAmount()))
	assert.Equal(t, 0, a.totalUnbond.Cmp(a.unbonds.GetUnbondAmount()))
}

func TestAccount_RemoveUnstaking(t *testing.T) {
	a := getTestAccount() // unstakes : [{value:5, bh: 10}, {10, 20}]
	us1 := NewUnstake(big.NewInt(5), 10)
	us2 := NewUnstake(big.NewInt(10), 20)
	assert.Contains(t, a.unstakes, us1)
	assert.Contains(t, a.unstakes, us2)
	expected := big.NewInt(15)
	assert.Equal(t, 0, expected.Cmp(a.GetUnstakeAmount()))

	//invalid height
	ra, err := a.RemoveUnstake(1)
	assert.Error(t, err)
	assert.Nil(t, ra)
	assert.Contains(t, a.unstakes, us1)
	assert.Contains(t, a.unstakes, us2)
	assert.Equal(t, 0, expected.Cmp(a.GetUnstakeAmount()))

	ra, err = a.RemoveUnstake(20)
	assert.NoError(t, err)
	assert.Contains(t, a.unstakes, us1)
	assert.NotContains(t, a.unstakes, us2)
	expected = big.NewInt(5)
	assert.Equal(t, 0, expected.Cmp(a.GetUnstakeAmount()))
	assert.Equal(t, 0, big.NewInt(10).Cmp(ra))

	ra, err = a.RemoveUnstake(10)
	assert.NoError(t, err)
	assert.NotContains(t, a.unbonds, us1)
	assert.NotContains(t, a.unbonds, us2)
	expected = big.NewInt(0)
	assert.Equal(t, 0, expected.Cmp(a.GetUnstakeAmount()))
	assert.Equal(t, 0, big.NewInt(5).Cmp(ra))
}

func TestAccount_SlashStake(t *testing.T) {
	a := getTestAccount() // a.stake = 100

	err := a.SlashStake(big.NewInt(10))
	assert.NoError(t, err)
	assert.Equal(t, 0, a.Stake().Cmp(big.NewInt(90)))

	err = a.SlashStake(big.NewInt(100))
	assert.Error(t, err)
	assert.Equal(t, 0, a.Stake().Cmp(big.NewInt(90)))

	err = a.SlashStake(big.NewInt(90))
	assert.NoError(t, err)
	assert.Equal(t, 0, a.Stake().Cmp(big.NewInt(0)))
}

func TestAccount_SlashBond(t *testing.T) {
	a := getTestAccount() //[{hx3, 10}, {hx4, 10}]
	amount := a.SlashBond(common.MustNewAddressFromString("hx3"), icutils.PercentToRate(10))
	assert.Equal(t, 0, amount.Cmp(big.NewInt(1)))
	b1 := a.Bonds()[0]
	assert.Equal(t, 0, b1.Amount().Cmp(big.NewInt(9)))
	bl := len(a.Bonds())
	assert.Equal(t, 2, bl)

	amount = a.SlashBond(common.MustNewAddressFromString("hx4"), icutils.PercentToRate(100))
	assert.Equal(t, 0, amount.Cmp(big.NewInt(10)))
	bl = len(a.Bonds())
	assert.Equal(t, 1, bl)
}

func TestAccount_SlashUnbond(t *testing.T) {
	a := getTestAccount() //[{hx5, value: 10, expire: 20}, {hx6, value: 10, expire: 30}]

	amount, eh := a.SlashUnbond(common.MustNewAddressFromString("hx5"), icutils.PercentToRate(10))
	assert.Equal(t, 0, amount.Cmp(big.NewInt(1)))
	assert.Equal(t, int64(-1), eh)
	u1 := a.Unbonds()[0]
	assert.Equal(t, 0, u1.Value().Cmp(big.NewInt(9)))
	ul := len(a.Unbonds())
	assert.Equal(t, 2, ul)

	amount, eh = a.SlashUnbond(common.MustNewAddressFromString("hx6"), icutils.PercentToRate(100))
	assert.Equal(t, 0, amount.Cmp(big.NewInt(10)))
	assert.Equal(t, int64(30), eh)
	ul = len(a.Unbonds())
	assert.Equal(t, 1, ul)
}

func equalTimerJobSlice(expected []TimerJobInfo, actual []TimerJobInfo) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := 0; i < len(expected); i++ {
		if (expected[i].Type != actual[i].Type) || (expected[i].Height != actual[i].Height) {
			return false
		}
	}
	return true
}

func TestAccountState_ReversedExpireHeight(t *testing.T) {
	slotMax := 3
	revision := icmodule.RevisionICON2R0
	a := &AccountState{}
	eh1 := int64(1)
	eh2 := int64(10)
	eh3 := int64(4)
	eh4 := int64(5)
	v1 := big.NewInt(10)
	v2 := big.NewInt(15)
	v3 := big.NewInt(20)
	v4 := big.NewInt(30)
	a.IncreaseUnstake(v1, eh1, slotMax, revision)
	a.IncreaseUnstake(v2, eh2, slotMax, revision)
	a.IncreaseUnstake(v3, eh3, slotMax, revision)

	unstakes := a.unstakes
	// [(v1, eh1), (v3, eh3), (v2, eh2)]
	assert.Equal(t, v1, unstakes[0].Value)
	assert.Equal(t, v3, unstakes[1].Value)
	assert.Equal(t, v2, unstakes[2].Value)
	assert.Equal(t, eh1, unstakes[0].Expire)
	assert.Equal(t, eh3, unstakes[1].Expire)
	assert.Equal(t, eh2, unstakes[2].Expire)

	a.IncreaseUnstake(v4, eh4, slotMax, revision)
	unstakes = a.unstakes
	// [(v1, eh1), (v3, eh3), (v2 + v4, eh2)]
	assert.Equal(t, v1, unstakes[0].Value)
	assert.Equal(t, v3, unstakes[1].Value)
	assert.Equal(t, new(big.Int).Add(v2, v4), unstakes[2].Value)
	assert.Equal(t, eh1, unstakes[0].Expire)
	assert.Equal(t, eh3, unstakes[1].Expire)
	assert.Equal(t, eh2, unstakes[2].Expire)
}

func TestAccountState_OverlappedExpireHeight(t *testing.T) {
	slotMax := 3
	revision := icmodule.RevisionICON2R0
	a := &AccountState{}
	eh1 := int64(1)
	eh2 := int64(10)
	eh3 := eh2
	eh4 := int64(5)
	v1 := big.NewInt(10)
	v2 := big.NewInt(15)
	v3 := big.NewInt(20)
	v4 := big.NewInt(30)
	a.IncreaseUnstake(v1, eh1, slotMax, revision)
	a.IncreaseUnstake(v2, eh2, slotMax, revision)
	a.IncreaseUnstake(v3, eh3, slotMax, revision)

	unstakes := a.unstakes
	// [(v1, eh1), (v2, eh2), (v3, eh3 = eh2)]
	assert.Equal(t, v1, unstakes[0].Value)
	assert.Equal(t, v2, unstakes[1].Value)
	assert.Equal(t, v3, unstakes[2].Value)
	assert.Equal(t, eh1, unstakes[0].Expire)
	assert.Equal(t, eh2, unstakes[1].Expire)
	assert.Equal(t, eh3, unstakes[2].Expire)

	a.IncreaseUnstake(v4, eh4, slotMax, revision)
	unstakes = a.unstakes
	// [(v1, eh1), (v2, eh2), (v3 + v4, eh3 = eh2)]
	assert.Equal(t, v1, unstakes[0].Value)
	assert.Equal(t, v2, unstakes[1].Value)
	assert.Equal(t, new(big.Int).Add(v3, v4), unstakes[2].Value)
	assert.Equal(t, eh1, unstakes[0].Expire)
	assert.Equal(t, eh2, unstakes[1].Expire)
	assert.Equal(t, eh3, unstakes[2].Expire)

	a.RemoveUnstake(eh2)
	unstakes = a.unstakes
	// [(v1, eh1)]
	assert.Equal(t, 1, len(unstakes))
	assert.Equal(t, v1, unstakes[0].Value)
	assert.Equal(t, eh1, unstakes[0].Expire)
}
