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

package icstage

import (
	"math/big"

	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

var (
	iScoreClaimPrefix = []byte{0x10}
)

type State struct {
	trie trie.MutableForObject
}

func (s *State) GetSnapshot() *Snapshot {
	return &Snapshot{
		trie: s.trie.GetSnapshot(),
	}
}

func (s *State) Reset(ss *Snapshot) {
	s.trie.Reset(ss.trie)
}

func (s *State) AddIScoreClaim(addr module.Address, amount *big.Int) error {
	key := scoredb.AppendKeys(iScoreClaimPrefix, addr)
	obj, err := s.trie.Get(key)
	if err != nil {
		return err
	}
	claim := ToIScoreClaim(obj)
	claim = claim.Added(amount)
	return s.trie.Set(key, icobject.New(TypeIScoreClaim, claim))
}

func NewStateFromSnapshot(ss *Snapshot) *State {
	return &State{
		trie: trie_manager.NewMutableFromImmutableForObject(ss.trie),
	}
}
