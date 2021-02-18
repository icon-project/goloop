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
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
)

type Voting interface {
	Iterator() icstate.VotingIterator
	ApplyVotes(deltas icstage.VoteList) error
}

type Bonding struct {
	icobject.NoDatabase
	icstate.Bonds
}

func (d *Bonding) Version() int {
	return 0
}

func (d *Bonding) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.Decode(&d.Bonds)
}

func (d *Bonding) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.Encode(d.Bonds)
}

func (d *Bonding) Equal(o icobject.Impl) bool {
	if d2, ok := o.(*Bonding); ok {
		return d.Bonds.Equal(d2.Bonds)
	} else {
		return false
	}
}

func (d *Bonding) Clone() *Bonding {
	if d == nil {
		return nil
	}
	nd := NewBonding()
	for _, ds := range d.Bonds {
		nd.Bonds = append(nd.Bonds,  ds.Clone())
	}
	return nd
}

func (d *Bonding) IsEmpty() bool {
	return len(d.Bonds) == 0
}

func (d *Bonding) ApplyVotes(deltas icstage.VoteList) error {
	var index int
	bonds := d.Bonds.Clone()
	add := make([]*icstate.Bond, 0)
	for _, vote := range deltas {
		index = -1
		for i, bond := range bonds {
			if bond.To().Equal(vote.To()) {
				index = i
				bond.Amount().Add(bond.Amount(), vote.Amount())
				switch bond.Amount().Sign() {
				case -1:
					return errors.Errorf("Negative bond value %d", bond.Amount().Int64())
				case 0:
					if err := bonds.Delete(i); err != nil {
						return err
					}
				}
				break
			}
		}
		if index == -1 { // add new bond
			if vote.Value.Sign() != 1 {
				return errors.Errorf("Negative bond value %v", vote)
			}
			nb := icstate.NewBond()
			nb.Address.Set(vote.To())
			nb.Amount().Set(vote.Amount())
			add = append(add, nb)
		}
	}
	bonds = append(bonds, add...)
	d.Bonds = bonds
	return nil
}

func newBonding(tag icobject.Tag) *Bonding {
	return NewBonding()
}

func NewBonding() *Bonding {
	d := new(Bonding)
	d.Bonds = make([]*icstate.Bond, 0)
	return d
}
