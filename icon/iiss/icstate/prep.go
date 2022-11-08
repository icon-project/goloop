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

func (p *PRep) ToJSON(blockHeight int64, bondRequirement int64, activeDSAMask int64) map[string]interface{} {
	pb := p.getPRepBaseState()
	jso := icutils.MergeMaps(pb.ToJSON(p.owner), p.PRepStatusState.ToJSON(blockHeight, bondRequirement, activeDSAMask))
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
	GetElectedPRepSize() int
	Size() int
	TotalBonded() *big.Int
	TotalDelegated() *big.Int
	GetTotalPower(br int64) *big.Int
	GetByIndex(i int) PRepSetEntry
	ToPRepSnapshots(electedPRepCount int, br int64) PRepSnapshots
	Sort(mainPRepCount, subPRepCount, extraMainPRepCount int, br int64, revision int)
	SortForQuery(br int64, revision int)
}

type PRepSetEntry interface {
	PRep() *PRep
	Status() Status
	Grade() Grade
	Power(br int64) *big.Int
	Delegated() *big.Int
	Bonded() *big.Int
	Owner() module.Address
	HasPubKey() bool
}

type prepSetEntry struct {
	prep   *PRep
	pubKey bool
}

func (p *prepSetEntry) PRep() *PRep {
	return p.prep
}

func (p *prepSetEntry) Status() Status {
	return p.prep.Status()
}

func (p *prepSetEntry) Grade() Grade {
	return p.prep.Grade()
}

func (p *prepSetEntry) Power(br int64) *big.Int {
	return p.prep.GetPower(br)
}

func (p *prepSetEntry) Delegated() *big.Int {
	return p.prep.Delegated()
}

func (p *prepSetEntry) Bonded() *big.Int {
	return p.prep.Bonded()
}

func (p *prepSetEntry) Owner() module.Address {
	return p.prep.Owner()
}

func (p *prepSetEntry) HasPubKey() bool {
	return p.pubKey
}

func NewPRepSetEntry(prep *PRep, pubKey bool) *prepSetEntry {
	return &prepSetEntry{
		prep:   prep,
		pubKey: pubKey,
	}
}

type prepSetImpl struct {
	totalBonded    *big.Int
	totalDelegated *big.Int // total delegated amount of all active P-Reps
	mainPReps      int
	subPReps       int
	entries        []PRepSetEntry
}

// OnTermEnd initializes all prep status including grade on term end
func (p *prepSetImpl) OnTermEnd(revision, mainPRepCount, subPRepCount, extraMainPRepCount, limit int, br int64) error {
	mainPReps := 0
	subPReps := 0
	electedPRepCount := mainPRepCount + subPRepCount

	var newGrade Grade
	for i, entry := range p.entries {
		if revision >= icmodule.RevisionBTP2 &&
			(entry.Power(br).Sign() == 0 || entry.HasPubKey() == false) {
			newGrade = GradeCandidate
		} else if i < mainPRepCount {
			newGrade = GradeMain
			mainPReps++
		} else if i < mainPRepCount+extraMainPRepCount && entry.Power(br).Sign() > 0 {
			// Prevent a prep with 0 power from being an extra main prep
			newGrade = GradeMain
			mainPReps++
		} else if i < electedPRepCount {
			newGrade = GradeSub
			subPReps++
		} else {
			newGrade = GradeCandidate
		}

		prep := entry.PRep()
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
	return len(p.entries)
}

func (p *prepSetImpl) TotalBonded() *big.Int {
	return p.totalBonded
}

func (p *prepSetImpl) TotalDelegated() *big.Int {
	return p.totalDelegated
}

func (p *prepSetImpl) GetTotalPower(br int64) *big.Int {
	totalPower := new(big.Int)
	for _, entry := range p.entries {
		totalPower.Add(totalPower, entry.Power(br))
	}
	return totalPower
}

func (p *prepSetImpl) GetByIndex(i int) PRepSetEntry {
	if i < 0 || i >= len(p.entries) {
		return nil
	}
	return p.entries[i]
}

func (p *prepSetImpl) ToPRepSnapshots(electedPRepCount int, br int64) PRepSnapshots {
	size := icutils.Min(len(p.entries), electedPRepCount)
	if size == 0 {
		return nil
	}

	ret := make(PRepSnapshots, size)
	for i := 0; i < size; i++ {
		entry := p.entries[i]
		ret[i] = NewPRepSnapshot(entry.Owner(), entry.Power(br))
	}
	return ret
}

func (p *prepSetImpl) Sort(mainPRepCount, subPRepCount, extraMainPRepCount int, br int64, rev int) {
	if rev < icmodule.RevisionExtraMainPReps {
		p.sort(br, nil)
	} else {
		if rev < icmodule.RevisionBTP2 {
			p.sort(br, nil)
			p.sortForExtraMainPRep(mainPRepCount, subPRepCount, extraMainPRepCount, br)
		} else {
			p.sort(br, cmpPubKey)
			var electable int
			p.visitAll(func(idx int, e1 PRepSetEntry) bool {
				if e1.Power(br).Sign() > 0 && e1.HasPubKey() {
					electable += 1
					return true
				} else {
					return false
				}
			})
			if electable > mainPRepCount {
				if electable < mainPRepCount+subPRepCount {
					subPRepCount = electable - mainPRepCount
				}
				p.sortForExtraMainPRep(mainPRepCount, subPRepCount, extraMainPRepCount, br)
			}
		}
	}
}

func (p *prepSetImpl) SortForQuery(br int64, revision int) {
	if revision >= icmodule.RevisionBTP2 {
		p.sort(br, cmpPubKey)
	} else {
		p.sort(br, nil)
	}
}

func (p *prepSetImpl) sort(br int64, cmp func(i, j PRepSetEntry) int) {
	sort.Slice(p.entries, func(i, j int) bool {
		p0, p1 := p.entries[i], p.entries[j]
		return lessByPower(p0, p1, br, cmp)
	})
}

func cmpPubKey(e0, e1 PRepSetEntry) int {
	if e0.HasPubKey() != e1.HasPubKey() {
		if e0.HasPubKey() {
			return 1
		}
		return -1
	}
	return 0
}

func lessByPower(e0, e1 PRepSetEntry, br int64, cmp func(i, j PRepSetEntry) int) bool {
	if cmp != nil {
		ret := cmp(e0, e1)
		if ret > 0 {
			return true
		} else if ret < 0 {
			return false
		}
	}
	ret := e0.Power(br).Cmp(e1.Power(br))
	if ret > 0 {
		return true
	} else if ret < 0 {
		return false
	}

	ret = e0.Delegated().Cmp(e1.Delegated())
	if ret > 0 {
		return true
	} else if ret < 0 {
		return false
	}

	return bytes.Compare(e0.Owner().Bytes(), e1.Owner().Bytes()) > 0
}

func (p *prepSetImpl) sortForExtraMainPRep(
	mainPRepCount, subPRepCount, extraMainPRepCount int, br int64) {
	// All counts are configuration values; Default: 22, 78, 3
	size := len(p.entries)
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

	// Copy sub preps from entries to subPReps
	subPRepEntries := p.entries[mainPRepCount:electedPRepCount]
	dupSubPRepEntries := make([]PRepSetEntry, len(subPRepEntries))
	copy(dupSubPRepEntries, subPRepEntries)

	// sort subPReps by LRU logic
	sortByLRU(subPRepEntries, br)

	// Add extra main preps to map
	i := 0
	extraMainPReps := make(map[string]bool)
	for _, entry := range subPRepEntries {
		if entry.Power(br).Sign() > 0 {
			// Prevent the prep whose power is 0 from being an extra main prep
			extraMainPReps[icutils.ToKey(entry.Owner())] = true
			subPRepEntries[i] = entry
			i++
			if i == extraMainPRepCount {
				// All extra main preps are selected
				break
			}
		}
	}

	// Append remaining sub preps excluding extra main preps
	for _, entry := range dupSubPRepEntries {
		// If prep is not an extra main prep
		if _, ok := extraMainPReps[icutils.ToKey(entry.Owner())]; !ok {
			subPRepEntries[i] = entry
			i++
		}
	}
}

func (p *prepSetImpl) visitAll(visit func(idx int, e1 PRepSetEntry) bool) {
	for i, e := range p.entries {
		if ok := visit(i, e); !ok {
			return
		}
	}
}

func sortByLRU(prepSet []PRepSetEntry, br int64) {
	sort.Slice(prepSet, func(i, j int) bool {
		return lessByLRU(prepSet[i].PRep(), prepSet[j].PRep(), br)
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

func NewPRepSet(prepList []PRepSetEntry) PRepSet {
	prepSet := &prepSetImpl{
		totalDelegated: new(big.Int),
		totalBonded:    new(big.Int),
		entries:        prepList,
	}

	for _, entry := range prepList {
		prepSet.totalBonded.Add(prepSet.totalBonded, entry.Bonded())
		prepSet.totalDelegated.Add(prepSet.totalDelegated, entry.Delegated())
		switch entry.Grade() {
		case GradeMain:
			prepSet.mainPReps += 1
		case GradeSub:
			prepSet.subPReps += 1
		case GradeCandidate:
			// Nothing to do
		default:
			panic(errors.Errorf("Invalid grade: %d", entry.Grade()))
		}
	}
	return prepSet
}
