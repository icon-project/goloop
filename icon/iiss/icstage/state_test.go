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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/iiss/icobject"
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
	vote1 := Vote{
		Address: addr1,
		Value:   big.NewInt(v1),
	}
	vote2 := Vote{
		Address: addr2,
		Value:   big.NewInt(v2),
	}

	type args struct {
		type_         int
		offset        int
		address       *common.Address
		votes         VoteList
		enable        bool
		irep          *big.Int
		rrep          *big.Int
		mainPRepCount int64
		pRepCount     int64
		validators    []*common.Address
	}

	tests := []struct {
		name string
		args args
	}{
		{
			"Delegation",
			args{
				type_:   TypeEventDelegation,
				offset:  offset1,
				address: addr1,
				votes:   VoteList{&vote1, &vote2},
			},
		},
		{
			"Bond",
			args{
				type_:   TypeEventBond,
				offset:  offset1,
				address: addr1,
				votes:   VoteList{&vote1, &vote2},
			},
		},
		{
			"Enable",
			args{
				type_:   TypeEventEnable,
				offset:  offset2,
				address: addr2,
				enable:  false,
			},
		},
	}
	for _, tt := range tests {
		a := tt.args
		t.Run(tt.name, func(t *testing.T) {
			switch a.type_ {
			case TypeEventDelegation:
				checkAddEventDelegation(t, s, a.offset, a.address, a.votes)
			case TypeEventBond:
				checkAddEventBond(t, s, a.offset, a.address, a.votes)
			case TypeEventEnable:
				checkAddEventEnable(t, s, a.offset, a.address, a.enable)
			}
		})
	}

	// check event size
	es, err := s.GetEventSize()
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

func checkAddEventDelegation(t *testing.T, s *State, offset int, address *common.Address, votes VoteList) {
	index, err := s.AddEventDelegation(offset, address, votes)
	assert.NoError(t, err)

	key := EventKey.Append(offset, index).Build()
	obj, err := icobject.GetFromMutableForObject(s.store, key)
	assert.NoError(t, err)
	event := ToEventVote(obj)
	assert.True(t, address.Equal(event.From))
	assert.True(t, votes.Equal(event.Votes))
}

func checkAddEventBond(t *testing.T, s *State, offset int, address *common.Address, votes VoteList) {
	index, err := s.AddEventBond(offset, address, votes)
	assert.NoError(t, err)

	key := EventKey.Append(offset, index).Build()
	obj, err := icobject.GetFromMutableForObject(s.store, key)
	assert.NoError(t, err)
	event := ToEventVote(obj)
	assert.True(t, address.Equal(event.From))
	assert.True(t, votes.Equal(event.Votes))
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
			"genesis block produce",
			args{
				offset:   offset1,
				proposer: addr1,
				voters:   []module.Address{},
			},
			wants{
				proposerIndex: 0,
				voteCount:     0,
				voteMask:      big.NewInt(int64(0b0000)),
			},
		},
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

	type args struct {
		version          int
		startHeight      int64
		offsetLimit      int
		irep             *big.Int
		rrep             *big.Int
		mainPRepCount    int
		electedPRepCount int
		period           int
		iglobal          *big.Int
		iprep            *big.Int
		ivoter           *big.Int
		bondRequirement  int
	}

	tests := []struct {
		name string
		args args
	}{
		{
			"Version 1",
			args{
				version:          GlobalVersion1,
				startHeight:      0,
				offsetLimit:      1000,
				irep:             big.NewInt(100),
				rrep:             big.NewInt(200),
				mainPRepCount:    22,
				electedPRepCount: 100,
			},
		},
		{
			"Version 2",
			args{
				version:          GlobalVersion2,
				startHeight:      0,
				offsetLimit:      1000,
				iglobal:          big.NewInt(100),
				iprep:            big.NewInt(50),
				ivoter:           big.NewInt(50),
				electedPRepCount: 100,
				bondRequirement:  5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			a := tt.args
			switch a.version {
			case GlobalVersion1:
				err = s.AddGlobalV1(
					a.startHeight,
					a.offsetLimit,
					a.irep,
					a.rrep,
					a.mainPRepCount,
					a.electedPRepCount,
				)
			case GlobalVersion2:
				err = s.AddGlobalV2(
					a.startHeight,
					a.offsetLimit,
					a.iglobal,
					a.iprep,
					a.ivoter,
					a.electedPRepCount,
					a.bondRequirement,
				)
			}
			assert.NoError(t, err)

			key := HashKey.Append(globalKey).Build()
			obj, err := s.store.Get(key)
			assert.NoError(t, err)
			g := ToGlobal(obj)
			assert.Equal(t, a.version, g.Version())

			switch a.version {
			case GlobalVersion1:
				global := g.GetV1()
				assert.NotNil(t, global)
				assert.Equal(t, a.version, global.Version())
				assert.Equal(t, a.offsetLimit, global.OffsetLimit)
				assert.Equal(t, 0, a.irep.Cmp(global.Irep))
				assert.Equal(t, 0, a.rrep.Cmp(global.Rrep))
				assert.Equal(t, a.mainPRepCount, global.MainPRepCount)
				assert.Equal(t, a.electedPRepCount, global.ElectedPRepCount)
			case GlobalVersion2:
				global := g.GetV2()
				assert.NotNil(t, global)
				assert.Equal(t, a.version, global.Version())
				assert.Equal(t, a.offsetLimit, global.OffsetLimit)
				assert.Equal(t, 0, a.iglobal.Cmp(global.Iglobal))
				assert.Equal(t, 0, a.iprep.Cmp(global.Iprep))
				assert.Equal(t, 0, a.ivoter.Cmp(global.Ivoter))
				assert.Equal(t, a.electedPRepCount, global.ElectedPRepCount)
				assert.Equal(t, a.bondRequirement, global.BondRequirement)
			}
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
