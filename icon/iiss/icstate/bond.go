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
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icutils"
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

func NewBond(addr *common.Address, v *big.Int) *Bond {
	return &Bond{
		Address: addr,
		Value:   new(common.HexInt).SetValue(v),
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
	n := new(Bond)
	n.Address = b.Address
	n.Value = b.Value
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
	nBigInt := new(big.Int).Sub(b.Value.Value(), slashAmount)
	b.Value = new(common.HexInt).SetValue(nBigInt)
	return slashAmount
}

func (b *Bond) String() string {
	return fmt.Sprintf("{address=%s, value=%s}", b.Address, b.Value)
}

func (b *Bond) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "Bond{address=%s value=%s}", b.Address, b.Value)
		} else {
			fmt.Fprintf(f, "Bond{%s %s}", b.Address, b.Value)
		}
	case 's':
		fmt.Fprint(f, b.String())
	}
}

type Bonds []*Bond

func (bs *Bonds) Has() bool {
	return len(*bs) > 0
}

func (bs *Bonds) Equal(bl2 Bonds) bool {
	if len(*bs) != len(bl2) {
		return false
	}
	for i, b := range *bs {
		if !b.Equal(bl2[i]) {
			return false
		}
	}
	return true
}

func (bs *Bonds) Clone() Bonds {
	if *bs == nil {
		return nil
	}
	bonds := make([]*Bond, len(*bs))
	for i, b := range *bs {
		bonds[i] = b.Clone()
	}
	return bonds
}

func (bs *Bonds) GetBondAmount() *big.Int {
	total := new(big.Int)
	for _, b := range *bs {
		total.Add(total, b.Amount())
	}
	return total
}

func (bs *Bonds) Delete(i int) error {
	if i < 0 || i >= len(*bs) {
		return errors.Errorf("Invalid index")
	}

	copy((*bs)[i:], (*bs)[i+1:])
	(*bs)[len(*bs)-1] = nil // or the zero value of T
	*bs = (*bs)[:len(*bs)-1]
	return nil
}

func (bs *Bonds) Slash(address module.Address, ratio int) *big.Int {
	bonds := *bs
	for idx, b := range *bs {
		if b.To().Equal(address) {
			if ratio == 100 {
				copy(bonds[idx:], bonds[idx+1:])
				bonds = bonds[0 : len(bonds)-1]
				if len(bonds) > 0 {
					*bs = bonds
				} else {
					*bs = nil
				}
				return b.Amount()
			} else {
				return b.Slash(ratio)
			}
		}
	}
	return new(big.Int)
}

func (bs *Bonds) ToJSON(v module.JSONVersion) []interface{} {
	if !bs.Has() {
		return nil
	}
	bonds := make([]interface{}, len(*bs))

	for idx, b := range *bs {
		bonds[idx] = b.ToJSON()
	}
	return bonds
}

func (bs *Bonds) ToMap() map[string]*Bond {
	if !bs.Has() {
		return nil
	}
	m := make(map[string]*Bond, len(*bs))

	for _, b := range *bs {
		m[icutils.ToKey(b.To())] = b
	}
	return m
}

func (bs *Bonds) getVotings() []Voting {
	size := len(*bs)
	votings := make([]Voting, size)
	if !bs.Has() {
		return votings
	}
	for i := 0; i < size; i++ {
		votings[i] = (*bs)[i]
	}
	return votings
}

func (bs *Bonds) Iterator() VotingIterator {
	if bs == nil {
		return nil
	}
	return NewVotingIterator(bs.getVotings())
}

func NewBonds(param []interface{}) (Bonds, error) {
	count := len(param)
	if count > getMaxBondsCount() {
		return nil, errors.Errorf("Too many bonds %d", count)
	}
	targets := make(map[string]struct{}, count)
	bonds := make([]*Bond, 0)
	for _, p := range param {
		bond := new(Bond)
		bs, err := json.Marshal(p)
		if err != nil {
			return nil, errors.IllegalArgumentError.Errorf("Failed to get bond %v", err)
		}
		if err = json.Unmarshal(bs, bond); err != nil {
			return nil, errors.IllegalArgumentError.Errorf("Failed to get bond %v", err)
		}
		if bond.Amount().Sign() == -1 {
			return nil, errors.IllegalArgumentError.Errorf("Can not set negative value to bond")
		}
		target := icutils.ToKey(bond.To())
		if _, ok := targets[target]; ok {
			return nil, errors.IllegalArgumentError.Errorf("Duplicated bond Address")
		}
		targets[target] = struct{}{}
		bonds = append(bonds, bond)
	}
	return bonds, nil
}
