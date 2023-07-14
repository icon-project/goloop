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
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

type Unbond struct {
	address *common.Address
	value   *big.Int
	expire  int64
}

func NewUnbond(a *common.Address, v *big.Int, e int64) *Unbond {
	return &Unbond{
		address: a,
		value:   v,
		expire:  e,
	}
}

func (u *Unbond) RLPDecodeSelf(decoder codec.Decoder) error {
	return decoder.DecodeListOf(
		&u.address,
		&u.value,
		&u.expire,
	)
}

func (u *Unbond) RLPEncodeSelf(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
		u.address,
		u.value,
		u.expire,
	)
}

func (u *Unbond) Equal(o *Unbond) bool {
	return u.address.Equal(o.address) && u.value.Cmp(o.value) == 0 && u.expire == o.expire
}

func (u *Unbond) Address() *common.Address {
	return u.address
}

func (u *Unbond) SetValue(v *big.Int) {
	u.value = v
}

func (u *Unbond) Value() *big.Int {
	return u.value
}

func (u *Unbond) SetExpire(e int64) {
	u.expire = e
}

func (u *Unbond) Expire() int64 {
	return u.expire
}

func (u *Unbond) Slash(rate icmodule.Rate) *big.Int {
	slashAmount := rate.MulBigInt(u.value)
	u.value = new(big.Int).Sub(u.value, slashAmount)
	return slashAmount
}

func (u *Unbond) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["address"] = u.address
	jso["value"] = u.value
	jso["expireBlockHeight"] = u.expire

	return jso
}

func (u *Unbond) Clone() *Unbond {
	return NewUnbond(u.address, u.value, u.expire)
}

func (u *Unbond) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "Unbond{address=%s value=%s expire=%d}", u.address, u.value, u.expire)
		} else {
			fmt.Fprintf(f, "Unbond{%s %s %d}", u.address, u.value, u.expire)
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

func (ul Unbonds) Contains(addr module.Address) bool {
	for _, u := range ul {
		if u.Address().Equal(addr) {
			return true
		}
	}
	return false
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
		total.Add(total, b.Value())
	}
	return total
}

func (ul Unbonds) GetUnbondByAddress(address module.Address) (*Unbond, int) {
	for i, ub := range ul {
		if address.Equal(ub.Address()) {
			return ub, i
		}
	}
	return nil, -1
}

func (ul Unbonds) MapByAddr() map[string]*Unbond {
	newMap := make(map[string]*Unbond)
	for _, ub := range ul {
		key := icutils.ToKey(ub.Address())
		newMap[key] = ub
	}
	return newMap
}

func (ul Unbonds) ExpireRefCount() map[int64]int {
	newMap := make(map[int64]int)
	for _, ub := range ul {
		key := ub.Expire()
		newMap[key] = newMap[key] + 1
	}
	return newMap
}

func (ul *Unbonds) Add(address module.Address, value *big.Int, expireHeight int64) {
	unbond := NewUnbond(common.AddressToPtr(address), value, expireHeight)
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

func (ul *Unbonds) Slash(address module.Address, rate icmodule.Rate) (Unbonds, *big.Int, int64) {
	expire := int64(-1)
	amount := big.NewInt(0)
	newUnbonds := make(Unbonds, 0)

	for _, u := range *ul {
		if u.Address().Equal(address) {
			unbond := u.Clone()
			amount = unbond.Slash(rate)

			percent := rate.Percent()
			if percent < 100 {
				newUnbonds = append(newUnbonds, unbond)
			} else if percent == 100 {
				expire = unbond.Expire()
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
