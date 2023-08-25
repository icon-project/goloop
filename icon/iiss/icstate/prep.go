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

func (p *PRep) ToJSON(sc icmodule.StateContext, bondRequirement icmodule.Rate, activeDSAMask int64) map[string]interface{} {
	pb := p.getPRepBaseState()
	jso := icutils.MergeMaps(
		pb.ToJSON(p.owner),
		p.PRepStatusState.ToJSON(sc, bondRequirement, activeDSAMask),
	)
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

func (p *PRep) Info() *PRepInfo {
	pb := p.getPRepBaseState()
	if pb == nil {
		return nil
	}
	return pb.info()
}

func (p *PRep) HasPubKey(dsaMask int64) bool {
	return p.GetDSAMask()&dsaMask == dsaMask
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
	OnTermEnd(sc icmodule.StateContext,
		mainPRepCount, subPRepCount, extraMainPRepCount, limit int, br icmodule.Rate, dsaMask int64) error
	GetPRepSize(grade Grade) int
	GetElectedPRepSize() int
	Size() int
	TotalBonded() *big.Int
	TotalDelegated() *big.Int
	GetTotalPower(br icmodule.Rate) *big.Int
	GetByIndex(i int) *PRep
	ToPRepSnapshots(electedPRepCount int, br icmodule.Rate) PRepSnapshots
	Sort(mainPRepCount, subPRepCount, extraMainPRepCount int, br icmodule.Rate, revision int, dsaMask int64)
	SortForQuery(br icmodule.Rate, revision int, dsaMask int64)
}

func isPRepElectable(p *PRep, br icmodule.Rate, dsaMask int64) bool {
	if p.GetPower(br).Sign() <= 0 {
		return false
	}
	if !p.HasPubKey(dsaMask) {
		return false
	}
	return p.IsJailInfoElectable()
}

type prepSetImpl struct {
	totalBonded    *big.Int
	totalDelegated *big.Int // total delegated amount of all active P-Reps
	mainPReps      int
	subPReps int
	preps    []*PRep
}

// OnTermEnd initializes all prep status including grade on term end
func (p *prepSetImpl) OnTermEnd(sc icmodule.StateContext,
	mainPRepCount, subPRepCount, extraMainPRepCount, limit int, br icmodule.Rate, dsaMask int64) error {
	revision := sc.Revision()
	mainPReps := 0
	subPReps := 0
	electedPRepCount := mainPRepCount + subPRepCount

	var newGrade Grade
	for i, prep := range p.preps {
		if revision >= icmodule.RevisionBTP2 && !isPRepElectable(prep, br, dsaMask) {
			newGrade = GradeCandidate
		} else if i < mainPRepCount {
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

		if err := prep.OnTermEnd(sc, newGrade, limit); err != nil {
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

func (p *prepSetImpl) GetPRepSize(grade Grade) int {
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

func (p *prepSetImpl) GetElectedPRepSize() int {
	return p.mainPReps + p.subPReps
}

func (p *prepSetImpl) Size() int {
	return len(p.preps)
}

func (p *prepSetImpl) TotalBonded() *big.Int {
	return p.totalBonded
}

func (p *prepSetImpl) TotalDelegated() *big.Int {
	return p.totalDelegated
}

func (p *prepSetImpl) GetTotalPower(br icmodule.Rate) *big.Int {
	totalPower := new(big.Int)
	for _, prep := range p.preps {
		totalPower.Add(totalPower, prep.GetPower(br))
	}
	return totalPower
}

func (p *prepSetImpl) GetByIndex(i int) *PRep {
	if i < 0 || i >= len(p.preps) {
		return nil
	}
	return p.preps[i]
}

func (p *prepSetImpl) ToPRepSnapshots(electedPRepCount int, br icmodule.Rate) PRepSnapshots {
	size := icutils.Min(len(p.preps), electedPRepCount)
	if size == 0 {
		return nil
	}

	ret := make(PRepSnapshots, size)
	for i := 0; i < size; i++ {
		prep := p.preps[i]
		ret[i] = NewPRepSnapshot(prep.Owner(), prep.GetPower(br))
	}
	return ret
}

// Sort sorts the PRepSet based on predefined criteria that may change with each revision
// PRepCount parameters are the metrics in configuration file
// Ex) mainPRepCount(22), subPRepCount(78), extraMainPRepCount(3)
func (p *prepSetImpl) Sort(mainPRepCount, subPRepCount, extraMainPRepCount int, br icmodule.Rate, rev int, dsaMask int64) {
	if rev < icmodule.RevisionExtraMainPReps {
		p.sort(br, dsaMask, nil)
	} else if rev < icmodule.RevisionBTP2 {
		p.sort(br, dsaMask, nil)
		p.sortForExtraMainPRep(mainPRepCount, subPRepCount, extraMainPRepCount, br)
	} else {
		p.sort(br, dsaMask, cmpByValidatorElectable)
		var electable int
		p.visitAll(func(idx int, e *PRep) bool {
			ok := isPRepElectable(e, br, dsaMask)
			if ok {
				electable += 1
			}
			return ok
		})
		if electable > mainPRepCount {
			if electable < mainPRepCount+subPRepCount {
				subPRepCount = electable - mainPRepCount
			}
			p.sortForExtraMainPRep(mainPRepCount, subPRepCount, extraMainPRepCount, br)
		}
	}
}

func (p *prepSetImpl) SortForQuery(br icmodule.Rate, revision int, dsaMask int64) {
	if revision >= icmodule.RevisionBTP2 {
		p.sort(br, dsaMask, cmpByValidatorElectable)
	} else {
		p.sort(br, dsaMask, nil)
	}
}

func (p *prepSetImpl) sort(br icmodule.Rate, dsaMask int64, cmp func(i, j *PRep, dsaMask int64) int) {
	sort.Slice(p.preps, func(i, j int) bool {
		p0, p1 := p.preps[i], p.preps[j]
		return lessByPower(p0, p1, br, dsaMask, cmp)
	})
}

func cmpByValidatorElectable(e0, e1 *PRep, dsaMask int64) int {
	if e0.HasPubKey(dsaMask) != e1.HasPubKey(dsaMask) {
		if e0.HasPubKey(dsaMask) {
			return 1
		}
		return -1
	}

	if e0.IsJailInfoElectable() != e1.IsJailInfoElectable() {
		if e0.IsJailInfoElectable() {
			return 1
		}
		return -1
	}
	return 0
}

func lessByPower(p0, p1 *PRep, br icmodule.Rate, dsaMask int64,
	cmp func(i, j *PRep, dsaMask int64) int) bool {
	var ret int
	if cmp != nil {
		if ret = cmp(p0, p1, dsaMask); ret != 0 {
			return ret > 0
		}
	}
	if ret = p0.GetPower(br).Cmp(p1.GetPower(br)); ret != 0 {
		return ret > 0
	}
	if ret = p0.Delegated().Cmp(p1.Delegated()); ret != 0 {
		return ret > 0
	}
	return bytes.Compare(p0.Owner().Bytes(), p1.Owner().Bytes()) > 0
}

func (p *prepSetImpl) sortForExtraMainPRep(
	mainPRepCount, subPRepCount, extraMainPRepCount int, br icmodule.Rate) {
	// All counts are configuration values; Default: 22, 78, 3
	size := len(p.preps)
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

	// Copy sub preps from preps to subPReps
	subPReps := p.preps[mainPRepCount:electedPRepCount]
	dupSubPReps := make([]*PRep, len(subPReps))
	copy(dupSubPReps, subPReps)

	// sort subPReps by LRU logic
	sortByLRU(subPReps, br)

	// Add extra main preps to map
	i := 0
	extraMainPReps := make(map[string]bool)
	for _, prep := range subPReps {
		if prep.GetPower(br).Sign() > 0 {
			// Prevent the prep whose power is 0 from being an extra main prep
			extraMainPReps[icutils.ToKey(prep.Owner())] = true
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

func (p *prepSetImpl) visitAll(visit func(idx int, e1 *PRep) bool) {
	for i, e := range p.preps {
		if ok := visit(i, e); !ok {
			return
		}
	}
}

func sortByLRU(prepSet []*PRep, br icmodule.Rate) {
	sort.Slice(prepSet, func(i, j int) bool {
		return lessByLRU(prepSet[i], prepSet[j], br)
	})
}

func lessByLRU(p0, p1 *PRep, br icmodule.Rate) bool {
	if p0.IsUnjailing() != p1.IsUnjailing() {
		return p0.IsUnjailing()
	}
	if p0.IsUnjailing() {
		// If both of preps are unjailing, compare their unjailRequestHeight
		if p0.UnjailRequestHeight() != p1.UnjailRequestHeight() {
			return p0.UnjailRequestHeight() < p1.UnjailRequestHeight()
		}
	}

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

func NewPRepSet(prepList []*PRep) PRepSet {
	prepSet := &prepSetImpl{
		totalDelegated: new(big.Int),
		totalBonded:    new(big.Int),
		preps:          prepList,
	}

	for _, prep := range prepList {
		prepSet.totalBonded.Add(prepSet.totalBonded, prep.Bonded())
		prepSet.totalDelegated.Add(prepSet.totalDelegated, prep.Delegated())
		switch prep.Grade() {
		case GradeMain:
			prepSet.mainPReps += 1
		case GradeSub:
			prepSet.subPReps += 1
		case GradeCandidate:
			// Nothing to do
		default:
			panic(errors.Errorf("Invalid grade: %d", prep.Grade()))
		}
	}
	return prepSet
}
