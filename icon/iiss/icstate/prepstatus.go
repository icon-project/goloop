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
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
	"math/big"
)

const (
	prepStatusVersion1 = iota + 1
	prepStatusVersion  = prepStatusVersion1
)

type Grade int

const (
	Main Grade = iota
	Sub
	Candidate
)

type Status int

const (
	Active Status = iota
	Unregistered
	Disqualified
)

type PRepStatus struct {
	grade        Grade
	status       Status
	penalty      int
	delegated    *big.Int
	bonded       *big.Int
	vTotal       int
	vFail        int
	vFailCount   int
	vPenaltyMask int
	lastState    int
	lastHeight   int
}

func (ps *PRepStatus) Bonded() *big.Int {
	return ps.bonded
}

func (ps *PRepStatus) Grade() Grade {
	return ps.grade
}

func (ps *PRepStatus) Status() Status {
	return ps.status
}

func (ps *PRepStatus) LastHeight() int {
	return ps.lastHeight
}

func (ps *PRepStatus) Delegated() *big.Int {
	return ps.delegated
}

func (ps *PRepStatus) GetBondedDelegation() *big.Int {
	// TODO: Not implemented
	return ps.delegated
}

func (ps *PRepStatus) VTotal() int {
	return ps.vTotal
}

func (ps *PRepStatus) VFail() int {
	return ps.vFail
}

func (ps *PRepStatus) ToJSON() map[string]interface{} {
	data := make(map[string]interface{})
	data["grade"] = ps.grade
	data["status"] = ps.status
	data["lastHeight"] = ps.lastHeight
	data["delegated"] = ps.delegated
	data["bonded"] = ps.bonded
	data["bondedDelegation"] = ps.GetBondedDelegation()
	data["totalBlocks"] = ps.vTotal
	data["validatedBlocks"] = ps.vTotal - ps.vFail
	return data
}

type PRepStatusSnapshot struct {
	icobject.NoDatabase
	PRepStatus
}

func (pss *PRepStatusSnapshot) Version() int {
	return 0
}

func (pss *PRepStatusSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&pss.grade,
		&pss.penalty,
		&pss.status,
		&pss.delegated,
		&pss.bonded,
		&pss.vTotal,
		&pss.vFail,
		&pss.vFailCount,
		&pss.vPenaltyMask,
		&pss.lastState,
		&pss.lastHeight,
	)
	return err
}

func (pss *PRepStatusSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		pss.grade,
		pss.penalty,
		pss.status,
		pss.delegated,
		pss.bonded,
		pss.vTotal,
		pss.vFail,
		pss.vFailCount,
		pss.vPenaltyMask,
		pss.lastState,
		pss.lastHeight,
	)
}

func (pss *PRepStatusSnapshot) Equal(o icobject.Impl) bool {
	pss1, ok := o.(*PRepStatusSnapshot)
	if !ok {
		return false
	}
	return pss.grade == pss1.grade &&
		pss.penalty == pss1.penalty &&
		pss.status == pss1.status &&
		pss.delegated.Cmp(pss1.delegated) == 0 &&
		pss.bonded.Cmp(pss.bonded) == 0 &&
		pss.vTotal == pss.vTotal &&
		pss.vFail == pss.vFail &&
		pss.vFailCount == pss.vFailCount &&
		pss.vPenaltyMask == pss.vPenaltyMask &&
		pss.lastState == pss.lastState &&
		pss.lastHeight == pss.lastHeight
}

func newPRepStatusSnapshot(_ icobject.Tag) *PRepStatusSnapshot {
	return &PRepStatusSnapshot{
		PRepStatus: PRepStatus{
			delegated: new(big.Int),
			bonded:    new(big.Int),
		},
	}
}

func NewPRepStatusSnapshot(grade Grade, delegated, bonded *big.Int) *PRepStatusSnapshot {
	return &PRepStatusSnapshot{
		PRepStatus: PRepStatus{
			grade:     grade,
			delegated: delegated,
			bonded:    bonded,
		},
	}
}

type PRepStatusState struct {
	address module.Address
	PRepStatus
}

func newPRepStatusState(address module.Address) *PRepStatusState {
	return &PRepStatusState{
		address: address,
		PRepStatus: PRepStatus{
			delegated: new(big.Int),
			bonded:    new(big.Int),
		},
	}
}

func (ps *PRepStatusState) Clear() {
	ps.grade = 0
	ps.penalty = 0
	ps.delegated = BigIntZero
	ps.bonded = BigIntZero
	ps.vTotal = 0
	ps.vFail = 0
	ps.vFailCount = 0
	ps.vPenaltyMask = 0
	ps.lastState = 0
	ps.lastHeight = 0
}

func (ps *PRepStatusState) Reset(pss *PRepStatusSnapshot) {
	ps.grade = pss.grade
}

func (ps *PRepStatusState) GetSnapshot() *PRepStatusSnapshot {
	return &PRepStatusSnapshot{PRepStatus: ps.PRepStatus}
}

func (ps PRepStatusState) IsEmpty() bool {
	return ps.status == Active && ps.grade == Main
}

func (ps PRepStatusState) GetAddress() module.Address {
	return ps.address
}

func (ps *PRepStatusState) SetGrade(g Grade) {
	ps.grade = g
}

func (ps *PRepStatusState) SetStatus(s Status) {
	ps.status = s
}

func (ps *PRepStatusState) GetPRepStatusInfo() map[string]interface{} {
	return ps.ToJSON()
}

func NewPRepStatusStateWithSnapshot(a module.Address, pss *PRepStatusSnapshot) *PRepStatusState {
	ps := newPRepStatusState(a)
	ps.Reset(pss)
	return ps
}
