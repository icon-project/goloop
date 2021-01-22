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
	"math/bits"
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

type ValidationState int

const (
	None ValidationState = iota
	Success
	Fail
)

type PRepStatus struct {
	icobject.NoDatabase
	StateAndSnapshot

	owner module.Address

	grade        Grade
	status       Status
	delegated    *big.Int
	bonded       *big.Int
	vTotal       int
	vFail        int
	vPenaltyMask uint32
	lastState    ValidationState
	lastHeight   int64
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

func (ps *PRepStatus) VPenaltyMask() uint32 {
	return ps.vPenaltyMask
}

func (ps *PRepStatus) GetVPenaltyCount() int {
	return bits.OnesCount32(ps.vPenaltyMask)
}

func (ps *PRepStatus) LastState() ValidationState {
	return ps.lastState
}

func (ps *PRepStatus) LastHeight() int64 {
	return ps.lastHeight
}

func (ps *PRepStatus) Delegated() *big.Int {
	return ps.delegated
}

func (ps *PRepStatus) SetDelegated(delegated *big.Int) {
	ps.delegated.Set(delegated)
}

// Bond Delegation formula
// totalDelegation = bond + delegation
// bondRatio = bond / totalDelegation * 100
// bondedDelegation = totalDelegation * (bondRatio / bondRequirement)
//                  = bond * 100 / bondRequirement
// if bondedDelegation > totalDelegation
//    bondedDelegation = totalDelegation
func (ps *PRepStatus) GetBondedDelegation(bondRequirement int) *big.Int {
	if bondRequirement < 1 || bondRequirement > 100 {
		// should not be 0 for bond requirement
		return big.NewInt(0)
	}
	totalDelegation := new(big.Int).Add(ps.delegated, ps.bonded)
	multiplier := big.NewInt(100)
	bondedDelegation := new(big.Int).Mul(ps.bonded, multiplier) // not divided by bond requirement yet

	br := big.NewInt(int64(bondRequirement))
	bondedDelegation.Div(bondedDelegation, br)

	if totalDelegation.Cmp(bondedDelegation) > 0 {
		return bondedDelegation
	} else {
		return totalDelegation
	}
}

func (ps *PRepStatus) VTotal() int {
	return ps.vTotal
}

// GetVTotal returns the calculated number of validation
func (ps *PRepStatus) GetVTotal(blockHeight int64) int {
	return ps.vTotal + ps.getContValue(blockHeight)
}

func (ps *PRepStatus) VFail() int {
	return ps.vFail
}

// GetVFail returns the calculated number of validation failures
func (ps *PRepStatus) GetVFail(blockHeight int64) int {
	return ps.vFail + ps.GetVFailCont(blockHeight)
}

// GetVFailCont returns the number of consecutive validation failures
func (ps *PRepStatus) GetVFailCont(blockHeight int64) int {
	if ps.lastState == Fail {
		return ps.getContValue(blockHeight)
	}
	return 0
}

func (ps *PRepStatus) getContValue(blockHeight int64) int {
	if ps.lastState == None {
		return 0
	}
	if blockHeight < ps.lastHeight {
		return 0
	} else {
		return int(blockHeight - ps.lastHeight) + 1
	}
}

func (ps *PRepStatus) equal(other *PRepStatus) bool {
	if ps == other {
		return true
	}

	return ps.grade == other.grade &&
		ps.status == other.status &&
		ps.delegated.Cmp(other.delegated) == 0 &&
		ps.bonded.Cmp(other.bonded) == 0 &&
		ps.vTotal == other.vTotal &&
		ps.vFail == other.vFail &&
		ps.vPenaltyMask == other.vPenaltyMask &&
		ps.lastState == other.lastState &&
		ps.lastHeight == other.lastHeight
}

func (ps *PRepStatus) Set(other *PRepStatus) {
	ps.checkWritable()
	ps.owner = other.owner
	ps.grade = other.grade
	ps.status = other.status
	ps.delegated.Set(other.delegated)
	ps.bonded.Set(other.bonded)
	ps.vTotal = other.vTotal
	ps.vFail = other.vFail
	ps.vPenaltyMask = other.vPenaltyMask
	ps.lastState = other.lastState
	ps.lastHeight = other.lastHeight
}

func (ps *PRepStatus) Clone() *PRepStatus {
	return &PRepStatus{
		owner:        ps.owner,
		grade:        ps.grade,
		status:       ps.status,
		delegated:    new(big.Int).Set(ps.delegated),
		bonded:       new(big.Int).Set(ps.bonded),
		vTotal:       ps.vTotal,
		vFail:        ps.vFail,
		vPenaltyMask: ps.vPenaltyMask,
		lastState:    ps.lastState,
		lastHeight:   ps.lastHeight,
	}
}


func (ps *PRepStatus) ToJSON(blockHeight int64, bondRequirement int) map[string]interface{} {
	jso := make(map[string]interface{})
	jso["grade"] = int(ps.grade)
	jso["status"] = int(ps.status)
	jso["lastHeight"] = ps.lastHeight
	jso["delegated"] = ps.delegated
	jso["bonded"] = ps.bonded
	jso["bondedDelegation"] = ps.GetBondedDelegation(bondRequirement)
	totalBlocks := ps.GetVTotal(blockHeight)
	jso["totalBlocks"] = totalBlocks
	jso["validatedBlocks"] = totalBlocks - ps.GetVFail(blockHeight)
	return jso
}

func (ps *PRepStatus) Version() int {
	return 0
}

func (ps *PRepStatus) RLPDecodeFields(decoder codec.Decoder) error {
	ps.checkWritable()
	return decoder.DecodeListOf(
		&ps.grade,
		&ps.status,
		&ps.delegated,
		&ps.bonded,
		&ps.vTotal,
		&ps.vFail,
		&ps.vPenaltyMask,
		&ps.lastState,
		&ps.lastHeight,
	)
}

func (ps *PRepStatus) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
		ps.grade,
		ps.status,
		ps.delegated,
		ps.bonded,
		ps.vTotal,
		ps.vFail,
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
	ps.delegated = big.NewInt(0)
	ps.bonded = big.NewInt(0)
	ps.vTotal = 0
	ps.vFail = 0
	ps.vPenaltyMask = 0
	ps.lastState = None
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

func (ps *PRepStatus) SetVPenaltyMask(p uint32) {
	ps.vPenaltyMask = p
}

func (ps *PRepStatus) ShiftVPenaltyMask(mask uint32) {
	ps.vPenaltyMask = (ps.vPenaltyMask << 1) & mask
}


func (ps *PRepStatus) SetLastState(l ValidationState) {
	ps.lastState = l
}

func (ps *PRepStatus) SetLastHeight(h int64) {
	ps.lastHeight = h
}

func newPRepStatusWithTag(_ icobject.Tag) *PRepStatus {
	return NewPRepStatus(nil)
}

func NewPRepStatus(owner module.Address) *PRepStatus {
	return &PRepStatus{
		owner:      owner,
		grade:      Candidate,
		delegated:  new(big.Int),
		bonded:     new(big.Int),
		vFail:      0,
		vTotal:     0,
		lastState:  None,
		lastHeight: 0,
	}
}
