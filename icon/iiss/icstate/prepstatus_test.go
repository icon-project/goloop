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

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestPRepStatus_Bytes(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	ss1 := NewPRepStatus().GetSnapshot()

	o1 := icobject.New(TypePRepStatus, ss1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())

	ss2 := ToPRepStatus(o2)
	assert.Equal(t, true, ss1.Equal(ss2))
	assert.Equal(t, true, ss2.Equal(ss1))
}

// test for GetBondedDelegation
func TestPRepStatus_GetBondedDelegation(t *testing.T) {
	type args struct {
		delegated int64
		bonded    int64
	}
	tests := []struct {
		name string
		args args
		bd   int64
	}{
		{
			"d=99, b=1, bd=20",
			args{
				int64(100),
				int64(0),
			},
			int64(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			br := int64(5)
			in := tt.args

			ps := NewPRepStatus()
			ps.SetDelegated(big.NewInt(in.delegated))
			ps.SetBonded(big.NewInt(in.bonded))
			assert.Equal(t, in.delegated, ps.Delegated().Int64())
			assert.Equal(t, in.bonded, ps.Bonded().Int64())
			assert.Equal(t, tt.bd, ps.GetBondedDelegation(br).Int64())
		})
	}
}

func TestPRepStatus_GetVTotal(t *testing.T) {
	type args struct {
		vTotal      int64
		lastState   VoteState
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			ps := &prepStatusData{
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
		lastState   VoteState
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			ps := &prepStatusData{
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
		lastState   VoteState
		lastBH      int64
		vFailCont   int64
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			ps := &prepStatusData{
				lastState:  in.lastState,
				lastHeight: in.lastBH,
				vFailCont:  in.vFailCont,
			}

			ret := ps.GetVFailCont(in.blockHeight)
			assert.Equal(t, int64(tt.want), ret)
		})
	}
}

func TestPRepStatus_buildPenaltyMask(t *testing.T) {
	var mask uint32
	mask = buildPenaltyMask(30)
	assert.Equal(t, uint32(0x3fffffff), mask)

	mask = buildPenaltyMask(1)
	assert.Equal(t, uint32(0x1), mask)

	mask = buildPenaltyMask(2)
	assert.Equal(t, uint32(0x3), mask)
}

func TestPRepStatus_ShiftVPenaltyMask(t *testing.T) {
	type args struct {
		vPenaltyMask uint32
		mask         int
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
				30,
			},
			0x6,
		},
		{
			"Masked",
			args{
				0x3fffffff,
				30,
			},
			0x3ffffffe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			ps := &PRepStatusState{prepStatusData: prepStatusData{
				vPenaltyMask: in.vPenaltyMask,
			}}

			ps.shiftVPenaltyMask(in.mask)

			assert.Equal(t, tt.want, ps.vPenaltyMask)
		})
	}
}

func TestPRepStatus_UpdateBlockVoteStats(t *testing.T) {
	type attr struct {
		lh  int64
		ls  VoteState
		vf  int64
		vt  int64
		vfc int64
		vpm uint32
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
			init: attr{lh: 10, ls: Success, vf: 1, vt: 8, vfc: 0, vpm: 0},
			in:   input{bh: 11, voted: true},
			out: output{
				attr:         attr{lh: 10, ls: Success, vf: 1, vt: 8, vfc: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    9,
				getVFailCont: 0,
			},
		},
		{
			name: "S,F,fv",
			init: attr{lh: 10, ls: Success, vf: 1, vt: 8, vfc: 0, vpm: 0},
			in:   input{bh: 11, voted: false},
			out: output{
				attr:         attr{lh: 11, ls: Failure, vf: 2, vt: 9, vfc: 1, vpm: 0},
				getVFail:     2,
				getVTotal:    9,
				getVFailCont: 1,
			},
		},
		{
			name: "F,F,fv",
			init: attr{lh: 10, ls: Failure, vf: 1, vt: 8, vfc: 0, vpm: 0},
			in:   input{bh: 11, voted: false},
			out: output{
				attr:         attr{lh: 10, ls: Failure, vf: 1, vt: 8, vfc: 0, vpm: 0},
				getVFail:     2,
				getVTotal:    9,
				getVFailCont: 1,
			},
		},
		{
			name: "F,S,tv",
			init: attr{lh: 10, ls: Failure, vf: 1, vt: 8, vfc: 0, vpm: 0},
			in:   input{bh: 11, voted: true},
			out: output{
				attr:         attr{lh: 11, ls: Success, vf: 1, vt: 9, vfc: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    9,
				getVFailCont: 0,
			},
		},
		{
			name: "N,S,tv",
			init: attr{lh: 10, ls: None, vf: 1, vt: 8, vfc: 0, vpm: 0},
			in:   input{bh: 11, voted: true},
			out: output{
				attr:         attr{lh: 11, ls: Success, vf: 1, vt: 9, vfc: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    9,
				getVFailCont: 0,
			},
		},
		{
			name: "N,F,fv",
			init: attr{lh: 10, ls: None, vf: 1, vt: 8, vfc: 0, vpm: 0},
			in:   input{bh: 11, voted: false},
			out: output{
				attr:         attr{lh: 11, ls: Failure, vf: 2, vt: 9, vfc: 1, vpm: 0},
				getVFail:     2,
				getVTotal:    9,
				getVFailCont: 1,
			},
		},
		{
			name: "S,F,fv",
			init: attr{lh: 33, ls: Success, vf: 0, vt: 1, vfc: 0, vpm: 0},
			in:   input{bh: 60, voted: false},
			out: output{
				attr:         attr{lh: 60, ls: Failure, vf: 1, vt: 28, vfc: 1, vpm: 0},
				getVFail:     1,
				getVTotal:    28,
				getVFailCont: 1,
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

			ps := &PRepStatusState{prepStatusData: prepStatusData{
				vFail:        init.vf,
				vTotal:       init.vt,
				vFailCont:    init.vfc,
				lastHeight:   init.lh,
				lastState:    init.ls,
				vPenaltyMask: init.vpm,
			}}

			err = ps.OnBlockVote(in.bh, in.voted)
			assert.NoError(t, err)
			assert.Equal(t, out.lh, ps.lastHeight)
			assert.Equal(t, out.ls, ps.lastState)
			assert.Equal(t, out.vf, ps.vFail)
			assert.Equal(t, out.vt, ps.vTotal)
			assert.Equal(t, out.vfc, ps.vFailCont)
			assert.Equal(t, out.vpm, ps.vPenaltyMask)
			assert.Equal(t, out.getVFail, ps.GetVFail(bh))
			assert.Equal(t, out.getVTotal, ps.GetVTotal(bh))
			assert.Equal(t, out.getVFailCont, ps.GetVFailCont(bh))
		})
	}
}

func TestPRepStatus_syncBlockVoteStats(t *testing.T) {
	type attr struct {
		lh  int64
		ls  VoteState
		vf  int64
		vt  int64
		vfc int64
		vpm uint32
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
			name: "S,0",
			init: attr{lh: 10, ls: Success, vf: 1, vt: 8, vfc: 0, vpm: 0},
			in:   input{bh: 10},
			out: output{
				attr:         attr{lh: 10, ls: Success, vf: 1, vt: 8, vfc: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    8,
				getVFailCont: 0,
			},
		},
		{
			name: "S,1",
			init: attr{lh: 10, ls: Success, vf: 1, vt: 8, vfc: 0, vpm: 0},
			in:   input{bh: 11},
			out: output{
				attr:         attr{lh: 11, ls: Success, vf: 1, vt: 9, vfc: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    9,
				getVFailCont: 0,
			},
		},
		{
			name: "S,2",
			init: attr{lh: 10, ls: Success, vf: 1, vt: 8, vfc: 0, vpm: 0},
			in:   input{bh: 12},
			out: output{
				attr:         attr{lh: 12, ls: Success, vf: 1, vt: 10, vfc: 0, vpm: 0},
				getVFail:     1,
				getVTotal:    10,
				getVFailCont: 0,
			},
		},
		{
			name: "F,0",
			init: attr{lh: 10, ls: Failure, vf: 1, vt: 8, vfc: 1, vpm: 0},
			in:   input{bh: 10},
			out: output{
				attr:         attr{lh: 10, ls: Failure, vf: 1, vt: 8, vfc: 1, vpm: 0},
				getVFail:     1,
				getVTotal:    8,
				getVFailCont: 1,
			},
		},
		{
			name: "F,1",
			init: attr{lh: 10, ls: Failure, vf: 1, vt: 8, vfc: 1, vpm: 0},
			in:   input{bh: 11},
			out: output{
				attr:         attr{lh: 11, ls: Failure, vf: 2, vt: 9, vfc: 2, vpm: 0},
				getVFail:     2,
				getVTotal:    9,
				getVFailCont: 2,
			},
		},
		{
			name: "F,2",
			init: attr{lh: 10, ls: Failure, vf: 1, vt: 8, vfc: 1, vpm: 0},
			in:   input{bh: 12},
			out: output{
				attr:         attr{lh: 12, ls: Failure, vf: 3, vt: 10, vfc: 3, vpm: 0},
				getVFail:     3,
				getVTotal:    10,
				getVFailCont: 3,
			},
		},
		{
			name: "N,0",
			init: attr{lh: 10, ls: None, vf: 1, vt: 8, vfc: 1, vpm: 0},
			in:   input{bh: 10},
			out: output{
				attr:         attr{lh: 10, ls: None, vf: 1, vt: 8, vfc: 1, vpm: 0},
				getVFail:     1,
				getVTotal:    8,
				getVFailCont: 1,
			},
		},
		{
			name: "N,1",
			init: attr{lh: 10, ls: None, vf: 1, vt: 8, vfc: 1, vpm: 0},
			in:   input{bh: 10},
			out: output{
				attr:         attr{lh: 10, ls: None, vf: 1, vt: 8, vfc: 1, vpm: 0},
				getVFail:     1,
				getVTotal:    8,
				getVFailCont: 1,
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

			ps := &PRepStatusState{prepStatusData: prepStatusData{
				vFail:        init.vf,
				vTotal:       init.vt,
				vFailCont:    init.vfc,
				lastHeight:   init.lh,
				lastState:    init.ls,
				vPenaltyMask: init.vpm,
			}}

			err = ps.syncBlockVoteStats(in.bh)
			assert.NoError(t, err)
			assert.Equal(t, out.lh, ps.lastHeight)
			assert.Equal(t, out.ls, ps.lastState)
			assert.Equal(t, out.vf, ps.vFail)
			assert.Equal(t, out.vt, ps.vTotal)
			assert.Equal(t, out.vfc, ps.vFailCont)
			assert.Equal(t, out.vpm, ps.vPenaltyMask)
			assert.Equal(t, out.getVFail, ps.GetVFail(bh))
			assert.Equal(t, out.getVTotal, ps.GetVTotal(bh))
			assert.Equal(t, out.getVFailCont, ps.GetVFailCont(bh))
		})
	}
}

func TestPRepStatus_OnPenaltyImposed(t *testing.T) {
	type attr struct {
		lh  int64
		ls  VoteState
		vf  int64
		vt  int64
		vfc int64
		vpm uint32
	}
	type input struct {
		bh int64
	}
	type output struct {
		attr
		getVFail         int64
		getVTotal        int64
		getVFailCont     int64
		getVPenaltyCount int
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
			name: "F,N",
			init: attr{lh: 10, ls: Failure, vf: 5, vt: 8, vfc: 5, vpm: 0x2},
			in:   input{bh: 10},
			out: output{
				attr:             attr{lh: 10, ls: Failure, vf: 5, vt: 8, vfc: 0, vpm: 0x3},
				getVFail:         5,
				getVTotal:        8,
				getVFailCont:     0,
				getVPenaltyCount: 2,
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

			ps := &PRepStatusState{prepStatusData: prepStatusData{
				vFail:        init.vf,
				vTotal:       init.vt,
				vFailCont:    init.vfc,
				lastHeight:   init.lh,
				lastState:    init.ls,
				vPenaltyMask: init.vpm,
			}}

			err = ps.OnPenaltyImposed(bh)
			assert.NoError(t, err)
			assert.Equal(t, out.lh, ps.lastHeight)
			assert.Equal(t, out.ls, ps.lastState)
			assert.Equal(t, out.vf, ps.vFail)
			assert.Equal(t, out.vt, ps.vTotal)
			assert.Equal(t, out.vfc, ps.vFailCont)
			assert.Equal(t, out.vpm, ps.vPenaltyMask)
			assert.Equal(t, out.getVFail, ps.GetVFail(bh))
			assert.Equal(t, out.getVTotal, ps.GetVTotal(bh))
			assert.Equal(t, out.getVFailCont, ps.GetVFailCont(bh))
			assert.Equal(t, out.getVPenaltyCount, ps.GetVPenaltyCount())
			assert.Equal(t, ps.Grade(), GradeCandidate)
		})
	}
}

func TestPRepStatusData_getPenaltyType(t *testing.T) {
	ps := NewPRepStatus()
	assert.Equal(t, icmodule.PenaltyNone, ps.getPenaltyType())

	for i := 0; i < 10; i += 2 {
		ps.vPenaltyMask = uint32(i)
		assert.Equal(t, icmodule.PenaltyNone, ps.getPenaltyType())
	}

	for i := 1; i < 10; i += 2 {
		ps.vPenaltyMask = uint32(i)
		assert.Equal(t, icmodule.PenaltyBlockValidation, ps.getPenaltyType())
	}

	ps.SetStatus(Disqualified)
	assert.Equal(t, icmodule.PenaltyPRepDisqualification, ps.getPenaltyType())
}

func TestPrepStatusData_ToJSON(t *testing.T) {
	ps := NewPRepStatus()
	jso := ps.ToJSON(100, 5, 0)

	penalty, ok := jso["penalty"].(int64)
	assert.True(t, ok)
	assert.Equal(t, int64(icmodule.PenaltyNone), penalty)

	grade, ok := jso["grade"].(int)
	assert.True(t, ok)
	assert.Equal(t, int(GradeCandidate), grade)

	status, ok := jso["status"].(int)
	assert.True(t, ok)
	assert.Equal(t, int(NotReady), status)

	power, ok := jso["power"].(*big.Int)
	assert.True(t, ok)
	assert.Zero(t, power.Sign())
}
