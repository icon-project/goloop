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
	OnTermEnd(revision, mainPRepCount, subPRepCount, limit int) error
	GetPRepSize(grade Grade) int
	Size() int
	TotalBonded() *big.Int
	TotalDelegated() *big.Int
	GetTotalBondedDelegation(br int64) *big.Int
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
func (p *prepsBase) OnTermEnd(revision, mainPRepCount, subPRepCount, limit int) error {
	mainPReps := 0
	subPReps := 0
	electedPRepCount := mainPRepCount + subPRepCount

	var newGrade Grade
	for i, prep := range p.orderedPReps {
		if i < mainPRepCount {
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

func (p *prepsBase) GetTotalBondedDelegation(br int64) *big.Int {
	tbd := new(big.Int)
	for _, prep := range p.orderedPReps {
		tbd.Add(tbd, prep.GetBondedDelegation(br))
	}
	return tbd
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
		ret[i] = NewPRepSnapshot(prep.Owner(), prep.GetBondedDelegation(br))
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

func (p *prepsBase) sortByBondedDelegation(br int64) {
	sort.Slice(p.orderedPReps, func(i, j int) bool {
		p0, p1 := p.orderedPReps[i], p.orderedPReps[j]
		return lessByBondedDelegation(p0, p1, br)
	})
}

func lessByBondedDelegation(p0, p1 *PRep, br int64) bool {
	ret := p0.GetBondedDelegation(br).Cmp(p1.GetBondedDelegation(br))
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

func NewPRepsOrderedByBondedDelegation(prepList []*PRep, br int64) PRepSet {
	preps := &prepsBase{
		totalDelegated: new(big.Int),
		totalBonded:    new(big.Int),
	}

	for _, prep := range prepList {
		preps.appendPRep(prep)
	}
	preps.sortByBondedDelegation(br)
	return preps
}

// ================================================================

type prepsIncludingExtraMainPRep struct {
	prepsBase
}

func (p *prepsIncludingExtraMainPRep) sort(
	mainPRepCount, extraMainPRepCount, electedPRepCount int, br int64) {
	p.sortByBondedDelegation(br)
	p.sortForExtraMainPRep(mainPRepCount, extraMainPRepCount, electedPRepCount, br)
}

func (p *prepsIncludingExtraMainPRep) sortForExtraMainPRep(
	mainPRepCount, extraMainPRepCount, electedPRepCount int, br int64) {

	// No need to consider extra main preps
	pureMainPRepCount := mainPRepCount - extraMainPRepCount
	size := len(p.orderedPReps)
	if size <= pureMainPRepCount || extraMainPRepCount == 0 {
		return
	}

	// Copy the rest of preps excluding pure main preps to dubRestPReps slice
	restPReps := p.orderedPReps[pureMainPRepCount:electedPRepCount]
	dubRestPReps := make([]*PRep, len(restPReps))
	copy(dubRestPReps, restPReps)

	// Sort restPReps by LRU logic
	sortByLRU(restPReps, br)

	// Add extra main preps to map
	extraMainPReps := make(map[string]*PRep)
	for i := 0; i < extraMainPRepCount; i++ {
		prep := restPReps[i]
		extraMainPReps[icutils.ToKey(prep.Owner())] = prep
	}

	// Append sub preps
	i := extraMainPRepCount
	for _, prep := range dubRestPReps {
		// If prep is not a extra main prep
		if _, ok := extraMainPReps[icutils.ToKey(prep.Owner())]; !ok {
			restPReps[i] = prep
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

	// Sort by bondedDelegation
	cmp := p0.GetBondedDelegation(br).Cmp(p1.GetBondedDelegation(br))
	if cmp > 0 {
		return true
	} else if cmp < 0 {
		return false
	}

	// Sort by address
	return bytes.Compare(p0.Owner().Bytes(), p1.Owner().Bytes()) > 0
}

func NewPRepsIncludingExtraMainPRep(
	prepList []*PRep, mainPRepCount, extraMainPRepCount, electedPRepCount int, br int64) PRepSet {
	preps := &prepsIncludingExtraMainPRep{
		prepsBase: prepsBase{
			totalDelegated: new(big.Int),
			totalBonded:    new(big.Int),
		},
	}

	for _, prep := range prepList {
		preps.appendPRep(prep)
	}
	preps.sort(mainPRepCount, extraMainPRepCount, electedPRepCount, br)
	return preps
}
