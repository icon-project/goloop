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

type Voting interface {
	Iterator() icstate.VotingIterator
	ApplyVotes(deltas icstage.VoteList) error
}

type Bonding struct {
	icobject.NoDatabase
	icstate.Bonds
}

func (b *Bonding) Version() int {
	return 0
}

func (b *Bonding) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.Decode(&b.Bonds)
}

func (b *Bonding) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.Encode(b.Bonds)
}

func (b *Bonding) Equal(o icobject.Impl) bool {
	if d2, ok := o.(*Bonding); ok {
		return b.Bonds.Equal(d2.Bonds)
	} else {
		return false
	}
}

func (b *Bonding) Clone() *Bonding {
	if b == nil {
		return nil
	}
	nd := NewBonding()
	for _, ds := range b.Bonds {
		nd.Bonds = append(nd.Bonds,  ds.Clone())
	}
	return nd
}

func (b *Bonding) IsEmpty() bool {
	return len(b.Bonds) == 0
}

func (b *Bonding) ApplyVotes(deltas icstage.VoteList) error {
	var nBonds icstate.Bonds

	// add Bond not in old Bonds
	deltaMap := deltas.ToMap()
	for _, bond := range b.Bonds {
		_, ok := deltaMap[icutils.ToKey(bond.To())]
		if !ok {
			nBonds = append(nBonds, bond)
		}
	}

	// apply deltas
	bondMap := b.Bonds.ToMap()
	for _, vote := range deltas {
		bond, ok := bondMap[icutils.ToKey(vote.To())]
		if ok {
			value := new(big.Int).Add(bond.Amount(), vote.Amount())
			switch value.Sign() {
			case -1:
				return errors.Errorf("Negative bond to %s, value %d = %d - %d", vote.To(), value, bond.Amount(), vote.Amount())
			case 0:
				continue
			case 1:
				bond = icstate.NewBond(common.AddressToPtr(bond.To()), value)
			}
		} else {
			switch vote.Amount().Sign() {
			case -1:
				return errors.Errorf("Negative bond to %s, value %d", vote.To(), vote.Amount())
			case 0:
				continue
			case 1:
				bond = icstate.NewBond(common.AddressToPtr(vote.To()), vote.Amount())
			}
		}
		nBonds = append(nBonds, bond)
	}

	b.Bonds = nBonds
	return nil
}

func (b *Bonding) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "Bonding{%+v}", b.Bonds)
		} else {
			fmt.Fprintf(f, "Bonding{%v}", b.Bonds)
		}
	case 's':
		fmt.Fprintf(f, "%s", b.Bonds)
	}
}

func newBonding(_ icobject.Tag) *Bonding {
	return new(Bonding)
}

func NewBonding() *Bonding {
	d := new(Bonding)
	d.Bonds = make([]*icstate.Bond, 0)
	return d
}
