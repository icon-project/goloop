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
	"fmt"
	"math/big"
	"math/bits"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type Grade int

const (
	Main Grade = iota
	Sub
	Candidate
)

func (g Grade) String() string {
	switch g {
	case Main:
		return "M"
	case Sub:
		return "S"
	case Candidate:
		return "C"
	default:
		return "X"
	}
}

type Status int

const (
	Active Status = iota
	Unregistered
	Disqualified
	NotReady
)

func (s Status) String() string {
	switch s {
	case Active:
		return "A"
	case Unregistered:
		return "U"
	case Disqualified:
		return "D"
	case NotReady:
		return "N"
	default:
		return "X"
	}
}

type ValidationState int

const (
	None ValidationState = iota
	Ready
	Success
	Failure
)

func (vs ValidationState) String() string {
	switch vs {
	case Ready:
		return "R"
	case None:
		return "N"
	case Success:
		return "S"
	case Failure:
		return "F"
	default:
		return "X"
	}
}

type PRepStatus struct {
	icobject.NoDatabase
	StateAndSnapshot

	grade           Grade
	status          Status
	delegated       *big.Int
	bonded          *big.Int
	vTotal          int64
	vFail           int64
	vFailContOffset int64
	vPenaltyMask    uint32
	lastState       ValidationState
	lastHeight      int64
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

func (ps *PRepStatus) IsActive() bool {
	return ps.status == Active
}

// IsAlreadyPenalized returns true if this PRep got penalized during this term
func (ps *PRepStatus) IsAlreadyPenalized() bool {
	return (ps.vPenaltyMask & 1) != 0
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

func (ps *PRepStatus) GetVoted() *big.Int {
	return new(big.Int).Add(ps.delegated, ps.bonded)
}

func (ps *PRepStatus) SetDelegated(delegated *big.Int) {
	ps.checkWritable()
	ps.delegated = delegated
}

// GetBondedDelegation return amount of bonded delegation
// Bonded delegation formula
// totalVoted = bond + delegation
// bondRatio = bond / totalVoted * 100
// bondedDelegation = totalVoted * (bondRatio / bondRequirement)
//                  = bond * 100 / bondRequirement
// if bondedDelegation > totalVoted
//    bondedDelegation = totalVoted
func (ps *PRepStatus) GetBondedDelegation(bondRequirement int64) *big.Int {
	if bondRequirement < 0 || bondRequirement > 100 {
		// should not be negative or over 100 for bond requirement
		return big.NewInt(0)
	}
	totalVoted := ps.GetVoted() // bonded + delegated
	if bondRequirement == 0 {
		// when bondRequirement is 0, it means no threshold for BondedRequirement,
		// so it returns 100% of totalVoted.
		// And it should not be divided by 0 in the following code that could occurs Panic.
		return totalVoted
	}
	multiplier := big.NewInt(100)
	bondedDelegation := new(big.Int).Mul(ps.bonded, multiplier) // not divided by bond requirement yet

	br := big.NewInt(bondRequirement)
	bondedDelegation.Div(bondedDelegation, br)

	if totalVoted.Cmp(bondedDelegation) > 0 {
		return bondedDelegation
	} else {
		return totalVoted
	}
}

func (ps *PRepStatus) VTotal() int64 {
	return ps.vTotal
}

// GetVTotal returns the calculated number of validation
func (ps *PRepStatus) GetVTotal(blockHeight int64) int64 {
	if ps.lastState == None {
		return ps.vTotal
	} else {
		return ps.vTotal + ps.getSafeHeightDiff(blockHeight)
	}
}

func (ps *PRepStatus) VFail() int64 {
	return ps.vFail
}

// GetVFail returns the calculated number of validation failures
func (ps *PRepStatus) GetVFail(blockHeight int64) int64 {
	diff := blockHeight - ps.lastHeight
	if ps.lastState == Failure && diff >= 0 {
		return ps.vFail + diff
	}
	return ps.vFail
}

// GetVFailCont returns the number of consecutive validation failures
func (ps *PRepStatus) GetVFailCont(blockHeight int64) int64 {
	switch ps.lastState {
	case Ready:
		return ps.vFailContOffset
	case None:
		return ps.vFailContOffset
	case Failure:
		diff := blockHeight - ps.lastHeight
		if diff >= 0 {
			return diff + ps.vFailContOffset
		}
	}
	return 0
}

func (ps *PRepStatus) getSafeHeightDiff(blockHeight int64) int64 {
	if blockHeight < ps.lastHeight {
		return 0
	}
	return blockHeight - ps.lastHeight
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
		ps.vFailContOffset == other.vFailContOffset &&
		ps.vPenaltyMask == other.vPenaltyMask &&
		ps.lastState == other.lastState &&
		ps.lastHeight == other.lastHeight
}

func (ps *PRepStatus) Set(other *PRepStatus) {
	ps.checkWritable()
	ps.grade = other.grade
	ps.status = other.status
	ps.delegated = other.delegated
	ps.bonded = other.bonded
	ps.vTotal = other.vTotal
	ps.vFail = other.vFail
	ps.vFailContOffset = other.vFailContOffset
	ps.vPenaltyMask = other.vPenaltyMask
	ps.lastState = other.lastState
	ps.lastHeight = other.lastHeight
}

func (ps *PRepStatus) Clone() *PRepStatus {
	return &PRepStatus{
		grade:           ps.grade,
		status:          ps.status,
		delegated:       ps.delegated,
		bonded:          ps.bonded,
		vTotal:          ps.vTotal,
		vFail:           ps.vFail,
		vFailContOffset: ps.vFailContOffset,
		vPenaltyMask:    ps.vPenaltyMask,
		lastState:       ps.lastState,
		lastHeight:      ps.lastHeight,
	}
}

func (ps *PRepStatus) ToJSON(blockHeight int64, bondRequirement int64) map[string]interface{} {
	jso := make(map[string]interface{})
	jso["grade"] = int(ps.grade)
	jso["status"] = int(ps.status)
	jso["lastHeight"] = ps.lastHeight
	jso["delegated"] = ps.delegated
	jso["bonded"] = ps.bonded
	//	jso["voted"] = ps.GetVotedAmount()
	jso["bondedDelegation"] = ps.GetBondedDelegation(bondRequirement)
	totalBlocks := ps.GetVTotal(blockHeight)
	jso["totalBlocks"] = totalBlocks
	jso["validatedBlocks"] = totalBlocks - ps.GetVFail(blockHeight)
	return jso
}

func (ps *PRepStatus) GetStatsInJSON(blockHeight int64) map[string]interface{} {
	jso := make(map[string]interface{})
	jso["grade"] = int(ps.grade)
	jso["status"] = int(ps.status)
	jso["lastHeight"] = ps.lastHeight
	jso["lastState"] = int(ps.lastState)
	jso["penalties"] = ps.GetVPenaltyCount()
	jso["total"] = ps.vTotal
	jso["fail"] = ps.vFail
	jso["failCont"] = ps.vFailContOffset
	jso["realTotal"] = ps.GetVTotal(blockHeight)
	jso["realFail"] = ps.GetVFail(blockHeight)
	jso["realFailCont"] = ps.GetVFailCont(blockHeight)
	return jso
}

func (ps *PRepStatus) Version() int {
	return 0
}

func (ps *PRepStatus) RLPDecodeFields(decoder codec.Decoder) error {
	ps.checkWritable()
	return decoder.DecodeAll(
		&ps.grade,
		&ps.status,
		&ps.delegated,
		&ps.bonded,
		&ps.vTotal,
		&ps.vFail,
		&ps.vFailContOffset,
		&ps.vPenaltyMask,
		&ps.lastState,
		&ps.lastHeight,
	)
}

func (ps *PRepStatus) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		ps.grade,
		ps.status,
		ps.delegated,
		ps.bonded,
		ps.vTotal,
		ps.vFail,
		ps.vFailContOffset,
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
	ps.status = NotReady
	ps.grade = Candidate
	ps.delegated = big.NewInt(0)
	ps.bonded = big.NewInt(0)
	ps.vTotal = 0
	ps.vFail = 0
	ps.vFailContOffset = 0
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
	return ps.grade == Candidate &&
		ps.delegated.Sign() == 0 &&
		ps.bonded.Sign() == 0 &&
		ps.vFail == 0 &&
		ps.vFailContOffset == 0 &&
		ps.vTotal == 0 &&
		ps.lastState == None &&
		ps.lastHeight == 0 &&
		ps.status == NotReady
}

func (ps *PRepStatus) SetBonded(v *big.Int) {
	ps.checkWritable()
	ps.bonded = v
}

func (ps *PRepStatus) SetGrade(g Grade) {
	ps.checkWritable()
	ps.grade = g
}

func (ps *PRepStatus) SetStatus(s Status) {
	ps.checkWritable()
	ps.status = s
}

func (ps *PRepStatus) SetVTotal(t int64) {
	ps.checkWritable()
	ps.vTotal = t
}

func (ps *PRepStatus) SetVFail(f int64) {
	ps.checkWritable()
	ps.vFail = f
}

func (ps *PRepStatus) ResetVFailContOffset() {
	ps.checkWritable()
	ps.vFailContOffset = 0
}

func (ps *PRepStatus) setVPenaltyMask(p uint32) {
	ps.checkWritable()
	ps.vPenaltyMask = p
}

func (ps *PRepStatus) setLastHeight(blockHeight int64) {
	ps.checkWritable()
	ps.lastHeight = blockHeight
}

func (ps *PRepStatus) setLastState(lastState ValidationState) {
	ps.checkWritable()
	ps.lastState = lastState
}

func (ps *PRepStatus) shiftVPenaltyMask(mask uint32) {
	ps.checkWritable()
	ps.vPenaltyMask = (ps.vPenaltyMask << 1) & mask
}

// UpdateBlockVoteStats updates Penalty-related info based on ConsensusInfo
func (ps *PRepStatus) UpdateBlockVoteStats(blockHeight int64, voted bool) error {
	ps.checkWritable()
	vs := Success
	if !voted {
		vs = Failure
	}

	ls := ps.LastState()
	switch ls {
	case Ready:
		// S,C -> M
		if vs == Failure {
			ps.vFail++
			ps.vFailContOffset++
		} else {
			ps.vFailContOffset = 0
		}
		ps.vTotal++
		ps.lastHeight = blockHeight
		ps.lastState = vs
	case None:
		// Received vote info after this node is not a mainPRep
		if vs == Failure {
			ps.vFail++
			ps.vFailContOffset++
		} else {
			ps.vFailContOffset = 0
		}
		ps.vTotal++
		ps.lastHeight = blockHeight
	default: // Success or Failure
		if vs != ls {
			diff := blockHeight - ps.lastHeight
			if vs == Failure {
				ps.vFail++
				ps.vFailContOffset++
			} else {
				ps.vFail += diff - 1
				ps.vFailContOffset = 0
			}
			ps.vTotal += diff
			ps.lastState = vs
			ps.lastHeight = blockHeight
		}
	}
	return nil
}

// syncBlockVoteStats updates vote stats data at a given blockHeight
func (ps *PRepStatus) syncBlockVoteStats(blockHeight int64) error {
	ps.checkWritable()
	lh := ps.lastHeight
	if blockHeight < lh {
		return errors.Errorf("blockHeight(%d) < lastHeight(%d)", blockHeight, lh)
	}
	if ps.lastState == None {
		return nil
	}
	ps.vFail = ps.GetVFail(blockHeight)
	ps.vTotal = ps.GetVTotal(blockHeight)
	ps.vFailContOffset = ps.GetVFailCont(blockHeight)
	ps.lastHeight = blockHeight
	ps.lastState = None
	return nil
}

func (ps *PRepStatus) ImposePenalty(blockHeight int64) error {
	ps.checkWritable()
	if err := ps.syncBlockVoteStats(blockHeight); err != nil {
		return err
	}
	ps.vPenaltyMask |= 1
	ps.vFailContOffset = 0
	ps.grade = Candidate
	return nil
}

func (ps *PRepStatus) ChangeGrade(newGrade Grade, blockHeight int64, penaltyMask int) error {
	ps.checkWritable()
	if ps.grade == newGrade {
		return nil
	}
	if ps.grade == Main && ps.lastState == None {
		panic(errors.Errorf("Invalid PRepStatus: grade=%v lastState=%v", ps.grade, ps.lastState))
	}
	if ps.grade != Main && ps.lastState != None {
		panic(errors.Errorf("Invalid PRepStatus: grade=%v lastState=%v", ps.grade, ps.lastState))
	}

	if newGrade == Main {
		if ps.lastState == None {
			ps.lastState = Ready
			ps.lastHeight = blockHeight
		}
		ps.shiftVPenaltyMask(buildPenaltyMask(penaltyMask))
	} else {
		if ps.lastState != None {
			if err := ps.syncBlockVoteStats(blockHeight); err != nil {
				return err
			}
		}
	}
	ps.grade = newGrade
	return nil
}

func (ps *PRepStatus) String() string {
	return fmt.Sprintf(
		"st=%s grade=%s ls=%s lh=%d vf=%d vt=%d vpc=%d vfco=%d",
		ps.status,
		ps.grade,
		ps.lastState,
		ps.lastHeight,
		ps.vFail,
		ps.vTotal,
		ps.GetVPenaltyCount(),
		ps.vFailContOffset,
	)
}

func (ps *PRepStatus) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(
				f,
				"PRepStatus{"+
					"status=%s grade=%s lastState=%s lastHeight=%d "+
					"vFail=%d vTotal=%d vPenaltyCount=%d vFailContOffset=%d "+
					"delegated=%s bonded=%s}",
				ps.status,
				ps.grade,
				ps.lastState,
				ps.lastHeight,
				ps.vFail,
				ps.vTotal,
				ps.GetVPenaltyCount(),
				ps.vFailContOffset,
				ps.delegated,
				ps.bonded,
			)
		} else {
			fmt.Fprintf(
				f, "PRepStatus{%s %s %s %d %d %d %d %d %s %s}",
				ps.status,
				ps.grade,
				ps.lastState,
				ps.lastHeight,
				ps.vFail,
				ps.vTotal,
				ps.GetVPenaltyCount(),
				ps.vFailContOffset,
				ps.delegated,
				ps.bonded,
			)
		}
	case 's':
		fmt.Fprint(f, ps.String())
	}
}

func newPRepStatusWithTag(_ icobject.Tag) *PRepStatus {
	return new(PRepStatus)
}

func NewPRepStatus() *PRepStatus {
	return &PRepStatus{
		grade:           Candidate,
		delegated:       new(big.Int),
		bonded:          new(big.Int),
		vFail:           0,
		vFailContOffset: 0,
		vTotal:          0,
		lastState:       None,
		lastHeight:      0,
		status:          NotReady,
	}
}
