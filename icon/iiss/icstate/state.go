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
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/ompt"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/iccache"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

var (
	IssueKey          = containerdb.ToKey(containerdb.HashBuilder, "issue_icx").Build()
	RewardCalcInfoKey = containerdb.ToKey(containerdb.HashBuilder, "reward_calc_info").Build()
	LastValidatorsKey = containerdb.ToKey(
		containerdb.HashBuilder, scoredb.ArrayDBPrefix, "last_validators",
	)
	UnstakeSlotMaxKey = containerdb.ToKey(
		containerdb.HashBuilder, scoredb.VarDBPrefix, "unstake_slot_max",
	)
)

type State struct {
	readonly            bool
	accountCache        *AccountCache
	activePRepCache     *ActivePRepCache
	nodeOwnerCache      *NodeOwnerCache
	prepBaseCache       *PRepBaseCache
	prepStatusCache     *PRepStatusCache
	unstakingTimerCache *TimerCache
	unbondingTimerCache *TimerCache
	termCache           *termCache
	store               *icobject.ObjectStoreState
}

func (s *State) Reset(ss *Snapshot) error {
	var err error
	s.store.Reset(ss.store.ImmutableForObject)
	s.accountCache.Reset()
	s.activePRepCache.Reset()
	s.nodeOwnerCache.Reset()
	s.prepBaseCache.Reset()
	s.prepStatusCache.Reset()
	s.unstakingTimerCache.Reset()
	s.unbondingTimerCache.Reset()
	if err = s.termCache.Reset(); err != nil {
		return err
	}
	return nil
}

func (s *State) Flush() error {
	s.accountCache.Flush()
	s.activePRepCache.Flush()
	s.nodeOwnerCache.Flush()
	s.prepBaseCache.Flush()
	s.prepStatusCache.Flush()
	s.unstakingTimerCache.Flush()
	s.unbondingTimerCache.Flush()
	return s.termCache.Flush()
}

func (s *State) GetSnapshot() *Snapshot {
	if err := s.Flush(); err != nil {
		panic(err)
	}
	return newSnapshotFromImmutableForObject(s.store.GetSnapshot())
}

func (s *State) GetAccountState(addr module.Address) *AccountState {
	a := s.accountCache.Get(addr, true)
	return a
}

func (s *State) GetAccountSnapshot(addr module.Address) *AccountSnapshot {
	return s.accountCache.GetSnapshot(addr)
}

func (s *State) GetUnstakingTimerState(height int64) *TimerState {
	timer := s.unstakingTimerCache.Get(height)
	return timer
}

func (s *State) GetUnstakingTimerSnapshot(height int64) *TimerSnapshot {
	return s.unstakingTimerCache.GetSnapshot(height)
}

func (s *State) GetUnbondingTimerState(height int64) *TimerState {
	timer := s.unbondingTimerCache.Get(height)
	return timer
}

func (s *State) GetUnbondingTimerSnapshot(height int64) *TimerSnapshot {
	timer := s.unbondingTimerCache.GetSnapshot(height)
	return timer
}

func (s *State) AddActivePRep(owner module.Address) {
	s.activePRepCache.Add(owner)
}

func (s *State) RemoveActivePRep(owner module.Address) error {
	return s.activePRepCache.Remove(owner)
}

func (s *State) GetActivePRepSize() int {
	return s.activePRepCache.Size()
}

func (s *State) GetActivePRep(i int) module.Address {
	return s.activePRepCache.Get(i)
}

/*func (s *State) AddPRepBase(base *PRepBase) {
	s.prepBaseCache.Add(base)
}*/

func (s *State) GetPRepBase(owner module.Address, createIfNotExist bool) *PRepBase {
	return s.prepBaseCache.Get(owner, createIfNotExist)
}

func (s *State) GetPRepStatus(owner module.Address, createIfNotExist bool) *PRepStatus {
	return s.prepStatusCache.Get(owner, createIfNotExist)
}

func NewStateFromSnapshot(ss *Snapshot, readonly bool) *State {
	t := trie_manager.NewMutableFromImmutableForObject(ss.store.ImmutableForObject)
	if c := iccache.StateNodeCacheOf(t.Database()); c != nil && !readonly {
		ompt.SetCacheOfMutableForObject(t, c)
	}
	return NewStateFromTrie(t, readonly)
}

func NewStateFromTrie(t trie.MutableForObject, readonly bool) *State {
	store := icobject.NewObjectStoreState(t)

	s := &State{
		readonly:            readonly,
		accountCache:        newAccountCache(store),
		activePRepCache:     newActivePRepCache(store),
		nodeOwnerCache:      newNodeOwnerCache(store),
		prepBaseCache:       newPRepBaseCache(store),
		prepStatusCache:     newPRepStatusCache(store),
		unstakingTimerCache: newTimerCache(store, unstakingTimerDictPrefix),
		unbondingTimerCache: newTimerCache(store, unbondingTimerDictPrefix),
		termCache:           newTermCache(store),
		store:               store,
	}

	return s
}

func (s *State) AddNodeToOwner(node, owner module.Address) error {
	return s.nodeOwnerCache.Add(node, owner)
}

func (s *State) GetOwnerByNode(node module.Address) module.Address {
	return s.nodeOwnerCache.Get(node)
}

func (s *State) SetIssue(issue *Issue) error {
	_, err := s.store.Set(IssueKey, icobject.New(TypeIssue, issue))
	if err != nil {
		return err
	}
	return nil
}

func (s *State) GetIssue() (*Issue, error) {
	obj, err := s.store.Get(IssueKey)
	if err != nil {
		return nil, err
	}
	issue := ToIssue(obj)
	if issue == nil {
		issue = NewIssue()
	}
	return issue, nil
}

func (s *State) GetTerm() *Term {
	return s.termCache.Get()
}

func (s *State) SetTerm(term *Term) error {
	return s.termCache.Set(term)
}

func (s *State) SetRewardCalcInfo(rc *RewardCalcInfo) error {
	_, err := s.store.Set(RewardCalcInfoKey, icobject.New(TypeRewardCalcInfo, rc))
	if err != nil {
		return err
	}
	return nil
}

func (s *State) GetRewardCalcInfo() (*RewardCalcInfo, error) {
	obj, err := s.store.Get(RewardCalcInfoKey)
	if err != nil {
		return nil, err
	}
	rc := ToRewardCalcInfo(obj)
	if rc == nil {
		rc = NewRewardCalcInfo()
	}
	return rc, nil
}

func (s *State) SetLastValidators(al []module.Address) error {
	var err error
	db := containerdb.NewArrayDB(s.store, LastValidatorsKey)
	size := db.Size()
	for i, a := range al {
		value := a.Bytes()
		if i < size {
			err = db.Set(i, value)
		} else {
			err = db.Put(value)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *State) GetLastValidators() []module.Address {
	db := containerdb.NewArrayDB(s.store, LastValidatorsKey)
	size := db.Size()
	al := make([]module.Address, size, size)
	for i := 0; i < size; i += 1 {
		al[i] = db.Get(i).Address()
	}
	return al
}

func (s *State) SetUnstakeSlotMax(v int64) error {
	db := containerdb.NewVarDB(s.store, UnstakeSlotMaxKey)
	err := db.Set(v)
	return err
}

func (s *State) GetUnstakeSlotMax() int64 {
	db := containerdb.NewVarDB(s.store, UnstakeSlotMaxKey)
	return db.Int64()
}

func (s *State) ClearCache() {
	s.accountCache.Clear()
	s.unstakingTimerCache.Clear()
	s.unbondingTimerCache.Clear()
	// TODO clear other caches
	s.store.ClearCache()
}
