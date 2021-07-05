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

package icreward

import (
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
)

type Delegating struct {
	icobject.NoDatabase
	icstate.Delegations
}

func (d *Delegating) Version() int {
	return 0
}

func (d *Delegating) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.Decode(&d.Delegations)
}

func (d *Delegating) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.Encode(d.Delegations)
}

func (d *Delegating) Equal(o icobject.Impl) bool {
	if d2, ok := o.(*Delegating); ok {
		return d.Delegations.Equal(d2.Delegations)
	} else {
		return false
	}
}

func (d *Delegating) Clone() *Delegating {
	if d == nil {
		return nil
	}
	nd := NewDelegating()
	for _, ds := range d.Delegations {
		nd.Delegations = append(nd.Delegations,  ds.Clone())
	}
	return nd
}

func (d *Delegating) IsEmpty() bool {
	return len(d.Delegations) == 0
}

func (d *Delegating) ApplyVotes(deltas icstage.VoteList) error {
	var nDelegations icstate.Delegations

	// add Delegation not in old Delegations
	deltaMap := deltas.ToMap()
	for _, dg := range d.Delegations {
		_, ok := deltaMap[icutils.ToKey(dg.To())]
		if !ok {
			nDelegations = append(nDelegations, dg)
		}
	}

	// apply deltas
	delegationMap := d.Delegations.ToMap()
	for _, vote := range deltas {
		dg, ok := delegationMap[icutils.ToKey(vote.To())]
		if ok {
			value := new(big.Int).Add(dg.Amount(), vote.Amount())
			switch value.Sign() {
			case -1:
				return errors.Errorf("Negative delegation to %s, value %d = %d - %d", vote.To(), value, dg.Amount(), vote.Amount())
			case 0:
				continue
			case 1:
				dg = icstate.NewDelegation(common.AddressToPtr(dg.To()), value)
			}
		} else {
			switch vote.Amount().Sign() {
			case -1:
				return errors.Errorf("Negative delegation to %s, value %d", vote.To(), vote.Amount())
			case 0:
				continue
			case 1:
				dg = icstate.NewDelegation(common.AddressToPtr(vote.To()), vote.Amount())
			}
		}
		nDelegations = append(nDelegations, dg)
	}

	d.Delegations = nDelegations
	return nil
}

func (d *Delegating) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "Delegating{%+v}", d.Delegations)
		} else {
			fmt.Fprintf(f, "Delegating{%v}", d.Delegations)
		}
	case 's':
		fmt.Fprintf(f, "%s", d.Delegations)
	}
}

func newDelegating(_ icobject.Tag) *Delegating {
	return new(Delegating)
}

func NewDelegating() *Delegating {
	d := new(Delegating)
	d.Delegations = make([]*icstate.Delegation, 0)
	return d
}
