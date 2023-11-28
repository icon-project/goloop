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

func TestRewardFund_NewSafeRewardFund(t *testing.T) {
	tests := []struct {
		name                               string
		iglobal                            int
		iprep, icps, irelay, ivoter, iwage int
		err                                bool
	}{
		{
			"success",
			1000000,
			4000, 1000, 2000, 3000, 3000,
			false,
		},
		{
			"invalid iglobal",
			-1,
			4000, 1000, 2000, 3000, 3000,
			true,
		},
		{
			"invalid iprep",
			1000000,
			-1, 5001, 2000, 3000, 3000,
			true,
		},
		{
			"invalid icps",
			1000000,
			6000, -1000, 2000, 3000, 3000,
			true,
		},
		{
			"invalid irelay",
			1000000,
			4000, 5000, -2000, 3000, 3000,
			true,
		},
		{
			"invalid ivoter/iwage",
			1000000,
			4000, 1000, 8000, -3000, -3000,
			true,
		},
		{
			"invalid sum",
			1000000,
			400, 100, 200, 300, 300,
			true,
		},
	}

	for _, ver := range []int{RFVersion1, RFVersion2} {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("Version%d-%s", ver, tt.name), func(t *testing.T) {
				var rf *RewardFund
				var err error
				if ver == RFVersion1 {
					rf, err = NewSafeRewardFundV1(
						big.NewInt(int64(tt.iglobal)),
						icmodule.Rate(tt.iprep),
						icmodule.Rate(tt.icps),
						icmodule.Rate(tt.irelay),
						icmodule.Rate(tt.ivoter),
					)
				} else {
					rf, err = NewSafeRewardFundV2(
						big.NewInt(int64(tt.iglobal)),
						icmodule.Rate(tt.iprep),
						icmodule.Rate(tt.iwage),
						icmodule.Rate(tt.icps),
						icmodule.Rate(tt.irelay),
					)
				}
				if tt.err {
					assert.Error(t, err)
				} else {
					assert.Equal(t, big.NewInt(int64(tt.iglobal)), rf.IGlobal())
					assert.Equal(t, icmodule.Rate(tt.iprep), rf.IPrep())
					assert.Equal(t, icmodule.Rate(tt.icps), rf.ICps())
					assert.Equal(t, icmodule.Rate(tt.irelay), rf.IRelay())
					if ver == RFVersion1 {
						assert.Equal(t, icmodule.Rate(tt.ivoter), rf.IVoter())
					} else {
						assert.Equal(t, icmodule.Rate(0), rf.IVoter())
					}
					if ver == RFVersion2 {
						assert.Equal(t, icmodule.Rate(tt.iwage), rf.Iwage())
					} else {
						assert.Equal(t, icmodule.Rate(0), rf.Iwage())
					}
				}
			})
		}
	}
}

func TestRewardFund_ToRewardFund2(t *testing.T) {
	iglobal := big.NewInt(100000)
	iprep := icmodule.Rate(5000)
	ivoter := icmodule.Rate(1000)
	icps := icmodule.Rate(3000)
	irelay := icmodule.Rate(1000)
	rf, err := NewSafeRewardFundV1(iglobal, iprep, icps, irelay, ivoter)
	assert.NoError(t, err)

	rf2 := rf.ToRewardFundV2()
	assert.Equal(t, RFVersion2, rf2.version)
	assert.Equal(t, iglobal, rf2.IGlobal())
	assert.Equal(t, iprep+ivoter, rf2.GetAllocationByKey(KeyIprep))
	assert.Equal(t, icps, rf2.GetAllocationByKey(KeyIcps))
	assert.Equal(t, irelay, rf2.GetAllocationByKey(KeyIrelay))
	assert.Equal(t, icmodule.Rate(0), rf2.GetAllocationByKey(KeyIwage))
}

func TestRFundKey(t *testing.T) {
	tests := []struct {
		key  RFundKey
		ver  int
		want bool
	}{
		{KeyIvoter, RFVersion1, true},
		{KeyIvoter, RFVersion2, false},
		{KeyIprep, RFVersion1, true},
		{KeyIprep, RFVersion2, true},
		{KeyIwage, RFVersion1, false},
		{KeyIwage, RFVersion2, true},
		{KeyIcps, RFVersion1, true},
		{KeyIcps, RFVersion2, true},
		{KeyIrelay, RFVersion1, true},
		{KeyIrelay, RFVersion2, true},
		{KeyIrelay, RFVersionReserved, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-RFVersion%d", tt.key, tt.ver), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.key.IsValid(tt.ver))
		})
	}
}

func TestRewardFund(t *testing.T) {
	iglobal := big.NewInt(100000)
	iprep := icmodule.Rate(5000)
	iwage := icmodule.Rate(3000)
	icps := icmodule.Rate(1000)
	irelay := icmodule.Rate(1000)

	rf := NewRewardFund(RFVersion2)
	assert.NotNil(t, rf.iGlobal)
	assert.NotNil(t, rf.allocation)

	rf, err := NewSafeRewardFundV2(iglobal, iprep, iwage, icps, irelay)
	assert.NoError(t, err)

	bs := rf.Bytes()

	rf2, err := NewRewardFundFromByte(bs)
	assert.NoError(t, err)

	assert.True(t, rf.Equal(rf2))

	assert.True(t, rf.Equal(rf2))
	assert.Equal(t, RFVersion2, rf2.version)
	assert.Equal(t, iglobal, rf2.IGlobal())
	assert.Equal(t, iprep.MulBigInt(iglobal), rf2.GetAmount(KeyIprep))
	assert.Equal(t, iwage.MulBigInt(iglobal), rf2.GetAmount(KeyIwage))
	assert.Equal(t, icps.MulBigInt(iglobal), rf2.GetAmount(KeyIcps))
	assert.Equal(t, irelay.MulBigInt(iglobal), rf2.GetAmount(KeyIrelay))
	assert.Equal(t, int64(0), rf2.GetAmount("invalid").Int64())
}

func TestRewardFundFromVersion1(t *testing.T) {
	iglobal := big.NewInt(100000)
	iprep := icmodule.ToRate(int64(50))
	icps := icmodule.ToRate(int64(10))
	irelay := icmodule.ToRate(int64(10))
	ivoter := icmodule.ToRate(int64(30))
	rf, err := NewSafeRewardFundV1(iglobal, iprep, icps, irelay, ivoter)
	assert.NoError(t, err)

	bs := rf.Bytes()

	rf2, err := NewRewardFundFromByte(bs)
	assert.NoError(t, err)

	assert.Equal(t, RFVersion1, rf2.version)
	assert.Equal(t, iglobal, rf2.IGlobal())
	assert.Equal(t, iprep, rf2.IPrep())
	assert.Equal(t, icps, rf2.ICps())
	assert.Equal(t, irelay, rf2.IRelay())
	assert.Equal(t, ivoter, rf2.IVoter())
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

func TestRewardFund_SetAllocation(t *testing.T) {
	tests := []struct {
		name       string
		version    int
		allocation map[RFundKey]icmodule.Rate
		err        bool
	}{
		{
			"success",
			RFVersion1,
			map[RFundKey]icmodule.Rate{
				KeyIprep:  icmodule.Rate(1000),
				KeyIcps:   icmodule.Rate(1000),
				KeyIrelay: icmodule.Rate(4000),
				KeyIvoter: icmodule.Rate(4000),
			},
			false,
		},

		{
			"invalid key",
			RFVersion1,
			map[RFundKey]icmodule.Rate{
				KeyIprep:  icmodule.Rate(1000),
				KeyIcps:   icmodule.Rate(1000),
				KeyIrelay: icmodule.Rate(4000),
				KeyIvoter: icmodule.Rate(4000),
				KeyIwage:  icmodule.Rate(0),
			},
			true,
		},
		{
			"invalid sum",
			RFVersion1,
			map[RFundKey]icmodule.Rate{
				KeyIprep:  icmodule.Rate(1000),
				KeyIcps:   icmodule.Rate(1000),
				KeyIrelay: icmodule.Rate(4000),
				KeyIvoter: icmodule.Rate(5000),
			},
			true,
		},
		{
			"success",
			RFVersion2,
			map[RFundKey]icmodule.Rate{
				KeyIprep:  icmodule.Rate(1000),
				KeyIcps:   icmodule.Rate(1000),
				KeyIrelay: icmodule.Rate(4000),
				KeyIwage:  icmodule.Rate(4000),
			},
			false,
		},
		{
			"invalid key",
			RFVersion2,
			map[RFundKey]icmodule.Rate{
				KeyIprep:  icmodule.Rate(1000),
				KeyIcps:   icmodule.Rate(1000),
				KeyIrelay: icmodule.Rate(4000),
				KeyIwage:  icmodule.Rate(4000),
				KeyIvoter: icmodule.Rate(0),
			},
			true,
		},
		{
			"invalid sum",
			RFVersion2,
			map[RFundKey]icmodule.Rate{
				KeyIprep:  icmodule.Rate(1000),
				KeyIcps:   icmodule.Rate(1000),
				KeyIrelay: icmodule.Rate(4000),
				KeyIwage:  icmodule.Rate(5000),
			},
			true,
		},
		{
			"success with less key",
			RFVersion1,
			map[RFundKey]icmodule.Rate{
				KeyIprep: icmodule.Rate(10000),
			},
			false,
		},
		{
			"success with less key",
			RFVersion2,
			map[RFundKey]icmodule.Rate{
				KeyIprep: icmodule.Rate(10000),
			},
			false,
		},
		{
			"invalid version",
			RFVersionReserved,
			map[RFundKey]icmodule.Rate{
				KeyIprep: icmodule.Rate(10000),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("RFVersion%d-%s", tt.version+1, tt.name), func(t *testing.T) {
			rf := NewRewardFund(tt.version)
			err := rf.SetAllocation(tt.allocation)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				for k, v := range tt.allocation {
					assert.Equal(t, v, rf.GetAllocationByKey(k))
				}
			}
		})
	}
}

func TestRewardFund_Format(t *testing.T) {
	iglobal := big.NewInt(100_000)
	icps := icmodule.ToRate(10)
	iprep := icmodule.ToRate(70)
	irelay := icmodule.ToRate(0)
	iwage := icmodule.ToRate(20)

	rf, err := NewSafeRewardFundV2(iglobal, iprep, iwage, icps, irelay)
	assert.NoError(t, err)

	exp := fmt.Sprintf("{%d %d %d %d %d %d}", RFVersion2, iglobal, icps, iprep, irelay, iwage)
	ret := fmt.Sprintf("%v", rf)
	assert.Equal(t, exp, ret)

	exp = fmt.Sprintf(
		"RewardFund{version=%d %s=%d %s=%d %s=%d %s=%d %s=%d}",
		RFVersion2, KeyIglobal, iglobal, KeyIcps, icps, KeyIprep, iprep, KeyIrelay, irelay, KeyIwage, iwage)
	assert.Equal(t, exp, fmt.Sprintf("%+v", rf))
	assert.Equal(t, exp, fmt.Sprintf("%s", rf))

	rf = &RewardFund{
		RFVersion2,
		iglobal,
		map[RFundKey]icmodule.Rate{
			KeyIprep: iprep,
			KeyIcps:  icps,
			KeyIwage: iwage,
		},
	}

	exp = fmt.Sprintf("{%d %d %d %d %d}", RFVersion2, iglobal, icps, iprep, iwage)
	ret = fmt.Sprintf("%v", rf)
	assert.Equal(t, exp, ret)

	exp = fmt.Sprintf(
		"RewardFund{version=%d %s=%d %s=%d %s=%d %s=%d}",
		RFVersion2, KeyIglobal, iglobal, KeyIcps, icps, KeyIprep, iprep, KeyIwage, iwage)
	assert.Equal(t, exp, fmt.Sprintf("%+v", rf))
	assert.Equal(t, exp, fmt.Sprintf("%s", rf))
}

func TestRewardFund_GetOrderAllocationKeys(t *testing.T) {

	tests := []struct {
		allocation map[RFundKey]icmodule.Rate
		expect     []RFundKey
	}{
		{
			map[RFundKey]icmodule.Rate{
				KeyIprep:  icmodule.Rate(1000),
				KeyIcps:   icmodule.Rate(1000),
				KeyIrelay: icmodule.Rate(4000),
				KeyIvoter: icmodule.Rate(4000),
			},
			[]RFundKey{KeyIcps, KeyIprep, KeyIrelay, KeyIvoter},
		},
		{
			map[RFundKey]icmodule.Rate{
				KeyIprep:  icmodule.Rate(1000),
				KeyIcps:   icmodule.Rate(1000),
				KeyIrelay: icmodule.Rate(4000),
				KeyIwage:  icmodule.Rate(4000),
			},
			[]RFundKey{KeyIcps, KeyIprep, KeyIrelay, KeyIwage},
		},
		{
			map[RFundKey]icmodule.Rate{
				KeyIprep: icmodule.Rate(1000),
				KeyIcps:  icmodule.Rate(1000),
				KeyIwage: icmodule.Rate(4000),
			},
			[]RFundKey{KeyIcps, KeyIprep, KeyIwage},
		},
		{
			map[RFundKey]icmodule.Rate{
				KeyIprep:  icmodule.Rate(1000),
				KeyIcps:   icmodule.Rate(1000),
				KeyIrelay: icmodule.Rate(4000),
				KeyIwage:  icmodule.Rate(4000),
				KeyIvoter: icmodule.Rate(4000),
			},
			[]RFundKey{KeyIcps, KeyIprep, KeyIrelay, KeyIvoter, KeyIwage},
		},
		{
			map[RFundKey]icmodule.Rate{
				"newKey":  icmodule.Rate(1000),
				KeyIprep:  icmodule.Rate(1000),
				KeyIcps:   icmodule.Rate(1000),
				KeyIrelay: icmodule.Rate(4000),
				KeyIwage:  icmodule.Rate(4000),
				KeyIvoter: icmodule.Rate(4000),
			},
			[]RFundKey{KeyIcps, KeyIprep, KeyIrelay, KeyIvoter, KeyIwage, "newKey"},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			rf := &RewardFund{
				RFVersion2,
				big.NewInt(10),
				tt.allocation,
			}

			assert.Equal(t, tt.expect, rf.GetOrderAllocationKeys())
		})
	}
}
