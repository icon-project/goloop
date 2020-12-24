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
			obj, err := icobject.GetFromMutableForObject(s.store, key)
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
				address:     addr1,
				delegations: icstate.Delegations{&d1, &d2},
			},
		},
		{
			"Enable",
			args{
				type_:   TypeEventEnable,
				offset:  offset1,
				address: addr2,
				enable:  false,
			},
		},
		{
			"Period",
			args{
				type_:   TypeEventPeriod,
				offset:  offset2,
				address: addr1,
				irep:    big.NewInt(v1),
				rrep:    big.NewInt(v2),
			},
		},
	}
	for _, tt := range tests {
		a := tt.args
		t.Run(tt.name, func(t *testing.T) {
			switch a.type_ {
			case TypeEventDelegation:
				checkAddEventDelegation(t, s, a.offset, a.address, a.delegations)
			case TypeEventEnable:
				checkAddEventEnable(t, s, a.offset, a.address, a.enable)
			case TypeEventPeriod:
				checkAddEventPeriod(t, s, a.offset, a.irep, a.rrep)
			}
		})
	}

	// check event size
	es, err := s.getEventSize()
	assert.NoError(t, err)
	assert.Equal(t, int64(len(tests)), es.Value.Int64())

	// check Filter
	ss := s.GetSnapshot()
	count := 0
	for iter := ss.Filter(EventKey.Build()); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		assert.NoError(t, err)
		assert.NotNil(t, o)

		keySplit, _ := containerdb.SplitKeys(key)
		assert.Equal(t, EventKey.Build(), keySplit[0])
		assert.Equal(t, tests[count].args.offset, int(intconv.BytesToInt64(keySplit[1])))

		count += 1
	}
	size, err := ss.GetEventSize()
	assert.NoError(t, err)
	assert.Equal(t, int64(len(tests)), size.Value.Int64())
}

func checkAddEventDelegation(t *testing.T, s *State, offset int, address *common.Address, delegations icstate.Delegations) {
	index, err := s.AddEventDelegation(offset, address, delegations)
	assert.NoError(t, err)

	key := EventKey.Append(offset, index).Build()
	obj, err := icobject.GetFromMutableForObject(s.store, key)
	assert.NoError(t, err)
	event := ToEventDelegation(obj)
	assert.True(t, address.Equal(event.From))
	assert.True(t, delegations.Equal(event.Delegations))
}

func checkAddEventEnable(t *testing.T, s *State, offset int, address *common.Address, enable bool) {
	index, err := s.AddEventEnable(offset, address, enable)
	assert.NoError(t, err)

	key := EventKey.Append(offset, index).Build()
	obj, err := icobject.GetFromMutableForObject(s.store, key)
	assert.NoError(t, err)
	event := ToEventEnable(obj)
	assert.True(t, address.Equal(event.Target))
	assert.Equal(t, enable, event.Enable)
}

func checkAddEventPeriod(t *testing.T, s *State, offset int, irep *big.Int, rrep *big.Int) {
	index, err := s.AddEventPeriod(offset, irep, rrep)
	assert.NoError(t, err)

	key := EventKey.Append(offset, index).Build()
	obj, err := icobject.GetFromMutableForObject(s.store, key)
	assert.NoError(t, err)
	event := ToEventPeriod(obj)
	assert.Equal(t, 0, irep.Cmp(event.Irep))
	assert.Equal(t, 0, rrep.Cmp(event.Rrep))
}

func TestState_AddBlockProduce(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	s := NewStateFromSnapshot(NewSnapshot(database, nil))

	offset1 := 0

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")
	addr4 := common.NewAddressFromString("hx4")
	addr5 := common.NewAddressFromString("hx5")

	addrs := []*common.Address{addr1, addr2, addr3, addr4, addr5}

	type args struct {
		offset   int
		proposer module.Address
		voters   []module.Address
	}

	type wants struct {
		proposerIndex int
		voteCount     int
		voteMask      *big.Int
	}

	tests := []struct {
		name  string
		args  args
		wants wants
	}{
		{
			"block produce 1",
			args{
				offset:   offset1,
				proposer: addr1,
				voters:   []module.Address{addr1, addr2, addr3, addr4},
			},
			wants{
				proposerIndex: 0,
				voteCount:     4,
				voteMask:      big.NewInt(int64(0b1111)),
			},
		},
		{
			"block produce 2",
			args{
				offset:   offset1 + 1,
				proposer: addr2,
				voters:   []module.Address{addr1, addr2, addr3, addr4},
			},
			wants{
				proposerIndex: 1,
				voteCount:     4,
				voteMask:      big.NewInt(int64(0b1111)),
			},
		},
		{
			"block produce 3",
			args{
				offset:   offset1 + 2,
				proposer: addr5,
				voters:   []module.Address{addr1, addr4, addr5},
			},
			wants{
				proposerIndex: 4,
				voteCount:     3,
				voteMask:      big.NewInt(int64(0b11001)),
			},
		},
	}
	for _, tt := range tests {
		a := tt.args
		w := tt.wants
		t.Run(tt.name, func(t *testing.T) {
			err := s.AddBlockProduce(a.offset, a.proposer, a.voters)
			assert.NoError(t, err)

			key := BlockProduceKey.Append(a.offset).Build()
			obj, err := icobject.GetFromMutableForObject(s.store, key)
			assert.NoError(t, err)
			assert.NotNil(t, obj)

			o := ToBlockProduce(obj)
			assert.Equal(t, w.proposerIndex, o.ProposerIndex)
			assert.Equal(t, w.voteCount, o.VoteCount)
			assert.Equal(t, 0, w.voteMask.Cmp(o.VoteMask))
		})
	}

	ss := s.GetSnapshot()
	count := 0
	for iter := ss.Filter(ValidatorKey.Build()); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		assert.NoError(t, err)
		v := ToValidator(o)

		keySplit, _ := containerdb.SplitKeys(key)
		assert.Equal(t, ValidatorKey.Build(), keySplit[0])
		assert.Equal(t, count, int(intconv.BytesToInt64(keySplit[1])))
		assert.True(t, addrs[count].Equal(v.Address))

		count += 1
	}
}

func TestState_AddGlobal(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	s := NewStateFromSnapshot(NewSnapshot(database, nil))

	offsetLimit := 1000

	type args struct {
		offsetLimit int
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"Set offsetLimit",
			args{
				offsetLimit,
			},
			offsetLimit,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.args
			err := s.AddGlobal(a.offsetLimit)
			assert.NoError(t, err)

			key := HashKey.Append(globalKey).Build()
			obj, err := icobject.GetFromMutableForObject(s.store, key)
			assert.NoError(t, err)
			global := ToGlobal(obj)
			assert.Equal(t, tt.want, global.OffsetLimit)
		})
	}
}

func TestState_AddLoadValidators(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	s := NewStateFromSnapshot(NewSnapshot(database, nil))

	datas := []struct {
		offset int
		addr   *common.Address
	}{
		{
			0,
			common.NewAddressFromString("hx1"),
		},
		{
			2,
			common.NewAddressFromString("hx2"),
		},
		{
			3,
			common.NewAddressFromString("hx3"),
		},
		{
			5,
			common.NewAddressFromString("hx5"),
		},
	}
	for _, data := range datas {
		err := s.addValidator(data.offset, data.addr)
		assert.NoError(t, err)

		key := ValidatorKey.Append(data.offset).Build()
		obj, err := icobject.GetFromMutableForObject(s.store, key)
		assert.NoError(t, err)
		validator := ToValidator(obj)
		assert.True(t, data.addr.Equal(validator.Address))
	}

	ss := s.GetSnapshot()
	err := s.loadValidators(ss)
	assert.NoError(t, err)

	for _, data := range datas {
		offset, ok := s.validatorToIdx[string(data.addr.Bytes())]
		assert.True(t, ok)
		assert.Equal(t, data.offset, offset)
	}
}
