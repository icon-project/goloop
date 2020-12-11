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
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
	"math/big"
)

var UnbondingPeriod = int64(10)

type Unbond struct {
	Address *common.Address
	Value   *big.Int
	Expire  int64
}

func newUnbond() *Unbond {
	return &Unbond{
		Address: new(common.Address),
		Value:   new(big.Int),
	}
}

func (ub *Unbond) Equal(ub2 *Unbond) bool {
	return ub.Address.Equal(ub2.Address) && ub.Value.Cmp(ub2.Value) == 0 && ub.Expire == ub2.Expire
}

func (ub *Unbond) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["address"] = ub.Address
	jso["value"] = intconv.FormatBigInt(ub.Value)
	jso["expireHeight"] = ub.Expire

	return jso
}

func (ub *Unbond) Clone() *Unbond {
	n := newUnbond()
	n.Address.Set(ub.Address)
	n.Value.Set(ub.Value)
	n.Expire = ub.Expire
	return n
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
	unbonds := make([]*Unbond, len(ul))
	for i, u := range ul {
		unbonds[i] = u.Clone()
	}
	return unbonds
}

func (ul Unbonds) GetUnbondAmount() *big.Int {
	total := new(big.Int)
	for _, b := range ul {
		total.Add(total, b.Value)
	}
	return total
}

func (ul Unbonds) ToJSON(_ module.JSONVersion) []interface{} {
	if !ul.Has() {
		return nil
	}
	unbonds := make([]interface{}, len(ul))

	for idx, u := range ul {
		unbonds[idx] = u.ToJSON()
	}
	return unbonds
}
