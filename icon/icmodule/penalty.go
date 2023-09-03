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

package icmodule

type PenaltyType int

const (
	PenaltyNone PenaltyType = iota
	PenaltyPRepDisqualification
	PenaltyAccumulatedValidationFailure
	PenaltyValidationFailure
	PenaltyMissedNetworkProposalVote
	PenaltyDoubleVote
	PenaltyReserved
)

var penaltyNames = []string{
	"",
	"prepDisqualification",
	"accumulatedValidationFailure",
	"validationFailure",
	"missedNetworkProposalVote",
	"doubleVote",
}

var penaltyTypes = []PenaltyType {
	PenaltyPRepDisqualification,
	PenaltyAccumulatedValidationFailure,
	PenaltyValidationFailure,
	PenaltyMissedNetworkProposalVote,
	PenaltyDoubleVote,
}

func (p PenaltyType) String() string {
	if p > PenaltyNone && p < PenaltyReserved {
		return penaltyNames[p]
	}
	return ""
}

func (p PenaltyType) IsValid() bool {
	return p > PenaltyNone && p < PenaltyReserved
}

func ToPenaltyType(name string) PenaltyType {
	for i, penaltyName := range penaltyNames {
		if name == penaltyName {
			return PenaltyType(i)
		}
	}
	return PenaltyNone
}

func PenaltyNames() []string {
	return penaltyNames
}

func GetPenaltyTypes() []PenaltyType {
	return penaltyTypes
}