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

func TestAccount_UpdateUnstake(t *testing.T) {
	maxUnstakeCount = 3
	address := common.MustNewAddressFromString("hx1")
	account := newAccount(address)

	ul := make(Unstakes, 0)
	assert.True(t, account.unstakes.Equal(ul))

	// unstake += 0
	bh0 := int64(0)
	v0 := big.NewInt(0)
	tl, e := account.UpdateUnstake(v0, bh0)
	expectedTL := make([]TimerJobInfo, 0)
	assert.NoError(t, e)
	assert.True(t, equalTimerJobSlice(expectedTL, tl))
	assert.Equal(t, 0, account.GetUnstakeAmount().Cmp(v0))

	// unstake += 10
	bh1 := int64(10)
	v1 := big.NewInt(10)
	tl, e = account.UpdateUnstake(new(big.Int).Neg(v1), bh1)
	expectedTL = []TimerJobInfo{{JobTypeAdd, bh1}}
	assert.True(t, equalTimerJobSlice(expectedTL, tl))
	assert.NoError(t, e)
	assert.Equal(t, 0, account.GetUnstakeAmount().Cmp(v1))

	// unstake += 20
	bh2 := int64(20)
	v2 := big.NewInt(20)
	tl, e = account.UpdateUnstake(new(big.Int).Neg(v2), bh2)
	expectedTL = []TimerJobInfo{{JobTypeAdd, bh2}}
	assert.True(t, equalTimerJobSlice(expectedTL, tl))
	assert.NoError(t, e)
	expectedUA := new(big.Int).Add(v1, v2)
	assert.Equal(t, 0, account.GetUnstakeAmount().Cmp(expectedUA))

	// unstake += 30
	bh3 := int64(15) // unstakes : [(10, 10), (30, 15), (20, 20)]
	v3 := big.NewInt(20)
	tl, e = account.UpdateUnstake(new(big.Int).Neg(v3), bh3)
	expectedTL = []TimerJobInfo{{JobTypeAdd, bh3}}
	assert.True(t, equalTimerJobSlice(expectedTL, tl))
	assert.NoError(t, e)
	expectedUA = new(big.Int).Add(expectedUA, v3)
	assert.Equal(t, 0, account.GetUnstakeAmount().Cmp(expectedUA))

	// unstake -= 20
	// unstakes : [(10, 10), (30, 15)]
	tl, e = account.UpdateUnstake(v2, 0) //expireHeight does not effect when decrease unstake
	expectedTL = []TimerJobInfo{{JobTypeRemove, bh2}}
	assert.True(t, equalTimerJobSlice(expectedTL, tl))
	assert.NoError(t, e)
	expectedUA = new(big.Int).Sub(expectedUA, v2)
	assert.Equal(t, 0, account.GetUnstakeAmount().Cmp(expectedUA))
	assert.Equal(t, 2, len(account.unstakes))

	// unstake -= 13
	// unstakes : [(10, 10), (17, 15)]
	dv := big.NewInt(13)
	tl, e = account.UpdateUnstake(dv, 0)
	expectedTL = []TimerJobInfo{}
	assert.True(t, equalTimerJobSlice(expectedTL, tl))
	assert.NoError(t, e)
	expectedUA = new(big.Int).Sub(expectedUA, dv)
	assert.Equal(t, 0, account.GetUnstakeAmount().Cmp(expectedUA))
	assert.Equal(t, 2, len(account.unstakes))

	// unstake -= 27
	// unstakes : []
	dv = new(big.Int).Add(account.unstakes[0].Amount, account.unstakes[1].Amount)
	tl, e = account.UpdateUnstake(dv, 0)
	expectedTL = []TimerJobInfo{{JobTypeRemove, bh3}, {JobTypeRemove, bh1}}
	assert.True(t, equalTimerJobSlice(expectedTL, tl))
	assert.NoError(t, e)
	expectedUA = new(big.Int).Sub(expectedUA, dv)
	assert.Equal(t, 0, account.GetUnstakeAmount().Cmp(expectedUA))
	assert.Equal(t, 0, len(account.unstakes))
}

func TestAccount_UpdateUnbonds(t *testing.T) {
	a := assTest.Clone() // unbonds : [{address: hx5, value:10, bh: 20}, {hx6, 10, 30}]

	ub1 := &Unbond{common.MustNewAddressFromString("hx5"), big.NewInt(10), 20}
	ub2 := &Unbond{common.MustNewAddressFromString("hx6"), big.NewInt(10), 30}
	assert.True(t, a.unbonds[0].Equal(ub1))
	assert.True(t, a.unbonds[1].Equal(ub2))
	ua := big.NewInt(20)
	assert.Equal(t, 0, ua.Cmp(a.unbonds.GetUnbondAmount()))

	// modify hx5 targeted unbonding, add a unbonding(hx7)
	// unbonds : [{address: hx5, value:40, bh: 100}, {hx6, 10, 30}, {hx7, 40, 50}]/
	add1 := &Unbond{common.MustNewAddressFromString("hx7"), big.NewInt(40), 50}
	mod1 := &Unbond{common.MustNewAddressFromString("hx5"), big.NewInt(40), 100}
	ul2Add := []*Unbond{add1}
	ul2Mod := []*Unbond{mod1}
	tl := a.UpdateUnbonds(ul2Add, ul2Mod)

	expectedTL := []TimerJobInfo{{JobTypeAdd, ul2Add[0].Expire}}
	assert.True(t, equalTimerJobSlice(expectedTL, tl))
	ub1 = &Unbond{common.MustNewAddressFromString("hx5"), big.NewInt(40), 100}
	ub2 = &Unbond{common.MustNewAddressFromString("hx6"), big.NewInt(10), 30}
	assert.True(t, a.unbonds[0].Equal(ub1))
	assert.True(t, a.unbonds[1].Equal(ub2))
	assert.True(t, a.unbonds[2].Equal(add1))
	ua = big.NewInt(90)
	assert.Equal(t, 0, ua.Cmp(a.unbonds.GetUnbondAmount()))

	// delete hx5 targeted unbonding, add 2 unbondings(hx8, hx9)
	// unbonds : [{address: hx5, value: 0, bh:100}, {hx6, 10, 30}, {hx7, 40, 50}, {hx8, 50, 50}, {hx9, 100, 3}]
	add1 = &Unbond{common.MustNewAddressFromString("hx8"), big.NewInt(50), 50}
	add2 := &Unbond{common.MustNewAddressFromString("hx9"), big.NewInt(100), 3}
	mod1 = &Unbond{common.MustNewAddressFromString("hx5"), big.NewInt(0), 100}
	ul2Add = []*Unbond{add1, add2}
	ul2Mod = []*Unbond{mod1}
	tl = a.UpdateUnbonds(ul2Add, ul2Mod)

	j1 := TimerJobInfo{JobTypeAdd, add1.Expire}
	j2 := TimerJobInfo{JobTypeAdd, add2.Expire}
	j3 := TimerJobInfo{JobTypeRemove, mod1.Expire}
	assert.Contains(t, tl, j1)
	assert.Contains(t, tl, j2)
	assert.Contains(t, tl, j3)
	ub1 = &Unbond{common.MustNewAddressFromString("hx5"), big.NewInt(0), 100}
	ub2 = &Unbond{common.MustNewAddressFromString("hx6"), big.NewInt(10), 30}
	ub3 := &Unbond{common.MustNewAddressFromString("hx7"), big.NewInt(40), 50}
	ub4 := &Unbond{common.MustNewAddressFromString("hx8"), big.NewInt(50), 50}
	ub5 := &Unbond{common.MustNewAddressFromString("hx9"), big.NewInt(100), 3}
	assert.True(t, a.unbonds[0].Equal(ub1))
	assert.True(t, a.unbonds[1].Equal(ub2))
	assert.True(t, a.unbonds[2].Equal(ub3))
	assert.True(t, a.unbonds[3].Equal(ub4))
	assert.True(t, a.unbonds[4].Equal(ub5))

	//remove all unbondings
	mod1 = &Unbond{common.MustNewAddressFromString("hx6"), big.NewInt(0), 150}
	mod2 := &Unbond{common.MustNewAddressFromString("hx7"), big.NewInt(0), 200}
	mod3 := &Unbond{common.MustNewAddressFromString("hx8"), big.NewInt(0), 100}
	mod4 := &Unbond{common.MustNewAddressFromString("hx9"), big.NewInt(0), 1000}
	ul2Add = []*Unbond{}
	ul2Mod = []*Unbond{mod1, mod2, mod3, mod4}
	tl = a.UpdateUnbonds(ul2Add, ul2Mod)

	j1 = TimerJobInfo{JobTypeRemove, mod1.Expire}
	j2 = TimerJobInfo{JobTypeRemove, mod2.Expire}
	j3 = TimerJobInfo{JobTypeRemove, mod3.Expire}
	j4 := TimerJobInfo{JobTypeRemove, mod4.Expire}
	assert.Contains(t, tl, j1)
	assert.Contains(t, tl, j2)
	assert.Contains(t, tl, j3)
	assert.Contains(t, tl, j4)
	ub1 = &Unbond{common.MustNewAddressFromString("hx5"), big.NewInt(0), 100}
	ub2 = &Unbond{common.MustNewAddressFromString("hx6"), big.NewInt(0), 150}
	ub3 = &Unbond{common.MustNewAddressFromString("hx7"), big.NewInt(0), 200}
	ub4 = &Unbond{common.MustNewAddressFromString("hx8"), big.NewInt(0), 100}
	ub5 = &Unbond{common.MustNewAddressFromString("hx9"), big.NewInt(0), 1000}
	assert.True(t, a.unbonds[0].Equal(ub1))
	assert.True(t, a.unbonds[1].Equal(ub2))
	assert.True(t, a.unbonds[2].Equal(ub3))
	assert.True(t, a.unbonds[3].Equal(ub4))
	assert.True(t, a.unbonds[4].Equal(ub5))
}

func TestAccount_RemoveUnbonding(t *testing.T) {
	a := assTest.Clone() // unbonds : [{address: hx5, value:10, bh: 20}, {hx6, 10, 30}]
	ub1 := &Unbond{common.MustNewAddressFromString("hx5"), big.NewInt(10), 20}
	ub2 := &Unbond{common.MustNewAddressFromString("hx6"), big.NewInt(10), 30}
	assert.Contains(t, a.unbonds, ub1)
	assert.Contains(t, a.unbonds, ub2)
	expected := big.NewInt(20)
	assert.Equal(t, 0, expected.Cmp(a.unbonds.GetUnbondAmount()))

	//invalid height
	err := a.RemoveUnbonding(1)
	assert.Error(t, err)
	assert.Contains(t, a.unbonds, ub1)
	assert.Contains(t, a.unbonds, ub2)
	assert.Equal(t, 0, expected.Cmp(a.unbonds.GetUnbondAmount()))

	err = a.RemoveUnbonding(30)
	assert.NoError(t, err)
	assert.Contains(t, a.unbonds, ub1)
	assert.NotContains(t, a.unbonds, ub2)
	expected = big.NewInt(10)
	assert.Equal(t, 0, expected.Cmp(a.unbonds.GetUnbondAmount()))

	err = a.RemoveUnbonding(20)
	assert.NoError(t, err)
	assert.NotContains(t, a.unbonds, ub1)
	assert.NotContains(t, a.unbonds, ub2)
	expected = big.NewInt(0)
	assert.Equal(t, 0, expected.Cmp(a.unbonds.GetUnbondAmount()))
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

func TestAccount_GetUnbondingInfo(t *testing.T) {
	a := assTest.Clone()
	//bonds : [{hx3, 10}, {hx4, 10}] , unbonds : [{address: hx5, value:10, bh: 20}, {hx6, 10, 30}]
	addr1 := common.MustNewAddressFromString("hx3")
	addr2 := common.MustNewAddressFromString("hx4")
	bh := int64(31)
	b1 := &Bond{addr1, common.NewHexInt(5)}
	b2 := &Bond{addr2, common.NewHexInt(5)}
	nbs := []*Bond{b1, b2}
	ubAdds, ubMods, uDiff := a.GetUnbondingInfo(nbs, bh) // 2 unbonds will added

	expectedUDiff := big.NewInt(10)
	ubAdd1 := &Unbond{addr1, big.NewInt(5), bh}
	ubAdd2 := &Unbond{addr2, big.NewInt(5), bh}
	assert.True(t, ubAdds[0].Equal(ubAdd1))
	assert.True(t, ubAdds[1].Equal(ubAdd2))
	assert.Equal(t, 2, len(ubAdds))
	assert.Equal(t, 0, len(ubMods))
	assert.Equal(t, 0, uDiff.Cmp(expectedUDiff))

	//add bond
	addr3 := common.MustNewAddressFromString("hx5")
	b1 = &Bond{addr3, common.NewHexInt(5)}
	a.bonds = append(a.bonds, b1)
	//bonds : [{hx3, 10}, {hx4, 10}, {hx5, 5}], unbonds : [{address: hx5, value:10, bh: 20}, {hx6, 10, 30}]
	b1 = &Bond{addr2, common.NewHexInt(5)}
	b2 = &Bond{addr3, common.NewHexInt(7)}
	nbs = []*Bond{b1, b2}
	ubAdds, ubMods, uDiff = a.GetUnbondingInfo(nbs, bh) // 1 will modified(hx5), 1 will added(hx4)

	expectedUDiff = big.NewInt(3)
	ubAdd1 = &Unbond{addr2, big.NewInt(5), bh}
	ubMod1 := &Unbond{addr3, big.NewInt(8), bh}
	assert.True(t, ubAdds[0].Equal(ubAdd1))
	assert.True(t, ubMods[0].Equal(ubMod1))
	assert.Equal(t, 1, len(ubAdds))
	assert.Equal(t, 1, len(ubMods))
	assert.Equal(t, 0, uDiff.Cmp(expectedUDiff))

	//bonds : [{hx3, 10}, {hx4, 10}, {hx5, 5}], unbonds : [{address: hx5, value:10, bh: 20}, {hx6, 10, 30}]
	b1 = &Bond{addr2, common.NewHexInt(5)}
	b2 = &Bond{addr3, common.NewHexInt(16)}
	nbs = []*Bond{b1, b2}
	ubAdds, ubMods, uDiff = a.GetUnbondingInfo(nbs, bh) // 1 will modified(hx5), 1 will added(hx4)

	expectedUDiff = big.NewInt(-5)
	ubAdd1 = &Unbond{addr2, big.NewInt(5), bh}
	ubMod1 = &Unbond{addr3, big.NewInt(0), bh}
	assert.True(t, ubAdds[0].Equal(ubAdd1))
	assert.True(t, ubMods[0].Equal(ubMod1))
	assert.Equal(t, 1, len(ubAdds))
	assert.Equal(t, 1, len(ubMods))
	assert.Equal(t, 0, uDiff.Cmp(expectedUDiff))
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
