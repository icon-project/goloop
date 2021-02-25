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

	TotalIssued     *big.Int // amount of issued ICX while current calculation period
	PrevTotalIssued *big.Int // amount of issued ICX while previous calculation period
	OverIssued      *big.Int // PrevTotalIssued - reward calculated by calculator
	IScoreRemains   *big.Int // not issued ICX
	PrevBlockFee    *big.Int
}

func newIssue(_ icobject.Tag) *Issue {
	return NewIssue()
}

func NewIssue() *Issue {
	return &Issue{
		TotalIssued:     new(big.Int),
		PrevTotalIssued: new(big.Int),
		OverIssued:      new(big.Int),
		IScoreRemains:   new(big.Int),
		PrevBlockFee:    new(big.Int),
	}
}

func (i *Issue) Version() int {
	return issueVersion
}

func (i *Issue) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.DecodeListOf(
		&i.TotalIssued,
		&i.PrevTotalIssued,
		&i.OverIssued,
		&i.IScoreRemains,
		&i.PrevBlockFee,
	)
}

func (i *Issue) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
		i.TotalIssued,
		i.PrevTotalIssued,
		i.OverIssued,
		i.IScoreRemains,
		i.PrevBlockFee,
	)
}

func (i *Issue) Equal(o icobject.Impl) bool {
	if i2, ok := o.(*Issue); ok {
		return i.TotalIssued.Cmp(i2.TotalIssued) == 0 &&
			i.PrevTotalIssued.Cmp(i2.PrevTotalIssued) == 0 &&
			i.OverIssued.Cmp(i2.OverIssued) == 0 &&
			i.IScoreRemains.Cmp(i2.IScoreRemains) == 0 &&
			i.PrevBlockFee.Cmp(i2.PrevBlockFee) == 0
	} else {
		return false
	}
}

func (i *Issue) Clone() *Issue {
	ni := NewIssue()
	ni.TotalIssued.Set(i.TotalIssued)
	ni.PrevTotalIssued.Set(i.PrevTotalIssued)
	ni.OverIssued.Set(i.OverIssued)
	ni.IScoreRemains.Set(i.IScoreRemains)
	ni.PrevBlockFee.Set(i.PrevBlockFee)
	return ni
}

func (i *Issue) ResetTotalIssued() {
	i.PrevTotalIssued.Set(i.TotalIssued)
	i.TotalIssued.SetInt64(0)
}
