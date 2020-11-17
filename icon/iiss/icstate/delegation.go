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
	"encoding/json"
	"github.com/icon-project/goloop/common/errors"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

const (
	maxDelegations = 100
)

var maxDelegationCount = maxDelegations

func getMaxDelegationCount() int {
	return maxDelegationCount
}

func setMaxDelegationCount(v int) {
	if v == 0 {
		maxDelegationCount = maxDelegations
	} else {
		maxDelegationCount = v
	}
}

type Delegation struct {
	Address *common.Address	`json:"address"`
	Value   *common.HexInt	`json:"value"`
}

func newDelegation() *Delegation {
	return &Delegation{
		Address: new(common.Address),
		Value: new(common.HexInt),
	}
}

func (d *Delegation) Clone() *Delegation {
	n := newDelegation()
	n.Address.Set(d.Address)
	n.Value.Set(d.Value.Value())
	return n
}

func (d *Delegation) Equal(d2 *Delegation) bool {
	if d == d2 {
		return true
	}
	return d.Address.Equal(d2.Address) &&
		d.Value.Cmp(d2.Value.Value()) == 0
}

func (d *Delegation) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})

	jso["address"] = d.Address
	jso["value"] = d.Value

	return jso
}

type Delegations []*Delegation

func (ds Delegations) Has() bool {
	return len(ds) > 0
}

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

func (ds Delegations) GetDelegationAmount() *big.Int {
	total := new(big.Int)
	for _, d := range ds {
		total.Add(total, d.Value.Value())
	}
	return total
}

func (ds Delegations) ToJSON(v module.JSONVersion) []interface{} {
	if !ds.Has() {
		return nil
	}
	delegations := make([]interface{}, len(ds))

	for idx, d := range ds {
		delegations[idx] = d.ToJSON()
	}
	return delegations
}

func NewDelegations(param []interface{}) (Delegations, error) {
	count := len(param)
	if count > getMaxDelegationCount() {
		return nil, errors.Errorf("Too many delegations %d", count)
	}
	targets := make(map[string]struct{}, count)
	delegations := make([]*Delegation, 0)
	for _, p := range param {
		dg := newDelegation()
		bs, err := json.Marshal(p)
		if err != nil {
			return nil, errors.IllegalArgumentError.Errorf("Failed to get delegation %v", err)
		}
		if err = json.Unmarshal(bs, dg); err != nil {
			return nil, errors.IllegalArgumentError.Errorf("Failed to get delegation %v", err)
		}
		target := dg.Address.String()
		if _, ok := targets[target]; ok {
			return nil, errors.IllegalArgumentError.Errorf("Duplicated delegation address")
		}
		targets[target] = struct{}{}
		delegations = append(delegations, dg)
	}

	return delegations, nil
}

