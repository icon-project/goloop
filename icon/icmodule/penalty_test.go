/*
 * Copyright 2023 ICON Foundation
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

package icmodule

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPenaltyType_IsValid(t *testing.T) {
	for _, pt := range GetPenaltyTypes() {
		assert.True(t, pt.IsValid())
	}
	assert.False(t, PenaltyNone.IsValid())
}

func TestToPenaltyType(t *testing.T) {
	args := []struct {
		name string
		pt   PenaltyType
	}{
		{"prepDisqualification", PenaltyPRepDisqualification},
		{"accumulatedValidationFailure", PenaltyAccumulatedValidationFailure},
		{"validationFailure", PenaltyValidationFailure},
		{"missedNetworkProposalVote", PenaltyMissedNetworkProposalVote},
		{"doubleSign", PenaltyDoubleSign},
		{"", PenaltyNone},
		{"invalid_name", PenaltyNone},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T){
			assert.Equal(t, arg.pt, ToPenaltyType(arg.name))
		})
	}
}

func TestPenaltyType_String(t *testing.T) {
	args := []struct {
		pt PenaltyType
		name string
	}{
		{PenaltyPRepDisqualification, "prepDisqualification"},
		{PenaltyAccumulatedValidationFailure, "accumulatedValidationFailure"},
		{PenaltyValidationFailure, "validationFailure"},
		{PenaltyMissedNetworkProposalVote, "missedNetworkProposalVote"},
		{PenaltyDoubleSign, "doubleSign"},
		{PenaltyNone, ""},
	}
	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T){
			assert.Equal(t, arg.name, arg.pt.String())
		})
	}
}