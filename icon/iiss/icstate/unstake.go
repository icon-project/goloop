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
	"github.com/icon-project/goloop/common"
)

type Unstake struct {
	Amount       common.HexInt
	ExpireHeight int64
}

func (u *Unstake) Clone() *Unstake {
	n := new(Unstake)
	n.Amount.Set(u.Amount.Value())
	n.ExpireHeight = u.ExpireHeight
	return n
}

func (u *Unstake) Equal(u2 *Unstake) bool {
	if u == u2 {
		return true
	}
	return u.Amount.Cmp(u2.Amount.Value()) == 0 &&
		u.ExpireHeight == u2.ExpireHeight
}

type Unstakes []*Unstake

func (us Unstakes) Clone() Unstakes {
	if us == nil {
		return nil
	}
	n := make([]*Unstake, len(us))
	for i, u := range us {
		n[i] = u.Clone()
	}
	return n
}

func (us Unstakes) Equal(us2 Unstakes) bool {
	if len(us) != len(us2) {
		return false
	}
	for i, u := range us {
		if !u.Equal(us2[i]) {
			return false
		}
	}
	return true
}
