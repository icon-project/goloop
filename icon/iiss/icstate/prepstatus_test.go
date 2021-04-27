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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestPRepStatus_Bytes(t *testing.T) {
	owner := common.NewAccountAddress(make([]byte, common.AddressIDBytes, common.AddressIDBytes))
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	ss1 := NewPRepStatus()
	g := Candidate
	ss1.grade = g

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
	assert.Equal(t, true, ss2.Equal(ss1))
	assert.Equal(t, false, ss2.readonly)
}

// test for GetBondedDelegation
func TestPRepStatus_GetBondedDelegation(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	s := NewStateFromSnapshot(NewSnapshot(database, nil), false)

	addr1 := common.MustNewAddressFromString("hx1")

	delegated := big.NewInt(int64(99))
	status1 := s.GetPRepStatus(addr1, true)
	status1.SetDelegated(delegated)
	bonded := big.NewInt(int64(1))
	status1.SetBonded(bonded)
	res := status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(20))))

	delegated = big.NewInt(int64(99))
	s.GetPRepStatus(addr1, true).SetDelegated(delegated)
	bonded = big.NewInt(int64(2))
	s.GetPRepStatus(addr1, true).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(40))))

	delegated = big.NewInt(int64(93))
	s.GetPRepStatus(addr1, true).SetDelegated(delegated)
	bonded = big.NewInt(int64(7))
	s.GetPRepStatus(addr1, true).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(100))))

	delegated = big.NewInt(int64(90))
	s.GetPRepStatus(addr1, true).SetDelegated(delegated)
	bonded = big.NewInt(int64(10))
	s.GetPRepStatus(addr1, true).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(100))))

	// 0 input, exptected 0 output
	delegated = big.NewInt(int64(0))
	s.GetPRepStatus(addr1, true).SetDelegated(delegated)
	bonded = big.NewInt(int64(0))
	s.GetPRepStatus(addr1, true).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(0))))

	// extreme
	delegated = big.NewInt(int64(99999999999))
	s.GetPRepStatus(addr1, true).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1, true).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(19980))))

	// different requirement
	delegated = big.NewInt(int64(99999))
	s.GetPRepStatus(addr1, true).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1, true).SetBonded(bonded)
	res = status1.GetBondedDelegation(4)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(24975))))

	// 0 for bond requirement
	delegated = big.NewInt(int64(99999))
	s.GetPRepStatus(addr1, true).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1, true).SetBonded(bonded)
	res = status1.GetBondedDelegation(0)
	assert.Equal(t, 0, res.Cmp(status1.GetVoted()))

	// 101 for bond requirement
	delegated = big.NewInt(int64(99999))
	s.GetPRepStatus(addr1, true).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1, true).SetBonded(bonded)
	res = status1.GetBondedDelegation(101)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(0))))

	// 100 for bond requirement
	delegated = big.NewInt(int64(99999))
	s.GetPRepStatus(addr1, true).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1, true).SetBonded(bonded)
	res = status1.GetBondedDelegation(100)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(999))))
}

func TestPRepStatus_GetVTotal(t *testing.T) {
	type args struct {
		vTotal      int64
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
				Failure,
				15,
				20,
			},
			10 + 20 - 15,
		},
		{
			"Success state",
			args{
				20,
				Success,
				50,
				22000,
			},
			20 + 22000 - 50,
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
				Failure,
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

			assert.Equal(t, int64(tt.want), ret)
		})
	}
}

func TestPRepStatus_GetVFail(t *testing.T) {
	type args struct {
		vFail       int64
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
				Failure,
				15,
				20,
			},
			10 + 20 - 15,
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
				Failure,
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

			assert.Equal(t, int64(tt.want), ret)
		})
	}
}

func TestPRepStatus_GetVFailCont(t *testing.T) {
	type args struct {
		lastState       ValidationState
		lastBH          int64
		vFailContOffset int64
		blockHeight     int64
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"Fail state",
			args{
				Failure,
				15,
				1,
				20,
			},
			20 - 15 + 1,
		},
		{
			"Success state",
			args{
				Success,
				50,
				0,
				22000,
			},
			0,
		},
		{
			"None state",
			args{
				None,
				200,
				3,
				1000,
			},
			3,
		},
		{
			"Invalid block height",
			args{
				Failure,
				200,
				0,
				1,
			},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			ps := &PRepStatus{
				lastState:       in.lastState,
				lastHeight:      in.lastBH,
				vFailContOffset: in.vFailContOffset,
			}

			ret := ps.GetVFailCont(in.blockHeight)

			assert.Equal(t, int64(tt.want), ret)
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
				vPenaltyMask: in.vPenaltyMask,
			}

			ps.ShiftVPenaltyMask(in.mask)

			assert.Equal(t, tt.want, ps.vPenaltyMask)
		})
	}
}

func TestPRepStatus_UpdateBlockVoteStats(t *testing.T) {
	type attr struct {
		lh   int64
		ls   ValidationState
		vf   int64
		vt   int64
		vfco int64
		vpm  uint32
	}
	type input struct {
		bh    int64
		voted bool
	}
	type output struct {
		attr
		getVFail     int64
		getVTotal    int64
		getVFailCont int64
	}
	type test struct {
		name string
		init attr
		in   input
		out  output
	}

	tests := [...]test{
		{
			name: "S,S,tv",
			init: attr{lh: 10, ls: Success, vf: 1, vt: 8, vfco: 0, vpm: 0},
			in:   input{bh: 11, voted: true},
			out: output{
				attr:         attr{lh: 10, ls: Success, vf: 1, vt: 8, vfco: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    9,
				getVFailCont: 0,
			},
		},
		{
			name: "S,F,fv",
			init: attr{lh: 10, ls: Success, vf: 1, vt: 8, vfco: 0, vpm: 0},
			in:   input{bh: 11, voted: false},
			out: output{
				attr:         attr{lh: 11, ls: Failure, vf: 2, vt: 9, vfco: 1, vpm: 0},
				getVFail:     2,
				getVTotal:    9,
				getVFailCont: 1,
			},
		},
		{
			name: "F,F,fv",
			init: attr{lh: 10, ls: Failure, vf: 1, vt: 8, vfco: 0, vpm: 0},
			in:   input{bh: 11, voted: false},
			out: output{
				attr:         attr{lh: 10, ls: Failure, vf: 1, vt: 8, vfco: 0, vpm: 0},
				getVFail:     2,
				getVTotal:    9,
				getVFailCont: 1,
			},
		},
		{
			name: "F,S,tv",
			init: attr{lh: 10, ls: Failure, vf: 1, vt: 8, vfco: 0, vpm: 0},
			in:   input{bh: 11, voted: true},
			out: output{
				attr:         attr{lh: 11, ls: Success, vf: 1, vt: 9, vfco: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    9,
				getVFailCont: 0,
			},
		},
		{
			name: "N,N,tv",
			init: attr{lh: 10, ls: None, vf: 1, vt: 8, vfco: 0, vpm: 0},
			in:   input{bh: 11, voted: true},
			out: output{
				attr:         attr{lh: 11, ls: None, vf: 1, vt: 9, vfco: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    9,
				getVFailCont: 0,
			},
		},
		{
			name: "N,N,fv",
			init: attr{lh: 10, ls: None, vf: 1, vt: 8, vfco: 0, vpm: 0},
			in:   input{bh: 11, voted: false},
			out: output{
				attr:         attr{lh: 11, ls: None, vf: 2, vt: 9, vfco: 1, vpm: 0},
				getVFail:     2,
				getVTotal:    9,
				getVFailCont: 1,
			},
		},
		{
			name: "R,S,tv",
			init: attr{lh: 10, ls: Ready, vf: 1, vt: 8, vfco: 1, vpm: 0},
			in:   input{bh: 11, voted: true},
			out: output{
				attr:         attr{lh: 11, ls: Success, vf: 1, vt: 9, vfco: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    9,
				getVFailCont: 0,
			},
		},
		{
			name: "R,F,fv",
			init: attr{lh: 9, ls: Ready, vf: 1, vt: 8, vfco: 1, vpm: 0},
			in:   input{bh: 12, voted: false},
			out: output{
				attr:         attr{lh: 12, ls: Failure, vf: 2, vt: 9, vfco: 2, vpm: 0},
				getVFail:     2,
				getVTotal:    9,
				getVFailCont: 2,
			},
		},
		{
			name: "R,S,tv",
			init: attr{lh: 9, ls: Ready, vf: 1, vt: 8, vfco: 1, vpm: 0},
			in:   input{bh: 12, voted: true},
			out: output{
				attr:         attr{lh: 12, ls: Success, vf: 1, vt: 9, vfco: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    9,
				getVFailCont: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			init := tt.init
			in := tt.in
			out := tt.out
			bh := in.bh

			ps := &PRepStatus{
				vFail:           init.vf,
				vTotal:          init.vt,
				vFailContOffset: init.vfco,
				lastHeight:      init.lh,
				lastState:       init.ls,
				vPenaltyMask:    init.vpm,
			}

			err = ps.UpdateBlockVoteStats(in.bh, in.voted)
			assert.NoError(t, err)
			assert.Equal(t, out.lh, ps.lastHeight)
			assert.Equal(t, out.ls, ps.lastState)
			assert.Equal(t, out.vf, ps.vFail)
			assert.Equal(t, out.vt, ps.vTotal)
			assert.Equal(t, out.vfco, ps.vFailContOffset)
			assert.Equal(t, out.vpm, ps.vPenaltyMask)
			assert.Equal(t, out.getVFail, ps.GetVFail(bh))
			assert.Equal(t, out.getVTotal, ps.GetVTotal(bh))
			assert.Equal(t, out.getVFailCont, ps.GetVFailCont(bh))
		})
	}
}

func TestPRepStatus_SyncBlockVoteStats(t *testing.T) {
	type attr struct {
		lh   int64
		ls   ValidationState
		vf   int64
		vt   int64
		vfco int64
		vpm  uint32
	}
	type input struct {
		bh int64
	}
	type output struct {
		attr
		getVFail     int64
		getVTotal    int64
		getVFailCont int64
	}
	type test struct {
		name string
		init attr
		in   input
		out  output
	}

	tests := [...]test{
		{
			// 0 == in.bh - init.lh
			name: "S,N,0",
			init: attr{lh: 10, ls: Success, vf: 1, vt: 8, vfco: 0, vpm: 0},
			in:   input{bh: 10},
			out: output{
				attr:         attr{lh: 10, ls: None, vf: 1, vt: 8, vfco: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    8,
				getVFailCont: 0,
			},
		},
		{
			name: "S,N,1",
			init: attr{lh: 10, ls: Success, vf: 1, vt: 8, vfco: 0, vpm: 0},
			in:   input{bh: 11},
			out: output{
				attr:         attr{lh: 11, ls: None, vf: 1, vt: 9, vfco: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    9,
				getVFailCont: 0,
			},
		},
		{
			name: "S,N,2",
			init: attr{lh: 10, ls: Success, vf: 1, vt: 8, vfco: 0, vpm: 0},
			in:   input{bh: 12},
			out: output{
				attr:         attr{lh: 12, ls: None, vf: 1, vt: 10, vfco: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    10,
				getVFailCont: 0,
			},
		},
		{
			name: "F,N,0",
			init: attr{lh: 10, ls: Failure, vf: 1, vt: 8, vfco: 1, vpm: 0},
			in:   input{bh: 10},
			out: output{
				attr:         attr{lh: 10, ls: None, vf: 1, vt: 8, vfco: 1, vpm: 0},
				getVFail:     1,
				getVTotal:    8,
				getVFailCont: 1,
			},
		},
		{
			name: "F,N,1",
			init: attr{lh: 10, ls: Failure, vf: 1, vt: 8, vfco: 1, vpm: 0},
			in:   input{bh: 11},
			out: output{
				attr:         attr{lh: 11, ls: None, vf: 2, vt: 9, vfco: 2, vpm: 0},
				getVFail:     2,
				getVTotal:    9,
				getVFailCont: 2,
			},
		},
		{
			name: "F,N,2",
			init: attr{lh: 10, ls: Failure, vf: 1, vt: 8, vfco: 1, vpm: 0},
			in:   input{bh: 12},
			out: output{
				attr:         attr{lh: 12, ls: None, vf: 3, vt: 10, vfco: 3, vpm: 0},
				getVFail:     3,
				getVTotal:    10,
				getVFailCont: 3,
			},
		},
		{
			name: "N,N,0",
			init: attr{lh: 10, ls: None, vf: 1, vt: 8, vfco: 1, vpm: 0},
			in:   input{bh: 10},
			out: output{
				attr:         attr{lh: 10, ls: None, vf: 1, vt: 8, vfco: 1, vpm: 0},
				getVFail:     1,
				getVTotal:    8,
				getVFailCont: 1,
			},
		},
		{
			name: "N,N,0,C,M",
			init: attr{lh: 0, ls: None, vf: 0, vt: 0, vfco: 0, vpm: 0},
			in:   input{bh: 9},
			out: output{
				attr:         attr{lh: 0, ls: None, vf: 0, vt: 0, vfco: 0, vpm: 0},
				getVFail:     0,
				getVTotal:    0,
				getVFailCont: 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			init := tt.init
			in := tt.in
			out := tt.out
			bh := in.bh

			ps := &PRepStatus{
				vFail:           init.vf,
				vTotal:          init.vt,
				vFailContOffset: init.vfco,
				lastHeight:      init.lh,
				lastState:       init.ls,
				vPenaltyMask:    init.vpm,
			}

			err = ps.SyncBlockVoteStats(in.bh)
			assert.NoError(t, err)
			assert.Equal(t, out.lh, ps.lastHeight)
			assert.Equal(t, out.ls, ps.lastState)
			assert.Equal(t, out.vf, ps.vFail)
			assert.Equal(t, out.vt, ps.vTotal)
			assert.Equal(t, out.vfco, ps.vFailContOffset)
			assert.Equal(t, out.vpm, ps.vPenaltyMask)
			assert.Equal(t, out.getVFail, ps.GetVFail(bh))
			assert.Equal(t, out.getVTotal, ps.GetVTotal(bh))
			assert.Equal(t, out.getVFailCont, ps.GetVFailCont(bh))
		})
	}
}
