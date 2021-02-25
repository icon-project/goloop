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
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/intconv"
)

type RewardFund struct {
	Iglobal *big.Int
	Iprep   *big.Int
	Icps    *big.Int
	Irelay  *big.Int
	Ivoter  *big.Int
}

func NewRewardFund() *RewardFund {
	return &RewardFund{
		Iglobal: new(big.Int),
		Iprep:   new(big.Int),
		Icps:    new(big.Int),
		Irelay:  new(big.Int),
		Ivoter:  new(big.Int),
	}
}

func newRewardFundFromByte(bs []byte) (*RewardFund, error) {
	rc := NewRewardFund()
	if bs == nil {
		return rc, nil
	}
	if _, err := codec.BC.UnmarshalFromBytes(bs, rc); err != nil {
		return nil, err
	}
	return rc, nil
}

func (rf *RewardFund) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(
		rf.Iglobal,
		rf.Iprep,
		rf.Icps,
		rf.Irelay,
		rf.Ivoter,
	)
}

func (rf *RewardFund) RLPDecodeSelf(d codec.Decoder) error {
	return d.DecodeListOf(
		&rf.Iglobal,
		&rf.Iprep,
		&rf.Icps,
		&rf.Irelay,
		&rf.Ivoter,
	)
}
func (rf *RewardFund) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(rf)
}

func (rf *RewardFund) IsEmpty() bool {
	return rf.Iglobal.Sign() == 0
}

func (rf *RewardFund) Equal(rc2 *RewardFund) bool {
	return rf.Iglobal.Cmp(rc2.Iglobal) == 0 &&
		rf.Iprep.Cmp(rc2.Iprep) == 0 &&
		rf.Icps.Cmp(rc2.Icps) == 0 &&
		rf.Irelay.Cmp(rc2.Irelay) == 0 &&
		rf.Ivoter.Cmp(rc2.Ivoter) == 0
}

func (rf *RewardFund) Clone() *RewardFund {
	rewardFund := NewRewardFund()
	rewardFund.Iglobal.Set(rf.Iglobal)
	rewardFund.Iprep.Set(rf.Iprep)
	rewardFund.Icps.Set(rf.Icps)
	rewardFund.Irelay.Set(rf.Irelay)
	rewardFund.Ivoter.Set(rf.Ivoter)
	return rewardFund
}

func (rf *RewardFund) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["Iglobal"] = intconv.FormatBigInt(rf.Iglobal)
	jso["Iprep"] = intconv.FormatBigInt(rf.Iprep)
	jso["Icps"] = intconv.FormatBigInt(rf.Icps)
	jso["Irelay"] = intconv.FormatBigInt(rf.Irelay)
	jso["Ivoter"] = intconv.FormatBigInt(rf.Ivoter)
	return jso
}

func (rf *RewardFund) GetPRepFund() *big.Int {
	fund := new(big.Int).Mul(rf.Iglobal, rf.Iprep)
	return fund.Div(fund, big.NewInt(100))
}

func (rf *RewardFund) GetVoterFund() *big.Int {
	fund := new(big.Int).Mul(rf.Iglobal, rf.Ivoter)
	return fund.Div(fund, big.NewInt(100))
}
