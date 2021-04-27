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

package iiss

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
)

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

	addr1 := common.MustNewAddressFromString("hx1")
	addr2 := common.MustNewAddressFromString("hx2")
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
			iScore := icreward.NewIScore(args.value)
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
			assert.Equal(t, 0, args.value.Cmp(iScore.Value()))
		})
	}
}

func TestCalculator_processBlockProduce(t *testing.T) {
	addr0 := common.MustNewAddressFromString("hx0")
	addr1 := common.MustNewAddressFromString("hx1")
	addr2 := common.MustNewAddressFromString("hx2")
	addr3 := common.MustNewAddressFromString("hx3")
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
				icstage.NewBlockProduce(0, 0, new(big.Int).SetInt64(int64(0b0))),
				new(big.Int),
			},
			err:   false,
			wants: [4]int64{0, 0, 0, 0},
		},
		{
			name: "All voted",
			args: args{
				icstage.NewBlockProduce(0, 4, new(big.Int).SetInt64(int64(0b1111))),
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
				icstage.NewBlockProduce(2, 3, new(big.Int).SetInt64(int64(0b0111))),
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
				icstage.NewBlockProduce(5, 3, new(big.Int).SetInt64(int64(0b0111))),
				variable,
			},
			err:   true,
			wants: [4]int64{0, 0, 0, 0},
		},
		{
			name: "There is no validator Info. for voter",
			args: args{
				icstage.NewBlockProduce(5, 16, new(big.Int).SetInt64(int64(0b01111111111111111))),
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
		name                string
		args                icstage.Global
		multiplier, divider int64
	}{
		{
			"Global Version1",
			icstage.NewGlobalV1(
				icstate.IISSVersion1,
				0,
				100-1,
				icmodule.RevisionIISS,
				big.NewInt(MonthBlock),
				big.NewInt(200),
				22,
				100,
			),
			// 	variable = irep * electedPRepCount * IScoreICXRatio / (2 * MonthBlock)
			MonthBlock * 100 * IScoreICXRatio,
			2 * MonthBlock,
		},
		{
			"Global Version1 - disabled",
			icstage.NewGlobalV1(
				icstate.IISSVersion1,
				0,
				100-1,
				icmodule.RevisionIISS,
				big.NewInt(0),
				big.NewInt(200),
				22,
				100,
			),
			0,
			MonthBlock * 2,
		},
		{
			"Global Version2",
			icstage.NewGlobalV2(
				icstate.IISSVersion2,
				0,
				1000-1,
				icmodule.Revision13,
				big.NewInt(10000),
				big.NewInt(50),
				big.NewInt(50),
				100,
				5,
			),
			// 	variable = iglobal * iprep * IScoreICXRatio / (100 * TermPeriod)
			10000 * 50 * IScoreICXRatio,
			100 * 1000,
		},
		{
			"Global Version2 - disabled",
			icstage.NewGlobalV2(
				icstate.IISSVersion2,
				0,
				-1,
				icmodule.Revision13,
				big.NewInt(0),
				big.NewInt(0),
				big.NewInt(0),
				0,
				0,
			),
			0,
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiplier, divider := varForVotedReward(tt.args)
			assert.Equal(t, tt.multiplier, multiplier.Int64())
			assert.Equal(t, tt.divider, divider.Int64())
		})
	}
}

func newVotedDataForTest(enable bool, delegated int64, bonded int64, bondRequirement int, iScore int64) *votedData {
	voted := icreward.NewVoted()
	voted.SetEnable(enable)
	voted.SetDelegated(big.NewInt(delegated))
	voted.SetBonded(big.NewInt(bonded))
	voted.SetBondedDelegation(big.NewInt(0))
	data := newVotedData(voted)
	data.iScore = big.NewInt(iScore)
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

func TestVotedInfo_setEnable(t *testing.T) {
	totalVoted := new(big.Int)
	vInfo := newVotedInfo(100)
	status := icstage.ESDisablePermanent
	for i := int64(1); i < 6; i += 1 {
		status = status % icstage.ESMax
		addr := common.MustNewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newVotedDataForTest(status.IsEnabled(), i, i, 1, 0)
		vInfo.addVotedData(addr, data)
		if status.IsEnabled() {
			totalVoted.Add(totalVoted, data.GetVotedAmount())
		}
	}
	assert.Equal(t, 0, totalVoted.Cmp(vInfo.totalVoted))

	status = icstage.ESEnable
	for key, vData := range vInfo.preps {
		status = status % icstage.ESMax
		addr, err := common.NewAddress([]byte(key))
		assert.NoError(t, err)

		if status.IsEnabled() != vData.Enable() {
			if status.IsEnabled() {
				totalVoted.Add(totalVoted, vData.GetVotedAmount())
			} else {
				totalVoted.Sub(totalVoted, vData.GetVotedAmount())
			}
		}
		vInfo.setEnable(addr, status)
		assert.Equal(t, status, vData.flag)
		assert.Equal(t, status.IsEnabled(), vData.Enable())
		assert.Equal(t, 0, totalVoted.Cmp(vInfo.totalVoted), "%s: %v\t%v", addr.String(), totalVoted, vInfo.totalVoted)
	}

	addr := common.MustNewAddressFromString("hx123412341234")
	vInfo.setEnable(addr, icstage.ESDisablePermanent)
	prep, ok := vInfo.preps[string(addr.Bytes())]
	assert.True(t, ok)
	assert.Equal(t, false, prep.Enable())
	assert.True(t, prep.voted.IsEmpty())
	assert.Equal(t, 0, prep.iScore.Sign())
	assert.Equal(t, 0, totalVoted.Cmp(vInfo.totalVoted))
}

func TestVotedInfo_updateDelegated(t *testing.T) {
	vInfo := newVotedInfo(100)
	votes := make([]*icstage.Vote, 0)
	enable := true
	for i := int64(1); i < 6; i += 1 {
		enable = !enable
		addr := common.MustNewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newVotedDataForTest(enable, i, i, 1, 0)
		vInfo.addVotedData(addr, data)

		votes = append(votes, icstage.NewVote(addr, big.NewInt(i)))
	}
	newAddr := common.MustNewAddressFromString("hx321321")
	votes = append(votes, icstage.NewVote(newAddr, big.NewInt(100)))

	totalVoted := new(big.Int).Set(vInfo.totalVoted)
	vInfo.updateDelegated(votes)
	for _, v := range votes {
		expect := v.Amount().Int64() * 2
		if v.To().Equal(newAddr) {
			expect = v.Amount().Int64()
		}
		vData := vInfo.preps[icutils.ToKey(v.To())]
		assert.Equal(t, expect, vData.GetDelegated().Int64())

		if vData.Enable() {
			totalVoted.Add(totalVoted, v.Amount())
		}
	}
	assert.Equal(t, 0, totalVoted.Cmp(vInfo.totalVoted))
}

func TestVotedInfo_updateBonded(t *testing.T) {
	vInfo := newVotedInfo(100)
	votes := make([]*icstage.Vote, 0)
	enable := true
	for i := int64(1); i < 6; i += 1 {
		enable = !enable
		addr := common.MustNewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newVotedDataForTest(enable, i, i, 1, 0)
		vInfo.addVotedData(addr, data)

		votes = append(votes, icstage.NewVote(addr, big.NewInt(i)))
	}
	newAddr := common.MustNewAddressFromString("hx321321")
	votes = append(votes, icstage.NewVote(newAddr, big.NewInt(100)))

	totalVoted := new(big.Int).Set(vInfo.totalVoted)
	vInfo.updateBonded(votes)
	for _, v := range votes {
		expect := v.Amount().Int64() * 2
		if v.To().Equal(newAddr) {
			expect = v.Amount().Int64()
		}
		vData := vInfo.preps[icutils.ToKey(v.To())]
		assert.Equal(t, expect, vData.GetBonded().Int64())

		if vData.Enable() {
			totalVoted.Add(totalVoted, v.Amount())
		}
	}
	assert.Equal(t, 0, totalVoted.Cmp(vInfo.totalVoted))
}

func TestVotedInfo_SortAndUpdateTotalBondedDelegation(t *testing.T) {
	d := newVotedInfo(100)
	total := int64(0)
	more := int64(10)
	maxIndex := int64(d.maxRankForReward) + more
	for i := int64(1); i <= maxIndex; i += 1 {
		addr := common.MustNewAddressFromString(fmt.Sprintf("hx%d", i))
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
		addr := common.MustNewAddressFromString(fmt.Sprintf("hx%d", maxIndex-int64(i)))
		assert.Equal(t, string(addr.Bytes()), rank)
	}
}

func TestVotedInfo_calculateReward(t *testing.T) {
	vInfo := newVotedInfo(100)
	total := int64(0)
	more := int64(10)
	maxIndex := int64(vInfo.maxRankForReward) + more
	for i := int64(1); i <= maxIndex; i += 1 {
		addr := common.MustNewAddressFromString(fmt.Sprintf("hx%d", i))
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
	divider := big.NewInt(1)
	period := 10000
	bigIntPeriod := big.NewInt(int64(period))

	vInfo.calculateReward(variable, divider, period)

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
	type args struct {
		global            icstage.Global
		totalVotingAmount *big.Int
	}
	type want struct {
		multiplier int64
		divider    int64
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"Global Version1",
			args{
				icstage.NewGlobalV1(
					icstate.IISSVersion1,
					0,
					100-1,
					icmodule.RevisionIISS,
					big.NewInt(MonthBlock),
					big.NewInt(20000000),
					22,
					100,
				),
				nil,
			},
			want{
				RrepMultiplier * 20000000 * IScoreICXRatio,
				YearBlock * RrepDivider,
			},
		},
		{
			"Global Version1 - disabled",
			args{
				icstage.NewGlobalV1(
					icstate.IISSVersion1,
					0,
					100-1,
					icmodule.RevisionIISS,
					big.NewInt(MonthBlock),
					big.NewInt(0),
					22,
					100,
				),
				nil,
			},
			want{
				0,
				0,
			},
		},
		{
			"Global Version2",
			args{
				icstage.NewGlobalV2(
					icstate.IISSVersion2,
					0,
					1000-1,
					icmodule.Revision13,
					big.NewInt(10000),
					big.NewInt(50),
					big.NewInt(50),
					100,
					5,
				),
				big.NewInt(10),
			},
			// 	multiplier = iglobal * ivoter * IScoreICXRatio / (100 * TermPeriod, totalVotingAmount)
			want{
				10000 * 50 * IScoreICXRatio,
				100 * 1000 * 10,
			},
		},
		{
			"Global Version2 - disabled",
			args{
				icstage.NewGlobalV2(
					icstate.IISSVersion2,
					0,
					0-1,
					icmodule.RevisionIISS,
					big.NewInt(0),
					big.NewInt(0),
					big.NewInt(0),
					0,
					0,
				),
				big.NewInt(10),
			},
			want{
				0,
				0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiplier, divider := varForVotingReward(tt.args.global, tt.args.totalVotingAmount)
			assert.Equal(t, tt.want.multiplier, multiplier.Int64())
			assert.Equal(t, tt.want.divider, divider.Int64())
		})
	}
}

func TestCalculator_VotingReward(t *testing.T) {
	addr1 := common.MustNewAddressFromString("hx1")
	addr2 := common.MustNewAddressFromString("hx2")
	addr3 := common.MustNewAddressFromString("hx3")
	addr4 := common.MustNewAddressFromString("hx4")
	prepInfo := map[string]*pRepEnable{
		string(addr1.Bytes()): {0, 0},
		string(addr2.Bytes()): {10, 0},
		string(addr3.Bytes()): {100, 200},
	}

	d1 := icstate.NewDelegation(addr1, big.NewInt(100))
	d2 := icstate.NewDelegation(addr2, big.NewInt(100))
	d3 := icstate.NewDelegation(addr3, big.NewInt(100))
	d4 := icstate.NewDelegation(addr4, big.NewInt(100))
	type args struct {
		multiplier int
		divider    int
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
				10,
				0,
				1000,
				icstate.Delegations{d1},
			},
			want: 100 * 100 * 1000 / 10,
		},
		{
			name: "PRep-enabled",
			args: args{
				100,
				10,
				0,
				1000,
				icstate.Delegations{d2},
			},
			want: 100 * 100 * (1000 - 10) / 10,
		},
		{
			name: "PRep-disabled",
			args: args{
				100,
				10,
				0,
				1000,
				icstate.Delegations{d3},
			},
			want: 100 * 100 * (200 - 100) / 10,
		},
		{
			name: "PRep-None",
			args: args{
				100,
				10,
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
				10,
				0,
				1000,
				icstate.Delegations{d1, d2, d3, d4},
			},
			want: (100*100*1000)/10 +
				(100*100*(1000-10))/10 +
				(100*100*(200-100))/10,
		},
	}

	calculator := NewCalculator()
	calculator.log = log.New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			reward := calculator.votingReward(
				big.NewInt(int64(args.multiplier)),
				big.NewInt(int64(args.divider)),
				args.from,
				args.to,
				prepInfo,
				args.delegating.Iterator(),
			)
			assert.Equal(t, tt.want, reward.Int64())
		})
	}
}
