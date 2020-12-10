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

package icreward

import (
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

var (
	DelegatedKey  = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x10})
	DelegatingKey = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x20})
	IScoreKey     = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x30})
	GlobalKey     = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x40})
	ValidatorsKey = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x50})
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

func (s *State) GetGlobal() (*Global, error) {
	key := GlobalKey.Build()
	obj, err := icobject.GetFromMutableForObject(s.trie, key)
	if err != nil {
		return nil, err
	}
	return ToGlobal(obj), nil
}

func (s *State) SetGlobal(global *Global) error {
	key := GlobalKey.Build()
	_, err := s.trie.Set(key, icobject.New(TypeGlobal, global))
	return err
}

func (s *State) GetIScore(addr module.Address) (*IScore, error) {
	key := IScoreKey.Append(addr).Build()
	obj, err := icobject.GetFromMutableForObject(s.trie, key)
	if err != nil {
		return nil, err
	}
	return ToIScore(obj), nil
}

func (s *State) SetIScore(addr module.Address, iScore *IScore) error {
	key := IScoreKey.Append(addr).Build()
	if iScore.IsEmpty() {
		_, err := s.trie.Delete(key)
		return err
	} else {
		_, err := s.trie.Set(key, icobject.New(TypeIScore, iScore))
		return err
	}
}

func (s *State) DeleteIScore(addr module.Address) error {
	key := IScoreKey.Append(addr).Build()
	_, err := s.trie.Delete(key)
	return err
}

func (s *State) SetDelegated(addr module.Address, delegated *Delegated) error {
	key := DelegatedKey.Append(addr).Build()
	_, err := s.trie.Set(key, icobject.New(TypeDelegated, delegated))
	return err
}

func (s *State) DeleteDelegated(addr module.Address) error {
	key := DelegatedKey.Append(addr).Build()
	_, err := s.trie.Delete(key)
	return err
}

func (s *State) GetDelegating(addr module.Address) (*Delegating, error) {
	key := DelegatingKey.Append(addr).Build()
	obj, err := icobject.GetFromMutableForObject(s.trie, key)
	if err != nil {
		return nil, err
	}
	return ToDelegating(obj), nil
}

func (s *State) SetDelegating(addr module.Address, delegating *Delegating) error {
	key := DelegatingKey.Append(addr).Build()
	_, err := s.trie.Set(key, icobject.New(TypeDelegating, delegating))
	return err
}

func (s *State) DeleteDelegating(addr module.Address) error {
	key := DelegatingKey.Append(addr).Build()
	_, err := s.trie.Delete(key)
	return err
}

func (s *State) SetValidators(vs *Validators) error {
	key := ValidatorsKey.Build()
	_, err := s.trie.Set(key, icobject.New(TypeValidator, vs))
	return err
}

func (s *State) GetValidators() (*Validators, error) {
	key := ValidatorsKey.Build()
	obj, err := icobject.GetFromMutableForObject(s.trie, key)
	if err != nil {
		return nil, err
	}
	return ToValidators(obj), nil
}

func NewStateFromSnapshot(ss *Snapshot) *State {
	return &State{
		trie: trie_manager.NewMutableFromImmutableForObject(ss.trie),
	}
}
func NewState(database db.Database, hash []byte) *State {
	database = icobject.AttachObjectFactory(database, newObjectImpl)
	return &State{
		trie: trie_manager.NewMutableForObject(database, hash, icobject.ObjectType),
	}
}
