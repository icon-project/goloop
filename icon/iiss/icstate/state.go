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
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

type State struct {
	mutableAccounts       map[string]*AccountState
	mutablePReps          map[string]*PRepState
	mutablePRepStatus     map[string]*PRepStatusState
	mutableUnstakingTimer map[int64]*TimerState
	mutableUnbondingTimer map[int64]*TimerState
	store                 *icobject.ObjectStoreState
}

func (s *State) Reset(ss *Snapshot) error {
	s.store.Reset(ss.store)
	for _, as := range s.mutableAccounts {
		key := crypto.SHA3Sum256(scoredb.AppendKeys(accountPrefix, as.Address()))
		value, err := icobject.GetFromMutableForObject(s.store, key)
		if err != nil {
			return err
		}
		if value == nil {
			as.Clear()
		} else {
			as.Reset(ToAccountSnapshot(value))
		}
	}
	for _, ps := range s.mutablePReps {
		key := crypto.SHA3Sum256(scoredb.AppendKeys(prepPrefix, ps.Owner()))
		value, err := icobject.GetFromMutableForObject(s.store, key)
		if err != nil {
			return err
		}
		if value == nil {
			ps.Clear()
		} else {
			ps.Reset(ToPRepSnapshot(value))
		}
	}
	for _, ps := range s.mutablePRepStatus {
		key := crypto.SHA3Sum256(scoredb.AppendKeys(prepStatusPrefix, ps.Address()))
		value, err := icobject.GetFromMutableForObject(s.store, key)
		if err != nil {
			return err
		}
		if value == nil {
			ps.Clear()
		} else {
			ps.Reset(ToPRepStatusSnapshot(value))
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

	for _, ps := range s.mutablePReps {
		key := crypto.SHA3Sum256(scoredb.AppendKeys(prepPrefix, ps.Owner()))
		value := icobject.New(TypePRep, ps.GetSnapshot())

		if ps.IsEmpty() {
			if _, err := s.store.Delete(key); err != nil {
				log.Errorf("Failed to delete prep key %x, err+%+v", key, err)
			}
		} else {
			if _, err := s.store.Set(key, value); err != nil {
				log.Errorf("Failed to set snapshot for %x, err+%+v", key, err)
			}
		}
	}

	for _, ps := range s.mutablePRepStatus {
		key := crypto.SHA3Sum256(scoredb.AppendKeys(prepStatusPrefix, ps.Address()))
		value := icobject.New(TypePRepStatus, ps.GetSnapshot())

		if ps.IsEmpty() {
			if _, err := s.store.Delete(key); err != nil {
				log.Errorf("Failed to delete prepStatus key %x, err+%+v", key, err)
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

	return newSnapshotFromImmutableForObject(s.store.GetSnapshot())
}

func (s *State) GetAccountState(addr module.Address) (*AccountState, error) {
	ids := addr.String()
	if a, ok := s.mutableAccounts[ids]; ok {
		return a, nil
	}
	key := crypto.SHA3Sum256(scoredb.AppendKeys(accountPrefix, addr))
	obj, err := icobject.GetFromMutableForObject(s.store, key)
	if err != nil {
		return nil, err
	}
	var ass *AccountSnapshot
	if obj != nil {
		ass = ToAccountSnapshot(obj)
	} else {
		ass = newAccountSnapshot(icobject.MakeTag(TypeAccount, accountVersion))
	}
	as := NewAccountStateWithSnapshot(addr, ass)
	s.mutableAccounts[ids] = as
	return as, nil
}

func (s *State) GetPRepState(addr module.Address) (*PRepState, error) {
	ids := addr.String()
	if a, ok := s.mutablePReps[ids]; ok {
		return a, nil
	}
	key := crypto.SHA3Sum256(scoredb.AppendKeys(prepPrefix, addr))
	obj, err := icobject.GetFromMutableForObject(s.store, key)
	if err != nil {
		return nil, err
	}
	var pss *PRepSnapshot
	if obj != nil {
		pss = ToPRepSnapshot(obj)
	} else {
		pss = newPRepSnapshot(icobject.MakeTag(TypePRep, prepVersion))
	}
	ps := NewPRepStateWithSnapshot(addr, pss)
	s.mutablePReps[ids] = ps
	return ps, nil
}

func (s *State) GetPRepStatusState(addr module.Address) (*PRepStatusState, error) {
	ids := addr.String()
	if a, ok := s.mutablePRepStatus[ids]; ok {
		return a, nil
	}
	key := crypto.SHA3Sum256(scoredb.AppendKeys(prepStatusPrefix, addr))
	obj, err := icobject.GetFromMutableForObject(s.store, key)
	if err != nil {
		return nil, err
	}
	var pss *PRepStatusSnapshot
	if obj != nil {
		pss = ToPRepStatusSnapshot(obj)
	} else {
		pss = newPRepStatusSnapshot(icobject.MakeTag(TypePRepStatus, prepStatusVersion))
	}
	ps := NewPRepStatusStateWithSnapshot(addr, pss)
	s.mutablePRepStatus[ids] = ps
	return ps, nil
}

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
func NewStateFromSnapshot(ss *Snapshot) *State {
	trie := trie_manager.NewMutableFromImmutableForObject(ss.store)

	return &State{
		mutableAccounts:       make(map[string]*AccountState),
		mutablePReps:          make(map[string]*PRepState),
		mutablePRepStatus:     make(map[string]*PRepStatusState),
		mutableUnstakingTimer: make(map[int64]*TimerState),
		mutableUnbondingTimer: make(map[int64]*TimerState),
		store:                 icobject.NewObjectStoreState(trie),
	}
}
