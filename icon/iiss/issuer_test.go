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
}

func setIssue(issue *icstate.Issue, totalIssued int64, prevTotalIssued int64, overIssued int64, iScoreRemains int64, prevBlockFee int64) {
	issue.TotalReward.SetInt64(totalIssued)
	issue.PrevTotalReward.SetInt64(prevTotalIssued)
	issue.OverIssued.SetInt64(overIssued)
	issue.IScoreRemains.SetInt64(iScoreRemains)
	issue.PrevBlockFee.SetInt64(prevBlockFee)
}

func TestIssuer_regulateIssueInfo(t *testing.T) {
	type values struct {
		prevtotalIssued int64
		totalIssued     int64
		overIssued      int64
		iScoreRemains   int64
		prevBlockFee    int64
	}

	tests := []struct {
		name   string
		in     values
		iScore int64
		out    values
	}{
		{
			"Zero iScore reward",
			values{
				0, 100, 0, 0, 0,
			},
			0,
			values{
				100, 0, 0, 0, 0,
			},
		},
		{
			"No overIssue",
			values{
				100, 200, 0, 100, 0,
			},
			100 * IScoreICXRatio,
			values{
				200, 0, 0, 100, 0,
			},
		},
		{
			"Positive overIssue",
			values{
				100, 200, 10, 1, 0,
			},
			90*IScoreICXRatio + 123,
			values{
				200, 0, 20, 124, 0,
			},
		},
		{
			"Negative overIssue",
			values{
				100, 200, 10, 1, 0,
			},
			200*IScoreICXRatio + 123,
			values{
				200, 0, -90,  124, 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			out := tt.out
			issue := icstate.NewIssue()
			setIssue(issue, in.totalIssued, in.prevtotalIssued, in.overIssued, in.iScoreRemains, in.prevBlockFee)
			regulateIssueInfo(issue, new(big.Int).SetInt64(tt.iScore))

			assert.Equal(t, out.totalIssued, issue.TotalReward.Int64())
			assert.Equal(t, out.prevtotalIssued, issue.PrevTotalReward.Int64())
			assert.Equal(t, out.overIssued, issue.OverIssued.Int64())
			assert.Equal(t, out.iScoreRemains, issue.IScoreRemains.Int64())
			assert.Equal(t, out.prevBlockFee, issue.PrevBlockFee.Int64())
		})
	}
}

func TestIssuer_calcRewardPerBlock(t *testing.T) {
	type values struct {
		irep           int64
		rrep           int64
		mainPRepCount  int64
		pRepCount  int64
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
				0, 0, 0, 0, 0,
			},
			0,
		},
		{
			"Prevote - beta3 only",
			values{
				100 * MonthBlock,
				10,
				0,
				0,
				100 * YearBlock,
			},
			10 * 100,
		},
		{
			"Prevote - too small delegation",
			values{
				100 * MonthBlock,
				10,
				0,
				0,
				100,
			},
			0,
		},
		{
			"Decentralized",
			values{
				100 * MonthBlock,
				10,
				22,
				100,
				100 * YearBlock,
			},
			100*22/2 + 100*100/2 + 10*100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			out := calcRewardPerBlock(
				big.NewInt(in.irep),
				big.NewInt(in.rrep),
				big.NewInt(in.mainPRepCount),
				big.NewInt(in.pRepCount),
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
		overIssued int64
		issue      int64
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
				0, 100,
			},
		},
		{
			"OverIssued",
			values{
				0, 0, 10, 0, 0,
			},
			100,
			wants{
				10, 100 - 10,
			},
		},
		{
			"OverIssued-larger than reward",
			values{
				0, 0, 300, 0, 0,
			},
			100,
			wants{
				100, 0,
			},
		},
		{
			"OverIssued and fee",
			values{
				0, 0, 10, 0, 20,
			},
			100,
			wants{
				10, 100 - 10 - 20,
			},
		},
		{
			"OverIssued and fee -larger than reward",
			values{
				0, 0, 300, 0, 20,
			},
			100,
			wants{
				80, 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			out := tt.out
			issue := icstate.NewIssue()
			setIssue(issue, in.totalIssued, in.prevTotalIssued, in.overIssued, in.iScoreRemains, in.prevBlockFee)
			overIssued, issued := calcIssueAmount(new(big.Int).SetInt64(tt.reward), issue)

			assert.Equal(t, out.issue, issued.Int64())
			assert.Equal(t, out.overIssued, overIssued.Int64())
		})
	}
}
