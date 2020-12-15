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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
)

const (
	globalKey = "global"
	eventsKey = "events"
)

var (
	IScoreClaimKey  = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x10})
	EventKey        = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x20})
	BlockProduceKey = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x30})
	ValidatorKey    = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x40})
	HashKey         = containerdb.ToKey(containerdb.PrefixedHashBuilder, []byte{0x70})
)

type State struct {
	validatorToIdx map[string]int
	trie           trie.MutableForObject
}

func (s *State) GetSnapshot() *Snapshot {
	return &Snapshot{
		trie: s.trie.GetSnapshot(),
	}
}

func (s *State) Reset(ss *Snapshot) {
	s.trie.Reset(ss.trie)
}

func (s *State) GetIScoreClaim(addr module.Address) (*IScoreClaim, error) {
	key := IScoreClaimKey.Append(addr).Build()
	obj, err := s.trie.Get(key)
	if err != nil {
		return nil, err
	}
	return ToIScoreClaim(obj), nil
}
func (s *State) AddIScoreClaim(addr module.Address, amount *big.Int) error {
	key := IScoreClaimKey.Append(addr).Build()
	obj, err := icobject.GetFromMutableForObject(s.trie, key)
	if err != nil {
		return err
	}
	claim := ToIScoreClaim(obj)
	claim = claim.Added(amount)
	_, err = s.trie.Set(key, icobject.New(TypeIScoreClaim, claim))
	return err
}

func (s *State) AddEventDelegation(offset int, from module.Address, delegations icstate.Delegations) (int64, error) {
	size, err := s.getEventSize()
	if err != nil {
		return 0, err
	}

	key := EventKey.Append(offset, size.Value).Build()
	ed := newEventDelegation(icobject.MakeTag(TypeEventDelegation, 0))
	ed.From = from.(*common.Address)
	ed.Delegations = delegations
	_, err = s.trie.Set(key, icobject.New(TypeEventDelegation, ed))
	if err != nil {
		return 0, err
	}

	index := size.Value.Int64()
	size.Value.Add(size.Value, intconv.BigIntOne)
	return index, s.setEventSize(size)
}

func (s *State) AddEventEnable(offset int, target module.Address, enable bool) (int64, error) {
	size, err := s.getEventSize()
	if err != nil {
		return 0, err
	}

	key := EventKey.Append(offset, size.Value).Build()
	obj := newEventEnable(icobject.MakeTag(TypeEventEnable, 0))
	obj.Target = target.(*common.Address)
	obj.Enable = enable
	_, err = s.trie.Set(key, icobject.New(TypeEventEnable, obj))
	if err != nil {
		return 0, err
	}

	index := size.Value.Int64()
	size.Value.Add(size.Value, intconv.BigIntOne)
	return index, s.setEventSize(size)
}

func (s *State) AddEventPeriod(offset int, irep *big.Int, rrep *big.Int) (int64, error) {
	size, err := s.getEventSize()
	if err != nil {
		return 0, err
	}

	key := EventKey.Append(offset, size.Value).Build()
	obj := newEventPeriod(icobject.MakeTag(TypeEventPeriod, 0))
	obj.Irep = irep
	obj.Rrep = rrep
	_, err = s.trie.Set(key, icobject.New(TypeEventPeriod, obj))
	if err != nil {
		return 0, err
	}

	index := size.Value.Int64()
	size.Value.Add(size.Value, intconv.BigIntOne)
	return index, s.setEventSize(size)
}

func (s *State) getEventSize() (*EventSize, error) {
	key := HashKey.Append(eventsKey).Build()
	o, err := icobject.GetFromMutableForObject(s.trie, key)
	if err != nil {
		return nil, err
	}

	size := ToEventSize(o)
	if size == nil {
		size = newEventSize(icobject.MakeTag(TypeEventSize, 0))
	}
	return size, nil
}

func (s *State) setEventSize(size *EventSize) error {
	key := HashKey.Append(eventsKey).Build()
	_, err := s.trie.Set(key, icobject.New(TypeEventSize, size))
	return err
}

func (s *State) AddBlockProduce(offset int, proposer module.Address, voters []module.Address) error {
	pKey := string(proposer.Bytes())
	pIdx, ok := s.validatorToIdx[pKey]
	if !ok {
		pIdx = len(s.validatorToIdx)
		s.validatorToIdx[pKey] = pIdx
		if err := s.addValidator(pIdx, proposer); err != nil {
			return err
		}
	}
	key := BlockProduceKey.Append(offset).Build()
	obj := newBlockProduce(icobject.MakeTag(TypeBlockProduce, 0))
	obj.ProposerIndex = pIdx
	obj.VoteCount = len(voters)
	voteMask := big.NewInt(0)
	for _, v := range voters {
		vKey := string(v.Bytes())
		idx, ok := s.validatorToIdx[vKey]
		if !ok {
			idx = len(s.validatorToIdx)
			s.validatorToIdx[vKey] = idx
			if err := s.addValidator(idx, v); err != nil {
				return err
			}
		}
		voteMask.SetBit(voteMask, idx, 1)
	}
	obj.VoteMask = voteMask
	_, err := s.trie.Set(key, icobject.New(TypeBlockProduce, obj))
	return err
}

func (s *State) addValidator(offset int, validator module.Address) error {
	key := ValidatorKey.Append(offset).Build()
	obj := newValidator(icobject.MakeTag(TypeValidator, 0))
	obj.Address = validator.(*common.Address)
	_, err := s.trie.Set(key, icobject.New(TypeValidator, obj))
	return err
}

func (s *State) AddGlobal(offsetLimit int) error {
	key := HashKey.Append(globalKey).Build()
	obj := newGlobal(icobject.MakeTag(TypeGlobal, 0))
	obj.OffsetLimit = offsetLimit
	_, err := s.trie.Set(key, icobject.New(TypeGlobal, obj))
	return err
}

func (s *State) loadValidators(ss *Snapshot) error {
	nvs := make(map[string]int)
	prefix := ValidatorKey.Build()
	for iter := ss.Filter(prefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return err
		}
		keySplit, err := containerdb.SplitKeys(key)
		if err != nil {
			return err
		}
		idx := int(intconv.BytesToInt64(keySplit[1]))
		v := ToValidator(o)
		nvs[string(v.Address.Bytes())] = idx
	}
	s.validatorToIdx = nvs
	return nil
}

func NewStateFromSnapshot(ss *Snapshot) *State {
	s := &State{
		trie: trie_manager.NewMutableFromImmutableForObject(ss.trie),
	}
	s.loadValidators(ss)
	return s
}

func NewState(database db.Database) *State {
	database = icobject.AttachObjectFactory(database, newObjectImpl)
	return &State{
		trie:           trie_manager.NewMutableForObject(database, nil, icobject.ObjectType),
		validatorToIdx: make(map[string]int),
	}
}
