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

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
)

const (
	issueVersion1 = iota + 1
	issueVersion  = issueVersion1
)

type Issue struct {
	icobject.NoDatabase

	totalReward      *big.Int // sum of rewards calculated by Issuer in current term
	prevTotalReward  *big.Int
	overIssuedIScore *big.Int // prevTotalReward - reward calculated by calculator
	prevBlockFee     *big.Int
}

func newIssue(_ icobject.Tag) *Issue {
	return new(Issue)
}

func NewIssue() *Issue {
	bigIntZero := new(big.Int)
	return &Issue{
		totalReward:      bigIntZero,
		prevTotalReward:  bigIntZero,
		overIssuedIScore: bigIntZero,
		prevBlockFee:     bigIntZero,
	}
}

func (i *Issue) Version() int {
	return issueVersion
}

func (i *Issue) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.DecodeAll(
		&i.totalReward,
		&i.prevTotalReward,
		&i.overIssuedIScore,
		&i.prevBlockFee,
	)
}

func (i *Issue) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		i.totalReward,
		i.prevTotalReward,
		i.overIssuedIScore,
		i.prevBlockFee,
	)
}

func (i *Issue) Equal(o icobject.Impl) bool {
	if i2, ok := o.(*Issue); ok {
		return i.totalReward.Cmp(i2.totalReward) == 0 &&
			i.prevTotalReward.Cmp(i2.prevTotalReward) == 0 &&
			i.overIssuedIScore.Cmp(i2.overIssuedIScore) == 0 &&
			i.prevBlockFee.Cmp(i2.prevBlockFee) == 0
	} else {
		return false
	}
}

func (i *Issue) Clone() *Issue {
	return &Issue{
		totalReward:      i.totalReward,
		prevTotalReward:  i.prevTotalReward,
		overIssuedIScore: i.overIssuedIScore,
		prevBlockFee:     i.prevBlockFee,
	}
}

func (i *Issue) TotalReward() *big.Int {
	return i.totalReward
}

func (i *Issue) SetTotalReward(v *big.Int) {
	i.totalReward = v
}

func (i *Issue) PrevTotalReward() *big.Int {
	return i.prevTotalReward
}

func (i *Issue) SetPrevTotalReward(v *big.Int) {
	i.prevTotalReward = v
}

func (i *Issue) OverIssuedIScore() *big.Int {
	return i.overIssuedIScore
}

func (i *Issue) SetOverIssuedIScore(v *big.Int) {
	i.overIssuedIScore = v
}

func (i *Issue) GetOverIssuedICX() *big.Int {
	return icutils.IScoreToICX(i.overIssuedIScore)
}

func (i *Issue) PrevBlockFee() *big.Int {
	return i.prevBlockFee
}

func (i *Issue) SetPrevBlockFee(v *big.Int) {
	i.prevBlockFee = v
}

func (i *Issue) Update(totalReward *big.Int, byFee *big.Int, byOverIssued *big.Int) *Issue {
	overIssuedDelta := new(big.Int).Add(byFee, byOverIssued)
	overIssuedDelta.Sub(overIssuedDelta, i.prevBlockFee)

	return &Issue{
		totalReward:      new(big.Int).Add(i.totalReward, totalReward),
		prevTotalReward:  i.prevTotalReward,
		overIssuedIScore: new(big.Int).Sub(i.overIssuedIScore, icutils.ICXToIScore(overIssuedDelta)),
		prevBlockFee:     i.prevBlockFee,
	}
}

func (i *Issue) ResetTotalReward() {
	i.prevTotalReward = i.totalReward
	i.totalReward = new(big.Int)
}

func (i *Issue) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "Issue{totalReward=%s prevTotalReward=%s overIssuedIScore=%s prevBlockFee=%s}",
				i.totalReward, i.prevTotalReward, i.overIssuedIScore, i.prevBlockFee)
		} else {
			fmt.Fprintf(f, "Issue{%s %s %s %s}",
				i.totalReward, i.prevTotalReward, i.overIssuedIScore, i.prevBlockFee)
		}
	}
}
