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
	"github.com/icon-project/goloop/common"
	"math/big"
)

type Unbond struct {
	target       *common.Address
	amount       *big.Int
	expireHeight int64
}

func newUnbond() *Unbond {
	return &Unbond{
		target: new(common.Address),
		amount: new(big.Int),
	}
}

func (ub *Unbond) Equal(ub2 *Unbond) bool {
	return ub.target.Equal(ub2.target) && ub.amount.Cmp(ub2.amount) == 0 && ub.expireHeight == ub2.expireHeight
}

type Unbonds []*Unbond

func (ul Unbonds) Has() bool {
	return len(ul) > 0
}

func (ul Unbonds) Equal(ul2 Unbonds) bool {
	if len(ul) != len(ul2) {
		return false
	}
	for i, b := range ul {
		if !b.Equal(ul2[i]) {
			return false
		}
	}
	return true
}

func (ul Unbonds) Clone() Unbonds {
	if ul == nil {
		return nil
	}
	unbondings := make([]*Unbond, len(ul))
	copy(unbondings, ul)
	return unbondings
}

