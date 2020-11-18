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
	"github.com/icon-project/goloop/module"
)

const (
	accountVersion1 = iota + 1
	accountVersion  = accountVersion1
)

var bigIntZero big.Int
var BigIntZero = &bigIntZero

type AccountSnapshot struct {
	NoDatabaseObject
	staked      *big.Int
	unstakes    Unstakes
	delegated   *big.Int
	delegations Delegations
	bonded      *big.Int
	bonds       Bonds
	unbonds     Unbonds
}

func (a *AccountSnapshot) Version() int {
	return accountVersion
}

func (a *AccountSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&a.staked,
		&a.unstakes,
		&a.delegated,
		&a.delegations,
		&a.bonded,
		&a.bonds,
		&a.unbonds,
	)
	return err
}

func (a *AccountSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		a.staked,
		a.unstakes,
		a.delegated,
		a.delegations,
		a.bonded,
		a.bonds,
		a.unbonds,
	)
}

func (a *AccountSnapshot) Equal(object ObjectImpl) bool {
	aa, ok := object.(*AccountSnapshot)
	if !ok {
		return false
	}
	if aa == a {
		return true
	}
	return a.staked.Cmp(aa.staked) == 0 &&
		a.unstakes.Equal(aa.unstakes) &&
		a.delegated.Cmp(aa.delegated) == 0 &&
		a.delegations.Equal(aa.delegations) &&
		a.bonded.Cmp(aa.bonded) == 0 &&
		a.bonds.Equal(aa.bonds) &&
		a.unbonds.Equal(aa.unbonds)
}

func newAccountSnapshot(tag Tag) *AccountSnapshot {
	// versioning with tag.Version() if necessary
	return &AccountSnapshot{
		staked:    new(big.Int),
		delegated: new(big.Int),
		bonded:    new(big.Int),
	}
}

type AccountState struct {
	address		module.Address
	staked      *big.Int
	unstakes    Unstakes
	delegated   *big.Int
	delegations Delegations
	bonded      *big.Int
	bonds       Bonds
	unbonds     Unbonds
}

func newAccountState(address module.Address) *AccountState {
	return &AccountState{
		address: address,
		staked: new(big.Int),
		delegated: new(big.Int),
		bonded: new(big.Int),
	}

}

func (as *AccountState) Clear() {
	as.staked = BigIntZero
	as.unstakes = nil
	as.delegated = BigIntZero
	as.delegations = nil
	as.bonded = BigIntZero
	as.bonds = nil
	as.unbonds = nil
}

func (as *AccountState) Reset(ass *AccountSnapshot) {
	as.staked = ass.staked
	as.unstakes = ass.unstakes.Clone()
	as.delegated = ass.delegated
	as.delegations = ass.delegations.Clone()
	as.bonded = ass.bonded
	as.bonds = ass.bonds.Clone()
	as.unbonds = ass.unbonds.Clone()
}

func (as *AccountState) GetSnapshot() *AccountSnapshot {
	ass := &AccountSnapshot{}
	ass.staked = as.staked
	ass.unstakes = as.unstakes.Clone()
	ass.delegated = as.delegated
	ass.delegations = as.delegations.Clone()
	ass.bonded = as.bonded
	ass.bonds = as.bonds.Clone()
	ass.unbonds = as.unbonds.Clone()
	return ass
}

// SetStake set stake amount
func (as *AccountState) SetStake(v *big.Int) error {
	if v.Sign() == -1 {
		return errors.Errorf("negative stake is not allowed")
	}
	as.staked = v

	return nil
}

// UpdateUnstake update unstakes
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

// GetStake return stake amount
func (as AccountState) GetStake() *big.Int {
	return as.staked
}

// GetUnstakeAmount return unstake amount
func (as AccountState) GetUnstakeAmount() *big.Int {
	return as.unstakes.GetUnstakeAmount()
}

// GetTotalStake return stake + unstake amount
func (as AccountState) GetTotalStake() *big.Int {
	return new(big.Int).Add(as.staked, as.unstakes.GetUnstakeAmount())
}

// GetStakeInfo return stake and unstake information as a json format
func (as AccountState) GetStakeInfo() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["stake"] = as.staked
	if unstakes := as.unstakes.ToJSON(module.JSONVersion3); unstakes != nil {
		jso["unstakes"] = unstakes
	}
	return jso
}


func (as *AccountState) SetDelegation(ds Delegations) {
	as.delegated.Set(ds.GetDelegationAmount())
	as.delegations = ds
}

func (as AccountState) GetDelegationInfo() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["totalDelegated"] = as.delegated
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
	return new(big.Int).Add(as.bonded, as.delegated)
}


func (as *AccountState) GetBond() *big.Int {
	return as.bonded
}

func NewAccountStateWithSnapshot(addr module.Address, ss *AccountSnapshot) *AccountState {
	as := newAccountState(addr)
	as.Reset(ss)
	return as
}
