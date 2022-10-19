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

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type Voted struct {
	icobject.NoDatabase
	enable           bool     // update via ENABLE event
	delegated        *big.Int // update via DELEGATE event
	bonded           *big.Int // update via BOND event
	bondedDelegation *big.Int // update when start calculation for P-Rep voted reward
}

func (v *Voted) Version() int {
	return 0
}

func (v *Voted) Enable() bool {
	return v.enable
}

func (v *Voted) SetEnable(enable bool) {
	v.enable = enable
}

func (v *Voted) Delegated() *big.Int {
	return v.delegated
}

func (v *Voted) SetDelegated(value *big.Int) {
	v.delegated = value
}

func (v *Voted) Bonded() *big.Int {
	return v.bonded
}

func (v *Voted) SetBonded(value *big.Int) {
	v.bonded = value
}

func (v *Voted) BondedDelegation() *big.Int {
	return v.bondedDelegation
}

func (v *Voted) SetBondedDelegation(value *big.Int) {
	v.bondedDelegation = value
}

func (v *Voted) UpdateBondedDelegation(bondRequirement int) {
	if bondRequirement == 0 {
		// IISS 2: bondedDelegation = delegated
		// IISS 3 and bondRequirement is disabled: bondedDelegation = delegated + bonded
		v.bondedDelegation = new(big.Int).Add(v.delegated, v.bonded)
	} else {
		// IISS 3 and bondRequirement is enabled
		voted := new(big.Int).Add(v.delegated, v.bonded)
		bondedDelegation := new(big.Int).Mul(v.bonded, big.NewInt(100))
		bondedDelegation.Div(bondedDelegation, big.NewInt(int64(bondRequirement)))
		if voted.Cmp(bondedDelegation) > 0 {
			v.bondedDelegation = bondedDelegation
		} else {
			v.bondedDelegation = voted
		}
	}
}

func (v *Voted) GetVotedAmount() *big.Int {
	return new(big.Int).Add(v.bonded, v.delegated)
}

func (v *Voted) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&v.enable, &v.delegated, &v.bonded, &v.bondedDelegation)
	return err
}

func (v *Voted) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(v.enable, v.delegated, v.bonded, v.bondedDelegation)
}

func (v *Voted) Equal(o icobject.Impl) bool {
	if ic2, ok := o.(*Voted); ok {
		return v.enable == ic2.enable &&
			v.delegated.Cmp(ic2.delegated) == 0 &&
			v.bonded.Cmp(ic2.bonded) == 0 &&
			v.bondedDelegation.Cmp(ic2.bondedDelegation) == 0
	} else {
		return false
	}
}

func (v *Voted) Clone() *Voted {
	if v == nil {
		return nil
	}
	nv := new(Voted)
	nv.enable = v.enable
	nv.delegated = v.delegated
	nv.bonded = v.bonded
	nv.bondedDelegation = v.bondedDelegation
	return nv
}

func (v *Voted) IsEmpty() bool {
	return v.enable == false && v.delegated.Sign() == 0 && v.bonded.Sign() == 0 && v.bondedDelegation.Sign() == 0
}

func (v *Voted) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "Voted{enable=%v delegated=%d bonded=%d bondedDelegation=%d}",
				v.enable, v.delegated, v.bonded, v.bondedDelegation)
		} else {
			fmt.Fprintf(f, "Voted{%v %d %d %d}",
				v.enable, v.delegated, v.bonded, v.bondedDelegation)
		}
	case 's':
		fmt.Fprintf(f, "enable=%v delegated=%d bonded=%d bondedDelegation=%d",
			v.enable, v.delegated, v.bonded, v.bondedDelegation)
	}
}

func newVoted(_ icobject.Tag) *Voted {
	return new(Voted)
}

func NewVoted() *Voted {
	return &Voted{
		delegated:        new(big.Int),
		bonded:           new(big.Int),
		bondedDelegation: new(big.Int),
	}
}
