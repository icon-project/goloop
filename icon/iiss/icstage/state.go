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
	"github.com/icon-project/goloop/common/db"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
)

var (
	IScoreClaimKey  = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x10})
	EventKey        = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x20})
	BlockProduceKey = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x30})
	GlobalKey       = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x40})
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
	key := IScoreClaimKey.Append(addr).Build()
	obj, err := s.trie.Get(key)
	if err != nil {
		return err
	}
	claim := ToIScoreClaim(obj)
	claim = claim.Added(amount)
	_, err = s.trie.Set(key, icobject.New(TypeIScoreClaim, claim))
	return err
}

func (s *State) AddEventDelegation(offset int, index int, from module.Address, delegations icstate.Delegations) error {
	key := EventKey.Append(offset, index).Build()
	ed := newEventDelegation(icobject.MakeTag(TypeEventDelegation, 0))
	ed.From = from.(*common.Address)
	ed.Delegations = delegations
	_, err := s.trie.Set(key, icobject.New(TypeEventDelegation, ed))
	return err
}

func (s *State) AddEventEnable(offset int, index int, target module.Address, enable bool) error {
	key := EventKey.Append(offset, index).Build()
	obj := newEventEnable(icobject.MakeTag(TypeEventEnable, 0))
	obj.Target = target.(*common.Address)
	obj.Enable = enable
	_, err := s.trie.Set(key, icobject.New(TypeEventEnable, obj))
	return err
}

func (s *State) AddEventPeriod(offset int, index int, irep *big.Int, rrep *big.Int) error {
	key := EventKey.Append(offset, index).Build()
	obj := newEventPeriod(icobject.MakeTag(TypeEventPeriod, 0))
	obj.Irep = irep
	obj.Rrep = rrep
	_, err := s.trie.Set(key, icobject.New(TypeEventPeriod, obj))
	return err
}

func (s *State) AddEventValidator(offset int, index int, validators []*common.Address) error {
	key := EventKey.Append(offset, index).Build()
	obj := newEventValidator(icobject.MakeTag(TypeEventValidator, 0))
	obj.validators = validators
	_, err := s.trie.Set(key, icobject.New(TypeEventValidator, obj))
	return err
}

func (s *State) AddBlockProduce(offset int, proposerIndex int, voteCount int, voteMask int64) error {
	key := BlockProduceKey.Append(offset).Build()
	obj := newBlockProduce(icobject.MakeTag(TypeBlockProduce, 0))
	obj.ProposerIndex = proposerIndex
	obj.VoteCount = voteCount
	obj.VoteMask = voteMask
	_, err := s.trie.Set(key, icobject.New(TypeBlockProduce, obj))
	return err
}

func (s *State) AddGlobal(blockHeight int64, offsetLimit int) error {
	key := GlobalKey.Build()
	obj := newGlobal(icobject.MakeTag(TypeGlobal, 0))
	obj.StartBlockHeight = blockHeight
	obj.OffsetLimit = offsetLimit
	_, err := s.trie.Set(key, icobject.New(TypeGlobal, obj))
	return err
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
