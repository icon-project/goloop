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
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
)

const (
	globalKey     = "global"
	eventsKey     = "events"
	validatorsKey = "validators"
)

var (
	IScoreClaimKey  = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x10})
	EventKey        = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x20})
	BlockProduceKey = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x30})
	ValidatorKey    = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x40})
	BTPKey          = containerdb.ToKey(containerdb.RLPBuilder, []byte{0x50})
	HashKey         = containerdb.ToKey(containerdb.PrefixedHashBuilder, []byte{0x70})
	GlobalKey       = containerdb.ToKey(containerdb.RawBuilder, HashKey.Append(globalKey).Build()).Build()
	EventSizeKey    = containerdb.ToKey(containerdb.RawBuilder, HashKey.Append(eventsKey).Build())
	ValidatorsKey   = containerdb.ToKey(containerdb.RawBuilder, HashKey.Append(validatorsKey).Build())
)

type State struct {
	validatorToIdx map[string]int
	store          *icobject.ObjectStoreState
}

func (s *State) GetSnapshot() *Snapshot {
	return &Snapshot{
		store: icobject.NewObjectStoreSnapshot(s.store.GetSnapshot()),
	}
}

func (s *State) Reset(ss *Snapshot) {
	s.store.Reset(ss.store.ImmutableForObject)
}

func (s *State) GetIScoreClaim(addr module.Address) (*IScoreClaim, error) {
	key := IScoreClaimKey.Append(addr).Build()
	obj, err := s.store.Get(key)
	if err != nil {
		return nil, err
	}
	return ToIScoreClaim(obj), nil
}

func (s *State) AddIScoreClaim(addr module.Address, amount *big.Int) (*IScoreClaim, error) {
	key := IScoreClaimKey.Append(addr).Build()
	obj, err := s.store.Get(key)
	if err != nil {
		return nil, err
	}
	claim := ToIScoreClaim(obj)
	claim = claim.Added(amount)
	_, err = s.store.Set(key, icobject.New(TypeIScoreClaim, claim))
	return claim, err
}

func (s *State) GetEvent(offset int, index int64) (*icobject.Object, error) {
	key := EventKey.Append(offset, index).Build()
	obj, err := s.store.Get(key)
	if err != nil || obj == nil {
		return nil, err
	}
	return obj.(*icobject.Object), nil
}

func (s *State) addEventVote(type_ int, offset int, from module.Address, votes VoteList) (int64, *icobject.Object, error) {
	index := s.getEventSize()
	key := EventKey.Append(offset, index).Build()
	event := NewEventVote(common.AddressToPtr(from), votes)
	obj := icobject.New(type_, event)
	_, err := s.store.Set(key, obj)
	if err != nil {
		return 0, nil, err
	}

	return index, obj, s.setEventSize(index + 1)
}

func (s *State) AddEventDelegation(offset int, from module.Address, votes VoteList) (int64, *icobject.Object, error) {
	return s.addEventVote(TypeEventDelegation, offset, from, votes)
}

func (s *State) AddEventDelegationV2(offset int, from module.Address, delegated VoteList, delegating VoteList) (int64, *icobject.Object, error) {
	index := s.getEventSize()
	key := EventKey.Append(offset, index).Build()
	event := NewEventDelegationV2(common.AddressToPtr(from), delegated, delegating)
	obj := icobject.New(TypeEventDelegationV2, event)
	_, err := s.store.Set(key, obj)
	if err != nil {
		return 0, obj, err
	}

	return index, obj, s.setEventSize(index + 1)
}

func (s *State) AddEventDelegated(offset int, from module.Address, votes VoteList) (int64, *icobject.Object, error) {
	return s.addEventVote(TypeEventDelegated, offset, from, votes)
}

func (s *State) AddEventBond(offset int, from module.Address, votes VoteList) (int64, *icobject.Object, error) {
	return s.addEventVote(TypeEventBond, offset, from, votes)
}

func (s *State) AddEventEnable(offset int, target module.Address, status EnableStatus) (int64, error) {
	index := s.getEventSize()
	key := EventKey.Append(offset, index).Build()
	obj := NewEventEnable(common.AddressToPtr(target), status)
	_, err := s.store.Set(key, icobject.New(TypeEventEnable, obj))
	if err != nil {
		return 0, err
	}

	return index, s.setEventSize(index + 1)
}

func (s *State) AddEventVotedReward(offset int) (int64, error) {
	index := s.getEventSize()
	key := EventKey.Append(offset, index).Build()
	obj := NewEventVotedReward()
	_, err := s.store.Set(key, icobject.New(TypeEventVotedReward, obj))
	if err != nil {
		return 0, err
	}

	return index, s.setEventSize(index + 1)
}

func (s *State) AddEventCommissionRate(offset int, target module.Address, value int) (int64, error) {
	index := s.getEventSize()
	key := EventKey.Append(offset, index).Build()
	obj := NewEventCommissionRate(common.AddressToPtr(target), value)
	_, err := s.store.Set(key, icobject.New(TypeEventCommissionRate, obj))
	if err != nil {
		return 0, err
	}

	return index, s.setEventSize(index + 1)
}

func (s *State) getEventSize() int64 {
	return containerdb.NewVarDB(s.store, EventSizeKey).Int64()
}

func (s *State) setEventSize(size int64) error {
	return containerdb.NewVarDB(s.store, EventSizeKey).Set(size)
}

func (s *State) ResetEventSize() error {
	return s.setEventSize(0)
}

func (s *State) getValidatorIndex(addr module.Address) (int, error) {
	vm := containerdb.NewDictDB(s.store, 1, ValidatorKey)
	if value := vm.Get(addr); value == nil {
		vs := containerdb.NewVarDB(s.store, ValidatorsKey)
		idx := vs.Int64()
		if err := vm.Set(addr, idx); err != nil {
			return 0, err
		}
		if err := vs.Set(idx + 1); err != nil {
			return 0, err
		}
		return int(idx), nil
	} else {
		return int(value.Int64()), nil
	}
}

func (s *State) AddBlockProduce(blockHeight int64, proposer module.Address, voters []module.Address) error {
	global, err := s.GetGlobal()
	if err != nil || global == nil {
		return err
	}
	offset := blockHeight - global.GetStartHeight() - 1
	pIdx, err := s.getValidatorIndex(proposer)
	if err != nil {
		return err
	}
	voteMask := big.NewInt(0)
	for _, v := range voters {
		idx, err := s.getValidatorIndex(v)
		if err != nil {
			return err
		}
		voteMask.SetBit(voteMask, idx, 1)
	}
	log.Tracef("BlockProduce(blockHeight=%d, offset=%d, proposer=%s, voter=%+v)", blockHeight, offset, proposer, voters)
	bp := NewBlockProduce(pIdx, len(voters), voteMask)
	bpv := containerdb.NewVarDB(s.store, BlockProduceKey.Append(offset))
	return bpv.Set(icobject.New(TypeBlockProduce, bp))
}

func (s *State) GetGlobal() (Global, error) {
	key := HashKey.Append(globalKey).Build()
	o, err := s.store.Get(key)
	if err != nil {
		return nil, err
	}
	return ToGlobal(o), nil
}

func (s *State) AddGlobalV1(revision int, startHeight int64, offsetLimit int, irep *big.Int, rrep *big.Int,
	mainPRepCount int, electedPRepCount int,
) error {
	g := NewGlobalV1(
		icstate.IISSVersion2,
		startHeight,
		offsetLimit,
		revision,
		irep,
		rrep,
		mainPRepCount,
		electedPRepCount,
	)
	_, err := s.store.Set(GlobalKey, icobject.New(TypeGlobal, g))
	return err
}

func (s *State) AddGlobalV2(revision int, startHeight int64, offsetLimit int, iglobal *big.Int, iprep *big.Int,
	ivoter *big.Int, icps *big.Int, irelay *big.Int, electedPRepCount int, bondRequirement int,
) error {
	g := NewGlobalV2(
		icstate.IISSVersion3,
		startHeight,
		offsetLimit,
		revision,
		iglobal,
		iprep,
		ivoter,
		icps,
		irelay,
		electedPRepCount,
		bondRequirement,
	)
	_, err := s.store.Set(GlobalKey, icobject.New(TypeGlobal, g))
	return err
}

func (s *State) AddBTPDSA(offset int, index int, value int) error {
	key := BTPKey.Append(offset, index).Build()
	obj := NewBTPDSA(value)
	_, err := s.store.Set(key, icobject.New(TypeBTPDSA, obj))
	return err
}

func (s *State) AddBTPPublicKey(offset int, index int, from module.Address, value int) error {
	key := BTPKey.Append(offset, index).Build()
	obj := NewBTPPublicKey(common.AddressToPtr(from), value)
	_, err := s.store.Set(key, icobject.New(TypeBTPPublicKey, obj))
	return err
}

func (s *State) ClearCache() {
	s.store.ClearCache()
}

func NewStateFromSnapshot(ss *Snapshot) *State {
	t := trie_manager.NewMutableFromImmutableForObject(ss.store.ImmutableForObject)
	s := &State{
		store: icobject.NewObjectStoreState(t),
	}
	return s
}

func NewState(database db.Database) *State {
	database = icobject.AttachObjectFactory(database, NewObjectImpl)
	t := trie_manager.NewMutableForObject(database, nil, icobject.ObjectType)
	return &State{
		store: icobject.NewObjectStoreState(t),
	}
}
