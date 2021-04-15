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
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
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
	Address *common.Address `json:"address"`
	Value   *common.HexInt  `json:"value"`
}

func NewDelegation() *Delegation {
	return &Delegation{
		Address: new(common.Address),
		Value:   new(common.HexInt),
	}
}

func (dg *Delegation) Clone() *Delegation {
	n := NewDelegation()
	n.Address.Set(dg.Address)
	n.Value.Set(dg.Value.Value())
	return n
}

func (dg *Delegation) Equal(d2 *Delegation) bool {
	if dg == d2 {
		return true
	}
	return dg.Address.Equal(d2.Address) &&
		dg.Value.Cmp(d2.Value.Value()) == 0
}

func (dg *Delegation) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})

	jso["address"] = dg.Address
	jso["value"] = dg.Value

	return jso
}

func (dg *Delegation) To() module.Address {
	return dg.Address
}

func (dg *Delegation) Amount() *big.Int {
	return dg.Value.Value()
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

func (ds *Delegations) Delete(i int) error {
	if i < 0 || i >= len(*ds) {
		return errors.Errorf("Invalid index")
	}

	copy((*ds)[i:], (*ds)[i+1:])
	(*ds)[len(*ds)-1] = nil // or the zero value of T
	*ds = (*ds)[:len(*ds)-1]
	return nil
}

func (ds Delegations) ToJSON(_ module.JSONVersion) []interface{} {
	jso := make([]interface{}, len(ds))
	for idx, d := range ds {
		jso[idx] = d.ToJSON()
	}
	return jso
}

func (ds *Delegations) getVotings() []Voting {
	size := len(*ds)
	votings := make([]Voting, size)
	if !ds.Has() {
		return votings
	}
	for i := 0; i < size; i++ {
		votings[i] = (*ds)[i]
	}
	return votings
}

func (ds *Delegations) Iterator() VotingIterator {
	if ds == nil {
		return nil
	}
	return NewVotingIterator(ds.getVotings())
}

func NewDelegations(param []interface{}) (Delegations, error) {
	count := len(param)
	if count > getMaxDelegationCount() {
		return nil, errors.Errorf("Too many delegations %d", count)
	}
	targets := make(map[string]struct{}, count)
	delegations := make([]*Delegation, 0)
	for _, p := range param {
		dg := NewDelegation()
		bs, err := json.Marshal(p)
		if err != nil {
			return nil, errors.IllegalArgumentError.Errorf("Failed to get delegation %v", err)
		}
		if err = json.Unmarshal(bs, dg); err != nil {
			return nil, errors.IllegalArgumentError.Errorf("Failed to get delegation %v", err)
		}
		if dg.Value.Sign() == -1 {
			return nil, errors.IllegalArgumentError.Errorf("Can not set negative value to delegation")
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
