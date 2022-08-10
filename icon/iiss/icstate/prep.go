package icstate

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

type PRep struct {
	owner module.Address
	state *State

	pb *PRepBaseState
	*PRepStatusState
}

func (p *PRep) Owner() module.Address {
	return p.owner
}

func (p *PRep) IRep() *big.Int {
	pb := p.getPRepBaseState()
	if pb == nil {
		return new(big.Int)
	}
	return pb.IRep()
}

func (p *PRep) NodeAddress() module.Address {
	pb := p.getPRepBaseState()
	if pb == nil {
		return nil
	}
	return pb.GetNode(p.owner)
}

func (p *PRep) ToJSON(blockHeight int64, bondRequirement int64) map[string]interface{} {
	pb := p.getPRepBaseState()
	jso := icutils.MergeMaps(pb.ToJSON(p.owner), p.PRepStatusState.ToJSON(blockHeight, bondRequirement))
	jso["address"] = p.owner
	return jso
}

func (p *PRep) init() error {
	ps := p.state.GetPRepStatusByOwner(p.owner, false)
	if ps == nil {
		return errors.Errorf("PRepStatus not found: %s", p.owner)
	}
	p.PRepStatusState = ps
	return nil
}

func (p *PRep) getPRepBaseState() *PRepBaseState {
	if p.pb == nil {
		p.pb = p.state.GetPRepBaseByOwner(p.owner, false)
	}
	return p.pb
}

func (p *PRep) info() *PRepInfo {
	pb := p.getPRepBaseState()
	if pb == nil {
		return nil
	}
	return pb.info()
}

func NewPRep(owner module.Address, state *State) *PRep {
	prep := &PRep{owner: owner, state: state}
	if err := prep.init(); err != nil {
		return nil
	}
	return prep
}

// ===============================================================

type PRepSet interface {
	OnTermEnd(revision, mainPRepCount, subPRepCount, extraMainPRepCount, limit int, br int64) error
	GetPRepSize(grade Grade) int
	Size() int
	TotalBonded() *big.Int
	TotalDelegated() *big.Int
	GetTotalPower(br int64) *big.Int
	GetPRepByIndex(i int) *PRep
	ToPRepSnapshots(electedPRepCount int, br int64) PRepSnapshots
}

type prepsBase struct {
	totalBonded    *big.Int
	totalDelegated *big.Int // total delegated amount of all active P-Reps
	mainPReps      int
	subPReps       int
	orderedPReps   []*PRep
}

// OnTermEnd initializes all prep status including grade on term end
func (p *prepsBase) OnTermEnd(revision, mainPRepCount, subPRepCount, extraMainPRepCount, limit int, br int64) error {
	mainPReps := 0
	subPReps := 0
	electedPRepCount := mainPRepCount + subPRepCount

	var newGrade Grade
	for i, prep := range p.orderedPReps {
		if i < mainPRepCount {
			newGrade = GradeMain
			mainPReps++
		} else if i < mainPRepCount+extraMainPRepCount && prep.GetPower(br).Sign() > 0 {
			// Prevent a prep with 0 power from being an extra main prep
			newGrade = GradeMain
			mainPReps++
		} else if i < electedPRepCount {
			newGrade = GradeSub
			subPReps++
		} else {
			newGrade = GradeCandidate
		}

		if err := prep.OnTermEnd(newGrade, limit); err != nil {
			return err
		}
		if revision == icmodule.RevisionResetPenaltyMask {
			prep.ResetVPenaltyMask()
		}
	}

	p.mainPReps = mainPReps
	p.subPReps = subPReps
	return nil
}

func (p *prepsBase) GetPRepSize(grade Grade) int {
	switch grade {
	case GradeMain:
		return p.mainPReps
	case GradeSub:
		return p.subPReps
	case GradeCandidate:
		return p.Size() - p.mainPReps - p.subPReps
	default:
		panic(errors.Errorf("Invalid grade: %d", grade))
	}
}

func (p *prepsBase) Size() int {
	return len(p.orderedPReps)
}

func (p *prepsBase) TotalBonded() *big.Int {
	return p.totalBonded
}

func (p *prepsBase) TotalDelegated() *big.Int {
	return p.totalDelegated
}

func (p *prepsBase) GetTotalPower(br int64) *big.Int {
	totalPower := new(big.Int)
	for _, prep := range p.orderedPReps {
		totalPower.Add(totalPower, prep.GetPower(br))
	}
	return totalPower
}

func (p *prepsBase) GetPRepByIndex(i int) *PRep {
	if i < 0 || i >= len(p.orderedPReps) {
		return nil
	}
	return p.orderedPReps[i]
}

func (p *prepsBase) ToPRepSnapshots(electedPRepCount int, br int64) PRepSnapshots {
	size := icutils.Min(len(p.orderedPReps), electedPRepCount)
	if size == 0 {
		return nil
	}

	ret := make(PRepSnapshots, size)
	for i := 0; i < size; i++ {
		prep := p.orderedPReps[i]
		ret[i] = NewPRepSnapshot(prep.Owner(), prep.GetPower(br))
	}
	return ret
}

func (p *prepsBase) appendPRep(prep *PRep) {
	if prep.PRepStatusState.Status() == Active {
		p.orderedPReps = append(p.orderedPReps, prep)
		p.totalBonded.Add(p.totalBonded, prep.Bonded())
		p.totalDelegated.Add(p.totalDelegated, prep.Delegated())
		p.adjustPRepSize(prep.Grade(), true)
	}
}

func (p *prepsBase) adjustPRepSize(grade Grade, increment bool) {
	delta := 1
	if !increment {
		delta = -1
	}

	switch grade {
	case GradeMain:
		p.mainPReps += delta
	case GradeSub:
		p.subPReps += delta
	case GradeCandidate:
		// Nothing to do
	default:
		panic(errors.Errorf("Invalid grade: %d", grade))
	}
}

func (p *prepsBase) sortByPower(br int64) {
	sort.Slice(p.orderedPReps, func(i, j int) bool {
		p0, p1 := p.orderedPReps[i], p.orderedPReps[j]
		return lessByPower(p0, p1, br)
	})
}

func lessByPower(p0, p1 *PRep, br int64) bool {
	ret := p0.GetPower(br).Cmp(p1.GetPower(br))
	if ret > 0 {
		return true
	} else if ret < 0 {
		return false
	}

	ret = p0.Delegated().Cmp(p1.Delegated())
	if ret > 0 {
		return true
	} else if ret < 0 {
		return false
	}

	return bytes.Compare(p0.Owner().Bytes(), p1.Owner().Bytes()) > 0
}

// ======================================================================

func NewPRepsOrderedByPower(prepList []*PRep, br int64) PRepSet {
	preps := &prepsBase{
		totalDelegated: new(big.Int),
		totalBonded:    new(big.Int),
	}

	for _, prep := range prepList {
		preps.appendPRep(prep)
	}
	preps.sortByPower(br)
	return preps
}

// ================================================================

type prepsIncludingExtraMainPRep struct {
	prepsBase
}

func (p *prepsIncludingExtraMainPRep) sort(
	mainPRepCount, subPRepCount, extraMainPRepCount int, br int64) {
	p.sortByPower(br)
	p.sortForExtraMainPRep(mainPRepCount, subPRepCount, extraMainPRepCount, br)
}

func (p *prepsIncludingExtraMainPRep) sortForExtraMainPRep(
	mainPRepCount, subPRepCount, extraMainPRepCount int, br int64) {
	// All counts are configuration values; Default: 22, 78, 3
	size := len(p.orderedPReps)
	if size <= mainPRepCount || extraMainPRepCount == 0 {
		// Not enough number of active preps to be extra main preps
		return
	}

	electedPRepCount := mainPRepCount + subPRepCount
	if electedPRepCount > size {
		electedPRepCount = size
	}

	// extraMainPRepCount MUST be larger than zero
	subPRepCount = size - mainPRepCount
	if subPRepCount < extraMainPRepCount {
		extraMainPRepCount = subPRepCount
	}

	// Copy sub preps from orderedPReps to subPReps
	subPReps := p.orderedPReps[mainPRepCount:electedPRepCount]
	dupSubPReps := make([]*PRep, len(subPReps))
	copy(dupSubPReps, subPReps)

	// Sort subPReps by LRU logic
	sortByLRU(subPReps, br)

	// Add extra main preps to map
	i := 0
	extraMainPReps := make(map[string]*PRep)
	for _, prep := range subPReps {
		if prep.GetPower(br).Sign() > 0 {
			// Prevent the prep whose power is 0 from being an extra main prep
			extraMainPReps[icutils.ToKey(prep.Owner())] = prep
			subPReps[i] = prep
			i++
			if i == extraMainPRepCount {
				// All extra main preps are selected
				break
			}
		}
	}

	// Append remaining sub preps excluding extra main preps
	for _, prep := range dupSubPReps {
		// If prep is not an extra main prep
		if _, ok := extraMainPReps[icutils.ToKey(prep.Owner())]; !ok {
			subPReps[i] = prep
			i++
		}
	}
}

func sortByLRU(preps []*PRep, br int64) {
	sort.Slice(preps, func(i, j int) bool {
		return lessByLRU(preps[i], preps[j], br)
	})
}

func lessByLRU(p0, p1 *PRep, br int64) bool {
	// Sort by lastState
	if p0.LastState() == None {
		if p1.LastState() != None {
			return true
		}
	} else {
		if p1.LastState() == None {
			return false
		}
	}

	// p0 and p1 have the same last states at this moment
	// Sort by lastHeight
	if p0.LastState() == None && p0.LastHeight() != p1.LastHeight() {
		return p0.LastHeight() < p1.LastHeight()
	}

	// Sort by power
	cmp := p0.GetPower(br).Cmp(p1.GetPower(br))
	if cmp == 0 {
		// Sort by address
		return bytes.Compare(p0.Owner().Bytes(), p1.Owner().Bytes()) > 0
	}
	return cmp > 0
}

// mainPRepCount does not include extraMainPRepCount
// Example: mainPRepCount: 22, subPRepCount: 78, extraMainPRepCount: 3
func NewPRepsIncludingExtraMainPRep(
	prepList []*PRep, mainPRepCount, subPRepCount, extraMainPRepCount int, br int64) PRepSet {
	preps := &prepsIncludingExtraMainPRep{
		prepsBase: prepsBase{
			totalDelegated: new(big.Int),
			totalBonded:    new(big.Int),
		},
	}

	for _, prep := range prepList {
		preps.appendPRep(prep)
	}
	preps.sort(mainPRepCount, subPRepCount, extraMainPRepCount, br)
	return preps
}
