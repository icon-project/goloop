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

type PRepStatusData struct {
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

func (ps *PRepStatusData) Bonded() *big.Int {
	return ps.bonded
}

func (ps *PRepStatusData) Grade() Grade {
	return ps.grade
}

func (ps *PRepStatusData) Status() Status {
	return ps.status
}

func (ps *PRepStatusData) LastHeight() int {
	return ps.lastHeight
}

func (ps *PRepStatusData) Delegated() *big.Int {
	return ps.delegated
}

func (ps *PRepStatusData) GetBondedDelegation() *big.Int {
	// TODO: Not implemented
	return ps.delegated
}

func (ps *PRepStatusData) VTotal() int {
	return ps.vTotal
}

func (ps *PRepStatusData) VFail() int {
	return ps.vFail
}

func (ps *PRepStatusData) Equal(other *PRepStatusData) bool {
	if ps == other {
		return true
	}

	return ps.grade == other.grade &&
		ps.penalty == other.penalty &&
		ps.status == other.status &&
		ps.delegated.Cmp(other.delegated) == 0 &&
		ps.bonded.Cmp(other.bonded) == 0 &&
		ps.vTotal == other.vTotal &&
		ps.vFail == other.vFail &&
		ps.vFailCount == other.vFailCount &&
		ps.vPenaltyMask == other.vPenaltyMask &&
		ps.lastState == other.lastState &&
		ps.lastHeight == other.lastHeight
}

func (ps *PRepStatusData) Set(other *PRepStatusData) {
	ps.grade = other.grade
	ps.penalty = other.penalty
	ps.status = other.status
	ps.delegated.Set(other.delegated)
	ps.bonded.Set(other.bonded)
	ps.vTotal = other.vTotal
	ps.vFail = other.vFail
	ps.vFailCount = other.vFailCount
	ps.vPenaltyMask = other.vPenaltyMask
	ps.lastState = other.lastState
	ps.lastHeight = other.lastHeight
}

func (ps *PRepStatusData) Clone() *PRepStatusData {
	return &PRepStatusData{
		grade:        ps.grade,
		penalty:      ps.penalty,
		status:       ps.status,
		delegated:    new(big.Int).Set(ps.delegated),
		bonded:       new(big.Int).Set(ps.bonded),
		vTotal:       ps.vTotal,
		vFail:        ps.vFail,
		vFailCount:   ps.vFailCount,
		vPenaltyMask: ps.vPenaltyMask,
		lastState:    ps.lastState,
		lastHeight:   ps.lastHeight,
	}
}

func (ps *PRepStatusData) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["grade"] = ps.grade
	jso["status"] = ps.status
	jso["lastHeight"] = ps.lastHeight
	jso["delegated"] = ps.delegated
	jso["bonded"] = ps.bonded
	jso["bondedDelegation"] = ps.GetBondedDelegation()
	jso["totalBlocks"] = ps.vTotal
	jso["validatedBlocks"] = ps.vTotal - ps.vFail
	return jso
}

type PRepStatusSnapshot struct {
	icobject.NoDatabase
	*PRepStatusData
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
	return pss.PRepStatusData.Equal(pss1.PRepStatusData)
}

func newPRepStatusSnapshot(_ icobject.Tag) *PRepStatusSnapshot {
	return &PRepStatusSnapshot{
		PRepStatusData: &PRepStatusData{
			delegated: new(big.Int),
			bonded:    new(big.Int),
		},
	}
}

type PRepStatusState struct {
	address module.Address
	*PRepStatusData
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
	if ps.PRepStatusData == nil {
		ps.PRepStatusData = pss.PRepStatusData.Clone()
	} else {
		ps.PRepStatusData.Set(pss.PRepStatusData)
	}
}

func (ps *PRepStatusState) GetSnapshot() *PRepStatusSnapshot {
	return &PRepStatusSnapshot{PRepStatusData: ps.PRepStatusData.Clone()}
}

func (ps PRepStatusState) IsEmpty() bool {
	return ps.status == Active && ps.grade == Main
}

func (ps PRepStatusState) Address() module.Address {
	return ps.address
}

func (ps *PRepStatusState) SetBonded(v *big.Int) {
	ps.bonded = v
}

func (ps *PRepStatusState) SetGrade(g Grade) {
	ps.grade = g
}

func (ps *PRepStatusState) SetStatus(s Status) {
	ps.status = s
}

func (ps *PRepStatusState) SetVTotal(t int) {
	ps.vTotal = t
}

func (ps *PRepStatusState) SetVFail(f int) {
	ps.vFail = f
}

func (ps *PRepStatusState) SetVFailCount(f int) {
	ps.vFailCount = f
}

func (ps *PRepStatusState) SetVPenaltyMask(p int) {
	ps.vPenaltyMask = p
}

func (ps *PRepStatusState) SetLastState(l int) {
	ps.lastState = l
}

func (ps *PRepStatusState) SetLastHeight(h int) {
	ps.lastHeight = h
}
func (ps *PRepStatusState) GetPRepStatusInfo() map[string]interface{} {
	return ps.ToJSON()
}

func NewPRepStatusStateWithSnapshot(a module.Address, pss *PRepStatusSnapshot) *PRepStatusState {
	return &PRepStatusState{
		address:        a,
		PRepStatusData: pss.PRepStatusData.Clone(),
	}
}
