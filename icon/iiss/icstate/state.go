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
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

type State struct {
	trie trie.MutableForObject
}

func (s *State) Reset(ss *Snapshot) {
	s.trie.Reset(ss.trie)
}

func (s *State) GetSnapshot() *Snapshot {
	return &Snapshot{
		trie: s.trie.GetSnapshot(),
	}
}

func (s *State) GetAccountState(addr module.Address) (*AccountState, error) {
	obj, err := s.trie.Get(crypto.SHA3Sum256(scoredb.AppendKeys(accountPrefix, addr)))
	if err != nil {
		return nil, err
	}
	as := new(AccountState)
	if obj != nil {
		as.Reset(obj.(*Object).Real().(*AccountSnapshot))
	} else {
		as.Reset(newAccountSnapshot(MakeTag(TypeAccount, accountVersion)))
	}
	return as, nil
}


func (s *State) SetAccountState(addr module.Address, as *AccountState) error {
	return s.trie.Set(
		crypto.SHA3Sum256(scoredb.AppendKeys(accountPrefix, addr)),
		NewObject(TypeAccount, as.GetSnapshot()),
	)
}

func NewStateFromSnapshot(ss *Snapshot) *State {
	return &State{
		trie: trie_manager.NewMutableFromImmutableForObject(ss.trie),
	}
}
