/*
 * Copyright 2021 ICON Foundation
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
	"encoding/json"
	"fmt"
	"math/big"
	"unicode"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/service/scoreresult"
)

type RewardFund struct {
	Iglobal *big.Int
	Iprep   icmodule.Rate
	Icps    icmodule.Rate
	Irelay  icmodule.Rate
	Ivoter  icmodule.Rate
}

func NewRewardFund() *RewardFund {
	return &RewardFund{
		Iglobal: new(big.Int),
	}
}

func NewSafeRewardFund(iglobal *big.Int, iprep, icps, irelay, ivoter icmodule.Rate) (*RewardFund, error) {
	if iglobal.Sign() < 0 {
		return nil, scoreresult.InvalidParameterError.Errorf("InvalidIglobal(%d)", iglobal)
	}
	if !(iprep.IsValid() && icps.IsValid() && irelay.IsValid() && ivoter.IsValid()) {
		return nil, scoreresult.InvalidParameterError.Errorf(
			"InvalidInflationRate(prep=%d,cps=%d,relay=%d,voter=%d)",
			iprep.Percent(), icps.Percent(), irelay.Percent(), ivoter.Percent())
	}
	isum := iprep + icps + irelay + ivoter
	if int64(isum) != icmodule.DenomInRate {
		return nil, icmodule.IllegalArgumentError.Errorf(
			"IllegalInflationRate(prep=%d,cps=%d,relay=%d,voter=%d)",
			iprep.Percent(), icps.Percent(), irelay.Percent(), ivoter.Percent())
	}
	return &RewardFund{
		Iglobal: iglobal,
		Iprep:   iprep,
		Icps:    icps,
		Irelay:  irelay,
		Ivoter:  ivoter,
	}, nil
}

func newRewardFundFromByte(bs []byte) (*RewardFund, error) {
	if bs == nil {
		return NewRewardFund(), nil
	}
	rc := &RewardFund{}
	if _, err := codec.BC.UnmarshalFromBytes(bs, rc); err != nil {
		return nil, err
	}
	return rc, nil
}

func (rf *RewardFund) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(
		rf.Iglobal,
		rf.Iprep.Percent(),
		rf.Icps.Percent(),
		rf.Irelay.Percent(),
		rf.Ivoter.Percent(),
	)
}

func (rf *RewardFund) RLPDecodeSelf(d codec.Decoder) error {
	var Iprep, Icps, Irelay, Ivoter int64
	err := d.DecodeListOf(
		&rf.Iglobal,
		&Iprep,
		&Icps,
		&Irelay,
		&Ivoter,
	)
	if err == nil {
		rf.Iprep = icmodule.ToRate(Iprep)
		rf.Icps = icmodule.ToRate(Icps)
		rf.Irelay = icmodule.ToRate(Irelay)
		rf.Ivoter = icmodule.ToRate(Ivoter)
	}
	return err
}
func (rf *RewardFund) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(rf)
}

func (rf *RewardFund) IsEmpty() bool {
	return rf.Iglobal.Sign() == 0
}

func (rf *RewardFund) Equal(other *RewardFund) bool {
	return rf.Iglobal.Cmp(other.Iglobal) == 0 &&
		rf.Iprep == other.Iprep &&
		rf.Icps == other.Icps &&
		rf.Irelay == other.Irelay &&
		rf.Ivoter == other.Ivoter
}

func (rf *RewardFund) Clone() *RewardFund {
	return &RewardFund{
		Iglobal: rf.Iglobal,
		Iprep:   rf.Iprep,
		Icps:    rf.Icps,
		Irelay:  rf.Irelay,
		Ivoter:  rf.Ivoter,
	}
}

func (rf *RewardFund) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["Iglobal"] = rf.Iglobal
	jso["Iprep"] = rf.Iprep.Percent()
	jso["Icps"] = rf.Icps.Percent()
	jso["Irelay"] = rf.Irelay.Percent()
	jso["Ivoter"] = rf.Ivoter.Percent()
	return jso
}

func (rf *RewardFund) GetPRepFund() *big.Int {
	return rf.Iprep.MulBigInt(rf.Iglobal)
}

func (rf *RewardFund) GetVoterFund() *big.Int {
	return rf.Ivoter.MulBigInt(rf.Iglobal)
}

func (rf *RewardFund) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "RewardFund{Iglobal=%d Iprep=%d Icps=%d Irelay=%d Ivoter=%d}",
				rf.Iglobal, rf.Iprep.Percent(), rf.Icps.Percent(), rf.Irelay.Percent(), rf.Ivoter.Percent())
		} else {
			fmt.Fprintf(f, "RewardFund{%d %d %d %d %d}",
				rf.Iglobal, rf.Iprep.Percent(), rf.Icps.Percent(), rf.Irelay.Percent(), rf.Ivoter.Percent())
		}
	}
}

func (rf *RewardFund) ToRewardFund2() *RewardFund2 {
	r2 := NewRewardFund2()
	r2.SetIGlobal(rf.Iglobal)
	r2.SetAllocationByKey(KeyIprep, rf.Iprep+rf.Ivoter)
	r2.SetAllocationByKey(KeyIcps, rf.Icps)
	r2.SetAllocationByKey(KeyIrelay, rf.Irelay)
	return r2
}

type RFundKey string

const (
	KeyIprep  RFundKey = "iprep"
	KeyIwage           = "iwage"
	KeyIcps            = "icps"
	KeyIrelay          = "irelay"
)

var rFundKeys = []RFundKey{KeyIprep, KeyIwage, KeyIcps, KeyIrelay}

func (r RFundKey) IsValid() bool {
	for _, v := range rFundKeys {
		if v == r {
			return true
		}
	}
	return false
}

func (r RFundKey) String() string {
	runes := []rune(r)
	runes[0] = unicode.ToUpper(runes[0])
	return string(r)
}

type RewardFund2 struct {
	iGlobal    *big.Int
	allocation map[RFundKey]icmodule.Rate
}

func (r *RewardFund2) IGlobal() *big.Int {
	return r.iGlobal
}

func (r *RewardFund2) SetIGlobal(value *big.Int) {
	r.iGlobal = value
}

func (r *RewardFund2) SetAllocation(alloc map[RFundKey]icmodule.Rate) {
	r.allocation = alloc
}

func (r *RewardFund2) GetAllocationByKey(key RFundKey) icmodule.Rate {
	if v, ok := r.allocation[key]; ok {
		return v
	} else {
		return 0
	}
}

func (r *RewardFund2) SetAllocationByKey(key RFundKey, value icmodule.Rate) {
	r.allocation[key] = value
}

func (r *RewardFund2) GetAmount(key RFundKey) *big.Int {
	return r.GetAllocationByKey(key).MulBigInt(r.iGlobal)
}

func (r *RewardFund2) Equal(r2 *RewardFund2) bool {
	if r.iGlobal.Cmp(r2.iGlobal) != 0 {
		return false
	}
	if len(r.allocation) != len(r2.allocation) {
		return false
	}
	for _, k := range rFundKeys {
		v1, ok1 := r.allocation[k]
		v2, ok2 := r2.allocation[k]
		if ok1 != ok2 {
			return false
		}
		if ok1 == true {
			if v1 != v2 {
				return false
			}
		}
	}
	return true
}

func (r *RewardFund2) RLPEncodeSelf(encoder codec.Encoder) error {
	return encoder.EncodeMulti(r.iGlobal, r.allocation)
}

func (r *RewardFund2) RLPDecodeSelf(d codec.Decoder) error {
	_, err := d.DecodeMulti(&r.iGlobal, &r.allocation)
	return err
}

func (r *RewardFund2) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(r)
}

func (r *RewardFund2) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["Iglobal"] = r.IGlobal()
	for k, v := range r.allocation {
		jso[k.String()] = v
	}
	return jso
}

func (r *RewardFund2) string(withName bool) string {
	ret := fmt.Sprintf("iGlobal=%d", r.iGlobal)
	for _, k := range rFundKeys {
		if v, ok := r.allocation[k]; ok {
			if len(ret) == 0 {
				if withName {
					ret = fmt.Sprintf("%s=%d", k, v)
				} else {
					ret = fmt.Sprintf("%d", v)
				}
			} else {
				if withName {
					ret = fmt.Sprintf("%s %s=%d", ret, k, v)
				} else {
					ret = fmt.Sprintf("%s %d", ret, v)
				}
			}
		}
	}
	if withName {
		ret = fmt.Sprintf("RewardFund2{%s}", ret)
	} else {
		ret = fmt.Sprintf("{%s}", ret)
	}
	return ret
}

func (r *RewardFund2) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "%s", r.string(true))
		} else {
			fmt.Fprintf(f, "%s", r.string(false))
		}
	case 's':
		fmt.Fprintf(f, "%s", r.string(true))
	}
}

func newRewardFund2FromByte(bs []byte) (*RewardFund2, error) {
	if bs == nil {
		return NewRewardFund2(), nil
	}
	rc := &RewardFund2{}
	if _, err := codec.BC.UnmarshalFromBytes(bs, rc); err != nil {
		return nil, err
	}
	return rc, nil
}

func NewRewardFund2() *RewardFund2 {
	return &RewardFund2{
		allocation: make(map[RFundKey]icmodule.Rate),
	}
}

type alloc struct {
	Name  RFundKey       `json:"name"`
	Value *common.HexInt `json:"value"`
}

func NewRewardFund2Allocation(param []interface{}) (map[RFundKey]icmodule.Rate, error) {
	allocation := make(map[RFundKey]icmodule.Rate)
	var a alloc
	for _, p := range param {
		bs, err := json.Marshal(p)
		if err != nil {
			return nil, scoreresult.IllegalFormatError.Wrapf(err, "failed to Reward Fund allocation")
		}
		if err = json.Unmarshal(bs, &a); err != nil {
			return nil, scoreresult.IllegalFormatError.Wrapf(err, "failed to Reward Fund allocation %+v", err)
		}
		if a.Name.IsValid() == false {
			return nil, scoreresult.InvalidParameterError.Errorf("invalid Reward Fund allocation name")
		}
		value := icmodule.Rate(a.Value.Int64())
		if value.IsValid() == false {
			return nil, scoreresult.InvalidParameterError.Errorf("invalid Reward Fund allocation value")
		}
		if _, ok := allocation[a.Name]; ok {
			return nil, scoreresult.InvalidParameterError.Errorf("duplicated Reward Fund allocation name")
		}
		allocation[a.Name] = value
	}

	sum := icmodule.Rate(0)
	for _, v := range allocation {
		sum += v
	}
	if sum.NumInt64() != icmodule.DenomInRate {
		return nil, scoreresult.InvalidParameterError.Errorf("sum of value is not %d", icmodule.DenomInRate)
	}

	return allocation, nil
}
