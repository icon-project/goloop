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
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
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
	store               *icobject.ObjectStoreState
}

func (s *State) Reset(ss *Snapshot) error {
	s.store.Reset(ss.store.ImmutableForObject)
	s.accountCache.Reset()
	s.activePRepCache.Reset()
	s.nodeOwnerCache.Reset()
	s.prepBaseCache.Reset()
	s.prepStatusCache.Reset()
	s.unstakingTimerCache.Reset()
	s.unbondingTimerCache.Reset()

	return nil
}

func (s *State) GetSnapshot() *Snapshot {
	s.accountCache.GetSnapshot()
	s.activePRepCache.GetSnapshot()
	s.nodeOwnerCache.GetSnapshot()
	s.prepBaseCache.GetSnapshot()
	s.prepStatusCache.GetSnapshot()
	s.unstakingTimerCache.GetSnapshot()
	s.unbondingTimerCache.GetSnapshot()

	return newSnapshotFromImmutableForObject(s.store.GetSnapshot())
}

func (s *State) GetAccount(addr module.Address) (*Account, error) {
	a := s.accountCache.Get(addr)
	return a, nil
}

func (s *State) GetUnstakingTimer(height int64) (*Timer, error) {
	timer := s.unstakingTimerCache.Get(height)
	return timer, nil
}

func (s *State) GetUnbondingTimer(height int64) (*Timer, error) {
	timer := s.unbondingTimerCache.Get(height)
	return timer, nil
}

func NewStateFromSnapshot(ss *Snapshot, readonly bool) *State {
	t := trie_manager.NewMutableFromImmutableForObject(ss.store.ImmutableForObject)
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
		store:               store,
	}

	return s
}

func (s *State) AddUnbondingTimerToCache(h int64) *Timer {
	t := newTimer(h)
	s.unbondingTimerCache.Add(t)
	return t
}

func (s *State) AddUnstakingTimerToCache(h int64) *Timer {
	t := newTimer(h)
	s.unstakingTimerCache.Add(t)
	return t
}

func (s *State) AddActivePRep(owner module.Address) {
	s.activePRepCache.Add(owner)
}

func (s *State) GetActivePRepSize() int {
	return s.activePRepCache.Size()
}

func (s *State) GetActivePRep(i int) module.Address {
	return s.activePRepCache.Get(i)
}

func (s *State) AddPRepBase(base *PRepBase) {
	s.prepBaseCache.Add(base)
}

func (s *State) GetPRepBase(owner module.Address) *PRepBase {
	return s.prepBaseCache.Get(owner)
}

func (s *State) RemovePRepBase(owner module.Address) error {
	return s.prepBaseCache.Remove(owner)
}

func (s *State) AddPRepStatus(status *PRepStatus) {
	s.prepStatusCache.Add(status)
}

func (s *State) GetPRepStatus(owner module.Address) *PRepStatus {
	return s.prepStatusCache.Get(owner)
}

func (s *State) RemovePRepStatus(owner module.Address) error {
	return s.prepBaseCache.Remove(owner)
}

func (s *State) AddNodeToOwner(node, owner module.Address) error {
	return s.nodeOwnerCache.Add(node, owner)
}

func (s *State) GetOwnerByNode(node module.Address) module.Address {
	return s.nodeOwnerCache.Get(node)
}
