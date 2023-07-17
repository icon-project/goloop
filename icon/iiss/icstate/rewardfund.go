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
	"fmt"
	"math/big"

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
			iprep, icps, irelay, ivoter)
	}
	isum := iprep + icps + irelay + ivoter
	if int64(isum) != icmodule.DenomInRate {
		return nil, icmodule.IllegalArgumentError.Errorf(
			"IllegalInflationRate(prep=%d,cps=%d,relay=%d,voter=%d)",
			iprep, icps, irelay, ivoter)
	}
	return &RewardFund{
		Iglobal: iglobal,
		Iprep: iprep,
		Icps: icps,
		Irelay: irelay,
		Ivoter: ivoter,
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
		Iprep: rf.Iprep,
		Icps: rf.Icps,
		Irelay: rf.Irelay,
		Ivoter: rf.Ivoter,
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
			fmt.Fprintf(f, "RewardFund{Iglobal=%s Iprep=%s Icps=%s Irelay=%s Ivoter=%s}",
				rf.Iglobal, rf.Iprep, rf.Icps, rf.Irelay, rf.Ivoter)
		} else {
			fmt.Fprintf(f, "RewardFund{%s %s %s %s %s}",
				rf.Iglobal, rf.Iprep, rf.Icps, rf.Irelay, rf.Ivoter)
		}
	}
}
