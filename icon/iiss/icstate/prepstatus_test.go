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
	"math/big"
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/bmizerany/assert"
)

func TestPRepStatus_Bytes(t *testing.T) {
	owner := common.NewAccountAddress(make([]byte, common.AddressIDBytes, common.AddressIDBytes))
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	ss1 := newPRepStatusWithTag(icobject.MakeTag(TypePRepStatus, prepStatusVersion))
	g := Candidate
	ss1.grade = g
	ss1.SetOwner(owner)

	o1 := icobject.New(TypePRepStatus, ss1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())

	ss2 := ToPRepStatus(o2, owner)
	assert.Equal(t, true, ss1.Equal(ss2))
	assert.Equal(t, false, ss2.readonly)
	assert.Equal(t, true, ss1.owner.Equal(owner))
	assert.Equal(t, true, ss2.owner.Equal(owner))
}

// test for GetBondedDelegation
func TestPRepStatus_GetBondedDelegation(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	s := NewStateFromSnapshot(NewSnapshot(database, nil), false)

	addr1 := common.NewAddressFromString("hx1")

	status1 := NewPRepStatus(addr1)
	base := NewPRepBase(addr1)
	s.AddPRepBase(base)
	s.AddPRepStatus(status1)

	delegated := big.NewInt(int64(99))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded := big.NewInt(int64(1))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res := status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(20))))

	delegated = big.NewInt(int64(99))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(2))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(40))))

	delegated = big.NewInt(int64(93))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(7))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(100))))

	delegated = big.NewInt(int64(90))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(10))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(100))))

	// 0 input, exptected 0 output
	delegated = big.NewInt(int64(0))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(0))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(0))))

	// extreme
	delegated = big.NewInt(int64(99999999999))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(19980))))

	// different requirement
	delegated = big.NewInt(int64(99999))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(4)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(24975))))

	// 0 for bond requirement
	delegated = big.NewInt(int64(99999))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(0)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(0))))

	// 101 for bond requirement
	delegated = big.NewInt(int64(99999))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(101)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(0))))

	// 1000 for bond requirement
	delegated = big.NewInt(int64(99999))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(100)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(999))))
}

func TestPRepStatus_getContValue(t *testing.T) {
	type args struct {
		lastState   ValidationState
		lastBH      int64
		blockHeight int64
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"Fail state",
			args{
				Fail,
				0,
				10,
			},
			11,
		},
		{
			"Success state",
			args{
				Success,
				10,
				22000,
			},
			21991,
		},
		{
			"None state",
			args{
				None,
				0,
				1000,
			},
			0,
		},
		{
			"Invalid block height",
			args{
				Fail,
				100,
				1,
			},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			ps := &PRepStatus{
				lastState:  in.lastState,
				lastHeight: in.lastBH,
			}

			ret := ps.getContValue(in.blockHeight)

			assert.Equal(t, tt.want, ret)
		})
	}
}

func TestPRepStatus_GetVTotal(t *testing.T) {
	type args struct {
		vTotal      int
		lastState   ValidationState
		lastBH      int64
		blockHeight int64
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"Fail state",
			args{
				10,
				Fail,
				15,
				20,
			},
			10 + 20 - 15 + 1,
		},
		{
			"Success state",
			args{
				20,
				Success,
				50,
				22000,
			},
			20 + 22000 - 50 + 1,
		},
		{
			"None state",
			args{
				100,
				None,
				200,
				1000,
			},
			100,
		},
		{
			"Invalid block height",
			args{
				100,
				Fail,
				200,
				1,
			},
			100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			ps := &PRepStatus{
				vTotal:     in.vTotal,
				lastState:  in.lastState,
				lastHeight: in.lastBH,
			}

			ret := ps.GetVTotal(in.blockHeight)

			assert.Equal(t, tt.want, ret)
		})
	}
}

func TestPRepStatus_GetVFail(t *testing.T) {
	type args struct {
		vFail       int
		lastState   ValidationState
		lastBH      int64
		blockHeight int64
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"Fail state",
			args{
				10,
				Fail,
				15,
				20,
			},
			10 + 20 - 15 + 1,
		},
		{
			"Success state",
			args{
				20,
				Success,
				50,
				22000,
			},
			20,
		},
		{
			"None state",
			args{
				100,
				None,
				200,
				1000,
			},
			100,
		},
		{
			"Invalid block height",
			args{
				100,
				Fail,
				200,
				1,
			},
			100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			ps := &PRepStatus{
				vFail:      in.vFail,
				lastState:  in.lastState,
				lastHeight: in.lastBH,
			}

			ret := ps.GetVFail(in.blockHeight)

			assert.Equal(t, tt.want, ret)
		})
	}
}

func TestPRepStatus_GetVFailCont(t *testing.T) {
	type args struct {
		lastState   ValidationState
		lastBH      int64
		blockHeight int64
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"Fail state",
			args{
				Fail,
				15,
				20,
			},
			20 - 15 + 1,
		},
		{
			"Success state",
			args{
				Success,
				50,
				22000,
			},
			0,
		},
		{
			"None state",
			args{
				None,
				200,
				1000,
			},
			0,
		},
		{
			"Invalid block height",
			args{
				Fail,
				200,
				1,
			},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			ps := &PRepStatus{
				lastState:  in.lastState,
				lastHeight: in.lastBH,
			}

			ret := ps.GetVFailCont(in.blockHeight)

			assert.Equal(t, tt.want, ret)
		})
	}
}

func TestPRepStatus_ShiftVPenaltyMask(t *testing.T) {
	type args struct {
		vPenaltyMask uint32
		mask         uint32
	}

	tests := []struct {
		name string
		args args
		want uint32
	}{
		{
			"Normal",
			args{
				0x3,
				0x3fffffff,
			},
			0x6,
		},
		{
			"Masked",
			args{
				0x3fffffff,
				0x3fffffff,
			},
			0x3ffffffe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			ps := &PRepStatus{
				vPenaltyMask:  in.vPenaltyMask,
			}

			ps.ShiftVPenaltyMask(in.mask)

			assert.Equal(t, tt.want, ps.vPenaltyMask)
		})
	}
}

