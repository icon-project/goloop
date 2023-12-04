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
	"bytes"
	"fmt"
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
	prevHeight       int64
	prevPeriod       int64
	prevHash         []byte
	prevReward       *big.Int
}

func newRewardCalcInfo(_ icobject.Tag) *RewardCalcInfo {
	return new(RewardCalcInfo)
}

func NewRewardCalcInfo() *RewardCalcInfo {
	return &RewardCalcInfo{
		prevReward:       new(big.Int),
	}
}

func (rc *RewardCalcInfo) Version() int {
	return rewardCalcInfoVersion
}

func (rc *RewardCalcInfo) StartHeight() int64 {
	return rc.startHeight
}

func (rc *RewardCalcInfo) SetStartHeight(height int64) {
	rc.startHeight = height
}

func (rc *RewardCalcInfo) PrevHeight() int64 {
	return rc.prevHeight
}

func (rc *RewardCalcInfo) SetPrevHeight(height int64) {
	rc.prevHeight = height
}

func (rc *RewardCalcInfo) PrevPeriod() int64 {
	return rc.prevPeriod
}

func (rc *RewardCalcInfo) PrevHash() []byte {
	return rc.prevHash
}

func (rc *RewardCalcInfo) SetPrevHash(hash []byte) {
	rc.prevHash = hash
}

func (rc *RewardCalcInfo) PrevCalcReward() *big.Int {
	return rc.prevReward
}

func (rc *RewardCalcInfo) SetPrevCalcReward(v *big.Int) {
	rc.prevReward = v
}

func (rc *RewardCalcInfo) GetResultInJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["iscore"] = rc.prevReward
	jso["estimatedICX"] = new(big.Int).Div(rc.prevReward, big.NewInt(1000))
	jso["startBlockHeight"] = rc.prevHeight
	jso["endBlockHeight"] = rc.prevHeight + rc.prevPeriod - 1
	jso["stateHash"] = rc.prevHash
	return jso
}

func (rc *RewardCalcInfo) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.DecodeAll(
		&rc.startHeight,
		&rc.prevHeight,
		&rc.prevPeriod,
		&rc.prevHash,
		&rc.prevReward,
	)
}

func (rc *RewardCalcInfo) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		rc.startHeight,
		rc.prevHeight,
		rc.prevPeriod,
		rc.prevHash,
		rc.prevReward,
	)
}

func (rc *RewardCalcInfo) Equal(o icobject.Impl) bool {
	if rc2, ok := o.(*RewardCalcInfo); ok {
		return rc.startHeight == rc2.startHeight &&
			rc.prevHeight == rc2.prevHeight &&
			rc.prevPeriod == rc2.prevPeriod &&
			bytes.Equal(rc.prevHash, rc2.prevHash) &&
			rc.prevReward.Cmp(rc2.prevReward) == 0
	} else {
		return false
	}
}

func (rc *RewardCalcInfo) Clone() *RewardCalcInfo {
	nrc := NewRewardCalcInfo()
	nrc.startHeight = rc.startHeight
	nrc.prevHeight = rc.prevHeight
	nrc.prevPeriod = rc.prevPeriod
	nrc.prevHash = rc.prevHash
	nrc.prevReward = rc.prevReward
	return nrc
}

func (rc *RewardCalcInfo) Update(blockHeight int64, reward *big.Int, hash []byte) {
	rc.prevPeriod = rc.startHeight - rc.prevHeight
	rc.prevHeight = rc.startHeight
	rc.startHeight = blockHeight
	rc.prevHash = hash
	rc.prevReward = reward
}

func (rc *RewardCalcInfo) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "rcInfo{start=%d prevHeight=%d prevPeriod=%d prevReward=%s}",
				rc.startHeight, rc.prevHeight, rc.prevPeriod, rc.prevReward)
		} else {
			fmt.Fprintf(f, "rcInfo{%d %v %d %s}",
				rc.startHeight, rc.prevHeight, rc.prevPeriod, rc.prevReward)

		}
	}
}
