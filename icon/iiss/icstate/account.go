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

type AccountSnapshot struct {
	icobject.NoDatabase
	staked      *big.Int
	unStakes    UnStakes
	delegated   *big.Int
	delegations Delegations
	bonded      *big.Int
	bonds       Bonds
	unBonds     UnBonds
}

func (a *AccountSnapshot) Version() int {
	return accountVersion
}

func (a *AccountSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&a.staked,
		&a.unStakes,
		&a.delegated,
		&a.delegations,
		&a.bonded,
		&a.bonds,
		&a.unBonds,
	)
	return err
}

func (a *AccountSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		a.staked,
		a.unStakes,
		a.delegated,
		a.delegations,
		a.bonded,
		a.bonds,
		a.unBonds,
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
	return a.staked.Cmp(aa.staked) == 0 &&
		a.unStakes.Equal(aa.unStakes) &&
		a.delegated.Cmp(aa.delegated) == 0 &&
		a.delegations.Equal(aa.delegations) &&
		a.bonded.Cmp(aa.bonded) == 0 &&
		a.bonds.Equal(aa.bonds) &&
		a.unBonds.Equal(aa.unBonds)
}

func newAccountSnapshot(tag icobject.Tag) *AccountSnapshot {
	// versioning with tag.Version() if necessary
	return &AccountSnapshot{
		staked:    new(big.Int),
		delegated: new(big.Int),
		bonded:    new(big.Int),
	}
}

type AccountState struct {
	address     module.Address
	staked      *big.Int
	unstakes    UnStakes
	delegating  *big.Int
	delegations Delegations
	bonding     *big.Int
	bonds       Bonds
	unbonds     UnBonds
}

func newAccountState(address module.Address) *AccountState {
	return &AccountState{
		address:    address,
		staked:     new(big.Int),
		delegating: new(big.Int),
		bonding:    new(big.Int),
	}

}

func (as *AccountState) Clear() {
	as.staked = BigIntZero
	as.unstakes = nil
	as.delegating = BigIntZero
	as.delegations = nil
	as.bonding = BigIntZero
	as.bonds = nil
	as.unbonds = nil
}

func (as *AccountState) Reset(ass *AccountSnapshot) {
	as.staked = ass.staked
	as.unstakes = ass.unStakes.Clone()
	as.delegating = ass.delegated
	as.delegations = ass.delegations.Clone()
	as.bonding = ass.bonded
	as.bonds = ass.bonds.Clone()
	as.unbonds = ass.unBonds.Clone()
}

func (as *AccountState) GetSnapshot() *AccountSnapshot {
	ass := &AccountSnapshot{}
	ass.staked = as.staked
	ass.unStakes = as.unstakes.Clone()
	ass.delegated = as.delegating
	ass.delegations = as.delegations.Clone()
	ass.bonded = as.bonding
	ass.bonds = as.bonds.Clone()
	ass.unBonds = as.unbonds.Clone()
	return ass
}

func (as AccountState) IsEmpty() bool {
	return as.staked.BitLen() == 0 && as.unstakes == nil
}

// SetStake set stake Value
func (as *AccountState) SetStake(v *big.Int) error {
	if v.Sign() == -1 {
		return errors.Errorf("negative stake is not allowed")
	}
	as.staked = v

	return nil
}

// UpdateUnstake update unStakes
func (as *AccountState) UpdateUnstake(stakeInc *big.Int, expireHeight int64) error {
	switch stakeInc.Sign() {
	case 1:
		if err := as.unstakes.decreaseUnstake(stakeInc); err != nil {
			return err
		}
	case -1:
		if err := as.unstakes.increaseUnstake(new(big.Int).Abs(stakeInc), expireHeight); err != nil {
			return err
		}
	}
	return nil
}

func (as AccountState) GetAddress() module.Address {
	return as.address
}

// GetStake return stake Value
func (as AccountState) GetStake() *big.Int {
	return as.staked
}

// GetUnstakeAmount return unstake Value
func (as AccountState) GetUnstakeAmount() *big.Int {
	return as.unstakes.GetUnstakeAmount()
}

// GetTotalStake return stake + unstake Value
func (as AccountState) GetTotalStake() *big.Int {
	return new(big.Int).Add(as.staked, as.unstakes.GetUnstakeAmount())
}

// GetStakeInfo return stake and unstake information as a json format
func (as AccountState) GetStakeInfo() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["stake"] = as.staked
	if unstakes := as.unstakes.ToJSON(module.JSONVersion3); unstakes != nil {
		jso["unStakes"] = unstakes
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
	jso["votingPower"] = new(big.Int).Sub(as.staked, as.GetVotedPower())

	if delegations := as.delegations.ToJSON(module.JSONVersion3); delegations != nil {
		jso["delegations"] = delegations
	}

	return jso
}

func (as *AccountState) GetVotingPower() *big.Int {
	return new(big.Int).Sub(as.staked, as.GetVotedPower())
}

func (as *AccountState) GetVotedPower() *big.Int {
	return new(big.Int).Add(as.bonding, as.delegating)
}

func (as *AccountState) GetBond() *big.Int {
	return as.bonding
}

func (as *AccountState) GetDelegation() *big.Int {
	return as.delegating
}

func (as *AccountState) Bonds() Bonds {
	return as.bonds
}

func (as *AccountState) UnBonds() UnBonds {
	return as.unbonds
}

func (as *AccountState) GetBondsInfo() []interface{} {
	return as.bonds.ToJSON(module.JSONVersion3)
}

func (as *AccountState) GetUnBondsInfo() []interface{} {
	return as.unbonds.ToJSON(module.JSONVersion3)
}

func (as *AccountState) GetUnBondingInfo(bonds Bonds, unBondingHeight int64) (UnBonds, UnBonds, *big.Int) {
	diff, uDiff := new(big.Int), new(big.Int)
	ubToAdd, ubToMod := make([]*Unbond, 0), make([]*Unbond, 0)
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

func (as *AccountState) UpdateUnBonds(ubToAdd UnBonds, ubToMod UnBonds) {
	as.unbonds = append(as.unbonds, ubToAdd...)
	for _, mod := range ubToMod {
		for _, ub := range as.unbonds {
			ub.Value = mod.Value
		}
	}
}

func NewAccountStateWithSnapshot(addr module.Address, ss *AccountSnapshot) *AccountState {
	as := newAccountState(addr)
	as.Reset(ss)
	return as
}
