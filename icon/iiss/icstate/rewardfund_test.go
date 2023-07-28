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

package icstate

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/icmodule"
)

func TestRewardFund(t *testing.T) {
	iglobal := int64(100000)
	iprep := int64(50)
	ivoter := int64(50)
	rf := NewRewardFund()
	rf.Iglobal = big.NewInt(iglobal)
	rf.Iprep = icmodule.ToRate(iprep)
	rf.Ivoter = icmodule.ToRate(ivoter)

	bs := rf.Bytes()

	rf2, err := newRewardFundFromByte(bs)
	assert.NoError(t, err)

	assert.True(t, rf.Equal(rf2))
	assert.Equal(t, 0, rf.Iglobal.Cmp(rf2.Iglobal))

	rf3 := rf.Clone()
	assert.True(t, rf.Equal(rf3))

	assert.Equal(t, iglobal*iprep/100, rf.GetPRepFund().Int64())
	assert.Equal(t, iglobal*ivoter/100, rf.GetVoterFund().Int64())
}

func TestRewardFund_NewSafeRewardFund(t *testing.T) {
	tests := []struct {
		name                        string
		iglobal                     int
		iprep, icps, irelay, ivoter int
		err                         bool
	}{
		{
			"success",
			1000000,
			4000, 1000, 2000, 3000,
			false,
		},
		{
			"invalid iglobal",
			-1,
			4000, 1000, 2000, 3000,
			true,
		},
		{
			"invalid iprep",
			1000000,
			-1, 5001, 2000, 3000,
			true,
		},
		{
			"invalid icps",
			1000000,
			6000, -1000, 2000, 3000,
			true,
		},
		{
			"invalid irelay",
			1000000,
			4000, 5000, -2000, 3000,
			true,
		},
		{
			"invalid ivoter",
			1000000,
			4000, 1000, 8000, -3000,
			true,
		},
		{
			"invalid sum",
			1000000,
			400, 100, 200, 300,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf, err := NewSafeRewardFund(
				big.NewInt(int64(tt.iglobal)),
				icmodule.Rate(tt.iprep),
				icmodule.Rate(tt.icps),
				icmodule.Rate(tt.irelay),
				icmodule.Rate(tt.ivoter),
			)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.Equal(t, big.NewInt(int64(tt.iglobal)), rf.Iglobal)
				assert.Equal(t, icmodule.Rate(tt.iprep), rf.Iprep)
				assert.Equal(t, icmodule.Rate(tt.icps), rf.Icps)
				assert.Equal(t, icmodule.Rate(tt.irelay), rf.Irelay)
				assert.Equal(t, icmodule.Rate(tt.ivoter), rf.Ivoter)
			}
		})
	}
}

func TestRewardFund_ToRewardFund2(t *testing.T) {
	iglobal := big.NewInt(100000)
	iprep := icmodule.Rate(5000)
	ivoter := icmodule.Rate(1000)
	icps := icmodule.Rate(3000)
	irelay := icmodule.Rate(1000)
	rf, err := NewSafeRewardFund(iglobal, iprep, icps, irelay, ivoter)
	assert.NoError(t, err)

	rf2 := rf.ToRewardFund2()
	assert.Equal(t, iglobal, rf2.IGlobal())
	assert.Equal(t, iprep+ivoter, rf2.GetAllocationByKey(KeyIprep))
	assert.Equal(t, icps, rf2.GetAllocationByKey(KeyIcps))
	assert.Equal(t, irelay, rf2.GetAllocationByKey(KeyIrelay))
	assert.Equal(t, icmodule.Rate(0), rf2.GetAllocationByKey(KeyIwage))
}

func TestRFundKey(t *testing.T) {
	tests := []struct {
		name string
		in   RFundKey
		want bool
	}{
		{
			"invalid",
			RFundKey("invalid"),
			false,
		},
		{
			"KeyIprep",
			KeyIprep,
			true,
		},
		{
			"KeyIwage",
			KeyIwage,
			true,
		},
		{
			"KeyIcps",
			KeyIcps,
			true,
		},
		{
			"KeyIrelay",
			KeyIrelay,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.in.IsValid())
		})
	}
}

func TestRewardFund2(t *testing.T) {
	iglobal := int64(100000)
	iprep := int64(50)
	iwage := int64(30)
	icps := int64(10)
	irelay := int64(10)
	rf := NewRewardFund2()
	assert.NotNil(t, rf.iGlobal)
	assert.NotNil(t, rf.allocation)
	rf.SetIGlobal(big.NewInt(iglobal))
	rf.SetAllocationByKey(KeyIprep, icmodule.ToRate(iprep))
	rf.SetAllocationByKey(KeyIwage, icmodule.ToRate(iwage))
	rf.SetAllocationByKey(KeyIcps, icmodule.ToRate(icps))
	rf.SetAllocationByKey(KeyIrelay, icmodule.ToRate(irelay))

	bs := rf.Bytes()

	rf2, err := newRewardFund2FromByte(bs)
	assert.NoError(t, err)

	assert.True(t, rf.Equal(rf2))

	assert.Equal(t, big.NewInt(iglobal*iprep/100), rf.GetAmount(KeyIprep))
	assert.Equal(t, big.NewInt(iglobal*iwage/100), rf.GetAmount(KeyIwage))
	assert.Equal(t, big.NewInt(iglobal*icps/100), rf.GetAmount(KeyIcps))
	assert.Equal(t, big.NewInt(iglobal*irelay/100), rf.GetAmount(KeyIrelay))
	assert.Equal(t, int64(0), rf.GetAmount("invalid").Int64())
}

func TestNewRewardFund2Allocation(t *testing.T) {
	tests := []struct {
		name  string
		param []interface{}
		err   bool
		len   int
	}{
		{"Nil param", nil, true, 0},
		{"Empty param", []interface{}{}, true, 0},
		{
			"Invalid name value",
			[]interface{}{
				map[string]interface{}{
					"name":  "invalid",
					"value": fmt.Sprintf("%#x", 10000),
				},
			},
			true,
			0,
		},
		{
			"Invalid value",
			[]interface{}{
				map[string]interface{}{
					"name":  KeyIprep,
					"value": fmt.Sprintf("%#x", 10000),
				},
				map[string]interface{}{
					"name":  KeyIcps,
					"value": fmt.Sprintf("%#x", 10000),
				},
				map[string]interface{}{
					"name":  KeyIrelay,
					"value": fmt.Sprintf("%#x", -10000),
				},
			},
			true,
			0,
		},
		{
			"Invalid key",
			[]interface{}{
				map[string]interface{}{
					"name":  KeyIprep,
					"value": fmt.Sprintf("%#x", 5000),
				},
				map[string]interface{}{
					"invalid_name":  KeyIcps,
					"invalid_value": fmt.Sprintf("%#x", 5000),
				},
			},
			true,
			0,
		},
		{
			"Duplicated name",
			[]interface{}{
				map[string]interface{}{
					"name":  KeyIprep,
					"value": fmt.Sprintf("%#x", 5000),
				},
				map[string]interface{}{
					"name":  KeyIprep,
					"value": fmt.Sprintf("%#x", 5000),
				},
			},
			true,
			0,
		},
		{
			"Sum is not 10000",
			[]interface{}{
				map[string]interface{}{
					"name":  KeyIprep,
					"value": fmt.Sprintf("%#x", 5000),
				},
				map[string]interface{}{
					"name":  KeyIcps,
					"value": fmt.Sprintf("%#x", 1000),
				},
			},
			true,
			0,
		},
		{
			"Success 4 elements",
			[]interface{}{
				map[string]interface{}{
					"name":  KeyIprep,
					"value": fmt.Sprintf("%#x", 5000),
				},
				map[string]interface{}{
					"name":  KeyIwage,
					"value": fmt.Sprintf("%#x", 3000),
				},
				map[string]interface{}{
					"name":  KeyIcps,
					"value": fmt.Sprintf("%#x", 1000),
				},
				map[string]interface{}{
					"name":  KeyIrelay,
					"value": fmt.Sprintf("%#x", 1000),
				},
			},
			false,
			4,
		},
		{
			"Success 4 elements with zero value",
			[]interface{}{
				map[string]interface{}{
					"name":  KeyIprep,
					"value": fmt.Sprintf("%#x", 5000),
				},
				map[string]interface{}{
					"name":  KeyIwage,
					"value": fmt.Sprintf("%#x", 3000),
				},
				map[string]interface{}{
					"name":  KeyIcps,
					"value": fmt.Sprintf("%#x", 2000),
				},
				map[string]interface{}{
					"name":  KeyIrelay,
					"value": fmt.Sprintf("%#x", 0),
				},
			},
			false,
			4,
		},
		{
			"Success 2 elements",
			[]interface{}{
				map[string]interface{}{
					"name":  KeyIprep,
					"value": fmt.Sprintf("%#x", 5000),
				},
				map[string]interface{}{
					"name":  KeyIcps,
					"value": fmt.Sprintf("%#x", 5000),
				},
			},
			false,
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allocation, err := NewRewardFund2Allocation(tt.param)
			if tt.err {
				assert.Error(t, err, "NewRewardFund2Allocation() was not failed for %v.", tt.param)
			} else {
				assert.NoError(t, err, "NewRewardFund2Allocation() was failed for %v. err=%v", tt.param, err)
				assert.Equal(t, tt.len, len(allocation))
			}
		})
	}
}

func TestRewardFund2_Format(t *testing.T) {
	iglobal := int64(100_000)
	icps := icmodule.ToRate(10)
	iprep := icmodule.ToRate(70)
	irelay := icmodule.ToRate(0)
	iwage := icmodule.ToRate(20)

	rf := NewRewardFund2()
	rf.SetIGlobal(big.NewInt(iglobal))
	rf.SetAllocationByKey(KeyIprep, iprep)
	rf.SetAllocationByKey(KeyIcps, icps)
	rf.SetAllocationByKey(KeyIrelay, irelay)
	rf.SetAllocationByKey(KeyIwage, iwage)

	exp := fmt.Sprintf("{%d %d %d %d %d}", iglobal, iprep, iwage, icps, irelay)
	ret := fmt.Sprintf("%v", rf)
	assert.Equal(t, exp, ret)

	exp = fmt.Sprintf(
		"RewardFund2{iGlobal=%d iprep=%d iwage=%d icps=%d irelay=%d}",
		iglobal, iprep, iwage, icps, irelay)
	assert.Equal(t, exp, fmt.Sprintf("%+v", rf))
	assert.Equal(t, exp, fmt.Sprintf("%s", rf))
}