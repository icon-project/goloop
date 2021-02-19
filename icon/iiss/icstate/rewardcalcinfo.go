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
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

const (
	rewardCalcInfoVersion1 = iota + 1
	rewardCalcInfoVersion  = rewardCalcInfoVersion1
)

type RewardCalcInfo struct {
	icobject.NoDatabase

	startHeight      int64
	period           int64
	isDecentralized  bool
	prevHeight       int64
	prevCalcReward   *big.Int
	additionalReward *big.Int
}

func newRewardCalcInfo(_ icobject.Tag) *RewardCalcInfo {
	return NewRewardCalcInfo()
}

func NewRewardCalcInfo() *RewardCalcInfo {
	return &RewardCalcInfo{
		prevCalcReward:   new(big.Int),
		additionalReward: new(big.Int),
	}
}

func (rc *RewardCalcInfo) Version() int {
	return rewardCalcInfoVersion
}

func (rc *RewardCalcInfo) StartHeight() int64 {
	return rc.startHeight
}

func (rc *RewardCalcInfo) Period() int64 {
	return rc.period
}

func (rc *RewardCalcInfo) IsDecentralized() bool {
	return rc.isDecentralized
}

func (rc *RewardCalcInfo) PrevHeight() int64 {
	return rc.prevHeight
}

func (rc *RewardCalcInfo) PrevCalcReward() *big.Int {
	return rc.prevCalcReward
}

func (rc *RewardCalcInfo) AdditionalReward() *big.Int {
	return rc.additionalReward
}

func (rc *RewardCalcInfo) GetEndHeight() int64 {
	return rc.startHeight + rc.period - 1
}

func (rc *RewardCalcInfo) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.DecodeListOf(
		&rc.startHeight,
		&rc.period,
		&rc.isDecentralized,
		&rc.prevHeight,
		&rc.prevCalcReward,
		&rc.additionalReward,
	)
}

func (rc *RewardCalcInfo) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
		rc.startHeight,
		rc.period,
		rc.isDecentralized,
		rc.prevHeight,
		rc.prevCalcReward,
		rc.additionalReward,
	)
}

func (rc *RewardCalcInfo) Equal(o icobject.Impl) bool {
	if rc2, ok := o.(*RewardCalcInfo); ok {
		return rc.startHeight == rc2.startHeight &&
			rc.period == rc2.period &&
			rc.isDecentralized == rc2.isDecentralized &&
			rc.prevHeight == rc2.prevHeight &&
			rc.prevCalcReward.Cmp(rc2.prevCalcReward) == 0 &&
			rc.additionalReward.Cmp(rc2.additionalReward) == 0
	} else {
		return false
	}
}

func (rc *RewardCalcInfo) Clone() *RewardCalcInfo {
	nrc := NewRewardCalcInfo()
	nrc.startHeight = rc.startHeight
	nrc.period = rc.period
	nrc.isDecentralized = rc.isDecentralized
	nrc.prevHeight = rc.prevHeight
	nrc.prevCalcReward = new(big.Int).Set(rc.prevCalcReward)
	nrc.additionalReward = new(big.Int).Set(rc.additionalReward)
	return nrc
}

func (rc *RewardCalcInfo) Start(
	blockHeight int64, period int64, isDecentralized bool, calcReward *big.Int, additionalReward *big.Int,
) {
	rc.prevHeight = rc.startHeight
	rc.startHeight = blockHeight
	rc.period = period
	rc.isDecentralized = isDecentralized
	rc.prevCalcReward.Set(calcReward)
	rc.additionalReward.Set(additionalReward)
}
