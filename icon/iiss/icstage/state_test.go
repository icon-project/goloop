/*
 * Copyright 2020 ICON Foundation
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package icstage

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
)

func TestState_AddIScoreClaim(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	s := NewStateFromSnapshot(NewSnapshot(database, nil))

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	v1 := int64(100)
	v2 := int64(200)

	type args struct {
		addr  module.Address
		value *big.Int
	}

	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			"Add Claim 100",
			args{
				addr1,
				big.NewInt(v1),
			},
			v1,
		},
		{
			"Add Claim 200",
			args{
				addr1,
				big.NewInt(v2),
			},
			v1 + v2,
		},
		{
			"Add Claim 200 to new address",
			args{
				addr2,
				big.NewInt(v2),
			},
			v2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			err := s.AddIScoreClaim(args.addr, args.value)
			assert.NoError(t, err)

			key := IScoreClaimKey.Append(args.addr).Build()
			obj, err := icobject.GetFromMutableForObject(s.trie, key)
			assert.NoError(t, err)
			claim := ToIScoreClaim(obj)
			assert.Equal(t, tt.want, claim.Value.Int64())
		})
	}

	ss := s.GetSnapshot()
	count := 0
	for iter := ss.Filter(IScoreClaimKey.Build()); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		assert.NoError(t, err)
		assert.NotNil(t, o)
		claim := ToIScoreClaim(o)
		assert.NotNil(t, claim)

		keySplit, _ := containerdb.SplitKeys(key)
		assert.Equal(t, IScoreClaimKey.Build(), keySplit[0])
		keyAddress, err := common.NewAddress(keySplit[1])
		assert.NoError(t, err)
		addr := addr1
		if count == 1 {
			addr = addr2
		}
		assert.True(t, addr.Equal(keyAddress))

		count += 1
	}
	assert.Equal(t, 2, count)
}

func TestState_AddEvent(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	s := NewStateFromSnapshot(NewSnapshot(database, nil))

	offset1 := 100
	offset2 := 200
	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	v1 := int64(100)
	v2 := int64(200)
	d1 := icstate.Delegation{
		Address: addr1,
		Value:   common.NewHexInt(v1),
	}
	d2 := icstate.Delegation{
		Address: addr2,
		Value:   common.NewHexInt(v2),
	}

	type args struct {
		type_       int
		offset      int
		index       int
		address     *common.Address
		delegations icstate.Delegations
		enable      bool
		irep        *big.Int
		rrep        *big.Int
		validators  []*common.Address
	}

	tests := []struct {
		name string
		args args
	}{
		{
			"Delegation",
			args{
				type_:       TypeEventDelegation,
				offset:      offset1,
				index:       0,
				address:     addr1,
				delegations: icstate.Delegations{&d1, &d2},
			},
		},
		{
			"Enable",
			args{
				type_:   TypeEventEnable,
				offset:  offset1,
				index:   1,
				address: addr2,
				enable:  false,
			},
		},
		{
			"Period",
			args{
				type_:   TypeEventPeriod,
				offset:  offset2,
				index:   0,
				address: addr1,
				irep:    big.NewInt(v1),
				rrep:    big.NewInt(v2),
			},
		},
	}
	for _, tt := range tests {
		args := tt.args
		t.Run(tt.name, func(t *testing.T) {
			switch args.type_ {
			case TypeEventDelegation:
				checkAddEventDelegation(t, s, args.offset, args.index, args.address, args.delegations)
			case TypeEventEnable:
				checkAddEventEnable(t, s, args.offset, args.index, args.address, args.enable)
			case TypeEventPeriod:
				checkAddEventPeriod(t, s, args.offset, args.index, args.irep, args.rrep)
			}
		})
	}

	// check Filter
	ss := s.GetSnapshot()
	count := 0
	for iter := ss.Filter(EventKey.Build()); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		assert.NoError(t, err)
		assert.NotNil(t, o)

		keySplit, _ := containerdb.SplitKeys(key)
		assert.Equal(t, EventKey.Build(), keySplit[0])
		if len(keySplit) == 1 {
			// size value
			continue
		}
		assert.Equal(t, tests[count].args.offset, int(intconv.BytesToInt64(keySplit[1])))
		assert.Equal(t, tests[count].args.index, int(intconv.BytesToInt64(keySplit[2])))

		count += 1
	}
	assert.Equal(t, len(tests), count)
}

func checkAddEventDelegation(t *testing.T, s *State, offset int, index int, address *common.Address, delegations icstate.Delegations) {
	err := s.AddEventDelegation(offset, index, address, delegations)
	assert.NoError(t, err)

	key := EventKey.Append(offset, index).Build()
	obj, err := icobject.GetFromMutableForObject(s.trie, key)
	assert.NoError(t, err)
	event := ToEventDelegation(obj)
	assert.True(t, address.Equal(event.From))
	assert.True(t, delegations.Equal(event.Delegations))
}

func checkAddEventEnable(t *testing.T, s *State, offset int, index int, address *common.Address, enable bool) {
	err := s.AddEventEnable(offset, index, address, enable)
	assert.NoError(t, err)

	key := EventKey.Append(offset, index).Build()
	obj, err := icobject.GetFromMutableForObject(s.trie, key)
	assert.NoError(t, err)
	event := ToEventEnable(obj)
	assert.True(t, address.Equal(event.Target))
	assert.Equal(t, enable, event.Enable)
}

func checkAddEventPeriod(t *testing.T, s *State, offset int, index int, irep *big.Int, rrep *big.Int) {
	err := s.AddEventPeriod(offset, index, irep, rrep)
	assert.NoError(t, err)

	key := EventKey.Append(offset, index).Build()
	obj, err := icobject.GetFromMutableForObject(s.trie, key)
	assert.NoError(t, err)
	event := ToEventPeriod(obj)
	assert.Equal(t, 0, irep.Cmp(event.Irep))
	assert.Equal(t, 0, rrep.Cmp(event.Rrep))
}

func TestState_AddBlockProduce(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	s := NewStateFromSnapshot(NewSnapshot(database, nil))

	offset1 := 0
	offset2 := 1

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")
	addr4 := common.NewAddressFromString("hx4")
	addr5 := common.NewAddressFromString("hx5")

	type args struct {
		type_        int
		offset       int
		proposeIndex int
		voteCount    int
		voteMask     int64
		validators   []*common.Address
	}

	tests := []struct {
		name string
		args args
	}{
		{
			"Validator 1",
			args{
				type_:      TypeValidator,
				offset:     offset1,
				validators: []*common.Address{addr1, addr2, addr3, addr4},
			},
		},
		{
			"block produce 1",
			args{
				type_:        TypeBlockProduce,
				offset:       offset1,
				proposeIndex: 1,
				voteCount:    4,
				voteMask:     0b1111,
			},
		},
		{
			"Validator 2",
			args{
				type_:      TypeValidator,
				offset:     offset2,
				validators: []*common.Address{addr1, addr2, addr3, addr5},
			},
		},
		{
			"block produce 2",
			args{
				type_:        TypeBlockProduce,
				offset:       offset2,
				proposeIndex: 3,
				voteCount:    3,
				voteMask:     0b1110,
			},
		},
	}
	for _, tt := range tests {
		args := tt.args
		t.Run(tt.name, func(t *testing.T) {
			switch args.type_ {
			case TypeBlockProduce:
				err := s.AddBlockVotes(args.offset, args.proposeIndex, args.voteCount, args.voteMask)
				assert.NoError(t, err)

				key := BlockProduceKey.Append(args.offset, suffixBlockVotes).Build()
				obj, err := icobject.GetFromMutableForObject(s.trie, key)
				assert.NoError(t, err)
				assert.NotNil(t, obj)

				o := ToBlockVotes(obj)
				assert.Equal(t, args.proposeIndex, o.ProposerIndex)
				assert.Equal(t, args.voteCount, o.VoteCount)
				assert.Equal(t, args.voteMask, o.VoteMask)
			case TypeValidator:
				err := s.AddValidators(args.offset, args.validators)
				assert.NoError(t, err)

				key := BlockProduceKey.Append(args.offset, suffixValidators).Build()
				obj, err := icobject.GetFromMutableForObject(s.trie, key)
				assert.NoError(t, err)
				assert.NotNil(t, obj)
				o := ToValidators(obj)
				assert.Equal(t, len(args.validators), len(o.Addresses))
				for i, v := range args.validators {
					assert.True(t, v.Equal(o.Addresses[i]))
				}
			}
		})
	}

	ss := s.GetSnapshot()
	count := 0
	for iter := ss.Filter(BlockProduceKey.Build()); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		assert.NoError(t, err)

		keySplit, _ := containerdb.SplitKeys(key)
		assert.Equal(t, BlockProduceKey.Build(), keySplit[0])
		assert.Equal(t, tests[count].args.offset, int(intconv.BytesToInt64(keySplit[1])))

		suffix := keySplit[2][0]
		switch suffix {
		case suffixBlockVotes:
			blockProduce := ToBlockVotes(o)
			assert.NotNil(t, blockProduce)
		case suffixValidators:
			validators := ToValidators(o)
			assert.NotNil(t, validators)
		}

		count += 1
	}
	assert.Equal(t, len(tests), count)
}
