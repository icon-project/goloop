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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

type Delegation struct {
	Target common.Address `jso:"target"`
	Amount common.HexInt  `jso:"amount"`
}

func (d *Delegation) Clone() *Delegation {
	n := new(Delegation)
	n.Target.Set(&d.Target)
	n.Amount.Set(&d.Amount.Int)
	return n
}

func (d *Delegation) Equal(d2 *Delegation) bool {
	if d == d2 {
		return true
	}
	return d.Target.Equal(&d2.Target) &&
		d.Amount.Cmp(d2.Amount.Value()) == 0
}

type Delegations []*Delegation

func (ds Delegations) Clone() Delegations {
	if ds == nil {
		return nil
	}
	ns := make([]*Delegation, len(ds))
	for i, d := range ds {
		ns[i] = d.Clone()
	}
	return ns
}

func (ds Delegations) Equal(ds2 Delegations) bool {
	if len(ds) != len(ds2) {
		return false
	}
	for i, d := range ds {
		if !d.Equal(ds2[i]) {
			return false
		}
	}
	return true
}

func (ds *Delegations) AddDelegation(addr module.Address, amount *big.Int) {
	for _, d := range *ds {
		if d.Target.Equal(addr) {
			d.Amount.Add(&d.Amount.Int, amount)
			return
		}
	}
	d := new(Delegation)
	d.Target.Set(addr)
	d.Amount.Set(amount)
	*ds = append(*ds, d)
}
