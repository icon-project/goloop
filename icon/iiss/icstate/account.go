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

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

const (
	accountVersion1 = iota + 1
	accountVersion  = accountVersion1
)

var bigIntZero big.Int
var BigIntZero = &bigIntZero

type AccountData struct {
	stake       *big.Int
	unstakes    UnStakes
	delegating  *big.Int
	delegations Delegations
	bonding     *big.Int
	bonds       Bonds
	unbonds     Unbonds
}

func (a *AccountData) Equal(other *AccountData) bool {
	if a == other {
		return true
	}

	return a.stake.Cmp(other.stake) == 0 &&
		a.unstakes.Equal(other.unstakes) &&
		a.delegating.Cmp(other.delegating) == 0 &&
		a.delegations.Equal(other.delegations) &&
		a.bonding.Cmp(other.bonding) == 0 &&
		a.bonds.Equal(other.bonds) &&
		a.unbonds.Equal(other.unbonds)
}

func (a *AccountData) Set(other *AccountData) {
	a.stake.Set(other.stake)
	a.unstakes = other.unstakes.Clone()
	a.delegating.Set(other.delegating)
	a.delegations = other.delegations.Clone()
	a.bonding.Set(other.bonding)
	a.bonds = other.bonds.Clone()
	a.unbonds = other.unbonds.Clone()
}

func (a *AccountData) Clone() *AccountData {
	return &AccountData{
		stake:       new(big.Int).Set(a.stake),
		unstakes:    a.unstakes.Clone(),
		delegating:  new(big.Int).Set(a.delegating),
		delegations: a.delegations.Clone(),
		bonding:     new(big.Int).Set(a.bonding),
		bonds:       a.bonds.Clone(),
		unbonds:     a.unbonds.Clone(),
	}
}

type AccountSnapshot struct {
	icobject.NoDatabase
	*AccountData
}

func (a *AccountSnapshot) Version() int {
	return accountVersion
}

func (a *AccountSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&a.stake,
		&a.unstakes,
		&a.delegating,
		&a.delegations,
		&a.bonding,
		&a.bonds,
		&a.unbonds,
	)
	return err
}

func (a *AccountSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		a.stake,
		a.unstakes,
		a.delegating,
		a.delegations,
		a.bonding,
		a.bonds,
		a.unbonds,
	)
}

func (a *AccountSnapshot) Equal(object icobject.Impl) bool {
	aa, ok := object.(*AccountSnapshot)
	if !ok {
		return false
	}
	if aa == a {
		return true
	}
	return a.AccountData.Equal(aa.AccountData)
}

func newAccountSnapshot(_ icobject.Tag) *AccountSnapshot {
	// versioning with tag.Version() if necessary
	return &AccountSnapshot{
		AccountData: &AccountData{
			stake:      new(big.Int),
			delegating: new(big.Int),
			bonding:    new(big.Int),
		},
	}
}

type AccountState struct {
	address module.Address
	*AccountData
}

func (as *AccountState) Clear() {
	as.stake = BigIntZero
	as.unstakes = nil
	as.delegating = BigIntZero
	as.delegations = nil
	as.bonding = BigIntZero
	as.bonds = nil
	as.unbonds = nil
}

func (as *AccountState) Reset(ass *AccountSnapshot) {
	as.AccountData.Set(ass.AccountData)
}

func (as *AccountState) GetSnapshot() *AccountSnapshot {
	return &AccountSnapshot{AccountData: as.AccountData.Clone()}
}

func (as AccountState) IsEmpty() bool {
	return as.stake.BitLen() == 0 && as.unstakes == nil
}

// SetStake set stake Value
func (as *AccountState) SetStake(v *big.Int) error {
	if v.Sign() == -1 {
		return errors.Errorf("negative stake is not allowed")
	}
	as.stake.Set(v)

	return nil
}

// UpdateUnstake update unStakes
func (as *AccountState) UpdateUnstake(stakeInc *big.Int, expireHeight int64) ([]TimerJobInfo, error) {
	tl := make([]TimerJobInfo, 0)
	var err error
	switch stakeInc.Sign() {
	case 1:
		if tl, err = as.unstakes.decreaseUnstake(stakeInc); err != nil {
			return nil, err
		}
	case -1:
		if err := as.unstakes.increaseUnstake(new(big.Int).Abs(stakeInc), expireHeight); err != nil {
			return nil, err
		}
		tl = append(tl, TimerJobInfo{JobTypeAdd, expireHeight})
	}
	return tl, nil
}

func (as AccountState) Address() module.Address {
	return as.address
}

// Stake return stake Value
func (as AccountState) Stake() *big.Int {
	return as.stake
}

// GetUnstakeAmount return unstake Value
func (as AccountState) GetUnstakeAmount() *big.Int {
	return as.unstakes.GetUnstakeAmount()
}

// GetTotalStake return stake + unstake Value
func (as AccountState) GetTotalStake() *big.Int {
	return new(big.Int).Add(as.stake, as.unstakes.GetUnstakeAmount())
}

// GetStakeInfo return stake and unstake information as a json format
func (as AccountState) GetStakeInfo() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["stake"] = as.stake
	if unstakes := as.unstakes.ToJSON(module.JSONVersion3); unstakes != nil {
		jso["unstakes"] = unstakes
	}
	return jso
}

func (as *AccountState) SetDelegation(ds Delegations) {
	as.delegations = ds
	as.delegating.Set(as.delegations.GetDelegationAmount())
}

func (as AccountState) GetDelegationInfo() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["totalDelegated"] = as.delegating
	jso["votingPower"] = new(big.Int).Sub(as.stake, as.GetVotedPower())

	if delegations := as.delegations.ToJSON(module.JSONVersion3); delegations != nil {
		jso["delegations"] = delegations
	}

	return jso
}

func (as *AccountState) GetVotingPower() *big.Int {
	return new(big.Int).Sub(as.stake, as.GetVotedPower())
}

func (as *AccountState) GetVotedPower() *big.Int {
	return new(big.Int).Add(as.bonding, as.delegating)
}

func (as *AccountState) Bond() *big.Int {
	return as.bonding
}

func (as *AccountState) GetDelegation() *big.Int {
	return as.delegating
}

func (as *AccountState) Bonds() Bonds {
	return as.bonds
}

func (as *AccountState) Unbonds() Unbonds {
	return as.unbonds
}

func (as *AccountState) GetBondsInfo() []interface{} {
	return as.bonds.ToJSON(module.JSONVersion3)
}

func (as *AccountState) GetUnBondsInfo() []interface{} {
	return as.unbonds.ToJSON(module.JSONVersion3)
}

func (as *AccountState) GetUnBondingInfo(bonds Bonds, unBondingHeight int64) (Unbonds, Unbonds, *big.Int) {
	diff, uDiff := new(big.Int), new(big.Int)
	var ubToAdd, ubToMod []*Unbond
	for _, nb := range bonds {
		for _, ob := range as.bonds {
			if nb.Address.Equal(ob.Address) {
				diff.Sub(ob.Value.Value(), nb.Value.Value())
				if diff.Sign() == 1 {
					unbond := Unbond{nb.Address, diff, unBondingHeight}
					ubToAdd = append(ubToAdd, &unbond)
					uDiff.Add(uDiff, diff)
				} else {
					for _, ub := range as.unbonds {
						if nb.Address.Equal(ub.Address) {
							unbond := Unbond{nb.Address, ub.Value.Add(ub.Value, diff), unBondingHeight}
							ubToMod = append(ubToMod, &unbond)
							uDiff.Add(uDiff, diff)
						}
					}
				}
			}
		}
	}
	return ubToAdd, ubToMod, uDiff
}

func (as *AccountState) SetBonds(bonds Bonds) {
	as.bonds = bonds
	as.bonding.Set(as.bonds.GetBondAmount())
}

func (as *AccountState) UpdateUnBonds(ubToAdd Unbonds, ubToMod Unbonds) []TimerJobInfo {
	var tl []TimerJobInfo
	as.unbonds = append(as.unbonds, ubToAdd...)
	for _, u := range ubToAdd {
		tl = append(tl, TimerJobInfo{JobTypeAdd, u.Expire})
	}
	for _, mod := range ubToMod {
		for _, ub := range as.unbonds {
			if ub.Address.Equal(mod.Address) {
				ub.Value = mod.Value
				ub.Expire = mod.Expire
				if ub.Value.Cmp(new(big.Int)) == 0 {
					tl = append(tl, TimerJobInfo{JobTypeRemove, ub.Expire})
				}
			}
		}
	}
	return tl
}

func (as *AccountState) RemoveUnBonding(height int64) error {
	var tmp Unbonds
	for _, u := range as.unbonds {
		if u.Expire != height {
			tmp = append(tmp, u)
		}
	}

	if len(tmp) == len(as.unbonds) {
		return errors.Errorf("%s does not have unBonding timer at %d", as.address.String(), height)
	}
	as.unbonds = tmp

	return nil
}

func (as *AccountState) RemoveUnStaking(height int64) (ra *big.Int, err error) {
	var tmp UnStakes
	ra = new(big.Int)
	for _, u := range as.unstakes {
		if u.ExpireHeight == height {
			ra.Set(u.Amount)
		} else {
			tmp = append(tmp, u)
		}
	}
	tl := len(tmp)
	ul := len(as.unstakes)

	if tl == ul {
		err = errors.Errorf("%s does not have unstaking timer at %d", as.address.String(), height)
	} else if tl != ul-1 {
		err = errors.Errorf("%s has too many unstaking timer at %d", as.address.String(), height)
	}
	as.unstakes = tmp

	return
}

func NewAccountStateWithSnapshot(addr module.Address, as *AccountSnapshot) *AccountState {
	return &AccountState{
		address:     addr,
		AccountData: as.AccountData.Clone(),
	}
}
