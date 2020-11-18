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
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

type State struct {
	mutableAccounts map[string]*AccountState
	trie            trie.MutableForObject
}

func (s *State) Reset(ss *Snapshot) error {
	s.trie.Reset(ss.trie)
	for _, as := range s.mutableAccounts {
		key := crypto.SHA3Sum256(scoredb.AppendKeys(accountPrefix, as.GetAddress()))
		value, err := s.trie.Get(key)
		if err != nil {
			return err
		}
		if value == nil {
			as.Clear()
		} else {
			as.Reset(value.(*Object).Account())
		}
	}
	return nil
}

func (s *State) GetSnapshot() *Snapshot {
	for _, as := range s.mutableAccounts {
		key := crypto.SHA3Sum256(scoredb.AppendKeys(accountPrefix, as.GetAddress()))
		value := NewObject(TypeAccount, as.GetSnapshot())

		if as.IsEmpty() {
			if err := s.trie.Delete(key); err != nil {
				log.Errorf("Failed to delete account key %x, err+%+v", key, err)
			}
		} else {
			if err := s.trie.Set(key, value); err != nil {
				log.Errorf("Failed to set snapshot for %x, err+%+v", key, err)
			}
		}
	}
	return &Snapshot{
		trie: s.trie.GetSnapshot(),
	}
}

func (s *State) GetAccountState(addr module.Address) (*AccountState, error) {
	ids := addr.String()
	if a, ok := s.mutableAccounts[ids]; ok {
		return a, nil
	}
	obj, err := s.trie.Get(crypto.SHA3Sum256(scoredb.AppendKeys(accountPrefix, addr)))
	if err != nil {
		return nil, err
	}
	var ass *AccountSnapshot
	if obj != nil {
		ass = obj.(*Object).Account()
	} else {
		ass = newAccountSnapshot(MakeTag(TypeAccount, accountVersion))
	}
	as := NewAccountStateWithSnapshot(addr, ass)
	s.mutableAccounts[ids] = as
	return as, nil
}

func NewStateFromSnapshot(ss *Snapshot) *State {
	return &State{
		mutableAccounts: make(map[string]*AccountState),
		trie: trie_manager.NewMutableFromImmutableForObject(ss.trie),
	}
}
