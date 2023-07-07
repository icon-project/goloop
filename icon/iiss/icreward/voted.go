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
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

const (
	votedVersion1 = iota
	votedVersion2
)

type Voted struct {
	icobject.NoDatabase
	version          int
	enable           bool     // update via ENABLE event
	delegated        *big.Int // update via DELEGATE event
	bonded           *big.Int // update via BOND event
	bondedDelegation *big.Int // update when start calculation for P-Rep voted reward
	commissionRate   *big.Int
}

func (v *Voted) Version() int {
	return v.version
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

func (v *Voted) CommissionRate() *big.Int {
	return v.commissionRate
}

func (v *Voted) SetCommissionRate(value *big.Int) {
	v.commissionRate = value
	v.version = votedVersion2
}

func (v *Voted) RLPDecodeFields(decoder codec.Decoder) error {
	var err error
	switch v.version {
	case votedVersion1:
		v.commissionRate = new(big.Int)
		_, err = decoder.DecodeMulti(&v.enable, &v.delegated, &v.bonded, &v.bondedDelegation)
	case votedVersion2:
		v.bondedDelegation = new(big.Int)
		_, err = decoder.DecodeMulti(&v.enable, &v.delegated, &v.bonded, &v.commissionRate)
	}
	return err
}

func (v *Voted) RLPEncodeFields(encoder codec.Encoder) error {
	switch v.version {
	case votedVersion1:
		return encoder.EncodeMulti(v.enable, v.delegated, v.bonded, v.bondedDelegation)
	case votedVersion2:
		return encoder.EncodeMulti(v.enable, v.delegated, v.bonded, v.commissionRate)
	default:
		return errors.IllegalArgumentError.Errorf("illegal Voted version %d", v.version)
	}
}

func (v *Voted) Equal(o icobject.Impl) bool {
	if v2, ok := o.(*Voted); ok {
		return v.enable == v2.enable &&
			v.delegated.Cmp(v2.delegated) == 0 &&
			v.bonded.Cmp(v2.bonded) == 0 &&
			v.bondedDelegation.Cmp(v2.bondedDelegation) == 0 &&
			v.commissionRate.Cmp(v2.commissionRate) == 0
	} else {
		return false
	}
}

func (v *Voted) Clone() *Voted {
	if v == nil {
		return nil
	}
	nv := NewVoted()
	nv.version = v.version
	nv.enable = v.enable
	nv.delegated.Set(v.delegated)
	nv.bonded.Set(v.bonded)
	nv.bondedDelegation.Set(v.bondedDelegation)
	nv.commissionRate.Set(v.commissionRate)
	return nv
}

func (v *Voted) IsEmpty() bool {
	return v.enable == false && v.delegated.Sign() == 0 && v.bonded.Sign() == 0 && v.bondedDelegation.Sign() == 0
}

func (v *Voted) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "Voted{version=%d enable=%v delegated=%d bonded=%d bondedDelegation=%d commissionRate=%d}",
				v.version, v.enable, v.delegated, v.bonded, v.bondedDelegation, v.commissionRate)
		} else {
			fmt.Fprintf(f, "Voted{%d %v %d %d %d %d}",
				v.version, v.enable, v.delegated, v.bonded, v.bondedDelegation, v.commissionRate)
		}
	case 's':
		fmt.Fprintf(f, "version=%d enable=%v delegated=%d bonded=%d bondedDelegation=%d commissionRate=%d",
			v.version, v.enable, v.delegated, v.bonded, v.bondedDelegation, v.commissionRate)
	}
}

func newVoted(tag icobject.Tag) *Voted {
	v := NewVoted()
	v.version = tag.Version()
	return v
}

func NewVoted() *Voted {
	return &Voted{
		delegated:        new(big.Int),
		bonded:           new(big.Int),
		bondedDelegation: new(big.Int),
		commissionRate:   new(big.Int),
	}
}
