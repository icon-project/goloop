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
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type Voted struct {
	icobject.NoDatabase
	Enable           bool     // update via ENABLE event
	Delegated        *big.Int // update via DELEGATE event
	Bonded           *big.Int // update via BOND event
	BondedDelegation *big.Int // update when start calculation for P-Rep voted reward
}

func (v *Voted) Version() int {
	return 0
}

func (v *Voted) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&v.Enable, &v.Delegated, &v.Bonded, &v.BondedDelegation)
	return err
}

func (v *Voted) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(v.Enable, v.Delegated, v.Bonded, v.BondedDelegation)
}

func (v *Voted) Equal(o icobject.Impl) bool {
	if ic2, ok := o.(*Voted); ok {
		return v.Enable == ic2.Enable &&
			v.Delegated.Cmp(ic2.Delegated) == 0 &&
			v.Bonded.Cmp(ic2.Bonded) == 0 &&
			v.BondedDelegation.Cmp(ic2.BondedDelegation) == 0
	} else {
		return false
	}
}

func (v *Voted) Clone() *Voted {
	if v == nil {
		return nil
	}
	nv := NewVoted()
	nv.Enable = v.Enable
	nv.Delegated.Set(v.Delegated)
	nv.Bonded.Set(v.Bonded)
	nv.BondedDelegation.Set(v.BondedDelegation)
	return nv
}

func (v *Voted) IsEmpty() bool {
	return v.Enable == false && v.Delegated.Sign() == 0 && v.Bonded.Sign() == 0 && v.BondedDelegation.Sign() == 0
}

func (v *Voted) SetEnable(enable bool) {
	v.Enable = enable
}

func (v *Voted) SetBonded(bonded *big.Int) {
	v.Bonded.Set(bonded)
}

func (v *Voted) UpdateBondedDelegation(bondRequirement int) {
	if bondRequirement == 0 {
		// IISSVersion1: bondedDelegation = delegated
		// IISSVersion2 and bondRequirement is disabled: bondedDelegation = delegated + bonded
		v.BondedDelegation.Set(new(big.Int).Add(v.Delegated, v.Bonded))
	} else {
		// IISSVersion2 and bondRequirement is enabled
		voted := new(big.Int).Add(v.Delegated, v.Bonded)
		bondedDelegation := new(big.Int).Mul(v.Bonded, big.NewInt(100))
		bondedDelegation.Div(bondedDelegation, big.NewInt(int64(bondRequirement)))
		if voted.Cmp(bondedDelegation) > 0 {
			v.BondedDelegation.Set(bondedDelegation)
		} else {
			v.BondedDelegation.Set(voted)
		}
	}
}

func (v *Voted) GetVoted() *big.Int {
	return new(big.Int).Add(v.Bonded, v.Delegated)
}

func newVoted(tag icobject.Tag) *Voted {
	return NewVoted()
}

func NewVoted() *Voted {
	return &Voted{
		Delegated:        new(big.Int),
		Bonded:           new(big.Int),
		BondedDelegation: new(big.Int),
	}
}
