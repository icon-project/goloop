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
	"github.com/icon-project/goloop/icon/icmodule"
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

type accountData struct {
	stake *big.Int

	unstakes    Unstakes
	delegations Delegations
	bonds       Bonds
	unbonds     Unbonds

	totalDelegation *big.Int
	totalBond       *big.Int
	totalUnbond     *big.Int
}

func (a *accountData) equal(other *accountData) bool {
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

func (a accountData) clone() accountData {
	return accountData{
		stake: a.stake,

		unstakes:    a.unstakes.Clone(),
		delegations: a.delegations.Clone(),
		bonds:       a.bonds.Clone(),
		unbonds:     a.unbonds.Clone(),

		totalDelegation: a.totalDelegation,
		totalBond:       a.totalBond,
		totalUnbond:     a.totalUnbond,
	}
}

func (a accountData) IsEmpty() bool {
	return (a.stake == nil || a.stake.Sign() == 0) && len(a.unstakes) == 0
}

func (a accountData) Stake() *big.Int {
	return a.stake
}

func (a accountData) UnStakes() Unstakes {
	return a.unstakes
}

func (a accountData) GetUnstakeAmount() *big.Int {
	return a.unstakes.GetUnstakeAmount()
}

func (a accountData) GetTotalStake() *big.Int {
	return new(big.Int).Add(a.stake, a.unstakes.GetUnstakeAmount())
}

func (a *accountData) Delegating() *big.Int {
	return a.totalDelegation
}

func (a *accountData) Delegations() Delegations {
	return a.delegations
}

func (a accountData) GetStakeInJSON(blockHeight int64) map[string]interface{} {
	jso := make(map[string]interface{})
	jso["stake"] = a.stake
	jso["unstakes"] = a.unstakes.ToJSON(module.JSONVersion3, blockHeight)
	return jso
}

func (a accountData) GetDelegationInJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["totalDelegated"] = a.totalDelegation
	jso["votingPower"] = a.GetVotingPower()
	jso["delegations"] = a.delegations.ToJSON(module.JSONVersion3)
	return jso
}

func (a *accountData) GetVotingPower() *big.Int {
	return new(big.Int).Sub(a.stake, a.UsingStake())
}

func (a *accountData) GetVoting() *big.Int {
	return new(big.Int).Add(a.Bond(), a.Delegating())
}

func (a *accountData) UsingStake() *big.Int {
	using := a.GetVoting()
	return using.Add(using, a.totalUnbond)
}

func (a *accountData) Bond() *big.Int {
	return a.totalBond
}

func (a *accountData) Bonds() Bonds {
	return a.bonds
}

func (a *accountData) Unbonds() Unbonds {
	return a.unbonds
}

func (a *accountData) Unbond() *big.Int {
	return a.totalUnbond
}

func (a *accountData) GetBondInJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["bonds"] = a.bonds.ToJSON(module.JSONVersion3)
	jso["unbonds"] = a.unbonds.ToJSON(module.JSONVersion3)
	jso["totalBonded"] = a.totalBond
	jso["votingPower"] = a.GetVotingPower()
	return jso
}

func (a *accountData) GetUnbondsInJSON() []interface{} {
	return a.unbonds.ToJSON(module.JSONVersion3)
}

func (a *accountData) String() string {
	return fmt.Sprintf(
		"stake=%s unstake=%s totalDelegation=%s totalBond=%s totalUnbond=%s",
		a.stake, a.unstakes.GetUnstakeAmount(), a.totalDelegation, a.totalBond, a.totalUnbond,
	)
}

func (a *accountData) Format(f fmt.State, c rune) {
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

type AccountSnapshot struct {
	icobject.NoDatabase
	accountData
}

func (a *AccountSnapshot) Equal(object icobject.Impl) bool {
	other, ok := object.(*AccountSnapshot)
	if !ok {
		return false
	}
	if a == other {
		return true
	}
	return a.equal(&other.accountData)
}

func (a *AccountSnapshot) Version() int {
	return accountVersion
}

func (a *AccountSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.DecodeAll(
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

func (a *AccountSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
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

var emptyAccountData = accountData{
	stake:           new(big.Int),
	totalDelegation: new(big.Int),
	totalBond:       new(big.Int),
	totalUnbond:     new(big.Int),
}

var emptyAccountSnapshot = &AccountSnapshot{
	accountData: emptyAccountData,
}

func newAccountWithTag(_ icobject.Tag) *AccountSnapshot {
	// versioning with tag.Version() if necessary
	return new(AccountSnapshot)
}

type AccountState struct {
	snapshot *AccountSnapshot
	accountData
}

func (a *AccountState) Reset(s *AccountSnapshot) {
	if a.snapshot == s {
		return
	}
	a.snapshot = s
	a.accountData = s.accountData.clone()
}

func (a *AccountState) GetSnapshot() *AccountSnapshot {
	if a.snapshot == nil {
		a.snapshot = &AccountSnapshot{
			accountData: a.accountData.clone(),
		}
	}
	return a.snapshot
}

func (a *AccountState) Clear() {
	a.accountData = emptyAccountData
	a.snapshot = emptyAccountSnapshot
}

func (a *AccountState) setDirty() {
	if a.snapshot != nil {
		a.snapshot = nil
	}
}

func (a *AccountState) SetStake(v *big.Int) error {
	if v.Sign() == -1 {
		return errors.Errorf("negative stake is not allowed")
	}
	a.stake = v
	a.setDirty()
	return nil
}

func (a *AccountState) DecreaseUnstake(stakeInc *big.Int, expireHeight int64, revision int) ([]TimerJobInfo, error) {
	if tj, err := a.unstakes.decreaseUnstake(stakeInc, expireHeight, revision); err != nil {
		return nil, err
	} else {
		a.setDirty()
		return tj, nil
	}
}

func (a *AccountState) IncreaseUnstake(stakeInc *big.Int, expireHeight int64, slotMax, revision int) ([]TimerJobInfo, error) {
	if tj, err := a.unstakes.increaseUnstake(new(big.Int).Abs(stakeInc), expireHeight, slotMax, revision); err != nil {
		return nil, err
	} else {
		a.setDirty()
		return tj, nil
	}
}

func (a *AccountState) SetDelegation(ds Delegations) {
	a.delegations = ds
	a.totalDelegation = a.delegations.GetDelegationAmount()
	a.setDirty()
}

func (a *AccountState) SetBonds(bonds Bonds) {
	a.bonds = bonds
	a.totalBond = a.bonds.GetBondAmount()
	a.setDirty()
}

func (a *AccountState) UpdateUnbonds(bondDelta map[string]*big.Int, expireHeight int64) ([]TimerJobInfo, error) {
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
	a.setDirty()
	return tl, nil
}

func (a *AccountState) RemoveUnbond(height int64) error {
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
		return errors.Errorf("Unbond timer not found at %d", height)
	}
	a.unbonds = tmp
	a.totalUnbond = new(big.Int).Sub(a.Unbond(), removed)
	a.setDirty()
	return nil
}

func (a *AccountState) RemoveUnstake(height int64) (ra *big.Int, err error) {
	var tmp Unstakes
	ra = new(big.Int)
	for _, u := range a.unstakes {
		if u.GetExpire() == height {
			ra.Add(ra, u.GetValue())
		} else {
			tmp = append(tmp, u)
		}
	}
	if len(tmp) == len(a.unstakes) {
		return nil, errors.Errorf("Unstaking timer not found at %d", height)
	}
	a.unstakes = tmp
	a.setDirty()
	return
}

func (a *AccountState) SlashStake(amount *big.Int) error {
	stake := new(big.Int).Set(a.Stake())
	stake.Sub(stake, amount)
	return a.SetStake(stake)
}

func (a *AccountState) SlashBond(address module.Address, rate icmodule.Rate) *big.Int {
	newBonds, amount := a.bonds.Slash(address, rate)
	a.bonds = newBonds
	a.totalBond = new(big.Int).Sub(a.totalBond, amount)
	a.setDirty()
	return amount
}

func (a *AccountState) SlashUnbond(address module.Address, rate icmodule.Rate) (*big.Int, int64) {
	newUnbonds, amount, expire := a.unbonds.Slash(address, rate)
	a.unbonds = newUnbonds
	a.totalUnbond = new(big.Int).Sub(a.totalUnbond, amount)
	a.setDirty()
	return amount, expire
}

func newAccountStateWithSnapshot(ass *AccountSnapshot) *AccountState {
	a := new(AccountState)
	if ass == nil {
		ass = emptyAccountSnapshot
	}
	a.Reset(ass)
	return a
}

func GetEmptyAccountSnapshot() *AccountSnapshot {
	return emptyAccountSnapshot
}
