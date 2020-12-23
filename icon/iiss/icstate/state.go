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
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
	"math/big"
)

type State struct {
	readonly            bool
	accountCache        *AccountCache
	unstakingtimerCache *TimerCache
	unbondingtimerCache *TimerCache
	store               *icobject.ObjectStoreState
	pm                  *PRepManager
}

func (s *State) Reset(ss *Snapshot) error {
	s.store.Reset(ss.store.ImmutableForObject)
	s.accountCache.Reset()
	s.unstakingtimerCache.Reset()
	s.unbondingtimerCache.Reset()

	if err := s.pm.Reset(); err != nil {
		return err
	}
	return nil
}

func (s *State) GetSnapshot() *Snapshot {
	s.accountCache.GetSnapshot()
	s.unstakingtimerCache.GetSnapshot()
	s.unbondingtimerCache.GetSnapshot()
	if err := s.pm.GetSnapshot(); err != nil {
		panic(err)
	}

	return newSnapshotFromImmutableForObject(s.store.GetSnapshot())
}

func (s *State) GetAccount(addr module.Address) (*Account, error) {
	a := s.accountCache.Get(addr)
	return a, nil
}

func (s *State) GetUnstakingTimer(height int64) (*Timer, error) {
	timer := s.unstakingtimerCache.Get(height)
	return timer, nil
}

func (s *State) GetUnbondingTimer(height int64) (*Timer, error) {
	timer := s.unbondingtimerCache.Get(height)
	return timer, nil
}

func (s *State) GetValidators() []module.Validator {
	return s.pm.GetValidators()
}

func (s *State) GetPRepsInJSON() map[string]interface{} {
	return s.pm.GetPRepsInJSON()
}

func (s *State) GetPRepInJSON(address module.Address) (map[string]interface{}, error) {
	prep := s.pm.GetPRepByOwner(address)
	if prep == nil {
		return nil, errors.Errorf("PRep not found: %s", address)
	}
	return prep.ToJSON(), nil
}

func (s *State) RegisterPRep(owner, node module.Address, params []string) error {
	return s.pm.RegisterPRep(owner, node, params)
}

func (s *State) UnregisterPRep(owner module.Address) error {
	return s.pm.UnregisterPRep(owner)
}

func (s *State) SetDelegation(from module.Address, ds Delegations) error {
	account, err := s.GetAccount(from)
	if err != nil {
		return err
	}

	if account.Stake().Cmp(new(big.Int).Add(ds.GetDelegationAmount(), account.Bond())) == -1 {
		return errors.Errorf("Not enough voting power")
	}

	account.SetDelegation(ds)
	return s.pm.ChangeDelegation(account.delegations, ds)
}

func (s *State) SetPRep(from, node module.Address, params []string) error {
	return s.pm.SetPRep(from, node, params)
}

func (s *State) SetBond(from module.Address, height int64, bonds Bonds) error {
	account, err := s.GetAccount(from)
	if err != nil {
		return err
	}

	bondAmount := big.NewInt(0)
	for _, bond := range bonds {
		bondAmount.Add(bondAmount, bond.Amount())

		prep := s.pm.GetPRepByOwner(bond.To())
		if prep == nil {
			return errors.Errorf("PRep not found: %v", from)
		}
		if !prep.BonderList().Contains(from) {
			return errors.Errorf("%s is not in bonder List of %s", from.String(), bond.Address.String())
		}

		prep.SetBonded(bond.Amount())
	}
	if account.Stake().Cmp(new(big.Int).Add(bondAmount, account.Delegating())) == -1 {
		return errors.Errorf("Not enough voting power")
	}

	ubToAdd, ubToMod, ubDiff := account.GetUnbondingInfo(bonds, height+UnbondingPeriod)
	votingAmount := new(big.Int).Add(account.Delegating(), bondAmount)
	votingAmount.Sub(votingAmount, account.Bond())
	unbondingAmount := new(big.Int).Add(account.Unbonds().GetUnbondAmount(), ubDiff)
	if account.Stake().Cmp(new(big.Int).Add(votingAmount, unbondingAmount)) == -1 {
		return errors.Errorf("Not enough voting power")
	}
	account.SetBonds(bonds)
	tl := account.UpdateUnbonds(ubToAdd, ubToMod)
	for _, t := range tl {
		ts, e := s.GetUnbondingTimer(t.Height)
		if e != nil {
			return errors.Errorf("Error while getting unbonding Timer")
		}
		if err = ScheduleTimerJob(ts, t, from); err != nil {
			return errors.Errorf("Error while scheduling Unbonding Timer Job")
		}
	}
	return nil
}

func (s *State) SetBonderList(from module.Address, bl BonderList) error {
	pb := s.pm.getPRepBase(from)
	if pb == nil {
		return errors.Errorf("PRep not found: %v", from)
	}

	var account *Account
	var err error
	for _, old := range pb.BonderList() {
		if !bl.Contains(old) {
			account, err = s.GetAccount(old)
			if err != nil {
				return err
			}
			if len(account.Bonds()) > 0 || len(account.Unbonds()) > 0 {
				return errors.Errorf("Bonding/Unbonding exist. bonds : %d, unbonds : %d", len(account.Bonds()), len(account.Unbonds()))
			}
		}
	}

	pb.SetBonderList(bl)
	return nil
}

func (s *State) GetBonderList(address module.Address) ([]interface{}, error) {
	pb := s.pm.getPRepBase(address)
	if pb == nil {
		return nil, errors.Errorf("PRep not found: %v", address)
	}
	return pb.GetBonderListInJSON(), nil
}

func NewStateFromSnapshot(ss *Snapshot, readonly bool) *State {
	t := trie_manager.NewMutableFromImmutableForObject(ss.store.ImmutableForObject)
	store := icobject.NewObjectStoreState(t)

	s := &State{
		readonly:            readonly,
		accountCache:        newAccountCache(store),
		unstakingtimerCache: newTimerCache(store, unstakingTimerDictPrefix),
		unbondingtimerCache: newTimerCache(store, unbondingTimerDictPrefix),
		store:               store,
		pm:                  newPRepManager(store, big.NewInt(0)),
	}

	return s
}
