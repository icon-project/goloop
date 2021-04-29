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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

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

func (u *Unbond) To() *common.Address {
	return u.Address
}

func (u *Unbond) SetTo(t *common.Address) {
	u.Address = t
}

func (u *Unbond) SetValue(v *big.Int) {
	u.Value = v
}

func (u *Unbond) Amount() *big.Int {
	return u.Value
}

func (u *Unbond) SetExpire(e int64) {
	u.Expire = e
}

func (u *Unbond) ExpireHeight() int64 {
	return u.Expire
}

func (u *Unbond) Slash(ratio int) *big.Int {
	slashAmount := new(big.Int).Mul(u.Value, big.NewInt(int64(ratio)))
	slashAmount.Div(slashAmount, big.NewInt(int64(100)))
	newValue := new(big.Int).Sub(u.Value, slashAmount)
	u.SetValue(newValue)
	return slashAmount
}

func (u *Unbond) Equal(ub2 *Unbond) bool {
	return u.Address.Equal(ub2.Address) && u.Value.Cmp(ub2.Value) == 0 && u.Expire == ub2.Expire
}

func (u *Unbond) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["address"] = u.Address
	jso["value"] = u.Value
	jso["expireBlockHeight"] = u.Expire

	return jso
}

func (u *Unbond) Clone() *Unbond {
	n := newUnbond()
	n.Address = u.Address
	n.Value = u.Value
	n.Expire = u.Expire
	return n
}

func (u *Unbond) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "Unbond{address=%s value=%s expire=%d}", u.Address, u.Value, u.Expire)
		} else {
			fmt.Fprintf(f, "Unbond{%s %s %d}", u.Address, u.Value, u.Expire)
		}
	}
}

type Unbonds []*Unbond

func (ul Unbonds) IsEmpty() bool {
	return len(ul) == 0
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

func (ul Unbonds) GetUnbondByAddress(address module.Address) (*Unbond, int) {
	for i, ub := range ul {
		if address.Equal(ub.Address) {
			return ub, i
		}
	}
	return nil, -1
}

func (ul Unbonds) MapByAddr() map[string]*Unbond {
	newMap := make(map[string]*Unbond)
	for _, ub := range ul {
		key := icutils.ToKey(ub.Address)
		newMap[key] = ub
	}
	return newMap
}

func (ul Unbonds) ExpireRefCount() map[int64]int {
	newMap := make(map[int64]int)
	for _, ub := range ul {
		key := ub.Expire
		newMap[key] = newMap[key] + 1
	}
	return newMap
}

func (ul *Unbonds) Add(address module.Address, value *big.Int, expireHeight int64) {
	unbond := newUnbond()
	unbond.SetTo(common.AddressToPtr(address))
	unbond.SetValue(value)
	unbond.Expire = expireHeight
	*ul = append(*ul, unbond)
}

func (ul *Unbonds) Delete(i int) error {
	if i < 0 || i >= len(*ul) {
		return errors.Errorf("Invalid index")
	}

	copy((*ul)[i:], (*ul)[i+1:])
	(*ul)[len(*ul)-1] = nil // or the zero value of T
	*ul = (*ul)[:len(*ul)-1]
	return nil
}

func (ul *Unbonds) DeleteByAddress(address module.Address) error {
	_, idx := ul.GetUnbondByAddress(address)
	return ul.Delete(idx)
}

func (ul *Unbonds) Slash(address module.Address, ratio int) (Unbonds, *big.Int, int64) {
	expire := int64(-1)
	amount := big.NewInt(0)
	newUnbonds := make(Unbonds, 0)

	for _, u := range *ul {
		if u.To().Equal(address) {
			unbond := u.Clone()
			amount = unbond.Slash(ratio)

			if ratio < 100 {
				newUnbonds = append(newUnbonds, unbond)
			} else if ratio == 100 {
				expire = unbond.ExpireHeight()
			}
		} else {
			newUnbonds = append(newUnbonds, u)
		}
	}
	return newUnbonds, amount, expire
}

func (ul Unbonds) ToJSON(_ module.JSONVersion) []interface{} {
	if ul.IsEmpty() {
		return nil
	}
	unbonds := make([]interface{}, len(ul))

	for idx, u := range ul {
		unbonds[idx] = u.ToJSON()
	}
	return unbonds
}
