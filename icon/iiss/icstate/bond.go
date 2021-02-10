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
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

const (
	maxBonds = 100
)

var maxBondCount = maxBonds

func getMaxBondsCount() int {
	return maxBondCount
}

func setMaxBondCount(v int) {
	if v > 0 {
		maxBondCount = v
	}
}

type Bond struct {
	Address *common.Address `json:"address"`
	Value   *common.HexInt  `json:"value"`
}

func NewBond() *Bond {
	return &Bond{
		Address: new(common.Address),
		Value:   new(common.HexInt),
	}
}

func (b *Bond) Equal(b2 *Bond) bool {
	return b.Address.Equal(b2.Address) && b.Value.Cmp(b2.Value.Value()) == 0
}

func (b *Bond) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["address"] = b.Address
	jso["value"] = b.Value
	return jso
}

func (b *Bond) Clone() *Bond {
	n := NewBond()
	n.Address.Set(b.Address)
	n.Value.Set(b.Value.Value())
	return n
}

func (b *Bond) To() module.Address {
	return b.Address
}

func (b *Bond) Amount() *big.Int {
	return b.Value.Value()
}

func (b *Bond) Slash(ratio int) *big.Int {
	slashAmount := new(big.Int).Mul(b.Value.Value(), big.NewInt(int64(ratio)))
	slashAmount.Div(slashAmount, big.NewInt(int64(100)))
	b.Value.Sub(b.Value.Value(), slashAmount)
	return slashAmount
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
	for i, b := range bl {
		bonds[i] = b.Clone()
	}
	return bonds
}

func (bl Bonds) GetBondAmount() *big.Int {
	total := new(big.Int)
	for _, b := range bl {
		total.Add(total, b.Value.Value())
	}
	return total
}

func (bl *Bonds) Delete(i int) error {
	if i < 0 || i >= len(*bl) {
		return errors.Errorf("Invalid index")
	}

	copy((*bl)[i:], (*bl)[i+1:])
	(*bl)[len(*bl)-1] = nil // or the zero value of T
	*bl = (*bl)[:len(*bl)-1]
	return nil
}

func (bl *Bonds) Slash(address module.Address, ratio int) *big.Int {
	bonds := *bl
	for idx, b := range *bl {
		if b.Address.Equal(address) {
			if ratio == 100 {
				copy(bonds[idx:], bonds[idx+1:])
				bonds = bonds[0 : len(bonds)-1]
				if len(bonds) > 0 {
					*bl = bonds
				} else {
					*bl = nil
				}
				return b.Value.Value()
			} else {
				return b.Slash(ratio)
			}
		}
	}
	return new(big.Int)
}

func (bl Bonds) ToJSON(v module.JSONVersion) []interface{} {
	if !bl.Has() {
		return nil
	}
	bonds := make([]interface{}, len(bl))

	for idx, b := range bl {
		bonds[idx] = b.ToJSON()
	}
	return bonds
}

func (bl *Bonds) getVotings() []Voting {
	size := len(*bl)
	votings := make([]Voting, size)
	if !bl.Has() {
		return votings
	}
	for i := 0; i < size; i++ {
		votings[i] = (*bl)[i]
	}
	return votings
}

func (bl *Bonds) Iterator() VotingIterator {
	if bl == nil {
		return nil
	}
	return NewVotingIterator(bl.getVotings())
}

func NewBonds(param []interface{}) (Bonds, error) {
	count := len(param)
	if count > getMaxBondsCount() {
		return nil, errors.Errorf("Too many bonds %d", count)
	}
	targets := make(map[string]struct{}, count)
	bonds := make([]*Bond, 0)
	for _, p := range param {
		bond := NewBond()
		bs, err := json.Marshal(p)
		if err != nil {
			return nil, errors.IllegalArgumentError.Errorf("Failed to get bond %v", err)
		}
		if err = json.Unmarshal(bs, bond); err != nil {
			return nil, errors.IllegalArgumentError.Errorf("Failed to get bond %v", err)
		}
		if bond.Value.Sign() == -1 {
			return nil, errors.IllegalArgumentError.Errorf("Can not set negative value to bond")
		}
		target := bond.Address.String()
		if _, ok := targets[target]; ok {
			return nil, errors.IllegalArgumentError.Errorf("Duplicated bond Address")
		}
		targets[target] = struct{}{}
		bonds = append(bonds, bond)
	}
	return bonds, nil
}
