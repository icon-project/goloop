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

	stake *big.Int

	unstakes    Unstakes
	delegations Delegations
	bonds       Bonds
	unbonds     Unbonds

	totalDelegation *big.Int
	totalBond       *big.Int
	totalUnbond     *big.Int
}

func (a *Account) equal(other *Account) bool {
	if a == other {
		return true
	}

	return a.stake.Cmp(other.stake) == 0 &&
		a.unstakes.Equal(other.unstakes) &&
		a.totalDelegation.Cmp(other.totalDelegation) == 0 &&
		a.delegations.Equal(other.delegations) &&
		a.totalBond.Cmp(other.totalBond) == 0 &&
		a.totalUnbond.Cmp(other.totalUnbond) == 0 &&
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

	a.stake = other.stake
	a.unstakes = other.unstakes.Clone()
	a.totalDelegation = other.totalDelegation
	a.delegations = other.delegations.Clone()
	a.totalBond = other.totalBond
	a.totalUnbond = other.totalUnbond
	a.bonds = other.bonds.Clone()
	a.unbonds = other.unbonds.Clone()
}

func (a *Account) Clone() *Account {
	return &Account{
		stake:           a.stake,
		unstakes:        a.unstakes.Clone(),
		delegations:     a.delegations.Clone(),
		bonds:           a.bonds.Clone(),
		unbonds:         a.unbonds.Clone(),
		totalDelegation: a.totalDelegation,
		totalBond:       a.totalBond,
		totalUnbond:     a.totalUnbond,
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
		&a.totalDelegation,
		&a.delegations,
		&a.totalBond,
		&a.totalUnbond,
		&a.bonds,
		&a.unbonds,
	)
}

func (a *Account) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
		a.stake,
		a.unstakes,
		a.totalDelegation,
		a.delegations,
		a.totalBond,
		a.totalUnbond,
		a.bonds,
		a.unbonds,
	)
}

func (a *Account) Clear() {
	a.checkWritable()
	a.stake = new(big.Int)
	a.unstakes = nil
	a.totalDelegation = new(big.Int)
	a.delegations = nil
	a.totalBond = new(big.Int)
	a.totalUnbond = new(big.Int)
	a.bonds = nil
	a.unbonds = nil
}

func (a *Account) IsEmpty() bool {
	return (a.stake == nil || a.stake.Sign() == 0) && len(a.unstakes) == 0
}

// SetStake set stake Value
func (a *Account) SetStake(v *big.Int) error {
	a.checkWritable()
	if v.Sign() == -1 {
		return errors.Errorf("negative stake is not allowed")
	}
	a.stake = v
	return nil
}

func (a *Account) DecreaseUnstake(stakeInc *big.Int, expireHeight int64, revision int) ([]TimerJobInfo, error) {
	a.checkWritable()
	return a.unstakes.decreaseUnstake(stakeInc, expireHeight, revision)
}

func (a *Account) IncreaseUnstake(stakeInc *big.Int, expireHeight int64, slotMax, revision int) ([]TimerJobInfo, error) {
	a.checkWritable()
	return a.unstakes.increaseUnstake(new(big.Int).Abs(stakeInc), expireHeight, slotMax, revision)
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
	jso["unstakes"] = a.unstakes.ToJSON(module.JSONVersion3, blockHeight)
	return jso
}

func (a *Account) Delegating() *big.Int {
	return a.totalDelegation
}

func (a *Account) Delegations() Delegations {
	return a.delegations
}

func (a *Account) SetDelegation(ds Delegations) {
	a.checkWritable()
	a.delegations = ds
	a.totalDelegation = a.delegations.GetDelegationAmount()
}

func (a Account) GetDelegationInJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["totalDelegated"] = a.totalDelegation
	jso["votingPower"] = a.GetVotingPower()
	jso["delegations"] = a.delegations.ToJSON(module.JSONVersion3)
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
	return using.Add(using, a.totalUnbond)
}

func (a *Account) Bond() *big.Int {
	return a.totalBond
}

func (a *Account) Bonds() Bonds {
	return a.bonds
}

func (a *Account) Unbonds() Unbonds {
	return a.unbonds
}

func (a *Account) Unbond() *big.Int {
	return a.totalUnbond
}

func (a *Account) GetBondsInJSON() []interface{} {
	return a.bonds.ToJSON(module.JSONVersion3)
}

func (a *Account) GetUnbondsInJSON() []interface{} {
	return a.unbonds.ToJSON(module.JSONVersion3)
}

func (a *Account) SetBonds(bonds Bonds) {
	a.checkWritable()
	a.bonds = bonds
	a.totalBond = a.bonds.GetBondAmount()
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

	ubs := a.unbonds.Clone()
	unbondsMapByAddr := ubs.MapByAddr()
	expireRefCount := ubs.ExpireRefCount() // Get ExpireRefCount

	for _, key := range keys {
		value := bondDelta[key]
		sign := value.Sign()
		if sign == 0 {
			// there is no change
			continue
		}

		unbond, ok := unbondsMapByAddr[key]
		var unbondExpireHeight int64
		if ok {
			unbondExpireHeight = unbond.Expire()
		}

		if sign == -1 { // value is negative. increase unbond value
			if expireRefCount[expireHeight] == 0 {
				// add new timer
				tl = append(tl, TimerJobInfo{JobTypeAdd, expireHeight})
			}
			expireRefCount[expireHeight]++

			if ok {
				if expireRefCount[unbond.Expire()] == 1 {
					tl = append(tl, TimerJobInfo{JobTypeRemove, unbondExpireHeight})
				}
				expireRefCount[unbondExpireHeight]--
				// update unbond
				unbond.SetValue(new(big.Int).Sub(unbond.Value(), value))
				unbond.SetExpire(expireHeight)
			} else {
				// add new unbond
				addr, err := common.NewAddress([]byte(key))
				if err != nil {
					return nil, err
				}

				ubs.Add(addr, new(big.Int).Neg(value), expireHeight)
			}
		} else { // value is positive. decrease unbond value
			if ok {
				// decrease unbond value
				unbond.SetValue(new(big.Int).Sub(unbond.Value(), value))
				if unbond.Value().Sign() <= 0 {
					// remove unbond
					addr, err := common.NewAddress([]byte(key))
					if err != nil {
						return nil, err
					}
					if err := ubs.DeleteByAddress(addr); err != nil {
						return nil, err
					}
					if expireRefCount[unbondExpireHeight] == 1 {
						// remove timer
						tl = append(tl, TimerJobInfo{JobTypeRemove, unbondExpireHeight})
					}
					expireRefCount[unbondExpireHeight]--
				}
			} else {
				// do nothing
			}
		}
	}
	a.unbonds = ubs
	a.totalUnbond = a.unbonds.GetUnbondAmount()
	return tl, nil
}

func (a *Account) RemoveUnbond(height int64) error {
	a.checkWritable()
	var tmp Unbonds
	removed := new(big.Int)
	for _, u := range a.unbonds {
		if u.Expire() != height {
			tmp = append(tmp, u)
		} else {
			removed.Add(removed, u.Value())
		}
	}

	if len(tmp) == len(a.unbonds) {
		return errors.Errorf("does not have totalUnbond entry with expire(%d)", height)
	}
	a.unbonds = tmp
	a.totalUnbond = new(big.Int).Sub(a.Unbond(), removed)

	return nil
}

func (a *Account) RemoveUnstake(height int64) (ra *big.Int, err error) {
	a.checkWritable()
	var tmp Unstakes
	ra = new(big.Int)
	for _, u := range a.unstakes {
		if u.Expire() == height {
			ra.Set(u.Value())
		} else {
			tmp = append(tmp, u)
		}
	}
	tl := len(tmp)
	ul := len(a.unstakes)

	if tl == ul {
		err = errors.Errorf("Unstaking timer not found at %d", height)
	} else if tl != ul-1 {
		err = errors.Errorf("Too many unstaking timer at %d", height)
	}
	a.unstakes = tmp

	return
}

func (a *Account) SlashStake(amount *big.Int) error {
	a.checkWritable()
	stake := new(big.Int).Set(a.Stake())
	stake.Sub(stake, amount)
	return a.SetStake(stake)
}

func (a *Account) SlashBond(address module.Address, ratio int) *big.Int {
	a.checkWritable()
	newBonds, amount := a.bonds.Slash(address, ratio)
	a.bonds = newBonds
	a.totalBond = new(big.Int).Sub(a.totalBond, amount)
	return amount
}

func (a *Account) SlashUnbond(address module.Address, ratio int) (*big.Int, int64) {
	a.checkWritable()
	newUnbonds, amount, expire := a.unbonds.Slash(address, ratio)
	a.unbonds = newUnbonds
	a.totalUnbond = new(big.Int).Sub(a.totalUnbond, amount)
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
		"stake=%s unstake=%s totalDelegation=%s totalBond=%s totalUnbond=%s",
		a.stake, a.unstakes.GetUnstakeAmount(), a.totalDelegation, a.totalBond, a.totalUnbond,
	)
}

func (a *Account) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "Account{stake=%d unstakes=%+v totalDelegation=%d delegations=%+v totalBond=%d totalUnbond=%d bonds=%+v unbonds=%+v}",
				a.stake, a.unstakes, a.totalDelegation, a.delegations, a.totalBond, a.totalUnbond, a.bonds, a.unbonds)
		} else {
			fmt.Fprintf(f, "Account{%d %v %d %v %d %d %v %v}",
				a.stake, a.unstakes, a.totalDelegation, a.delegations, a.totalBond, a.totalUnbond, a.bonds, a.unbonds)
		}
	case 's':
		fmt.Fprint(f, a.String())
	}
}

func newAccountWithTag(_ icobject.Tag) *Account {
	// versioning with tag.Version() if necessary
	return new(Account)
}

func newAccount() *Account {
	return &Account{
		stake:           new(big.Int),
		totalDelegation: new(big.Int),
		totalBond:       new(big.Int),
		totalUnbond:     new(big.Int),
	}
}
