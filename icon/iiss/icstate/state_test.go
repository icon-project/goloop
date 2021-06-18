/*
 * Copyright 2021 ICON Foundation
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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/module"
)

func newNodeOnlyRegInfo(node module.Address) *RegInfo {
	city := ""
	country := ""
	email := ""
	website := ""
	details := ""
	name := ""
	p2pEndpoint := ""

	return NewRegInfo(city, country, details, email, name, p2pEndpoint, website, node)
}

func TestState_RegisterPRep(t *testing.T) {
	var err error
	size := 10
	irep := BigIntInitialIRep
	state := newDummyState(false)

	for i := 0; i < size; i++ {
		owner := newDummyAddress(i)
		ri := newDummyRegInfo(i)
		err = state.RegisterPRep(owner, ri, irep)
		assert.NoError(t, err)
		err = state.Flush()
		assert.NoError(t, err)

		riApplied := ri.Clone()
		riApplied.SetNode(owner)
		pb, _ := state.GetPRepBaseByOwner(owner, false)
		assert.NotNil(t, pb)
		assert.True(t, pb.RegInfo.Equal(riApplied))

		ps, _ := state.GetPRepStatusByOwner(owner, false)
		assert.NotNil(t, ps)
		assert.Equal(t, Candidate, ps.Grade())
		assert.Equal(t, Active, ps.Status())
		assert.Zero(t, ps.Delegated().Int64())
		assert.Zero(t, ps.Bonded().Int64())
		assert.Equal(t, None, ps.LastState())
		assert.Zero(t, ps.LastHeight())
		assert.Zero(t, ps.VTotal())
		assert.Zero(t, ps.VFail())
	}
}

func TestState_SetPRep(t *testing.T) {
	var err error
	size := 10
	irep := BigIntInitialIRep
	bh := int64(100)
	state := newDummyState(false)

	for i := 0; i < size; i++ {
		owner := newDummyAddress(i)
		ri := newDummyRegInfo(i)
		err = state.RegisterPRep(owner, ri, irep)
		assert.NoError(t, err)

		err = state.Flush()
		assert.NoError(t, err)

		node := newDummyAddress(i + 100)
		assert.False(t, node.Equal(owner))
		ri = newNodeOnlyRegInfo(node)
		err = state.SetPRep(bh, owner, ri)
		assert.NoError(t, err)

		err = state.Flush()
		assert.NoError(t, err)

		node2 := state.GetNodeByOwner(owner)
		assert.True(t, node2.Equal(node))
	}
}

func Test_checkValidationPenalty(t *testing.T) {
	ValidationPenaltyCondition := int64(30)

	type args struct {
		vPenaltyMask uint32
		lastState    ValidationState
		lastBH       int64
		blockHeight  int64
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"True",
			args{
				0,
				Failure,
				0,
				ValidationPenaltyCondition,
			},
			true,
		},
		{
			"False - State(None)",
			args{
				0,
				None,
				0,
				ValidationPenaltyCondition,
			},
			false,
		},
		{
			"False - State(Success)",
			args{
				0,
				Success,
				0,
				ValidationPenaltyCondition,
			},
			false,
		},
		{
			"False - Not enough fail count)",
			args{
				0,
				Failure,
				0,
				ValidationPenaltyCondition - 100,
			},
			false,
		},
		{
			"False - already got penalty)",
			args{
				1,
				Failure,
				0,
				ValidationPenaltyCondition,
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			ps := NewPRepStatus()
			ps.setVPenaltyMask(in.vPenaltyMask)
			ps.setLastState(in.lastState)
			ps.setLastHeight(in.lastBH)

			ret := checkValidationPenalty(ps, in.blockHeight, ValidationPenaltyCondition)

			assert.Equal(t, tt.want, ret)
		})
	}
}


