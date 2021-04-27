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
)

const (
	issueVersion1 = iota + 1
	issueVersion  = issueVersion1
)

type Issue struct {
	icobject.NoDatabase

	totalIssued     *big.Int // amount of issued ICX while current calculation period
	prevTotalIssued *big.Int // amount of issued ICX while previous calculation period
	overIssued      *big.Int // prevTotalIssued - reward calculated by calculator
	iScoreRemains   *big.Int // not issued ICX
	prevBlockFee    *big.Int
}

func newIssue(_ icobject.Tag) *Issue {
	return new(Issue)
}

func NewIssue() *Issue {
	return &Issue{
		totalIssued:     new(big.Int),
		prevTotalIssued: new(big.Int),
		overIssued:      new(big.Int),
		iScoreRemains:   new(big.Int),
		prevBlockFee:    new(big.Int),
	}
}

func (i *Issue) Version() int {
	return issueVersion
}

func (i *Issue) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.DecodeListOf(
		&i.totalIssued,
		&i.prevTotalIssued,
		&i.overIssued,
		&i.iScoreRemains,
		&i.prevBlockFee,
	)
}

func (i *Issue) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
		i.totalIssued,
		i.prevTotalIssued,
		i.overIssued,
		i.iScoreRemains,
		i.prevBlockFee,
	)
}

func (i *Issue) Equal(o icobject.Impl) bool {
	if i2, ok := o.(*Issue); ok {
		return i.totalIssued.Cmp(i2.totalIssued) == 0 &&
			i.prevTotalIssued.Cmp(i2.prevTotalIssued) == 0 &&
			i.overIssued.Cmp(i2.overIssued) == 0 &&
			i.iScoreRemains.Cmp(i2.iScoreRemains) == 0 &&
			i.prevBlockFee.Cmp(i2.prevBlockFee) == 0
	} else {
		return false
	}
}

func (i *Issue) Clone() *Issue {
	ni := NewIssue()
	ni.totalIssued = i.totalIssued
	ni.prevTotalIssued = i.prevTotalIssued
	ni.overIssued = i.overIssued
	ni.iScoreRemains = i.iScoreRemains
	ni.prevBlockFee = i.prevBlockFee
	return ni
}

func (i *Issue) TotalIssued() *big.Int {
	return i.totalIssued
}

func (i *Issue) SetTotalIssued(v *big.Int) {
	i.totalIssued = v
}

func (i *Issue) PrevTotalIssued() *big.Int {
	return i.prevTotalIssued
}

func (i *Issue) SetPrevTotalIssued(v *big.Int) {
	i.prevTotalIssued = v
}

func (i *Issue) OverIssued() *big.Int {
	return i.overIssued
}

func (i *Issue) SetOverIssued(v *big.Int) {
	i.overIssued = v
}

func (i *Issue) IScoreRemains() *big.Int {
	return i.iScoreRemains
}

func (i *Issue) SetIScoreRemains(v *big.Int) {
	i.iScoreRemains = v
}

func (i *Issue) PrevBlockFee() *big.Int {
	return i.prevBlockFee
}

func (i *Issue) SetPrevBlockFee(v *big.Int) {
	i.prevBlockFee = v
}

func (i *Issue) Update(totalReward *big.Int, byFee *big.Int, byOverIssued *big.Int) *Issue {
	issue := i.Clone()
	issue.totalIssued = new(big.Int).Add(issue.totalIssued, totalReward)
	if byFee.Sign() != 0 {
		issue.prevBlockFee = new(big.Int).Sub(issue.prevBlockFee, byFee)
	}
	if byOverIssued.Sign() != 0 {
		issue.overIssued = new(big.Int).Sub(issue.overIssued, byOverIssued)
	}
	return issue
}

func (i *Issue) ResetTotalIssued() {
	i.prevTotalIssued = i.totalIssued
	i.totalIssued = new(big.Int)
}

func (i *Issue) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "Issue{totalIssued=%s prevTotalIssued=%s overIssued=%s iscoreRemains=%s prevBlockFee=%s}",
				i.totalIssued, i.prevTotalIssued, i.overIssued, i.iScoreRemains, i.prevBlockFee)
		} else {
			fmt.Fprintf(f, "Issue{%s %s %s %s %s}",
				i.totalIssued, i.prevTotalIssued, i.overIssued, i.iScoreRemains, i.prevBlockFee)
		}
	}
}
