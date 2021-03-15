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
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
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

	prep2, err := parseIssuePRepData(bs)
	assert.NoError(t, err)

	assert.True(t, prep1.equal(prep2))

	assert.Equal(t, 0, prep1.IRep.Cmp(prep2.IRep.Value()))
	assert.Equal(t, 0, prep1.RRep.Cmp(prep2.RRep.Value()))
	assert.Equal(t, 0, prep1.TotalDelegation.Cmp(prep2.TotalDelegation.Value()))
	assert.Equal(t, 0, prep1.Value.Cmp(prep2.Value.Value()))

	prep3, err := parseIssuePRepData(nil)
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

	result2, err := parseIssueResultData(bs)
	assert.NoError(t, err)

	assert.True(t, result1.equal(result2))

	assert.Equal(t, 0, result1.ByFee.Cmp(result2.ByFee.Value()))
	assert.Equal(t, 0, result1.ByOverIssuedICX.Cmp(result2.ByOverIssuedICX.Value()))
	assert.Equal(t, 0, result1.Issue.Cmp(result2.Issue.Value()))

	result3, err := parseIssueResultData(nil)
	assert.NoError(t, err)
	assert.Nil(t, result3)
}

func setIssue(issue *icstate.Issue, totalIssued int64, prevTotalIssued int64, overIssued int64, iScoreRemains int64, prevBlockFee int64) {
	issue.TotalIssued.SetInt64(totalIssued)
	issue.PrevTotalIssued.SetInt64(prevTotalIssued)
	issue.OverIssued.SetInt64(overIssued)
	issue.IScoreRemains.SetInt64(iScoreRemains)
	issue.PrevBlockFee.SetInt64(prevBlockFee)
}

func TestIssuer_RegulateIssueInfo(t *testing.T) {
	type values struct {
		prevtotalIssued int64
		totalIssued     int64
		overIssued      int64
		iScoreRemains   int64
		prevBlockFee    int64
	}

	tests := []struct {
		name             string
		in               values
		iScore           *big.Int
		additionalReward *big.Int
		err              bool
		out              values
	}{
		{
			"Nill iScore reward",
			values{
				0, 100, 0, 0, 0,
			},
			nil,
			new(big.Int).SetInt64(0),
			false,
			values{
				0, 0, 0, 0, 0,
			},
		},
		{
			"Zero iScore reward",
			values{
				0, 100, 0, 0, 0,
			},
			new(big.Int).SetInt64(0),
			new(big.Int).SetInt64(0),
			false,
			values{
				0, 0, 0, 0, 0,
			},
		},
		{
			"No overIssue",
			values{
				100, 200, 0, 100, 0,
			},
			new(big.Int).SetInt64(100 * IScoreICXRatio),
			new(big.Int).SetInt64(0),
			false,
			values{
				0, 0, 0, 100, 0,
			},
		},
		{
			"No overIssue with additionalReward",
			values{
				100, 200, 0, 100, 0,
			},
			new(big.Int).SetInt64(50 * IScoreICXRatio),
			new(big.Int).SetInt64(50),
			false,
			values{
				0, 0, 0, 100, 0,
			},
		},
		{
			"Positive overIssue",
			values{
				100, 200, 10, 1, 0,
			},
			new(big.Int).SetInt64(90*IScoreICXRatio + 123),
			new(big.Int).SetInt64(0),
			false,
			values{
				0, 0, 20, 124, 0,
			},
		},
		{
			"Positive overIssue with additionalReward",
			values{
				100, 200, 10, 1, 0,
			},
			new(big.Int).SetInt64(50*IScoreICXRatio + 123),
			new(big.Int).SetInt64(40),
			false,
			values{
				0, 0, 20, 124, 0,
			},
		},
		{
			"Negative overIssue",
			values{
				100, 200, 10, 1, 0,
			},
			new(big.Int).SetInt64(200*IScoreICXRatio + 123),
			new(big.Int).SetInt64(0),
			true,
			values{},
		},
		{
			"Negative overIssue with additionReward",
			values{
				100, 200, 10, 1, 0,
			},
			new(big.Int).SetInt64(50*IScoreICXRatio + 123),
			new(big.Int).SetInt64(150),
			true,
			values{},
		},
	}

	var err error
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			out := tt.out
			issue := icstate.NewIssue()
			setIssue(issue, in.totalIssued, in.prevtotalIssued, in.overIssued, in.iScoreRemains, in.prevBlockFee)
			err = RegulateIssueInfo(issue, tt.iScore, tt.additionalReward)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, out.overIssued, issue.OverIssued.Int64())
				assert.Equal(t, out.iScoreRemains, issue.IScoreRemains.Int64())
				assert.Equal(t, out.prevBlockFee, issue.PrevBlockFee.Int64())
			}
		})
	}
}

func TestIssuer_calcRewardPerBlock(t *testing.T) {
	type values struct {
		irep           int64
		rrep           int64
		mainPRepCount  int64
		totalDelegated int64
	}

	tests := []struct {
		name string
		in   values
		want int64
	}{
		{
			"No reward",
			values{
				0, 0, 0, 0,
			},
			0,
		},
		{
			"Prevote - voting only",
			values{
				100 * MonthBlock,
				1000,
				0,
				100 * YearBlock,
			},
			(100 * MonthBlock) / (MonthBlock * 2) * 100 +
				RrepMultiplier * 1000 * 100 / RrepDivider,
		},
		{
			"Prevote - too small delegation",
			values{
				100 * MonthBlock,
				1000,
				0,
				100,
			},
			(100 * MonthBlock) / (MonthBlock * 2) * 100 + 0,
		},
		{
			"Decentralized",
			values{
				100 * MonthBlock,
				1000,
				22,
				100 * YearBlock,
			},
			(100 * MonthBlock) / (MonthBlock * 2) * 22 +
				(100 * MonthBlock) / (MonthBlock * 2) * 100 +
				RrepMultiplier * 1000 * 100 / RrepDivider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			out := calcRewardPerBlock(
				big.NewInt(in.irep),
				big.NewInt(in.rrep),
				big.NewInt(in.mainPRepCount),
				big.NewInt(in.totalDelegated),
			)

			assert.Equal(t, tt.want, out.Int64())
		})
	}
}

func TestIssuer_calcIssueAmount(t *testing.T) {
	type values struct {
		prevTotalIssued int64
		totalIssued     int64
		overIssued      int64
		iScoreRemains   int64
		prevBlockFee    int64
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
				0, 0, 0, 0, 0,
			},
			100,
			wants{
				0, 0, 100,
			},
		},
		{
			"OverIssued",
			values{
				0, 0, 10, 0, 0,
			},
			100,
			wants{
				0, 10, 100 - 10,
			},
		},
		{
			"OverIssued-larger than reward",
			values{
				0, 0, 300, 0, 0,
			},
			100,
			wants{
				0, 100, 0,
			},
		},
		{
			"Fee",
			values{
				0, 0, 000, 0, 10,
			},
			100,
			wants{
				10, 0, 90,
			},
		},
		{
			"Fee-larger than reward",
			values{
				0, 0, 0, 0, 200,
			},
			100,
			wants{
				100, 0, 0,
			},
		},
		{
			"OverIssued and fee",
			values{
				0, 0, 10, 0, 20,
			},
			100,
			wants{
				20, 10, 100 - 10 - 20,
			},
		},
		{
			"OverIssued and fee - larger than reward (overIssued has priority",
			values{
				0, 0, 300, 0, 200,
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
			setIssue(issue, in.totalIssued, in.prevTotalIssued, in.overIssued, in.iScoreRemains, in.prevBlockFee)
			issued, byOverIssued, byFee := calcIssueAmount(new(big.Int).SetInt64(tt.reward), issue)

			assert.Equal(t, out.issue, issued.Int64())
			assert.Equal(t, out.byOverIssued, byOverIssued.Int64())
			assert.Equal(t, out.byFee, byFee.Int64())
		})
	}
}
