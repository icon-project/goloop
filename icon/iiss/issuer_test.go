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
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
)

func TestIssuer_IssuePRepJSON(t *testing.T) {
	prep1 := &IssuePRepJSON{
		IRep:            common.NewHexInt(10),
		RRep:            common.NewHexInt(20),
		TotalDelegation: common.NewHexInt(1000),
		Value:           common.NewHexInt(900),
	}
	bs, err := json.Marshal(prep1)
	assert.NoError(t, err)

	prep2, err := ParseIssuePRepData(bs)
	assert.NoError(t, err)

	assert.True(t, prep1.Equal(prep2))

	assert.Equal(t, 0, prep1.GetIRep().Cmp(prep2.GetIRep()))
	assert.Equal(t, 0, prep1.GetRRep().Cmp(prep2.GetRRep()))
	assert.Equal(t, 0, prep1.GetTotalDelegation().Cmp(prep2.GetTotalDelegation()))
	assert.Equal(t, 0, prep1.GetValue().Cmp(prep2.GetValue()))

	prep3, err := ParseIssuePRepData(nil)
	assert.NoError(t, err)
	assert.Nil(t, prep3)
}

func TestIssuer_IssueResultJSON(t *testing.T) {
	result1 := &IssueResultJSON{
		ByFee:           common.NewHexInt(10),
		ByOverIssuedICX: common.NewHexInt(20),
		Issue:           common.NewHexInt(1000),
	}
	bs, err := json.Marshal(result1)
	assert.NoError(t, err)

	result2, err := ParseIssueResultData(bs)
	assert.NoError(t, err)

	assert.True(t, result1.Equal(result2))

	assert.Equal(t, 0, result1.GetByFee().Cmp(result2.GetByFee()))
	assert.Equal(t, 0, result1.GetByOverIssuedICX().Cmp(result2.GetByOverIssuedICX()))
	assert.Equal(t, 0, result1.GetIssue().Cmp(result2.GetIssue()))

	result3, err := ParseIssueResultData(nil)
	assert.NoError(t, err)
	assert.Nil(t, result3)
}

func setIssue(issue *icstate.Issue, totalIssued int64, prevTotalIssued int64, overIssued int64, prevBlockFee int64) {
	issue.SetTotalReward(big.NewInt(totalIssued))
	issue.SetPrevTotalReward(big.NewInt(prevTotalIssued))
	issue.SetOverIssuedIScore(big.NewInt(overIssued))
	issue.SetPrevBlockFee(big.NewInt(prevBlockFee))
}

func TestIssuer_RegulateIssueInfo(t *testing.T) {
	type values struct {
		prevTotalIssued  int64
		totalIssued      int64
		overIssuedIScore int64
		prevBlockFee     int64
	}

	tests := []struct {
		name   string
		in     values
		reward *big.Int
		out    values
	}{
		{
			"Nil iScore reward",
			values{
				0, 100, 0, 0,
			},
			nil,
			values{
				0, 0, 0, 0,
			},
		},
		{
			"Zero iScore reward",
			values{
				0, 100, 0, 0,
			},
			new(big.Int).SetInt64(0),
			values{
				0, 0, 0, 0,
			},
		},
		{
			"No overIssue",
			values{
				100, 200, 100, 0,
			},
			new(big.Int).SetInt64(100 * icmodule.IScoreICXRatio),
			values{
				0, 0, 100, 0,
			},
		},
		{
			"Positive overIssue",
			values{
				100, 200, 10001, 0,
			},
			new(big.Int).SetInt64(90*icmodule.IScoreICXRatio + 123),
			values{
				0, 0, 100*icmodule.IScoreICXRatio + 10001 - (90*icmodule.IScoreICXRatio + 123), 0,
			},
		},
		{
			"Negative overIssue",
			values{
				100, 200, 1, 0,
			},
			new(big.Int).SetInt64(100*icmodule.IScoreICXRatio + 123),
			values{
				100, 200, 1 - 123, 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			out := tt.out
			issue := icstate.NewIssue()
			setIssue(issue, in.totalIssued, in.prevTotalIssued, in.overIssuedIScore, in.prevBlockFee)
			RegulateIssueInfo(issue, tt.reward)
			assert.Equal(t, out.overIssuedIScore, issue.OverIssuedIScore().Int64())
			assert.Equal(t, out.prevBlockFee, issue.PrevBlockFee().Int64())
		})
	}
}

func TestIssuer_calcRewardPerBlock(t *testing.T) {
	type values struct {
		irep           *big.Int
		rrep           *big.Int
		mainPRepCount  *big.Int
		totalDelegated *big.Int
	}

	tests := []struct {
		name string
		in   values
		want *big.Int
	}{
		{
			"No reward",
			values{
				new(big.Int),
				new(big.Int),
				new(big.Int),
				new(big.Int),
			},
			new(big.Int),
		},
		{
			"Prevote - voting only",
			values{
				new(big.Int).SetInt64(100 * icmodule.MonthBlock),
				new(big.Int).SetInt64(1000),
				new(big.Int),
				new(big.Int).SetInt64(100 * icmodule.YearBlock),
			},
			new(big.Int).SetInt64(
				(100*icmodule.MonthBlock)/(icmodule.MonthBlock*2)*100 +
					icmodule.RrepMultiplier*1000*100/icmodule.RrepDivider,
			),
		},
		{
			"Prevote - too small delegation",
			values{
				new(big.Int).SetInt64(100 * icmodule.MonthBlock),
				new(big.Int).SetInt64(1000),
				new(big.Int),
				new(big.Int).SetInt64(100),
			},
			new(big.Int).SetInt64((100*icmodule.MonthBlock)/(icmodule.MonthBlock*2)*100 + 0),
		},
		{
			"Decentralized",
			values{
				new(big.Int).SetInt64(100 * icmodule.MonthBlock),
				new(big.Int).SetInt64(1000),
				new(big.Int).SetInt64(22),
				new(big.Int).SetInt64(100 * icmodule.YearBlock),
			},
			new(big.Int).SetInt64(
				(100*icmodule.MonthBlock)/(icmodule.MonthBlock*2)*22 +
					(100*icmodule.MonthBlock)/(icmodule.MonthBlock*2)*100 +
					icmodule.RrepMultiplier*1000*100/icmodule.RrepDivider,
			),
		},
		{
			"MainNet-10,362,083-Decentralized",
			values{
				icmodule.BigIntInitialIRep,
				new(big.Int).SetInt64(0x2ac),
				new(big.Int).SetInt64(22),
				new(big.Int).Add(
					new(big.Int).Mul(new(big.Int).SetInt64(170075049), icutils.BigIntICX),
					new(big.Int).SetInt64(583626807627704134),
				),
			},
			new(big.Int).SetInt64(0x3fcd641964f21cea),
		},
		{
			"MainNet-10,405,202",
			values{
				icmodule.BigIntInitialIRep,
				new(big.Int).SetInt64(0x2ac),
				new(big.Int).SetInt64(22),
				new(big.Int).Add(
					new(big.Int).Mul(new(big.Int).SetInt64(170774443), icutils.BigIntICX),
					new(big.Int).SetInt64(514041607082338118),
				),
			},
			new(big.Int).SetInt64(0x3fee2d05334c7b8d),
		},
		{
			"MainNet-27,523,843-NP-setIRep",
			values{
				new(big.Int).Mul(new(big.Int).SetInt64(10_000), icutils.BigIntICX),
				new(big.Int).SetInt64(0x19d),
				new(big.Int).SetInt64(22),
				new(big.Int).Add(
					new(big.Int).Mul(new(big.Int).SetInt64(326594583), icutils.BigIntICX),
					new(big.Int).SetInt64(659661834421744157),
				),
			},
			new(big.Int).SetInt64(0x2aa4110d9a9c3a7a),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			out := calcRewardPerBlock(
				in.irep,
				in.rrep,
				in.mainPRepCount,
				in.totalDelegated,
			)

			assert.Equal(t, 0, tt.want.Cmp(out))
		})
	}
}

func TestIssuer_calcIssueAmount(t *testing.T) {
	type values struct {
		prevTotalIssued  int64
		totalIssued      int64
		overIssuedIScore int64
		prevBlockFee     int64
	}
	type wants struct {
		byFee        int64
		byOverIssued int64
		issue        int64
	}

	tests := []struct {
		name   string
		in     values
		reward int64
		out    wants
	}{
		{
			"First",
			values{
				0, 0, 0, 0,
			},
			100,
			wants{
				0, 0, 100,
			},
		},
		{
			"OverIssued",
			values{
				0, 0, 10 * icmodule.IScoreICXRatio, 0,
			},
			100,
			wants{
				0, 10, 100 - 10,
			},
		},
		{
			"OverIssued-larger than reward",
			values{
				0, 0, 300 * icmodule.IScoreICXRatio, 0,
			},
			100,
			wants{
				0, 100, 0,
			},
		},
		{
			"Fee",
			values{
				0, 0, 0, 10,
			},
			100,
			wants{
				10, 0, 90,
			},
		},
		{
			"Fee-larger than reward",
			values{
				0, 0, 0, 200,
			},
			100,
			wants{
				100, 0, 0,
			},
		},
		{
			"OverIssued and fee",
			values{
				0, 0, 10 * icmodule.IScoreICXRatio, 20,
			},
			100,
			wants{
				20, 10, 100 - 10 - 20,
			},
		},
		{
			"OverIssued and fee - larger than reward (overIssuedIScore has priority",
			values{
				0, 0, 300 * icmodule.IScoreICXRatio, 200,
			},
			100,
			wants{
				0, 100, 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			out := tt.out
			issue := icstate.NewIssue()
			setIssue(issue, in.totalIssued, in.prevTotalIssued, in.overIssuedIScore, in.prevBlockFee)
			issued, byOverIssued, byFee := calcIssueAmount(new(big.Int).SetInt64(tt.reward), issue)

			assert.Equal(t, out.issue, issued.Int64())
			assert.Equal(t, out.byOverIssued, byOverIssued.Int64())
			assert.Equal(t, out.byFee, byFee.Int64())
		})
	}
}
