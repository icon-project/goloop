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
	NotReady
)

type PRepStatus struct {
	icobject.NoDatabase
	StateAndSnapshot

	owner module.Address

	grade        Grade
	status       Status
	penalty      int
	delegated    *big.Int
	bonded       *big.Int
	vTotal       int
	vFail        int
	vFailCont    int
	vPenaltyMask int
	lastState    int
	lastHeight   int
}

func (ps *PRepStatus) Owner() module.Address {
	return ps.owner
}

func (ps *PRepStatus) SetOwner(owner module.Address) {
	ps.checkWritable()
	ps.owner = owner
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

func (ps *PRepStatus) SetDelegated(delegated *big.Int) {
	ps.delegated.Set(delegated)
}

func (ps *PRepStatus) GetBondedDelegation(bondRequirement int64) *big.Int {
	if bondRequirement == 0 || bondRequirement > 100 {
		// should not be 0 for bond requirement
		return big.NewInt(0)
	}
	sum := new(big.Int).Add(ps.delegated, ps.bonded)
	multiplier := big.NewInt(100)
	calc := new(big.Int).Mul(ps.bonded, multiplier)

	br := big.NewInt(bondRequirement)
	calc.Div(calc, br)

	if sum.Cmp(calc) > 0 {
		return calc
	} else {
		return sum
	}
}

func (ps *PRepStatus) VTotal() int {
	return ps.vTotal
}

func (ps *PRepStatus) VFail() int {
	return ps.vFail
}

func (ps *PRepStatus) VFailCont() int {
	return ps.vFailCont
}

func (ps *PRepStatus) equal(other *PRepStatus) bool {
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
		ps.vFailCont == other.vFailCont &&
		ps.vPenaltyMask == other.vPenaltyMask &&
		ps.lastState == other.lastState &&
		ps.lastHeight == other.lastHeight
}

func (ps *PRepStatus) Set(other *PRepStatus) {
	ps.checkWritable()

	ps.grade = other.grade
	ps.penalty = other.penalty
	ps.status = other.status
	ps.delegated.Set(other.delegated)
	ps.bonded.Set(other.bonded)
	ps.vTotal = other.vTotal
	ps.vFail = other.vFail
	ps.vFailCont = other.vFailCont
	ps.vPenaltyMask = other.vPenaltyMask
	ps.lastState = other.lastState
	ps.lastHeight = other.lastHeight
}

func (ps *PRepStatus) Clone() *PRepStatus {
	return &PRepStatus{
		owner:        ps.owner,
		grade:        ps.grade,
		penalty:      ps.penalty,
		status:       ps.status,
		delegated:    new(big.Int).Set(ps.delegated),
		bonded:       new(big.Int).Set(ps.bonded),
		vTotal:       ps.vTotal,
		vFail:        ps.vFail,
		vFailCont:    ps.vFailCont,
		vPenaltyMask: ps.vPenaltyMask,
		lastState:    ps.lastState,
		lastHeight:   ps.lastHeight,
	}
}

func (ps *PRepStatus) ToJSON(br int64) map[string]interface{} {
	jso := make(map[string]interface{})
	jso["grade"] = int(ps.grade)
	jso["status"] = int(ps.status)
	jso["lastHeight"] = ps.lastHeight
	jso["delegated"] = ps.delegated
	jso["bonded"] = ps.bonded
	jso["bondedDelegation"] = ps.GetBondedDelegation(br)
	jso["totalBlocks"] = ps.vTotal
	jso["validatedBlocks"] = ps.vTotal - ps.vFail
	return jso
}

func (ps *PRepStatus) Version() int {
	return 0
}

func (ps *PRepStatus) RLPDecodeFields(decoder codec.Decoder) error {
	ps.checkWritable()
	return decoder.DecodeListOf(
		&ps.grade,
		&ps.penalty,
		&ps.status,
		&ps.delegated,
		&ps.bonded,
		&ps.vTotal,
		&ps.vFail,
		&ps.vFailCont,
		&ps.vPenaltyMask,
		&ps.lastState,
		&ps.lastHeight,
	)
}

func (ps *PRepStatus) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
		ps.grade,
		ps.penalty,
		ps.status,
		ps.delegated,
		ps.bonded,
		ps.vTotal,
		ps.vFail,
		ps.vFailCont,
		ps.vPenaltyMask,
		ps.lastState,
		ps.lastHeight,
	)
}

func (ps *PRepStatus) Equal(o icobject.Impl) bool {
	other, ok := o.(*PRepStatus)
	if !ok {
		return false
	}
	return ps.equal(other)
}

func (ps *PRepStatus) Clear() {
	ps.checkWritable()
	ps.owner = nil
	ps.status = Active
	ps.grade = Candidate
	ps.penalty = 0
	ps.delegated = BigIntZero
	ps.bonded = BigIntZero
	ps.vTotal = 0
	ps.vFail = 0
	ps.vFailCont = 0
	ps.vPenaltyMask = 0
	ps.lastState = 0
	ps.lastHeight = 0
}

func (ps *PRepStatus) GetSnapshot() *PRepStatus {
	if ps.IsReadonly() {
		return ps
	}
	ret := ps.Clone()
	ret.freeze()
	return ret
}

func (ps *PRepStatus) IsEmpty() bool {
	return ps == nil || ps.owner == nil
}

func (ps *PRepStatus) SetBonded(v *big.Int) {
	ps.bonded.Set(v)
}

func (ps *PRepStatus) SetGrade(g Grade) {
	ps.grade = g
}

func (ps *PRepStatus) SetStatus(s Status) {
	ps.status = s
}

func (ps *PRepStatus) SetVTotal(t int) {
	ps.vTotal = t
}

func (ps *PRepStatus) SetVFail(f int) {
	ps.vFail = f
}

func (ps *PRepStatus) SetVFailCont(f int) {
	ps.vFailCont = f
}

func (ps *PRepStatus) SetVPenaltyMask(p int) {
	ps.vPenaltyMask = p
}

func (ps *PRepStatus) SetLastState(l int) {
	ps.lastState = l
}

func (ps *PRepStatus) SetLastHeight(h int) {
	ps.lastHeight = h
}

func newPRepStatusWithTag(_ icobject.Tag) *PRepStatus {
	return NewPRepStatus(nil)
}

func NewPRepStatus(owner module.Address) *PRepStatus {
	return &PRepStatus{
		owner:     owner,
		grade:     Candidate,
		delegated: new(big.Int),
		bonded:    new(big.Int),
		vFail: 0,
		vFailCont: 0,
		vTotal: 0,
	}
}
