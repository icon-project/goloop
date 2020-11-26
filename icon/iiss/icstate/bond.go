/*
 * Copyright 2020 ICON Foundation
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package icstate

import (
	"math/big"

	"github.com/icon-project/goloop/common"
)

type Bond struct {
	Target *common.Address
	Amount *big.Int
}

func newBond() *Bond {
	return &Bond{
		Target: new(common.Address),
		Amount: new(big.Int),
	}
}

func (b *Bond) Equal(b2 *Bond) bool {
	return b.Target.Equal(b2.Target) && b.Amount.Cmp(b2.Amount) == 0
}

type Bonds []*Bond

func (bl Bonds) Has() bool {
	return len(bl) > 0
}

func (bl Bonds) Equal(bl2 Bonds) bool {
	if len(bl) != len(bl2) {
		return false
	}
	for i, b := range bl {
		if !b.Equal(bl2[i]) {
			return false
		}
	}
	return true
}

func (bl Bonds) Clone() Bonds {
	if bl == nil {
		return nil
	}
	bonds := make([]*Bond, len(bl))
	copy(bonds, bl)
	return bonds
}
