/*
 * Copyright 2024 ICON Foundation
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
)

type BondRequirementInfo struct {
	version  int
	rate     icmodule.Rate
	nextRate icmodule.Rate
}

func (info *BondRequirementInfo) Version() int {
	return info.version
}

func (info *BondRequirementInfo) Rate() icmodule.Rate {
	return info.rate
}

func (info *BondRequirementInfo) SetRate(rate icmodule.Rate) {
	info.rate = rate
}

func (info *BondRequirementInfo) NextRate() icmodule.Rate {
	return info.nextRate
}

func (info *BondRequirementInfo) SetNextRate(rate icmodule.Rate) {
	info.nextRate = rate
}

func (info *BondRequirementInfo) String() string {
	return fmt.Sprintf("BondRequirementInfo{rate=%d nextRate=%d}", info.rate, info.nextRate)
}

func (info *BondRequirementInfo) RLPDecodeSelf(decoder codec.Decoder) error {
	return decoder.DecodeListOf(&info.version, &info.rate, &info.nextRate)
}

func (info *BondRequirementInfo) RLPEncodeSelf(encoder codec.Encoder) error {
	return encoder.EncodeListOf(info.version, info.rate, info.nextRate)
}

func (info *BondRequirementInfo) Equal(other *BondRequirementInfo) bool {
	if info == other {
		return true
	}
	return info.version == other.version &&
		info.rate == other.rate &&
		info.nextRate == other.nextRate
}

func (info *BondRequirementInfo) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(info)
}

func (info *BondRequirementInfo) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"current": info.rate,
		"next":    info.nextRate,
	}
}

func NewBondRequirementInfo(rate, nextRate icmodule.Rate) *BondRequirementInfo {
	return &BondRequirementInfo{
		version:  0,
		rate:     rate,
		nextRate: nextRate,
	}
}

func NewBondRequirementInfoFromByte(bs []byte) (*BondRequirementInfo, error) {
	info := new(BondRequirementInfo)
	if _, err := codec.BC.UnmarshalFromBytes(bs, info); err != nil {
		return nil, err
	}
	return info, nil
}
