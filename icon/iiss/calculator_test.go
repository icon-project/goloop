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

package iiss

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
)

func TestCalculator(t *testing.T) {
	database := db.NewMapDB()
	c := NewCalculator()

	err := c.Init(database)
	assert.NoError(t, err)
	assert.Equal(t, database, c.dbase)
	assert.Equal(t, int64(0), c.blockHeight)

	c.blockHeight = 100
	c.stats.blockProduce.SetInt64(int64(100))
	c.stats.Voted.SetInt64(int64(200))
	c.stats.voting.SetInt64(int64(300))
	err = c.Flush()
	assert.NoError(t, err)

	c2 := NewCalculator()
	err = c2.Init(database)
	assert.NoError(t, err)
	assert.Equal(t, c.dbase, c2.dbase)
	assert.Equal(t, c.blockHeight, c2.blockHeight)
	assert.True(t, c.stats.equal(c2.stats))
}

func MakeCalculator(database db.Database, back *icstage.Snapshot) *Calculator {
	c := NewCalculator()
	c.back = back
	c.base = icreward.NewSnapshot(database, nil)
	c.temp = c.base.NewState()

	return c
}

func TestCalculator_processClaim(t *testing.T) {
	database := db.NewMapDB()
	s := icstage.NewState(database)

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	v1 := int64(100)
	v2 := int64(200)

	type args struct {
		addr  *common.Address
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
			"Add Claim 200 to new address",
			args{
				addr2,
				big.NewInt(v2),
			},
			v2,
		},
	}

	c := MakeCalculator(database, s.GetSnapshot())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			iScore := icreward.NewIScore()
			iScore.Value.Set(args.value)
			err := c.temp.SetIScore(args.addr, iScore)
			assert.NoError(t, err)

			err = s.AddIScoreClaim(args.addr, args.value)
			assert.NoError(t, err)
		})
	}

	err := c.processClaim()
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			iScore, err := c.temp.GetIScore(args.addr)
			assert.NoError(t, err)
			assert.Equal(t, 0, args.value.Cmp(iScore.Value))
		})
	}
}

func TestCalculator_processBlockProduce(t *testing.T) {
	addr0 := common.NewAddressFromString("hx0")
	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")
	variable := big.NewInt(int64(YearBlock * IScoreICXRatio))
	rewardGenerate := variable.Int64()
	rewardValidate := variable.Int64()

	type args struct {
		bp       *icstage.BlockProduce
		variable *big.Int
	}
	tests := []struct {
		name  string
		args  args
		err   bool
		wants [4]int64
	}{
		{
			name: "Zero Irep",
			args: args{
				&icstage.BlockProduce{
					ProposerIndex: 0,
					VoteCount:     0,
					VoteMask:      new(big.Int).SetInt64(int64(0b0)),
				},
				new(big.Int),
			},
			err:   false,
			wants: [4]int64{0, 0, 0, 0},
		},
		{
			name: "All voted",
			args: args{
				&icstage.BlockProduce{
					ProposerIndex: 0,
					VoteCount:     4,
					VoteMask:      new(big.Int).SetInt64(int64(0b1111)),
				},
				variable,
			},
			err: false,
			wants: [4]int64{
				rewardGenerate + rewardValidate/4,
				rewardValidate / 4,
				rewardValidate / 4,
				rewardValidate / 4,
			},
		},
		{
			name: "3 P-Rep voted",
			args: args{
				&icstage.BlockProduce{
					ProposerIndex: 2,
					VoteCount:     3,
					VoteMask:      new(big.Int).SetInt64(int64(0b0111)),
				},
				variable,
			},
			err: false,
			wants: [4]int64{
				rewardValidate / 3,
				rewardValidate / 3,
				rewardGenerate + rewardValidate/3,
				0,
			},
		},
		{
			name: "Invalid proposerIndex",
			args: args{
				&icstage.BlockProduce{
					ProposerIndex: 5,
					VoteCount:     3,
					VoteMask:      new(big.Int).SetInt64(int64(0b0111)),
				},
				variable,
			},
			err:   true,
			wants: [4]int64{0, 0, 0, 0},
		},
		{
			name: "There is no validator Info. for voter",
			args: args{
				&icstage.BlockProduce{
					ProposerIndex: 5,
					VoteCount:     16,
					VoteMask:      new(big.Int).SetInt64(int64(0b01111111111111111)),
				},
				variable,
			},
			err:   true,
			wants: [4]int64{0, 0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			vs := makeVS(addr0, addr1, addr2, addr3)
			err := processBlockProduce(in.bp, in.variable, vs)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				for i, v := range vs {
					assert.Equal(t, tt.wants[i], v.iScore.Int64(), "index %d", i)
				}
			}
		})
	}
}

func makeVS(addrs ...*common.Address) []*validator {
	vs := make([]*validator, 0)

	for _, addr := range addrs {
		vs = append(vs, newValidator(addr))
	}
	return vs
}

func TestCalculator_varForVotedReward(t *testing.T) {
	tests := []struct {
		name string
		args *icstage.Global
		want int64
	}{
		{
			"Global Version1",
			&icstage.Global{
				GlobalImpl: &icstage.GlobalV1{
					IISSVersion:      icstate.IISSVersion1,
					OffsetLimit:      100,
					Irep:             big.NewInt(MonthBlock),
					Rrep:             big.NewInt(200),
					MainPRepCount:    22,
					ElectedPRepCount: 100,
				},
			},
			// 	variable = irep * electedPRepCount * IScoreICXRatio / (2 * MonthBlock)
			MonthBlock * 100 * IScoreICXRatio / (2 * MonthBlock),
		},
		{
			"Global Version1 - disabled",
			&icstage.Global{
				GlobalImpl: &icstage.GlobalV1{
					IISSVersion:      icstate.IISSVersion1,
					OffsetLimit:      100,
					Irep:             big.NewInt(0),
					Rrep:             big.NewInt(200),
					MainPRepCount:    22,
					ElectedPRepCount: 100,
				},
			},
			0,
		},
		{
			"Global Version2",
			&icstage.Global{
				GlobalImpl: &icstage.GlobalV2{
					IISSVersion:      icstate.IISSVersion2,
					OffsetLimit:      1000,
					Iglobal:          big.NewInt(10000),
					Iprep:            big.NewInt(50),
					Ivoter:           big.NewInt(50),
					ElectedPRepCount: 100,
					BondRequirement:  5,
				},
			},
			// 	variable = iglobal * iprep * IScoreICXRatio / (100 * TermPeriod)
			10000 * 50 * IScoreICXRatio / (100 * 1000),
		},
		{
			"Global Version2 - disabled",
			&icstage.Global{
				GlobalImpl: &icstage.GlobalV2{
					IISSVersion:      icstate.IISSVersion2,
					OffsetLimit:      0,
					Iglobal:          big.NewInt(0),
					Iprep:            big.NewInt(0),
					Ivoter:           big.NewInt(0),
					ElectedPRepCount: 0,
					BondRequirement:  0,
				},
			},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret := varForVotedReward(tt.args)
			assert.Equal(t, tt.want, ret.Int64())
		})
	}
}

func newVotedDataForTest(enable bool, delegated int64, bonded int64, bondRequirement int, iScore int64) *votedData {
	data := &votedData{
		voted: &icreward.Voted{
			Enable:           enable,
			Delegated:        big.NewInt(delegated),
			Bonded:           big.NewInt(bonded),
			BondedDelegation: big.NewInt(0),
		},
		iScore: big.NewInt(iScore),
	}
	data.voted.UpdateBondedDelegation(bondRequirement)
	return data
}

func TestDelegatedData_compare(t *testing.T) {
	d1 := newVotedDataForTest(true, 10, 0, 0, 10)
	d2 := newVotedDataForTest(true, 20, 0, 0, 20)
	d3 := newVotedDataForTest(true, 20, 0, 0, 21)
	d4 := newVotedDataForTest(false, 30, 0, 0, 30)
	d5 := newVotedDataForTest(false, 31, 0, 0, 31)
	type args struct {
		d1 *votedData
		d2 *votedData
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"x<y",
			args{d1, d2},
			-1,
		},
		{
			"x<y,disable",
			args{d5, d2},
			-1,
		},
		{
			"x==y",
			args{d2, d3},
			0,
		},
		{
			"x==y,disable",
			args{d4, d5},
			0,
		},
		{
			"x>y",
			args{d3, d1},
			1,
		},
		{
			"x>y,disable",
			args{d1, d4},
			1,
		},
	}
	for _, tt := range tests {
		args := tt.args
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, args.d1.compare(args.d2))
		})
	}
}

func TestDelegated_setEnable(t *testing.T) {
	totalVoted := new(big.Int)
	vInfo := newVotedInfo(100)
	enable := false
	for i := int64(1); i < 6; i += 1 {
		enable = !enable
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newVotedDataForTest(enable, i, i, 1, 0)
		vInfo.addVotedData(addr, data)
		if enable {
			totalVoted.Add(totalVoted, data.GetVotedAmount())
		}
	}
	assert.Equal(t, 0, totalVoted.Cmp(vInfo.totalVoted))

	enable = true
	for key, vData := range vInfo.preps {
		enable = !enable
		addr, err := common.NewAddress([]byte(key))
		assert.NoError(t, err)

		if enable != vData.voted.Enable {
			if enable {
				totalVoted.Add(totalVoted, vData.GetVotedAmount())
			} else {
				totalVoted.Sub(totalVoted, vData.GetVotedAmount())
			}
		}
		vInfo.setEnable(addr, enable)
		assert.Equal(t, enable, vData.voted.Enable)
		assert.Equal(t, 0, totalVoted.Cmp(vInfo.totalVoted), "%s: %v\t%v", addr.String(), totalVoted, vInfo.totalVoted)
	}

	addr := common.NewAddressFromString("hx123412341234")
	vInfo.setEnable(addr, false)
	prep, ok := vInfo.preps[string(addr.Bytes())]
	assert.True(t, ok)
	assert.Equal(t, false, prep.Enable())
	assert.True(t, prep.voted.IsEmpty())
	assert.Equal(t, 0, prep.iScore.Sign())
	assert.Equal(t, 0, totalVoted.Cmp(vInfo.totalVoted))
}

func TestDelegated_updateDelegated(t *testing.T) {
	vInfo := newVotedInfo(100)
	votes := make([]*icstage.Vote, 0)
	enable := true
	for i := int64(1); i < 6; i += 1 {
		enable = !enable
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newVotedDataForTest(enable, i, i, 1, 0)
		vInfo.addVotedData(addr, data)

		votes = append(
			votes,
			&icstage.Vote{
				Address: addr,
				Value:   big.NewInt(i),
			},
		)
	}
	newAddr := common.NewAddressFromString("hx321321")
	votes = append(
		votes,
		&icstage.Vote{
			Address: newAddr,
			Value:   big.NewInt(100),
		},
	)

	totalVoted := new(big.Int).Set(vInfo.totalVoted)
	vInfo.updateDelegated(votes)
	for _, v := range votes {
		expect := v.Value.Int64() * 2
		if v.Address.Equal(newAddr) {
			expect = v.Value.Int64()
		}
		voted := vInfo.preps[string(v.Address.Bytes())].voted
		assert.Equal(t, expect, voted.Delegated.Int64())

		if voted.Enable {
			totalVoted.Add(totalVoted, v.Value)
		}
	}
	assert.Equal(t, 0, totalVoted.Cmp(vInfo.totalVoted))
}

func TestDelegated_updateBonded(t *testing.T) {
	vInfo := newVotedInfo(100)
	votes := make([]*icstage.Vote, 0)
	enable := true
	for i := int64(1); i < 6; i += 1 {
		enable = !enable
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newVotedDataForTest(enable, i, i, 1, 0)
		vInfo.addVotedData(addr, data)

		votes = append(
			votes,
			&icstage.Vote{
				Address: addr,
				Value:   big.NewInt(i),
			},
		)
	}
	newAddr := common.NewAddressFromString("hx321321")
	votes = append(
		votes,
		&icstage.Vote{
			Address: newAddr,
			Value:   big.NewInt(100),
		},
	)

	totalVoted := new(big.Int).Set(vInfo.totalVoted)
	vInfo.updateBonded(votes)
	for _, v := range votes {
		expect := v.Value.Int64() * 2
		if v.Address.Equal(newAddr) {
			expect = v.Value.Int64()
		}
		voted := vInfo.preps[string(v.Address.Bytes())].voted
		assert.Equal(t, expect, voted.Bonded.Int64())

		if voted.Enable {
			totalVoted.Add(totalVoted, v.Value)
		}
	}
	assert.Equal(t, 0, totalVoted.Cmp(vInfo.totalVoted))
}

func TestDelegated_SortAndUpdateTotalBondedDelegation(t *testing.T) {
	d := newVotedInfo(100)
	total := int64(0)
	more := int64(10)
	maxIndex := int64(d.maxRankForReward) + more
	for i := int64(1); i <= maxIndex; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newVotedDataForTest(true, i, 0, 0, i)
		d.addVotedData(addr, data)
		if i > more {
			total += i
		}
	}
	d.sort()
	d.updateTotalBondedDelegation()
	assert.Equal(t, total, d.totalBondedDelegation.Int64())

	for i, rank := range d.rank {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", maxIndex-int64(i)))
		assert.Equal(t, string(addr.Bytes()), rank)
	}
}

func TestDelegated_calculateReward(t *testing.T) {
	vInfo := newVotedInfo(100)
	total := int64(0)
	more := int64(10)
	maxIndex := int64(vInfo.maxRankForReward) + more
	for i := int64(1); i <= maxIndex; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newVotedDataForTest(true, i, 0, 0, 0)
		vInfo.addVotedData(addr, data)
		if i > more {
			total += i
		}
	}
	vInfo.sort()
	vInfo.updateTotalBondedDelegation()
	assert.Equal(t, total, vInfo.totalBondedDelegation.Int64())

	variable := big.NewInt(YearBlock)
	period := 10000
	bigIntPeriod := big.NewInt(int64(period))

	vInfo.calculateReward(variable, period)

	for i, addr := range vInfo.rank {
		expect := big.NewInt(maxIndex - int64(i))
		if i >= vInfo.maxRankForReward {
			expect.SetInt64(0)
		} else {
			expect.Mul(expect, variable)
			expect.Mul(expect, bigIntPeriod)
			expect.Div(expect, vInfo.totalBondedDelegation)
		}
		assert.Equal(t, expect.Int64(), vInfo.preps[addr].iScore.Int64())
	}
}

func TestCalculator_varForVotingReward(t *testing.T) {
	tests := []struct {
		name string
		args *icstage.Global
		want int64
	}{
		{
			"Global Version1",
			&icstage.Global{
				GlobalImpl: &icstage.GlobalV1{
					IISSVersion:      icstate.IISSVersion1,
					OffsetLimit:      100,
					Irep:             big.NewInt(MonthBlock),
					Rrep:             big.NewInt(20000000),
					MainPRepCount:    22,
					ElectedPRepCount: 100,
				},
			},
			// 	variable = rrep * IScoreICXRatio / YearBlock
			20000000 * IScoreICXRatio / YearBlock,
		},
		{
			"Global Version1 - disabled",
			&icstage.Global{
				GlobalImpl: &icstage.GlobalV1{
					IISSVersion:      icstate.IISSVersion1,
					OffsetLimit:      100,
					Irep:             big.NewInt(MonthBlock),
					Rrep:             big.NewInt(0),
					MainPRepCount:    22,
					ElectedPRepCount: 100,
				},
			},
			0,
		},
		{
			"Global Version2",
			&icstage.Global{
				GlobalImpl: &icstage.GlobalV2{
					IISSVersion:      icstate.IISSVersion2,
					OffsetLimit:      1000,
					Iglobal:          big.NewInt(10000),
					Iprep:            big.NewInt(50),
					Ivoter:           big.NewInt(50),
					ElectedPRepCount: 100,
					BondRequirement:  5,
				},
			},
			// 	variable = iglobal * ivoter * IScoreICXRatio / (100 * TermPeriod)
			10000 * 50 * IScoreICXRatio / (100 * 1000),
		},
		{
			"Global Version2 - disabled",
			&icstage.Global{
				GlobalImpl: &icstage.GlobalV2{
					IISSVersion:      icstate.IISSVersion2,
					OffsetLimit:      0,
					Iglobal:          big.NewInt(0),
					Iprep:            big.NewInt(0),
					Ivoter:           big.NewInt(0),
					ElectedPRepCount: 0,
					BondRequirement:  0,
				},
			},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret := varForVotingReward(tt.args)
			assert.Equal(t, tt.want, ret.Int64())
		})
	}
}

func TestCalculator_DelegatingReward(t *testing.T) {
	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")
	addr4 := common.NewAddressFromString("hx4")
	prepInfo := map[string]*pRepEnable{
		string(addr1.Bytes()): {0, 0},
		string(addr2.Bytes()): {10, 0},
		string(addr3.Bytes()): {100, 200},
	}

	d1 := &icstate.Delegation{
		Address: addr1,
		Value:   common.NewHexInt(100),
	}
	d2 := &icstate.Delegation{
		Address: addr2,
		Value:   common.NewHexInt(100),
	}
	d3 := &icstate.Delegation{
		Address: addr3,
		Value:   common.NewHexInt(100),
	}
	d4 := &icstate.Delegation{
		Address: addr4,
		Value:   common.NewHexInt(100),
	}
	type args struct {
		rrep       int
		from       int
		to         int
		delegating icstate.Delegations
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "PRep-full",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d1},
			},
			want: 100 * 100 * 1000 * 1000 / YearBlock,
		},
		{
			name: "PRep-enabled",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d2},
			},
			want: 100 * 100 * (1000 - 10) * 1000 / YearBlock,
		},
		{
			name: "PRep-disabled",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d3},
			},
			want: 100 * 100 * (200 - 100) * 1000 / YearBlock,
		},
		{
			name: "PRep-None",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d4},
			},
			want: 0,
		},
		{
			name: "PRep-combination",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d1, d2, d3, d4},
			},
			want: (100 * 100 * 1000 * 1000 / YearBlock) +
				(100 * 100 * (1000 - 10) * 1000 / YearBlock) +
				(100 * 100 * (200 - 100) * 1000 / YearBlock),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			reward := delegatingReward(
				big.NewInt(int64(args.rrep)),
				args.from,
				args.to,
				prepInfo,
				args.delegating,
			)
			assert.Equal(t, tt.want, reward.Int64())
		})
	}
}

func TestCalculator_processDelegating(t *testing.T) {
	database := db.NewMapDB()
	s := icstage.NewState(database)
	c := MakeCalculator(database, s.GetSnapshot())

	variable := 100
	varBigInt := big.NewInt(int64(variable))
	from := 0
	to := 100
	offset := 50

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")
	addr4 := common.NewAddressFromString("hx4")
	addr5 := common.NewAddressFromString("hx5")

	d1Value := 100
	d2Value := 200
	d1 := &icstate.Delegation{
		Address: addr1,
		Value:   common.NewHexInt(int64(d1Value)),
	}
	d2 := &icstate.Delegation{
		Address: addr2,
		Value:   common.NewHexInt(int64(d2Value)),
	}
	d3 := &icstate.Delegation{
		Address: addr3,
		Value:   common.NewHexInt(int64(d2Value)),
	}
	d5 := &icstate.Delegation{
		Address: addr5,
		Value:   common.NewHexInt(int64(d2Value)),
	}
	ds1 := icstate.Delegations{d1}
	ds2 := icstate.Delegations{d2}
	ds3 := icstate.Delegations{d3}
	ds5 := icstate.Delegations{d5}

	vote1 := &icstage.Vote{
		Address: addr1,
		Value:   big.NewInt(int64(d1Value)),
	}
	votes1 := icstage.VoteList{vote1}

	// make pRepInfo.
	prepInfo := make(map[string]*pRepEnable)
	prepInfo[string(addr1.Bytes())] = &pRepEnable{0, 0}
	prepInfo[string(addr3.Bytes())] = &pRepEnable{0, offset}
	prepInfo[string(addr5.Bytes())] = &pRepEnable{offset, 0}

	// write delegating data to base
	dting1 := icreward.NewDelegating()
	dting1.Delegations = ds1
	dting2 := icreward.NewDelegating()
	dting2.Delegations = ds2
	dting3 := icreward.NewDelegating()
	dting3.Delegations = ds3
	dting5 := icreward.NewDelegating()
	dting5.Delegations = ds5
	c.temp.SetDelegating(addr2, dting1.Clone())
	c.temp.SetDelegating(addr3, dting2.Clone())
	c.temp.SetDelegating(addr4, dting3.Clone())
	c.temp.SetDelegating(addr5, dting5.Clone())
	c.base = c.temp.GetSnapshot()

	// make delegationMap
	delegationMap := make(map[string]map[int]icstage.VoteList)
	delegationMap[string(addr1.Bytes())] = make(map[int]icstage.VoteList)
	delegationMap[string(addr1.Bytes())][from+offset] = votes1

	err := c.processDelegating(varBigInt, from, to, prepInfo, delegationMap)
	assert.NoError(t, err)

	type args struct {
		addr *common.Address
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "Modify voting configuration",
			args: args{addr1},
			want: int64(0),
		},
		{
			name: "Delegated to P-Rep",
			args: args{addr2},
			want: int64(variable * d1Value * (to - from) * IScoreICXRatio / YearBlock),
		},
		{
			name: "Delegated to none P-Rep",
			args: args{addr3},
			want: 0,
		},
		{
			name: "Delegated to P-Rep and got penalty",
			args: args{addr4},
			want: int64(variable * d2Value * (offset - from) * IScoreICXRatio / YearBlock),
		},
		{
			name: "Delegated to none P-Rep and register P-Rep later",
			args: args{addr5},
			want: int64(variable * d2Value * (to - offset) * IScoreICXRatio / YearBlock),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args

			iScore, err := c.temp.GetIScore(in.addr)
			assert.NoError(t, err)
			if tt.want == 0 {
				if iScore != nil && iScore.Value.Int64() != 0 {
					t.Errorf("FAIL: tt.name")
				}
			} else {
				assert.Equal(t, tt.want, iScore.Value.Int64())
			}
		})
	}
}

func TestCalculator_processDelegateEvent(t *testing.T) {
	database := db.NewMapDB()
	s := icstage.NewState(database)
	c := MakeCalculator(database, s.GetSnapshot())

	variable := 100
	varBigInt := big.NewInt(int64(variable))
	from := 0
	to := 100
	offset := 50

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")

	d1Value := 100
	d2Value := 200
	d1 := &icstate.Delegation{
		Address: addr1,
		Value:   common.NewHexInt(int64(d1Value)),
	}
	d2 := &icstate.Delegation{
		Address: addr1,
		Value:   common.NewHexInt(int64(d2Value)),
	}
	ds1 := icstate.Delegations{d1}
	ds2 := icstate.Delegations{d2}

	// make pRepInfo. all enabled
	prepInfo := make(map[string]*pRepEnable)
	prepInfo[string(addr1.Bytes())] = &pRepEnable{0, 0}

	vote1 := &icstage.Vote{
		Address: addr1,
		Value:   big.NewInt(int64(d1Value)),
	}
	vote1Negative := &icstage.Vote{
		Address: addr1,
		Value:   big.NewInt(int64(-d1Value)),
	}
	vote2 := &icstage.Vote{
		Address: addr1,
		Value:   big.NewInt(int64(d2Value)),
	}
	votes1 := icstage.VoteList{vote1}
	votes2 := icstage.VoteList{vote2}
	votes1Negative := icstage.VoteList{vote1Negative}

	// write delegating data to base
	dting1 := icreward.NewDelegating()
	dting1.Delegations = ds1
	dting2 := icreward.NewDelegating()
	dting2.Delegations = ds2
	c.temp.SetDelegating(addr2, dting2.Clone())
	c.temp.SetDelegating(addr3, dting1.Clone())
	c.base = c.temp.GetSnapshot()

	// make delegationMap
	delegationMap := make(map[string]map[int]icstage.VoteList)
	delegationMap[string(addr1.Bytes())] = make(map[int]icstage.VoteList)
	delegationMap[string(addr1.Bytes())][from+offset] = votes2
	delegationMap[string(addr2.Bytes())] = make(map[int]icstage.VoteList)
	delegationMap[string(addr2.Bytes())][from] = votes1
	delegationMap[string(addr2.Bytes())][from+offset] = votes2
	delegationMap[string(addr3.Bytes())] = make(map[int]icstage.VoteList)
	delegationMap[string(addr3.Bytes())][from+offset] = votes1Negative

	err := c.processDelegateEvent(varBigInt, to, prepInfo, delegationMap)
	assert.NoError(t, err)

	type args struct {
		addr *common.Address
	}
	tests := []struct {
		name       string
		args       args
		want       int64
		delegating *icreward.Delegating
	}{
		{
			name:       "Delegate New",
			args:       args{addr1},
			want:       int64(variable * d2Value * (to - offset) * IScoreICXRatio / YearBlock),
			delegating: dting2,
		},
		{
			name: "Delegated and modified",
			args: args{addr2},
			want: int64(variable*(d1Value+d2Value)*(offset-from)*IScoreICXRatio/YearBlock) +
				int64(variable*(d2Value+d1Value+d2Value)*(to-offset)*IScoreICXRatio/YearBlock),
			delegating: &icreward.Delegating{
				Delegations: icstate.Delegations{
					&icstate.Delegation{
						Address: addr1,
						Value:   common.NewHexInt(int64(d1Value + d2Value*2)),
					},
				},
			},
		},
		{
			name:       "Delegating removed",
			args:       args{addr3},
			want:       int64(variable * d1Value * (offset - from) * IScoreICXRatio / YearBlock),
			delegating: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args

			iScore, err := c.temp.GetIScore(in.addr)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, iScore.Value.Int64())

			delegating, err := c.temp.GetDelegating(in.addr)
			assert.NoError(t, err)
			if tt.delegating != nil {
				assert.NotNil(t, delegating)
				assert.True(t, delegating.Equal(tt.delegating), "%v\n%v", tt.delegating, delegating)
			} else {
				assert.Nil(t, delegating)
			}
		})
	}
}
func TestCalculator_VotingReward(t *testing.T) {
	vInfo := newVotedInfo(100)
	maxIndex := int64(vInfo.maxRankForReward)
	var enable bool
	for i := int64(1); i <= maxIndex; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		delegated := i * 1000
		bonded := i * 2000
		if i%2 != 0 {
			enable = false
		} else {
			enable = true
		}
		data := newVotedDataForTest(enable, delegated, bonded, 0, 0)
		vInfo.addVotedData(addr, data)
	}
	vInfo.sort()
	vInfo.updateTotalBondedDelegation()

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")
	addr4 := common.NewAddressFromString("hx4")

	d1 := &icstate.Delegation{
		Address: addr1,
		Value:   common.NewHexInt(100),
	}
	d2 := &icstate.Delegation{
		Address: addr2,
		Value:   common.NewHexInt(100),
	}
	b3 := &icstate.Bond{
		Address: addr3,
		Value:   common.NewHexInt(100),
	}
	b4 := &icstate.Bond{
		Address: addr4,
		Value:   common.NewHexInt(100),
	}
	nonePRep := &icstate.Delegation{
		Address: common.NewAddressFromString("hxffffffffff"),
		Value:   common.NewHexInt(100),
	}
	type args struct {
		variable int
		from     int
		to       int
		vInfo    *votedInfo
		iter     icstate.VotingIterator
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "variable is zero",
			args: args{
				0,
				0,
				1000,
				vInfo,
				icstate.NewVotingIterator([]icstate.Voting{d2, b4}),
			},
			// reward = variable * period * voting / total_voting
			want: 0,
		},
		{
			name: "period is zero",
			args: args{
				10000,
				1000,
				1000,
				vInfo,
				icstate.NewVotingIterator([]icstate.Voting{d2, b4}),
			},
			// reward = variable * period * voting / total_voting
			want: 0,
		},
		{
			name: "vInfo is nil",
			args: args{
				10000,
				0,
				1000,
				nil,
				icstate.NewVotingIterator([]icstate.Voting{d2, b4}),
			},
			// reward = variable * period * voting / total_voting
			want: 0,
		},
		{
			name: "empty vInfo",
			args: args{
				10000,
				0,
				1000,
				newVotedInfo(100),
				icstate.NewVotingIterator([]icstate.Voting{d2, b4}),
			},
			// reward = variable * period * voting / total_voting
			want: 0,
		},
		{
			name: "iter is nil",
			args: args{
				10000,
				0,
				1000,
				vInfo,
				nil,
			},
			// reward = variable * period * voting / total_voting
			want: 0,
		},
		{
			name: "empty iter",
			args: args{
				10000,
				0,
				1000,
				vInfo,
				icstate.NewVotingIterator([]icstate.Voting{}),
			},
			// reward = variable * period * voting / total_voting
			want: 0,
		},
		{
			name: "voting to disabled P-Rep",
			args: args{
				10000,
				0,
				1000,
				vInfo,
				icstate.NewVotingIterator([]icstate.Voting{d1, b3}),
			},
			// reward = variable * period * voting / total_voting
			want: 0,
		},
		{
			name: "voting to none P-Rep",
			args: args{
				10000,
				0,
				1000,
				vInfo,
				icstate.NewVotingIterator([]icstate.Voting{nonePRep}),
			},
			// reward = variable * period * voting / total_voting
			want: 0,
		},
		{
			name: "voting to enabled P-Rep",
			args: args{
				10000,
				0,
				1000,
				vInfo,
				icstate.NewVotingIterator([]icstate.Voting{d2, b4}),
			},
			// reward = variable * period * voting / total_voting
			want: 10000*(1000-0)*100/vInfo.totalVoted.Int64() +
				10000*(1000-0)*100/vInfo.totalVoted.Int64(),
		},
		{
			name: "voting to P-Rep",
			args: args{
				10000,
				0,
				1000,
				vInfo,
				icstate.NewVotingIterator([]icstate.Voting{d1, d2, b3, b4}),
			},
			// reward = variable * period * voting / total_voting
			want: 10000*(1000-0)*100/vInfo.totalVoted.Int64() +
				0 +
				10000*(1000-0)*100/vInfo.totalVoted.Int64() +
				0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			reward := votingReward(
				big.NewInt(int64(in.variable)),
				in.from,
				in.to,
				in.vInfo,
				in.iter,
			)
			assert.Equal(t, tt.want, reward.Int64())
		})
	}
}

func TestCalculator_processVoting(t *testing.T) {
	database := db.NewMapDB()
	s := icstage.NewState(database)
	c := MakeCalculator(database, s.GetSnapshot())

	vInfo := newVotedInfo(100)
	maxIndex := int64(vInfo.maxRankForReward)
	for i := int64(1); i <= maxIndex; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		delegated := i * 1000
		bonded := i * 2000
		data := newVotedDataForTest(true, delegated, bonded, 0, 0)
		vInfo.addVotedData(addr, data)
	}
	vInfo.sort()
	vInfo.updateTotalBondedDelegation()

	variable := 10000
	varBigInt := big.NewInt(int64(variable))
	from := 0
	to := 1000
	value := 100

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")
	addr4 := common.NewAddressFromString("hx4")
	addr5 := common.NewAddressFromString("hx5")

	d1 := &icstate.Delegation{
		Address: addr1,
		Value:   common.NewHexInt(int64(value)),
	}
	d2 := &icstate.Delegation{
		Address: addr2,
		Value:   common.NewHexInt(int64(value)),
	}
	dNonePRep := &icstate.Delegation{
		Address: common.NewAddressFromString("hx32123ffffff"),
		Value:   common.NewHexInt(int64(value)),
	}
	b4 := &icstate.Bond{
		Address: addr4,
		Value:   common.NewHexInt(int64(value)),
	}
	b5 := &icstate.Bond{
		Address: addr5,
		Value:   common.NewHexInt(int64(value)),
	}
	bNonePRep := &icstate.Bond{
		Address: common.NewAddressFromString("hx32123ffffff"),
		Value:   common.NewHexInt(int64(value)),
	}
	ds1 := icstate.Delegations{d1, d2}
	ds2 := icstate.Delegations{d2, dNonePRep}
	ds3 := icstate.Delegations{dNonePRep}
	bs1 := icstate.Bonds{b4, b5}
	bs2 := icstate.Bonds{bNonePRep}

	// write delegating and bonding data to base
	dting1 := icreward.NewDelegating()
	dting1.Delegations = ds1
	dting2 := icreward.NewDelegating()
	dting2.Delegations = ds2
	dting3 := icreward.NewDelegating()
	dting3.Delegations = ds3
	bondig1 := icreward.NewBonding()
	bondig1.Bonds = bs1
	bondig2 := icreward.NewBonding()
	bondig2.Bonds = bs2

	c.temp.SetDelegating(addr1, dting1.Clone())

	c.temp.SetBonding(addr2, bondig1.Clone())

	c.temp.SetDelegating(addr3, dting2.Clone())
	c.temp.SetBonding(addr3, bondig1.Clone())

	c.temp.SetDelegating(addr4, dting3.Clone())

	c.temp.SetBonding(addr5, bondig2.Clone())

	c.base = c.temp.GetSnapshot()

	err := c.processVoting(varBigInt, from, to, vInfo)
	assert.NoError(t, err)

	type args struct {
		addr *common.Address
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "delegating only",
			args: args{addr1},
			// reward = variable * period * voting / total_voting
			want: int64(variable*(to-from)*value)/vInfo.totalVoted.Int64() +
				int64(variable*(to-from)*value)/vInfo.totalVoted.Int64(),
		},
		{
			name: "bonding only",
			args: args{addr2},
			want: int64(variable*(to-from)*value)/vInfo.totalVoted.Int64() +
				int64(variable*(to-from)*value)/vInfo.totalVoted.Int64(),
		},
		{
			name: "delegating and bonding",
			args: args{addr3},
			want: int64(variable*(to-from)*value)/vInfo.totalVoted.Int64() +
				int64(variable*(to-from)*value)/vInfo.totalVoted.Int64() +
				int64(variable*(to-from)*value)/vInfo.totalVoted.Int64(),
		},
		{
			name: "delegating to none P-Rep",
			args: args{addr4},
			want: int64(0),
		},
		{
			name: "bonding to none P-Rep",
			args: args{addr5},
			want: int64(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args

			iScore, err := c.temp.GetIScore(in.addr)
			assert.NoError(t, err)
			if tt.want == 0 {
				if iScore != nil && iScore.Value.Int64() != 0 {
					t.Errorf("FAIL: tt.name")
				}
			} else {
				assert.Equal(t, tt.want, iScore.Value.Int64())
			}
		})
	}
}
