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
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
)

type Delegation struct {
	Address *common.Address `json:"address"`
	Value   *common.HexInt  `json:"value"`
}

func NewDelegation(addr *common.Address, v *big.Int) *Delegation {
	return &Delegation{
		Address: addr,
		Value:   new(common.HexInt).SetValue(v),
	}
}

func (dg *Delegation) Clone() *Delegation {
	n := new(Delegation)
	n.Address = dg.Address
	n.Value = dg.Value
	return n
}

func (dg *Delegation) Equal(d2 *Delegation) bool {
	if dg == d2 {
		return true
	}
	if dg == nil || d2 == nil {
		return false
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

func (dg *Delegation) String() string {
	return fmt.Sprintf("{address=%s value=%s}", dg.Address, dg.Value)
}

func (dg *Delegation) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			_, _ = fmt.Fprintf(f, "Delegation{address=%s value=%s}", dg.Address, dg.Value)
		} else {
			_, _ = fmt.Fprintf(f, "Delegation{%s %s}", dg.Address, dg.Value)
		}
	case 's':
		_, _ = fmt.Fprint(f, dg.String())
	}
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

func (ds Delegations) Delta(ds2 Delegations) map[string]*big.Int {
	delta := make(map[string]*big.Int)

	for _, d := range ds {
		key := icutils.ToKey(d.To())
		delta[key] = new(big.Int).Neg(d.Amount())
	}
	for _, d := range ds2 {
		key := icutils.ToKey(d.To())
		if delta[key] == nil {
			delta[key] = new(big.Int)
		}
		delta[key].Add(delta[key], d.Amount())
	}
	return delta
}

func (ds Delegations) GetDelegationAmount() *big.Int {
	total := new(big.Int)
	for _, d := range ds {
		total.Add(total, d.Amount())
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

func (ds *Delegations) ToMap() map[string]*Delegation {
	if !ds.Has() {
		return nil
	}
	m := make(map[string]*Delegation, len(*ds))

	for _, d := range *ds {
		m[icutils.ToKey(d.To())] = d
	}
	return m
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

func NewDelegations(param []interface{}, max int) (Delegations, error) {
	count := len(param)
	if count > max {
		return nil, scoreresult.InvalidParameterError.Errorf("Too many delegations %d", count)
	}
	targets := make(map[string]struct{}, count)
	delegations := make([]*Delegation, 0, count)
	for _, p := range param {
		dg := new(Delegation)
		bs, err := json.Marshal(p)
		if err != nil {
			return nil, scoreresult.IllegalFormatError.Wrapf(err, "Failed to get delegation")
		}
		if err = json.Unmarshal(bs, dg); err != nil {
			return nil, scoreresult.IllegalFormatError.Wrapf(err, "Failed to get delegation")
		}
		target := icutils.ToKey(dg.To())
		if _, ok := targets[target]; ok {
			return nil, scoreresult.InvalidParameterError.Errorf("Duplicated delegation address")
		}
		targets[target] = struct{}{}
		switch dg.Amount().Sign() {
		case -1:
			return nil, scoreresult.InvalidParameterError.Errorf("Can not set negative value to delegation")
		case 0:
			continue
		}
		delegations = append(delegations, dg)
	}

	return delegations, nil
}

var IllegalDelegationPrefix = containerdb.ToKey(
	containerdb.HashBuilder,
	scoredb.DictDBPrefix,
	"illegal_delegation",
)

type IllegalDelegation struct {
	icobject.NoDatabase

	address     *common.Address
	delegations Delegations
}

func NewIllegalDelegationWithTag(_ icobject.Tag) *IllegalDelegation {
	return new(IllegalDelegation)
}

func NewIllegalDelegation(addr module.Address, ds Delegations) *IllegalDelegation {
	return &IllegalDelegation{
		address:     common.AddressToPtr(addr),
		delegations: ds,
	}
}

func (d *IllegalDelegation) Version() int {
	return 1
}

func (d *IllegalDelegation) Address() module.Address {
	return d.address
}

func (d *IllegalDelegation) Delegations() Delegations {
	return d.delegations
}

func (d *IllegalDelegation) SetDelegations(ds Delegations) {
	d.delegations = ds
}

func (d *IllegalDelegation) Clone() *IllegalDelegation {
	return &IllegalDelegation{
		address: d.address,
		delegations:  d.delegations,
	}
}

func (d *IllegalDelegation) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.DecodeAll(
		&d.address,
		&d.delegations,
	)
}

func (d *IllegalDelegation) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		d.address,
		d.delegations,
	)
}

func (d *IllegalDelegation) Equal(o icobject.Impl) bool {
	if d2, ok := o.(*IllegalDelegation); ok {
		return d.address.Equal(d2.address) &&
			d.delegations.Equal(d2.delegations)
	} else {
		return false
	}
}

func (d *IllegalDelegation) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			_, _ = fmt.Fprintf(f, "IllegalDelegation{address=%s delegations=%+v}",
				d.address, d.delegations)
		} else {
			_, _ = fmt.Fprintf(f, "IllegalDelegation{%s %v}", d.address, d.delegations)
		}
	}
}
