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
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
)

var assTest = &Account{
	address: common.MustNewAddressFromString("hx0"),
	stake:   big.NewInt(100),
	unstakes: []*Unstake{
		{
			Amount:       big.NewInt(5),
			ExpireHeight: 10,
		},
		{
			Amount:       big.NewInt(10),
			ExpireHeight: 20,
		},
	},
	delegating: big.NewInt(20),
	delegations: []*Delegation{
		{
			Address: common.MustNewAddressFromString("hx1"),
			Value:   common.NewHexInt(10),
		},
		{
			Address: common.MustNewAddressFromString("hx2"),
			Value:   common.NewHexInt(10),
		},
	},
	bonding: big.NewInt(20),
	bonds: []*Bond{
		{
			Address: common.MustNewAddressFromString("hx3"),
			Value:   common.NewHexInt(10),
		},
		{
			Address: common.MustNewAddressFromString("hx4"),
			Value:   common.NewHexInt(10),
		},
	},
	unbonding: big.NewInt(20),
	unbonds: []*Unbond{
		{
			Address: common.MustNewAddressFromString("hx5"),
			Value:   big.NewInt(10),
			Expire:  20,
		},
		{
			Address: common.MustNewAddressFromString("hx6"),
			Value:   big.NewInt(10),
			Expire:  30,
		},
	},
}

func TestAccount_Bytes(t *testing.T) {
	address, err := common.NewAddress(make([]byte, common.AddressBytes, common.AddressBytes))
	if err != nil {
		t.Errorf("Failed to create an address")
	}
	assTest.SetAddress(address)
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

	ass2 := ToAccount(o2, address)
	assert.Equal(t, true, assTest.Equal(ass2))
}

func TestAccount_SetStake(t *testing.T) {
	address := common.MustNewAddressFromString("hx1")
	account := newAccount(address)

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
	a := assTest.Clone() // unbonds : [{address: hx5, value:10, bh: 20}, {hx6, 10, 30}]

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
		&Unbond{common.MustNewAddressFromString("hx5"), big.NewInt(40), 50},
		&Unbond{common.MustNewAddressFromString("hx6"), big.NewInt(10), 30},
		&Unbond{common.MustNewAddressFromString("hx7"), big.NewInt(40), 50},
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
		&Unbond{common.MustNewAddressFromString("hx6"), big.NewInt(10), 30},
		&Unbond{common.MustNewAddressFromString("hx7"), big.NewInt(40), 50},
		&Unbond{common.MustNewAddressFromString("hx8"), big.NewInt(50), 100},
		&Unbond{common.MustNewAddressFromString("hx9"), big.NewInt(50), 100},
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
	a := assTest.Clone() // unbonds : [{address: hx5, value:10, bh: 20}, {hx6, 10, 30}]
	ub1 := &Unbond{common.MustNewAddressFromString("hx5"), big.NewInt(10), 20}
	ub2 := &Unbond{common.MustNewAddressFromString("hx6"), big.NewInt(10), 30}
	assert.Contains(t, a.unbonds, ub1)
	assert.Contains(t, a.unbonds, ub2)
	expected := big.NewInt(20)
	assert.Equal(t, 0, expected.Cmp(a.unbonds.GetUnbondAmount()))
	assert.Equal(t, 0, a.unbonding.Cmp(a.unbonds.GetUnbondAmount()))

	//invalid height
	err := a.RemoveUnbonding(1)
	assert.Error(t, err)
	assert.Contains(t, a.unbonds, ub1)
	assert.Contains(t, a.unbonds, ub2)
	assert.Equal(t, 0, expected.Cmp(a.unbonds.GetUnbondAmount()))
	assert.Equal(t, 0, a.unbonding.Cmp(a.unbonds.GetUnbondAmount()))

	err = a.RemoveUnbonding(30)
	assert.NoError(t, err)
	assert.Contains(t, a.unbonds, ub1)
	assert.NotContains(t, a.unbonds, ub2)
	expected = big.NewInt(10)
	assert.Equal(t, 0, expected.Cmp(a.unbonds.GetUnbondAmount()))
	assert.Equal(t, 0, a.unbonding.Cmp(a.unbonds.GetUnbondAmount()))

	err = a.RemoveUnbonding(20)
	assert.NoError(t, err)
	assert.NotContains(t, a.unbonds, ub1)
	assert.NotContains(t, a.unbonds, ub2)
	expected = big.NewInt(0)
	assert.Equal(t, 0, expected.Cmp(a.unbonds.GetUnbondAmount()))
	assert.Equal(t, 0, a.unbonding.Cmp(a.unbonds.GetUnbondAmount()))
}

func TestAccount_RemoveUnstaking(t *testing.T) {
	a := assTest.Clone() // unstakes : [{value:5, bh: 10}, {10, 20}]
	us1 := &Unstake{big.NewInt(5), 10}
	us2 := &Unstake{big.NewInt(10), 20}
	assert.Contains(t, a.unstakes, us1)
	assert.Contains(t, a.unstakes, us2)
	expected := big.NewInt(15)
	assert.Equal(t, 0, expected.Cmp(a.GetUnstakeAmount()))

	//invalid height
	ra, err := a.RemoveUnstaking(1)
	assert.Error(t, err)
	assert.Contains(t, a.unstakes, us1)
	assert.Contains(t, a.unstakes, us2)
	assert.Equal(t, 0, expected.Cmp(a.GetUnstakeAmount()))
	assert.Equal(t, 0, big.NewInt(0).Cmp(ra))

	ra, err = a.RemoveUnstaking(20)
	assert.NoError(t, err)
	assert.Contains(t, a.unstakes, us1)
	assert.NotContains(t, a.unstakes, us2)
	expected = big.NewInt(5)
	assert.Equal(t, 0, expected.Cmp(a.GetUnstakeAmount()))
	assert.Equal(t, 0, big.NewInt(10).Cmp(ra))

	ra, err = a.RemoveUnstaking(10)
	assert.NoError(t, err)
	assert.NotContains(t, a.unbonds, us1)
	assert.NotContains(t, a.unbonds, us2)
	expected = big.NewInt(0)
	assert.Equal(t, 0, expected.Cmp(a.GetUnstakeAmount()))
	assert.Equal(t, 0, big.NewInt(5).Cmp(ra))
}

func TestAccount_SlashStake(t *testing.T) {
	a := assTest.Clone() // a.stake = 100

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
	a := assTest.Clone() //[{hx3, 10}, {hx4, 10}]

	amount := a.SlashBond(common.MustNewAddressFromString("hx3"), 10)
	assert.Equal(t, 0, amount.Cmp(big.NewInt(1)))
	b1 := a.Bonds()[0]
	assert.Equal(t, 0, b1.Value.Cmp(big.NewInt(9)))
	bl := len(a.Bonds())
	assert.Equal(t, 2, bl)

	amount = a.SlashBond(common.MustNewAddressFromString("hx4"), 100)
	assert.Equal(t, 0, amount.Cmp(big.NewInt(10)))
	bl = len(a.Bonds())
	assert.Equal(t, 1, bl)
}

func TestAccount_SlashUnbond(t *testing.T) {
	a := assTest.Clone() //[{hx5, value: 10, expire: 20}, {hx6, value: 10, expire: 30}]

	amount, eh := a.SlashUnbond(common.MustNewAddressFromString("hx5"), 10)
	assert.Equal(t, 0, amount.Cmp(big.NewInt(1)))
	assert.Equal(t, int64(-1), eh)
	u1 := a.Unbonds()[0]
	assert.Equal(t, 0, u1.Value.Cmp(big.NewInt(9)))
	ul := len(a.Unbonds())
	assert.Equal(t, 2, ul)

	amount, eh = a.SlashUnbond(common.MustNewAddressFromString("hx6"), 100)
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
