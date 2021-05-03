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

package iiss

import (
	"fmt"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	ValidationPenaltyCondition = 30
)

func TestPenalty_checkValidationPenalty(t *testing.T) {

	type args struct {
		vPenaltyMask uint32
		lastState    icstate.ValidationState
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
				icstate.Failure,
				0,
				int64(ValidationPenaltyCondition),
			},
			true,
		},
		{
			"False - State(None)",
			args{
				0,
				icstate.None,
				0,
				int64(ValidationPenaltyCondition),
			},
			false,
		},
		{
			"False - State(Success)",
			args{
				0,
				icstate.Success,
				0,
				int64(ValidationPenaltyCondition),
			},
			false,
		},
		{
			"False - Not enough fail count)",
			args{
				0,
				icstate.Failure,
				0,
				int64(ValidationPenaltyCondition - 100),
			},
			false,
		},
		{
			"False - already got penalty)",
			args{
				1,
				icstate.Failure,
				0,
				int64(ValidationPenaltyCondition),
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			ps := icstate.NewPRepStatus()
			ps.SetVPenaltyMask(in.vPenaltyMask)
			ps.SetLastState(in.lastState)
			ps.SetLastHeight(in.lastBH)

			ret := checkValidationPenalty(ps, in.blockHeight, ValidationPenaltyCondition)

			assert.Equal(t, tt.want, ret)
		})
	}
}

func TestPenalty_bitOperation(t *testing.T) {
	input := new(big.Int).SetInt64(30)
	res := buildPenaltyMask(input)
	str := fmt.Sprintf("%x", res)
	assert.Equal(t, "3fffffff", str)

	input = new(big.Int).SetInt64(1)
	res = buildPenaltyMask(input)
	str = fmt.Sprintf("%x", res)
	assert.Equal(t, "1", str)

	input = new(big.Int).SetInt64(2)
	res = buildPenaltyMask(input)
	str = fmt.Sprintf("%x", res)
	assert.Equal(t, "3", str)
}
