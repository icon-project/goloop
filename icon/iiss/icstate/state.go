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
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"math/big"
)

type State struct {
	readonly              bool
	mutableAccounts       map[string]*Account
	mutableUnstakingTimer map[int64]*TimerState
	mutableUnbondingTimer map[int64]*TimerState
	store                 *icobject.ObjectStoreState
	pm                    *PRepManager
}

func (s *State) Reset(ss *Snapshot) error {
	s.store.Reset(ss.store.ImmutableForObject)
	for _, as := range s.mutableAccounts {
		address := as.Address()
		key := crypto.SHA3Sum256(scoredb.AppendKeys(accountPrefix, address))
		value, err := icobject.GetFromMutableForObject(s.store, key)
		if err != nil {
			return err
		}
		if value == nil {
			as.Clear()
		} else {
			as.Set(ToAccount(value, address))
		}
	}
	for _, ubt := range s.mutableUnbondingTimer {
		key := crypto.SHA3Sum256(scoredb.AppendKeys(unbondingTimerPrefix, ubt.Height))
		value, err := s.store.Get(key)
		if err != nil {
			return err
		}
		if value == nil {
			ubt.Clear()
		} else {
			ubt.Reset(ToTimerSnapshot(value))
		}
	}
	for _, ust := range s.mutableUnstakingTimer {
		key := crypto.SHA3Sum256(scoredb.AppendKeys(unstakingTimerPrefix, ust.Height))
		value, err := s.store.Get(key)
		if err != nil {
			return err
		}
		if value == nil {
			ust.Clear()
		} else {
			ust.Reset(ToTimerSnapshot(value))
		}
	}

	if err := s.pm.Reset(); err != nil {
		return err
	}
	return nil
}

func (s *State) GetSnapshot() *Snapshot {
	for _, as := range s.mutableAccounts {
		key := crypto.SHA3Sum256(scoredb.AppendKeys(accountPrefix, as.Address()))
		value := icobject.New(TypeAccount, as.GetSnapshot())

		if as.IsEmpty() {
			if _, err := s.store.Delete(key); err != nil {
				log.Errorf("Failed to delete account key %x, err+%+v", key, err)
			}
		} else {
			if _, err := s.store.Set(key, value); err != nil {
				log.Errorf("Failed to set snapshot for %x, err+%+v", key, err)
			}
		}
	}

	for _, timer := range s.mutableUnstakingTimer {
		key := crypto.SHA3Sum256(scoredb.AppendKeys(unstakingTimerPrefix, timer.Height))
		value := icobject.New(TypePRepStatus, timer.GetSnapshot())

		if timer.IsEmpty() {
			if _, err := s.store.Delete(key); err != nil {
				log.Errorf("Failed to delete Timer key %x, err+%+v", key, err)
			}
		} else {
			if _, err := s.store.Set(key, value); err != nil {
				log.Errorf("Failed to set snapshot for %x, err+%+v", key, err)
			}
		}
	}
	for _, timer := range s.mutableUnbondingTimer {
		key := crypto.SHA3Sum256(scoredb.AppendKeys(unbondingTimerPrefix, timer.Height))
		value := icobject.New(TypePRepStatus, timer.GetSnapshot())

		if timer.IsEmpty() {
			if _, err := s.store.Delete(key); err != nil {
				log.Errorf("Failed to delete Timer key %x, err+%+v", key, err)
			}
		} else {
			if _, err := s.store.Set(key, value); err != nil {
				log.Errorf("Failed to set snapshot for %x, err+%+v", key, err)
			}
		}
	}

	if err := s.pm.GetSnapshot(); err != nil {
		panic(err)
	}

	return newSnapshotFromImmutableForObject(s.store.GetSnapshot())
}

func (s *State) GetAccount(addr module.Address) (*Account, error) {
	ids := addr.String()
	if a, ok := s.mutableAccounts[ids]; ok {
		return a, nil
	}
	key := crypto.SHA3Sum256(scoredb.AppendKeys(accountPrefix, addr))
	obj, err := icobject.GetFromMutableForObject(s.store, key)
	if err != nil {
		return nil, err
	}
	var as *Account
	if obj != nil {
		as = ToAccount(obj, addr)
	} else {
		as = newAccountWithTag(icobject.MakeTag(TypeAccount, accountVersion))
	}
	s.mutableAccounts[ids] = as
	return as, nil
}

//func (s *State) GetPRepBase(owner module.Address) (*PRepBase, error) {
//	ids := icutils.ToKey(owner)
//	if a, ok := s.mutablePRepBases[ids]; ok {
//		return a, nil
//	}
//
//	key := crypto.SHA3Sum256(scoredb.AppendKeys(prepPrefix, owner))
//	obj, err := icobject.GetFromMutableForObject(s.store, key)
//	if err != nil {
//		return nil, err
//	}
//
//	var pb *PRepBase
//	if obj != nil {
//		pb = ToPRepBase(obj, owner)
//	} else {
//		pb = newPRepBaseWithTag(icobject.MakeTag(TypePRepBase, prepVersion))
//	}
//
//	pb.SetOwner(owner)
//	s.mutablePRepBases[ids] = pb
//	return pb, nil
//}

//func (s *State) GetPRepStatus(owner module.Address) (*PRepStatus, error) {
//	ids := icutils.ToKey(owner)
//	if a, ok := s.mutablePRepStatuses[ids]; ok {
//		return a, nil
//	}
//	key := crypto.SHA3Sum256(scoredb.AppendKeys(prepStatusPrefix, owner))
//	obj, err := icobject.GetFromMutableForObject(s.store, key)
//	if err != nil {
//		return nil, err
//	}
//
//	var ps *PRepStatus
//	if obj != nil {
//		ps = ToPRepStatus(obj, owner)
//	} else {
//		ps = newPRepStatusWithTag(icobject.MakeTag(TypePRepStatus, prepStatusVersion))
//	}
//
//	s.mutablePRepStatuses[ids] = ps
//	return ps, nil
//}

func (s *State) GetUnstakingTimerState(height int64) (*TimerState, error) {
	if a, ok := s.mutableUnstakingTimer[height]; ok {
		return a, nil
	}
	obj, err := s.store.Get(crypto.SHA3Sum256(scoredb.AppendKeys(unstakingTimerPrefix, height)))
	if err != nil {
		return nil, err
	}
	var tss *TimerSnapshot
	if obj != nil {
		tss = ToTimerSnapshot(obj)
	} else {
		tss = newTimerSnapshot(icobject.MakeTag(TypeTimer, timerVersion))
	}
	ts := NewTimerStateWithSnapshot(height, tss)
	s.mutableUnstakingTimer[height] = ts
	return ts, nil
}

func (s *State) GetUnbondingTimerState(height int64) (*TimerState, error) {
	if a, ok := s.mutableUnbondingTimer[height]; ok {
		return a, nil
	}
	obj, err := s.store.Get(crypto.SHA3Sum256(scoredb.AppendKeys(unbondingTimerPrefix, height)))
	if err != nil {
		return nil, err
	}
	var tss *TimerSnapshot
	if obj != nil {
		tss = ToTimerSnapshot(obj)
	} else {
		tss = newTimerSnapshot(icobject.MakeTag(TypeTimer, timerVersion))
	}
	ts := NewTimerStateWithSnapshot(height, tss)
	s.mutableUnbondingTimer[height] = ts
	return ts, nil
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
		ts, e := s.GetUnbondingTimerState(t.Height)
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
		readonly:              readonly,
		mutableAccounts:       make(map[string]*Account),
		mutableUnstakingTimer: make(map[int64]*TimerState),
		mutableUnbondingTimer: make(map[int64]*TimerState),
		store:                 store,
		pm:                    newPRepManager(store, big.NewInt(0)),
	}

	return s
}
