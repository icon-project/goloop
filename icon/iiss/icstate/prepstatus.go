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
	"io"
	"math/big"
	"math/bits"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

type Grade int

const (
	GradeMain Grade = iota
	GradeSub
	GradeCandidate
	GradeNone
)

func (g Grade) String() string {
	switch g {
	case GradeMain:
		return "M"
	case GradeSub:
		return "S"
	case GradeCandidate:
		return "C"
	case GradeNone:
		return "N"
	default:
		return "X"
	}
}

func (g Grade) Cmp(g2 Grade) int {
	switch {
	case g == g2:
		return 0
	case g < g2:
		return 1
	default:
		return -1
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

type VoteState int

const (
	None VoteState = iota
	Ready
	Success
	Failure
)

func (vs VoteState) String() string {
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

type prepStatusData struct {
	grade     Grade
	status    Status
	delegated *big.Int
	bonded    *big.Int
	vTotal    int64
	vFail     int64
	vFailCont int64
	// ValidationFailurePenaltyMask
	vPenaltyMask uint32
	lastState    VoteState
	lastHeight   int64
	dsaMask      int64
	// Since IISS-4.0
	ji JailInfo

	// Data not stored in DB
	effectiveDelegated *big.Int
}

func (ps *prepStatusData) Bonded() *big.Int {
	return ps.bonded
}

func (ps *prepStatusData) Grade() Grade {
	return ps.grade
}

func (ps *prepStatusData) Status() Status {
	return ps.status
}

func (ps *prepStatusData) IsActive() bool {
	return ps.status == Active
}

// IsAlreadyPenalized returns true if this PRep got penalized during this term
func (ps *prepStatusData) IsAlreadyPenalized() bool {
	return (ps.vPenaltyMask & 1) != 0
}

func (ps *prepStatusData) GetVPenaltyCount() int {
	return bits.OnesCount32(ps.vPenaltyMask)
}

func (ps *prepStatusData) LastState() VoteState {
	return ps.lastState
}

func (ps *prepStatusData) LastHeight() int64 {
	return ps.lastHeight
}

func (ps *prepStatusData) Delegated() *big.Int {
	return ps.delegated
}

func (ps *prepStatusData) EffectiveDelegated() *big.Int {
	return ps.effectiveDelegated
}

func (ps *prepStatusData) getVoted() *big.Int {
	return new(big.Int).Add(ps.delegated, ps.bonded)
}

// GetBondedDelegation return amount of bonded delegation
// Bonded delegation formula
// totalVoted = bond + delegation
// bondRatio = bond / totalVoted * 100
// bondedDelegation = totalVoted * (bondRatio / bondRequirement)
//                  = bond * 100 / bondRequirement
// if bondedDelegation > totalVoted
//    bondedDelegation = totalVoted
func (ps *prepStatusData) GetBondedDelegation(br icmodule.Rate) *big.Int {
	if !br.IsValid() {
		// should not be negative or over 100 for bond requirement
		return big.NewInt(0)
	}
	return icutils.CalcPower(br, ps.bonded, ps.getVoted())
}

// GetPower returns the power score of a PRep.
// Power is the same as delegated of a given PRep before rev 14
// and will be bondedDelegation since rev 14.
// But the calculation formula for power can be changed in the future.
func (ps *prepStatusData) GetPower(bondRequirement icmodule.Rate) *big.Int {
	return ps.GetBondedDelegation(bondRequirement)
}

func (ps *prepStatusData) VTotal() int64 {
	return ps.vTotal
}

// GetVTotal returns the calculated number of validation
func (ps *prepStatusData) GetVTotal(blockHeight int64) int64 {
	if ps.lastState == None {
		return ps.vTotal
	}
	return ps.vTotal + ps.getSafeHeightDiff(blockHeight)
}

func (ps *prepStatusData) VFail() int64 {
	return ps.vFail
}

// GetVFail returns the calculated number of validation failures
func (ps *prepStatusData) GetVFail(blockHeight int64) int64 {
	if ps.lastState == Failure {
		return ps.vFail + ps.getSafeHeightDiff(blockHeight)
	}
	return ps.vFail
}

// GetVFailCont returns the number of consecutive validation failures
func (ps *prepStatusData) GetVFailCont(blockHeight int64) int64 {
	if ps.lastState == Failure {
		return ps.vFailCont + ps.getSafeHeightDiff(blockHeight)
	}
	return ps.vFailCont
}

func (ps *prepStatusData) GetDSAMask() int64 {
	return ps.dsaMask
}

func (ps *prepStatusData) getSafeHeightDiff(blockHeight int64) int64 {
	diff := blockHeight - ps.lastHeight
	if diff < 0 {
		panic(errors.Errorf("Invalid blockHeight: blockHeight=%d < lastHeight=%d", blockHeight, ps.lastHeight))
	}
	return diff
}

func (ps *prepStatusData) equal(other *prepStatusData) bool {
	if ps == other {
		return true
	}
	return ps.grade == other.grade &&
		ps.status == other.status &&
		ps.delegated.Cmp(other.delegated) == 0 &&
		ps.bonded.Cmp(other.bonded) == 0 &&
		ps.vTotal == other.vTotal &&
		ps.vFail == other.vFail &&
		ps.vFailCont == other.vFailCont &&
		ps.vPenaltyMask == other.vPenaltyMask &&
		ps.lastState == other.lastState &&
		ps.lastHeight == other.lastHeight &&
		ps.dsaMask == other.dsaMask &&
		ps.ji == other.ji
}

func (ps *prepStatusData) clone() prepStatusData {
	return prepStatusData{
		grade:        ps.grade,
		status:       ps.status,
		delegated:    ps.delegated,
		bonded:       ps.bonded,
		vTotal:       ps.vTotal,
		vFail:        ps.vFail,
		vFailCont:    ps.vFailCont,
		vPenaltyMask: ps.vPenaltyMask,
		lastState:    ps.lastState,
		lastHeight:   ps.lastHeight,
		dsaMask:      ps.dsaMask,
		ji:           ps.ji,

		// Data not stored in DB
		effectiveDelegated: ps.effectiveDelegated,
	}
}

func (ps *prepStatusData) ToJSON(sc icmodule.StateContext) map[string]interface{} {
	blockHeight := sc.BlockHeight()
	br := sc.GetBondRequirement()
	activeDSAMask := sc.GetActiveDSAMask()

	jso := make(map[string]interface{})
	jso["grade"] = int(ps.grade)
	jso["status"] = int(ps.status)
	jso["penalty"] = int(ps.getPenaltyType(sc))
	jso["lastHeight"] = ps.lastHeight
	jso["delegated"] = ps.delegated
	jso["bonded"] = ps.bonded
	jso["power"] = ps.GetPower(br)
	totalBlocks := ps.GetVTotal(blockHeight)
	jso["totalBlocks"] = totalBlocks
	jso["validatedBlocks"] = totalBlocks - ps.GetVFail(blockHeight)
	if activeDSAMask != 0 {
		jso["hasPublicKey"] = (ps.GetDSAMask() & activeDSAMask) == activeDSAMask
	}
	ps.ji.ToJSON(sc, jso)
	return jso
}

func (ps *prepStatusData) getPenaltyType(sc icmodule.StateContext) icmodule.PenaltyType {
	if sc.TermIISSVersion() >= IISSVersion4 {
		return ps.getPenaltyTypeV1()
	}
	return ps.getPenaltyTypeV0()
}

func (ps *prepStatusData) getPenaltyTypeV0() icmodule.PenaltyType {
	if ps.status == Disqualified {
		return icmodule.PenaltyPRepDisqualification
	}
	if (ps.vPenaltyMask & 1) != 0 {
		return icmodule.PenaltyValidationFailure
	}
	return icmodule.PenaltyNone
}

func (ps *prepStatusData) getPenaltyTypeV1() icmodule.PenaltyType {
	if ps.status == Disqualified {
		return icmodule.PenaltyPRepDisqualification
	}
	if icutils.ContainsAll(ps.ji.Flags(), JFlagDoubleSign) {
		return icmodule.PenaltyDoubleSign
	}
	if (ps.vPenaltyMask & 1) != 0 {
		if icutils.ContainsAll(ps.ji.Flags(), JFlagAccumulatedValidationFailure) {
			return icmodule.PenaltyAccumulatedValidationFailure
		}
		return icmodule.PenaltyValidationFailure
	}
	return icmodule.PenaltyNone
}

func (ps *prepStatusData) GetStatsInJSON(blockHeight int64) map[string]interface{} {
	jso := make(map[string]interface{})
	jso["grade"] = int(ps.grade)
	jso["status"] = int(ps.status)
	jso["lastHeight"] = ps.lastHeight
	jso["lastState"] = int(ps.lastState)
	jso["penalties"] = ps.GetVPenaltyCount()
	jso["total"] = ps.vTotal
	jso["fail"] = ps.vFail
	jso["failCont"] = ps.vFailCont
	jso["realTotal"] = ps.GetVTotal(blockHeight)
	jso["realFail"] = ps.GetVFail(blockHeight)
	jso["realFailCont"] = ps.GetVFailCont(blockHeight)
	return jso
}

func (ps *prepStatusData) IsEmpty() bool {
	return ps.grade == GradeCandidate &&
		ps.delegated.Sign() == 0 &&
		ps.bonded.Sign() == 0 &&
		ps.vFail == 0 &&
		ps.vFailCont == 0 &&
		ps.vTotal == 0 &&
		ps.lastState == None &&
		ps.lastHeight == 0 &&
		ps.status == NotReady &&
		ps.dsaMask == 0 &&
		ps.ji.IsEmpty()
}

func (ps *prepStatusData) String() string {
	return fmt.Sprintf(
		"st=%s grade=%s ls=%s lh=%d vf=%d vt=%d vpc=%d vfco=%d "+
			"dd=%s bd=%s vote=%s ed=%d dm=%d ji=%v",
		ps.status,
		ps.grade,
		ps.lastState,
		ps.lastHeight,
		ps.vFail,
		ps.vTotal,
		ps.GetVPenaltyCount(),
		ps.vFailCont,
		ps.delegated,
		ps.bonded,
		ps.getVoted(),
		ps.effectiveDelegated,
		ps.dsaMask,
		ps.ji,
	)
}

func (ps *prepStatusData) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		var format string
		if f.Flag('+') {
			format = "PRepStatus{" +
				"status=%s grade=%s lastState=%s lastHeight=%d " +
				"vFail=%d vTotal=%d vPenaltyCount=%d vFailCont=%d " +
				"delegated=%s bonded=%s effectiveDelegated=%d dsaMask=%d ji=%v}"
		} else {
			format = "PRepStatus{%s %s %s %d %d %d %d %d %s %s %d %d %v}"
		}
		_, _ = fmt.Fprintf(
			f, format,
			ps.status,
			ps.grade,
			ps.lastState,
			ps.lastHeight,
			ps.vFail,
			ps.vTotal,
			ps.GetVPenaltyCount(),
			ps.vFailCont,
			ps.delegated,
			ps.bonded,
			ps.effectiveDelegated,
			ps.dsaMask,
			ps.ji,
		)
	case 's':
		_, _ = fmt.Fprint(f, ps.String())
	}
}

func (ps *prepStatusData) JailFlags() int {
	return ps.ji.Flags()
}

func (ps *prepStatusData) IsInJail() bool {
	return ps.ji.IsInJail()
}

func (ps *prepStatusData) IsUnjailing() bool {
	return ps.ji.IsUnjailing()
}

func (ps *prepStatusData) IsUnjailable() bool {
	return ps.ji.IsUnjailable()
}

func (ps *prepStatusData) IsJailInfoElectable() bool {
	return ps.ji.IsElectable()
}

func (ps *prepStatusData) UnjailRequestHeight() int64 {
	return ps.ji.UnjailRequestHeight()
}

func (ps *prepStatusData) MinDoubleSignHeight() int64 {
	return ps.ji.MinDoubleSignHeight()
}

func (ps *prepStatusData) IsDoubleSignReportable(sc icmodule.StateContext, dsBlockHeight int64) bool {
	if !ps.IsActive() {
		return false
	}
	if icutils.ContainsAll(ps.ji.Flags(), JFlagDoubleSign) {
		// Already in Jail due to DoubleSignReport
		return false
	}
	if dsBlockHeight <= ps.ji.MinDoubleSignHeight() {
		// DoubleSignReport is too old to accept
		return false
	}
	return true
}

type PRepStatusSnapshot struct {
	icobject.NoDatabase
	prepStatusData
}

func (ps *PRepStatusSnapshot) Version() int {
	return 0
}

func (ps *PRepStatusSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	n, err := decoder.DecodeMulti(
		&ps.grade,
		&ps.status,
		&ps.delegated,
		&ps.bonded,
		&ps.vTotal,
		&ps.vFail,
		&ps.vFailCont,
		&ps.vPenaltyMask,
		&ps.lastState,
		&ps.lastHeight,
		&ps.dsaMask,
		&ps.ji,
	)
	if err == io.EOF {
		if n != 10 && n != 11 {
			return icmodule.InvalidStateError.Errorf("InvalidFormat(n=%d)", n)
		}
		err = nil
	}
	return err
}

func (ps *PRepStatusSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	if err := encoder.EncodeMulti(
		ps.grade,
		ps.status,
		ps.delegated,
		ps.bonded,
		ps.vTotal,
		ps.vFail,
		ps.vFailCont,
		ps.vPenaltyMask,
		ps.lastState,
		ps.lastHeight,
	); err != nil {
		return err
	}

	if !ps.ji.IsEmpty() {
		return encoder.EncodeMulti(ps.dsaMask, &ps.ji)
	} else {
		if ps.dsaMask != 0 {
			return encoder.Encode(ps.dsaMask)
		}
	}
	return nil
}

func (ps *PRepStatusSnapshot) Equal(o icobject.Impl) bool {
	other, ok := o.(*PRepStatusSnapshot)
	if !ok {
		return false
	}
	return ps.equal(&other.prepStatusData)
}

var emptyPRepStatusSnapshot = &PRepStatusSnapshot{
	prepStatusData: prepStatusData{
		grade:      GradeCandidate,
		delegated:  new(big.Int),
		bonded:     new(big.Int),
		vFail:      0,
		vFailCont:  0,
		vTotal:     0,
		lastState:  None,
		lastHeight: 0,
		status:     NotReady,
		dsaMask:    0,
	},
}

type PRepStatusState struct {
	owner module.Address
	prepStatusData
	last *PRepStatusSnapshot
}

func (ps *PRepStatusState) Owner() module.Address {
	return ps.owner
}

func (ps *PRepStatusState) Reset(ss *PRepStatusSnapshot) *PRepStatusState {
	if ps.last != ss {
		ed := ps.effectiveDelegated
		ps.last = ss
		ps.prepStatusData = ss.prepStatusData.clone()
		ps.effectiveDelegated = ed
	}
	return ps
}

func (ps *PRepStatusState) setDirty() {
	if ps.last != nil {
		ps.last = nil
	}
}

func (ps *PRepStatusState) Clear() {
	ps.Reset(emptyPRepStatusSnapshot)
}

func (ps *PRepStatusState) GetSnapshot() *PRepStatusSnapshot {
	if ps.last == nil {
		ps.last = &PRepStatusSnapshot{
			prepStatusData: ps.prepStatusData.clone(),
		}
	}
	return ps.last
}

func (ps *PRepStatusState) SetDelegated(delegated *big.Int) {
	ps.delegated = delegated
	ps.setDirty()
}

func (ps *PRepStatusState) SetEffectiveDelegated(value *big.Int) {
	ps.effectiveDelegated = value
	ps.setDirty()
}

func (ps *PRepStatusState) SetBonded(v *big.Int) {
	ps.bonded = v
	ps.setDirty()
}

func (ps *PRepStatusState) SetStatus(s Status) {
	ps.status = s
	ps.setDirty()
}

func (ps *PRepStatusState) Activate() error {
	if ps.status != NotReady {
		return errors.InvalidStateError.Errorf("AlreadyUsed")
	}
	ps.status = Active
	ps.setDirty()
	return nil
}

func (ps *PRepStatusState) SetVTotal(t int64) {
	ps.vTotal = t
	ps.setDirty()
}

func (ps *PRepStatusState) SetDSAMask(m int64) {
	if ps.dsaMask != m {
		ps.dsaMask = m
		ps.setDirty()
	}
}

func (ps *PRepStatusState) resetVFailCont() {
	if ps.IsAlreadyPenalized() {
		ps.vFailCont = 0
	}
}

func buildPenaltyMask(input int) (res uint32) {
	res = uint32((uint64(1) << input) - 1)
	return
}

func (ps *PRepStatusState) shiftVPenaltyMask(limit int) {
	ps.vPenaltyMask = (ps.vPenaltyMask << 1) & buildPenaltyMask(limit)
}

func (ps *PRepStatusState) OnEvent(
	sc icmodule.StateContext, event icmodule.PRepEvent, data ...interface{}) error {
	switch event {
	case icmodule.PRepEventBlockVote:
		return ps.onBlockVote(sc, data[0].(bool))
	case icmodule.PRepEventMainIn:
		return ps.onMainPRepIn(sc, data[0].(int), false)
	case icmodule.PRepEventImposePenalty:
		return ps.onPenaltyImposed(sc, data[0].(icmodule.PenaltyType))
	case icmodule.PRepEventRequestUnjail:
		return ps.onUnjailRequested(sc)
	case icmodule.PRepEventTermEnd:
		return ps.onTermEnd(sc, data[0].(Grade), data[1].(int))
	case icmodule.PRepEventValidatorOut:
		return ps.onValidatorOut(sc)
	default:
		panic("UnknownPRepEvent")
	}
	return nil
}

func (ps *PRepStatusState) onBlockVote(sc icmodule.StateContext, voted bool) error {
	voteState := Success
	if !voted {
		voteState = Failure
	}

	if ps.lastState == voteState {
		return nil
	}

	blockHeight := sc.BlockHeight()
	if err := ps.syncBlockVoteStats(blockHeight - 1); err != nil {
		return err
	}

	if voted {
		ps.vFailCont = 0
	} else {
		ps.vFail++
		ps.vFailCont++
	}

	// Common part
	ps.vTotal++
	ps.lastHeight = blockHeight
	ps.lastState = voteState
	ps.setDirty()
	return nil
}

func (ps *PRepStatusState) onMainPRepIn(sc icmodule.StateContext, limit int, termEnd bool) error {
	if termEnd == false {
		if ps.grade != GradeSub {
			return errors.Errorf("Invalid grade: %v -> M", ps.grade)
		}
	}

	ps.grade = GradeMain
	ps.shiftVPenaltyMask(limit)
	if ps.ji.IsUnjailing() {
		status := icmodule.ESEnable
		if termEnd {
			status = icmodule.ESEnableAtNextTerm
		}
		if err := sc.AddEventEnable(ps.owner, status); err != nil {
			return err
		}
	}

	if err := ps.ji.OnMainPRepIn(sc); err != nil {
		return err
	}

	ps.setDirty()
	return nil
}

// OnValidatorOut is called when this PRep node address disappears from ConsensusInfo
func (ps *PRepStatusState) onValidatorOut(sc icmodule.StateContext) error {
	blockHeight := sc.BlockHeight() - 1
	lh := ps.lastHeight
	if blockHeight < lh {
		return errors.Errorf("blockHeight(%d) < lastHeight(%d)", blockHeight, lh)
	}
	if err := ps.syncBlockVoteStats(blockHeight); err != nil {
		return err
	}
	ps.lastState = None
	ps.setDirty()
	return nil
}

func (ps *PRepStatusState) onPenaltyImposed(sc icmodule.StateContext, pt icmodule.PenaltyType) error {
	if pt != icmodule.PenaltyValidationFailure &&
		pt != icmodule.PenaltyAccumulatedValidationFailure &&
		pt != icmodule.PenaltyDoubleSign {
		return nil
	}

	if pt == icmodule.PenaltyValidationFailure {
		blockHeight := sc.BlockHeight()
		if err := ps.syncBlockVoteStats(blockHeight); err != nil {
			return err
		}
		ps.vFailCont = 0
		ps.vPenaltyMask |= 1
	}

	if err := ps.ji.OnPenaltyImposed(sc, pt); err != nil {
		return err
	}
	ps.grade = GradeCandidate
	ps.setDirty()
	return nil
}

func (ps *PRepStatusState) onTermEnd(sc icmodule.StateContext, newGrade Grade, limit int) error {
	ps.resetVFailCont()
	if newGrade == GradeMain {
		if err := ps.onMainPRepIn(sc, limit, true); err != nil {
			return err
		}
	} else {
		ps.grade = newGrade
	}
	if sc.RevisionValue() == icmodule.RevisionResetPenaltyMask {
		ps.vPenaltyMask = 0
	}
	ps.setDirty()
	return nil
}

func (ps *PRepStatusState) onUnjailRequested(sc icmodule.StateContext) error {
	if err := ps.ji.OnUnjailRequested(sc); err != nil {
		return err
	}
	if err := sc.AddEventEnable(ps.owner, icmodule.ESUnjail); err != nil {
		return err
	}
	ps.setDirty()
	return nil
}

// syncBlockVoteStats updates vote stats data at a given blockHeight
func (ps *PRepStatusState) syncBlockVoteStats(blockHeight int64) error {
	if blockHeight < ps.lastHeight {
		return errors.Errorf("blockHeight(%d) < lastHeight(%d)", blockHeight, ps.lastHeight)
	}
	if blockHeight == ps.lastHeight || ps.lastState == None {
		return nil
	}

	ps.vFail = ps.GetVFail(blockHeight)
	ps.vTotal = ps.GetVTotal(blockHeight)
	ps.vFailCont = ps.GetVFailCont(blockHeight)
	ps.lastHeight = blockHeight
	return nil
}

func (ps *PRepStatusState) DisableAs(status Status) (Grade, error) {
	switch ps.status {
	case Active:
		grade := ps.grade
		ps.grade = GradeNone
		ps.status = status
		ps.setDirty()
		return grade, nil
	default:
		return ps.grade, errors.InvalidStateError.Errorf("InvalidState(status=%s)", ps.status)
	}
}

func (ps *PRepStatusState) GetStatsInJSON(sc icmodule.StateContext) map[string]interface{} {
	jso := ps.prepStatusData.GetStatsInJSON(sc.BlockHeight())
	if sc.RevisionValue() >= icmodule.RevisionUpdatePRepStats {
		jso["address"] = ps.owner
	}
	return jso
}

func newPRepStatusWithTag(_ icobject.Tag) *PRepStatusSnapshot {
	return new(PRepStatusSnapshot)
}

func NewPRepStatusWithSnapshot(owner module.Address, snapshot *PRepStatusSnapshot) *PRepStatusState {
	ps := &PRepStatusState{owner: owner}
	return ps.Reset(snapshot)
}

func NewPRepStatus(owner module.Address) *PRepStatusState {
	return NewPRepStatusWithSnapshot(owner, emptyPRepStatusSnapshot)
}
