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
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/service/scoreresult"
)

func NewSafeRewardFundV1(iglobal *big.Int, iprep, icps, irelay, ivoter icmodule.Rate) (*RewardFund, error) {
	rf := NewRewardFund(RFVersion1)
	if err := rf.SetIGlobal(iglobal); err != nil {
		return nil, err
	}
	if err := rf.SetAllocation(map[RFundKey]icmodule.Rate{
		KeyIprep:  iprep,
		KeyIcps:   icps,
		KeyIrelay: irelay,
		KeyIvoter: ivoter,
	}); err != nil {
		return nil, err
	}
	return rf, nil
}

func NewSafeRewardFundV2(iglobal *big.Int, iprep, iwage, icps, irelay icmodule.Rate) (*RewardFund, error) {
	rf := NewRewardFund(RFVersion2)
	if err := rf.SetIGlobal(iglobal); err != nil {
		return nil, err
	}
	if err := rf.SetAllocation(map[RFundKey]icmodule.Rate{
		KeyIprep:  iprep,
		KeyIwage:  iwage,
		KeyIcps:   icps,
		KeyIrelay: irelay,
	}); err != nil {
		return nil, err
	}
	return rf, nil
}

type RFundKey string

const (
	KeyIprep  RFundKey = "Iprep"
	KeyIwage  RFundKey = "Iwage"
	KeyIcps   RFundKey = "Icps"
	KeyIrelay RFundKey = "Irelay"
	KeyIvoter RFundKey = "Ivoter"
)

var rFundKeys = map[int][]RFundKey{
	RFVersion1: {KeyIprep, KeyIcps, KeyIrelay, KeyIvoter},
	RFVersion2: {KeyIprep, KeyIwage, KeyIcps, KeyIrelay},
}

func (r RFundKey) IsValid(version int) bool {
	keys, ok := rFundKeys[version]
	if !ok {
		return false
	}
	for _, v := range keys {
		if v == r {
			return true
		}
	}
	return false
}

func (r RFundKey) String() string {
	runes := []rune(r)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

const (
	RFVersion1 = iota
	RFVersion2
	RFVersionReserved

	KeyIglobal = "Iglobal"
)

type RewardFund struct {
	version    int
	iGlobal    *big.Int
	allocation map[RFundKey]icmodule.Rate
}

func (r *RewardFund) IGlobal() *big.Int {
	return r.iGlobal
}

func (r *RewardFund) GetOrderAllocationKeys() []RFundKey {
	keys := make([]RFundKey, 0, len(r.allocation))
	for k := range r.allocation {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return keys
}

func (r *RewardFund) SetIGlobal(value *big.Int) error {
	if value.Sign() == -1 {
		return scoreresult.InvalidParameterError.Errorf("InvalidIglobal(%d)", value)
	}
	r.iGlobal = value
	return nil
}

func (r *RewardFund) validateAllocation(alloc map[RFundKey]icmodule.Rate) error {
	sum := icmodule.Rate(0)
	for k, v := range alloc {
		if k.IsValid(r.version) == false {
			return scoreresult.InvalidParameterError.Errorf("InvalidInflationRate(not supported key %s)", k)
		}
		if v.IsValid() == false {
			return scoreresult.InvalidParameterError.Errorf("InvalidInflationRate(%s=%d)", k, v)
		}
		sum += v
	}
	if int64(sum) != icmodule.DenomInRate {
		return icmodule.IllegalArgumentError.Errorf("IllegalInflationRate(sum of rates is not %d)", icmodule.DenomInRate)
	}
	return nil
}

func (r *RewardFund) SetAllocation(alloc map[RFundKey]icmodule.Rate) error {
	if err := r.validateAllocation(alloc); err != nil {
		return err
	}
	r.allocation = alloc
	return nil
}

func (r *RewardFund) GetAllocationByKey(key RFundKey) icmodule.Rate {
	if v, ok := r.allocation[key]; ok {
		return v
	} else {
		return icmodule.Rate(0)
	}
}

func (r *RewardFund) IPrep() icmodule.Rate {
	return r.GetAllocationByKey(KeyIprep)
}

func (r *RewardFund) ICps() icmodule.Rate {
	return r.GetAllocationByKey(KeyIcps)
}
func (r *RewardFund) IRelay() icmodule.Rate {
	return r.GetAllocationByKey(KeyIrelay)
}

func (r *RewardFund) IVoter() icmodule.Rate {
	return r.GetAllocationByKey(KeyIvoter)
}

func (r *RewardFund) Iwage() icmodule.Rate {
	return r.GetAllocationByKey(KeyIwage)
}

func (r *RewardFund) GetAmount(key RFundKey) *big.Int {
	return r.GetAllocationByKey(key).MulBigInt(r.iGlobal)
}

func (r *RewardFund) Equal(r2 *RewardFund) bool {
	if r.version != r2.version {
		return false
	}
	if r.iGlobal.Cmp(r2.iGlobal) != 0 {
		return false
	}
	if len(r.allocation) != len(r2.allocation) {
		return false
	}
	for k, v1 := range r.allocation {
		v2, ok2 := r2.allocation[k]
		if !ok2 {
			return false
		}
		if v1 != v2 {
			return false
		}
	}
	return true
}

func (r *RewardFund) RLPEncodeSelf(encoder codec.Encoder) error {
	if r.version == RFVersion1 {
		return encoder.EncodeListOf(
			r.IGlobal(),
			r.IPrep().Percent(),
			r.ICps().Percent(),
			r.IRelay().Percent(),
			r.IVoter().Percent(),
		)
	}
	return encoder.EncodeListOf(r.version, r.iGlobal, r.allocation)
}

func (r *RewardFund) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	var iGlobal *big.Int
	if err = d2.Decode(&iGlobal); err != nil {
		return err
	}
	if iGlobal.Cmp(big.NewInt(RFVersionReserved)) > 0 {
		r.version = RFVersion1
		r.iGlobal = iGlobal
		var Iprep, Icps, Irelay, Ivoter int64
		_, err = d2.DecodeMulti(
			&Iprep,
			&Icps,
			&Irelay,
			&Ivoter,
		)
		if err == nil {
			r.allocation = make(map[RFundKey]icmodule.Rate)
			r.allocation[KeyIprep] = icmodule.ToRate(Iprep)
			r.allocation[KeyIcps] = icmodule.ToRate(Icps)
			r.allocation[KeyIrelay] = icmodule.ToRate(Irelay)
			r.allocation[KeyIvoter] = icmodule.ToRate(Ivoter)
		}
	} else {
		r.version = int(iGlobal.Int64())
		_, err = d2.DecodeMulti(&r.iGlobal, &r.allocation)
	}
	return err
}

func (r *RewardFund) ToRewardFundV2() *RewardFund {
	if r.version == RFVersion1 {
		rf := NewRewardFund(RFVersion2)
		rf.SetIGlobal(r.IGlobal())
		allocation := map[RFundKey]icmodule.Rate{
			KeyIprep:  r.IPrep() + r.IVoter(),
			KeyIcps:   r.ICps(),
			KeyIrelay: r.IRelay(),
			KeyIwage:  icmodule.Rate(0),
		}
		rf.SetAllocation(allocation)
		return rf
	}
	return r
}

func (r *RewardFund) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(r)
}

func (r *RewardFund) Clone() *RewardFund {
	nr := NewRewardFund(r.version)
	nr.SetIGlobal(r.iGlobal)
	for k, v := range r.allocation {
		nr.allocation[k] = v
	}
	return nr
}

func (r *RewardFund) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso[KeyIglobal] = r.IGlobal()
	for k, v := range r.allocation {
		if r.version == RFVersion1 {
			jso[k.String()] = v.Percent()
		} else {
			jso[k.String()] = v.NumInt64()
		}
	}
	return jso
}

func (r *RewardFund) string(withName bool) string {
	var sb strings.Builder

	if withName {
		sb.WriteString("RewardFund{version=")
	} else {
		sb.WriteByte('{')
	}
	fmt.Fprintf(&sb, "%d", r.version)

	sb.WriteByte(' ')
	if withName {
		sb.WriteString(KeyIglobal)
		sb.WriteByte('=')
	}
	sb.WriteString(r.iGlobal.String())

	for _, k := range r.GetOrderAllocationKeys() {
		sb.WriteByte(' ')
		if withName {
			sb.WriteString(string(k))
			sb.WriteByte('=')
		}
		sb.WriteString(strconv.FormatInt(r.allocation[k].NumInt64(), 10))
	}

	sb.WriteByte('}')
	return sb.String()
}

func (r *RewardFund) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		fmt.Fprintf(f, "%s", r.string(f.Flag('+')))
	case 's':
		fmt.Fprintf(f, "%s", r.string(true))
	}
}

func NewRewardFundFromByte(bs []byte) (*RewardFund, error) {
	if bs == nil {
		return NewRewardFund(RFVersion2), nil
	}
	rc := &RewardFund{}
	if _, err := codec.BC.UnmarshalFromBytes(bs, rc); err != nil {
		return nil, err
	}
	return rc, nil
}

func NewRewardFund(version int) *RewardFund {
	return &RewardFund{
		version:    version,
		iGlobal:    new(big.Int),
		allocation: make(map[RFundKey]icmodule.Rate),
	}
}

type alloc struct {
	Name  RFundKey       `json:"name"`
	Value *common.HexInt `json:"value"`
}

func NewRewardFund2Allocation(param []interface{}) (map[RFundKey]icmodule.Rate, error) {
	allocation := make(map[RFundKey]icmodule.Rate)
	for _, p := range param {
		var a alloc
		bs, err := json.Marshal(p)
		if err != nil {
			return nil, scoreresult.IllegalFormatError.Wrapf(err, "failed to Reward Fund allocation")
		}
		if err = json.Unmarshal(bs, &a); err != nil {
			return nil, scoreresult.IllegalFormatError.Wrapf(err, "failed to Reward Fund allocation %+v", err)
		}
		if a.Name.IsValid(RFVersion2) == false {
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
