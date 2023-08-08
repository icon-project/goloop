/*
 * Copyright 2023 ICON Foundation
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

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/service/scoreresult"
)

type CommissionInfo struct {
	rate          icmodule.Rate
	maxRate       icmodule.Rate
	maxChangeRate icmodule.Rate
}

func (ci *CommissionInfo) Rate() icmodule.Rate {
	return ci.rate
}

func (ci *CommissionInfo) MaxRate() icmodule.Rate {
	return ci.maxRate
}

func (ci *CommissionInfo) MaxChangeRate() icmodule.Rate {
	return ci.maxChangeRate
}

func (ci *CommissionInfo) SetRate(rate icmodule.Rate) error {
	if !(rate >= 0 && rate <= ci.maxRate) {
		return scoreresult.InvalidParameterError.Errorf(
			"InvalidCommissionRate(rate=%d,maxRate=%d)", rate, ci.maxRate)
	}
	ci.rate = rate
	return nil
}

func (ci *CommissionInfo) Equal(other *CommissionInfo) bool {
	return ci.rate == other.rate &&
		ci.maxRate == other.maxRate &&
		ci.maxChangeRate == other.maxChangeRate
}

func (ci *CommissionInfo) String() string {
	return fmt.Sprintf(
		"CommissionInfo{rate=%d maxRate=%d maxChangeRate=%d}",
		ci.rate, ci.maxRate, ci.maxChangeRate)
}

func (ci *CommissionInfo) RLPDecodeSelf(d codec.Decoder) error {
	return d.DecodeAll(&ci.rate, &ci.maxRate, &ci.maxChangeRate)
}

func (ci *CommissionInfo) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeMulti(ci.rate, ci.maxRate, ci.maxChangeRate)
}

func (ci *CommissionInfo) Clone() *CommissionInfo {
	ret, _ := NewCommissionInfo(ci.rate, ci.maxRate, ci.maxChangeRate)
	return ret
}

func (ci *CommissionInfo) ToJSON(jso map[string]interface{}) map[string]interface{} {
	if jso == nil {
		jso = make(map[string]interface{})
	}
	jso["commissionRate"] = ci.rate
	jso["maxCommissionRate"] = ci.maxRate
	jso["maxCommissionChangeRate"] = ci.maxChangeRate
	return jso
}

func NewCommissionInfo(rate, maxRate, maxChangeRate icmodule.Rate) (*CommissionInfo, error) {
	if !(rate.IsValid() && maxRate.IsValid() && maxChangeRate.IsValid()) {
		return nil, scoreresult.InvalidParameterError.Errorf(
			"InvalidCommissionInfo(rate=%d,maxRate=%d,maxChangeRate=%d)", rate, maxRate, maxChangeRate)
	}
	if rate > maxRate || maxChangeRate > maxRate {
		return nil, icmodule.IllegalArgumentError.Errorf(
			"IllegalCommissionInfo(rate=%d,maxRate=%d,maxChangeRate=%d)", rate, maxRate, maxChangeRate)
	}
	return &CommissionInfo{
		rate:          rate,
		maxRate:       maxRate,
		maxChangeRate: maxChangeRate,
	}, nil
}

func NewEmptyCommissionInfo() *CommissionInfo {
	return &CommissionInfo{}
}
