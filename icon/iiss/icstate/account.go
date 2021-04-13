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
	"fmt"
	"math/big"
	"sort"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

const (
	accountVersion1 = iota + 1
	accountVersion  = accountVersion1
)

var AccountDictPrefix = containerdb.ToKey(
	containerdb.HashBuilder,
	scoredb.DictDBPrefix,
	"account_db",
)

// Account containing IISS information
type Account struct {
	icobject.NoDatabase
	StateAndSnapshot
	address module.Address

	stake       *big.Int
	unstakes    Unstakes
	delegating  *big.Int
	delegations Delegations
	bonding     *big.Int
	unbonding   *big.Int
	bonds       Bonds
	unbonds     Unbonds
}

func (a *Account) Address() module.Address {
	return a.address
}

func (a *Account) SetAddress(address module.Address) {
	a.checkWritable()
	a.address = address
}

func (a *Account) equal(other *Account) bool {
	if a == other {
		return true
	}

	return icutils.EqualAddress(a.address, other.address) &&
		a.stake.Cmp(other.stake) == 0 &&
		a.unstakes.Equal(other.unstakes) &&
		a.delegating.Cmp(other.delegating) == 0 &&
		a.delegations.Equal(other.delegations) &&
		a.bonding.Cmp(other.bonding) == 0 &&
		a.unbonding.Cmp(other.unbonding) == 0 &&
		a.bonds.Equal(other.bonds) &&
		a.unbonds.Equal(other.unbonds)
}

func (a *Account) Equal(object icobject.Impl) bool {
	other, ok := object.(*Account)
	if !ok {
		return false
	}
	if a == other {
		return true
	}

	return a.equal(other)
}

func (a *Account) Set(other *Account) {
	a.checkWritable()
	a.address = other.address

	a.stake.Set(other.stake)
	a.unstakes = other.unstakes.Clone()
	a.delegating.Set(other.delegating)
	a.delegations = other.delegations.Clone()
	a.bonding.Set(other.bonding)
	a.unbonding.Set(other.unbonding)
	a.bonds = other.bonds.Clone()
	a.unbonds = other.unbonds.Clone()
}

func (a *Account) Clone() *Account {
	return &Account{
		address:     a.address,
		stake:       new(big.Int).Set(a.stake),
		unstakes:    a.unstakes.Clone(),
		delegating:  new(big.Int).Set(a.delegating),
		delegations: a.delegations.Clone(),
		bonding:     new(big.Int).Set(a.bonding),
		unbonding:   new(big.Int).Set(a.unbonding),
		bonds:       a.bonds.Clone(),
		unbonds:     a.unbonds.Clone(),
	}
}

func (a *Account) Version() int {
	return accountVersion
}

func (a *Account) RLPDecodeFields(decoder codec.Decoder) error {
	a.checkWritable()
	return decoder.DecodeListOf(
		&a.stake,
		&a.unstakes,
		&a.delegating,
		&a.delegations,
		&a.bonding,
		&a.unbonding,
		&a.bonds,
		&a.unbonds,
	)
}

func (a *Account) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
		a.stake,
		a.unstakes,
		a.delegating,
		a.delegations,
		a.bonding,
		a.unbonding,
		a.bonds,
		a.unbonds,
	)
}

func (a *Account) Clear() {
	a.checkWritable()
	a.address = nil
	a.stake = big.NewInt(0)
	a.unstakes = nil
	a.delegating = big.NewInt(0)
	a.delegations = nil
	a.bonding = big.NewInt(0)
	a.unbonding = big.NewInt(0)
	a.bonds = nil
	a.unbonds = nil
}

func (a *Account) IsEmpty() bool {
	return a.address == nil
}

// SetStake set stake Value
func (a *Account) SetStake(v *big.Int) error {
	a.checkWritable()
	if v.Sign() == -1 {
		return errors.Errorf("negative stake is not allowed")
	}
	a.stake.Set(v)
	return nil
}

func (a *Account) DecreaseUnstake(stakeInc *big.Int) ([]TimerJobInfo, error) {
	a.checkWritable()
	return a.unstakes.decreaseUnstake(stakeInc)
}

func (a *Account) IncreaseUnstake(stakeInc *big.Int, expireHeight int64, slotMax int) ([]TimerJobInfo, error) {
	a.checkWritable()
	return a.unstakes.increaseUnstake(new(big.Int).Abs(stakeInc), expireHeight, slotMax)
}

// Stake return stake Value
func (a Account) Stake() *big.Int {
	return a.stake
}

// GetUnstakeAmount return unstake Value
func (a Account) GetUnstakeAmount() *big.Int {
	return a.unstakes.GetUnstakeAmount()
}

// GetTotalStake return stake + unstake Value
func (a Account) GetTotalStake() *big.Int {
	return new(big.Int).Add(a.stake, a.unstakes.GetUnstakeAmount())
}

// GetStakeInJSON returns stake and unstake information in json format
func (a Account) GetStakeInJSON(blockHeight int64) map[string]interface{} {
	jso := make(map[string]interface{})
	jso["stake"] = a.stake
	if unstakes := a.unstakes.ToJSON(module.JSONVersion3, blockHeight); unstakes != nil {
		jso["unstakes"] = unstakes
	}
	return jso
}

func (a *Account) Delegating() *big.Int {
	return a.delegating
}

func (a *Account) Delegations() Delegations {
	return a.delegations
}

func (a *Account) SetDelegation(ds Delegations) {
	a.checkWritable()
	a.delegations = ds
	a.delegating.Set(a.delegations.GetDelegationAmount())
}

func (a Account) GetDelegationInfo() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["totalDelegated"] = a.delegating
	jso["votingPower"] = a.GetVotingPower()

	if delegations := a.delegations.ToJSON(module.JSONVersion3); delegations != nil {
		jso["delegations"] = delegations
	}

	return jso
}

func (a *Account) GetVotingPower() *big.Int {
	return new(big.Int).Sub(a.stake, a.UsingStake())
}

func (a *Account) GetVoting() *big.Int {
	return new(big.Int).Add(a.Bond(), a.Delegating())
}

func (a *Account) UsingStake() *big.Int {
	using := a.GetVoting()
	return using.Add(using, a.unbonding)
}

func (a *Account) Bond() *big.Int {
	return a.bonding
}

func (a *Account) Bonds() Bonds {
	return a.bonds
}

func (a *Account) Unbonds() Unbonds {
	return a.unbonds
}

func (a *Account) Unbond() *big.Int {
	return a.unbonding
}

func (a *Account) GetBondsInfo() []interface{} {
	return a.bonds.ToJSON(module.JSONVersion3)
}

func (a *Account) GetUnbondsInfo() []interface{} {
	return a.unbonds.ToJSON(module.JSONVersion3)
}

func (a *Account) SetBonds(bonds Bonds) {
	a.checkWritable()
	a.bonds = bonds
	a.bonding.Set(a.bonds.GetBondAmount())
}

func (a *Account) UpdateUnbonds(bondDelta map[string]*big.Int, expireHeight int64) ([]TimerJobInfo, error) {
	a.checkWritable()
	var tl []TimerJobInfo

	// sort key of bondDelta
	size := len(bondDelta)
	keys := make([]string, 0, size)
	for key := range bondDelta {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	unbondsMapByAddr := a.unbonds.MapByAddr()
	expireRefCount := a.unbonds.ExpireRefCount()

	for _, key := range keys {
		value := bondDelta[key]
		sign := value.Sign()
		if sign == 0 {
			// there is no change
			continue
		}
		unbond, ok := unbondsMapByAddr[key]
		if sign == -1 { // value is negative. increase unbond value
			if expireRefCount[expireHeight] == 0 {
				// add new timer
				tl = append(tl, TimerJobInfo{JobTypeAdd, expireHeight})
			}
			expireRefCount[expireHeight]++

			if ok {
				if expireRefCount[unbond.Expire] == 1 {
					tl = append(tl, TimerJobInfo{JobTypeRemove, unbond.Expire})
				}
				expireRefCount[unbond.Expire]--
				// update unbond
				unbond.Value.Sub(unbond.Value, value)
				unbond.Expire = expireHeight
			} else {
				// add new unbond
				addr, err := common.NewAddress([]byte(key))
				if err != nil {
					return nil, err
				}

				a.unbonds.Add(addr, new(big.Int).Neg(value), expireHeight)
			}
		} else { // value is positive. decrease unbond value
			if ok {
				// decrease unbond value
				unbond.Value.Sub(unbond.Value, value)
				if unbond.Value.Sign() <= 0 {
					// remove unbond
					addr, err := common.NewAddress([]byte(key))
					if err != nil {
						return nil, err
					}
					if err := a.unbonds.DeleteByAddress(addr); err != nil {
						return nil, err
					}
					if expireRefCount[unbond.Expire] == 1 {
						// remove timer
						tl = append(tl, TimerJobInfo{JobTypeRemove, unbond.Expire})
					}
					expireRefCount[unbond.Expire]--
				}
			} else {
				// do nothing
			}
		}
	}
	a.unbonding.Set(a.unbonds.GetUnbondAmount())
	return tl, nil
}

func (a *Account) RemoveUnbonding(height int64) error {
	a.checkWritable()
	var tmp Unbonds
	removed := new(big.Int)
	for _, u := range a.unbonds {
		if u.Expire != height {
			tmp = append(tmp, u)
		} else {
			removed.Add(removed, u.Value)
		}
	}

	if len(tmp) == len(a.unbonds) {
		return errors.Errorf("%s does not have unbonding entry with expire(%d)", a.address, height)
	}
	a.unbonds = tmp
	a.unbonding.Sub(a.Unbond(), removed)

	return nil
}

func (a *Account) RemoveUnstaking(height int64) (ra *big.Int, err error) {
	a.checkWritable()
	var tmp Unstakes
	ra = new(big.Int)
	for _, u := range a.unstakes {
		if u.ExpireHeight == height {
			ra.Set(u.Amount)
		} else {
			tmp = append(tmp, u)
		}
	}
	tl := len(tmp)
	ul := len(a.unstakes)

	if tl == ul {
		err = errors.Errorf("%s does not have unstaking timer at %d", a.address.String(), height)
	} else if tl != ul-1 {
		err = errors.Errorf("%s has too many unstaking timer at %d", a.address.String(), height)
	}
	a.unstakes = tmp

	return
}

func (a *Account) SlashStake(amount *big.Int) error {
	stake := new(big.Int).Set(a.Stake())
	stake.Sub(stake, amount)
	return a.SetStake(stake)
}

func (a *Account) SlashBond(address module.Address, ratio int) *big.Int {
	amount := a.bonds.Slash(address, ratio)
	a.bonding.Sub(a.bonding, amount)
	return amount
}

func (a *Account) SlashUnbond(address module.Address, ratio int) (*big.Int, int64) {
	amount, expire := a.unbonds.Slash(address, ratio)
	a.unbonding.Sub(a.unbonding, amount)
	return amount, expire
}

func (a *Account) GetSnapshot() *Account {
	if a.IsReadonly() {
		return a
	}
	ret := a.Clone()
	ret.freeze()
	return ret
}

func (a *Account) String() string {
	return fmt.Sprintf(
		"addr=%s stake=%s unstake=%s delegating=%s bonding=%s unbonding=%s",
		a.address, a.stake, a.unstakes.GetUnstakeAmount(), a.delegating, a.bonding, a.unbonding,
	)
}

func (a *Account) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "Account{stake=%d unstakes=%+v delegating=%d delegations=%+v bonding=%d unbonding=%d bonds=%+v unbonds=%+v}",
				a.stake, a.unstakes, a.delegating, a.delegations, a.bonding, a.unbonding, a.bonds, a.unbonds)
		} else {
			fmt.Fprintf(f, "Account{%d %v %d %v %d %d %v %v}",
				a.stake, a.unstakes, a.delegating, a.delegations, a.bonding, a.unbonding, a.bonds, a.unbonds)
		}
	case 's':
		fmt.Fprint(f, a.String())
	}
}

func newAccountWithTag(_ icobject.Tag) *Account {
	// versioning with tag.Version() if necessary
	return &Account{}
}

func newAccount(addr module.Address) *Account {
	return &Account{
		address:    addr,
		stake:      new(big.Int),
		delegating: new(big.Int),
		bonding:    new(big.Int),
		unbonding:  new(big.Int),
	}
}
