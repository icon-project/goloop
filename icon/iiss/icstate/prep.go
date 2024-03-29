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

func (p *PRep) ToJSON(sc icmodule.StateContext) map[string]interface{} {
	pb := p.getPRepBaseState()
	jso := icutils.MergeMaps(pb.ToJSON(p.owner), p.PRepStatusState.ToJSON(sc))
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

func (p *PRep) IsElectable(sc icmodule.StateContext) bool {
	if !p.IsActive() {
		return false
	}

	if p.GetPower(sc.GetBondRequirement()).Sign() <= 0 {
		return false
	}

	rev := sc.RevisionValue()
	if rev >= icmodule.RevisionBTP2 {
		if !p.HasPubKey(sc.GetActiveDSAMask()) {
			return false
		}
	}
	if rev >= icmodule.RevisionIISS4R1 {
		if !p.IsJailInfoElectable() {
			return false
		}
	}
	return true
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
	OnTermEnd(sc icmodule.StateContext, limit int) error
	GetPRepSize(grade Grade) int
	Size() int
	GetByIndex(i int) *PRep
	ToPRepSnapshots(br icmodule.Rate) PRepSnapshots
}

type prepSetImpl struct {
	mainPRepCount    int // including extraMainPReps
	electedPRepCount int
	// | mainPReps | extraMainPReps | subPReps |
	preps            []*PRep
}

func (p *prepSetImpl) OnTermEnd(sc icmodule.StateContext, limit int) error {
	// Assume that p.preps has been already sorted properly according to the current revision
	var newGrade Grade
	for i, prep := range p.preps {
		if i < p.mainPRepCount {
			// Prevent a prep with 0 power from being an extra main prep
			newGrade = GradeMain
		} else if i < p.electedPRepCount {
			newGrade = GradeSub
		} else {
			newGrade = GradeCandidate
		}

		if err := prep.OnEvent(sc, icmodule.PRepEventTermEnd, newGrade, limit); err != nil {
			return err
		}
	}
	return nil
}

func (p *prepSetImpl) GetPRepSize(grade Grade) int {
	switch grade {
	case GradeMain:
		return p.mainPRepCount
	case GradeSub:
		return p.electedPRepCount - p.mainPRepCount
	case GradeCandidate:
		return p.Size() - p.electedPRepCount
	default:
		panic(errors.Errorf("Invalid grade: %d", grade))
	}
}

func (p *prepSetImpl) Size() int {
	return len(p.preps)
}

func (p *prepSetImpl) GetByIndex(i int) *PRep {
	if i < 0 || i >= len(p.preps) {
		return nil
	}
	return p.preps[i]
}

func (p *prepSetImpl) ToPRepSnapshots(br icmodule.Rate) PRepSnapshots {
	size := p.electedPRepCount
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
// PRepCount parameters are the metrics in network value configuration
// Ex) mainPRepCount(22), subPRepCount(78), extraMainPRepCount(3)
func (p *prepSetImpl) init(sc icmodule.StateContext, cfg PRepCountConfig) {
	rev := sc.RevisionValue()
	if rev < icmodule.RevisionBTP2 {
		p.sortBeforeRevBTP2(sc, cfg)
	} else {
		p.sortAfterRevBTP2(sc, cfg)
	}
}

func cmpByValidatorElectable(p0, p1 *PRep, dsaMask int64) int {
	if p0.HasPubKey(dsaMask) != p1.HasPubKey(dsaMask) {
		if p0.HasPubKey(dsaMask) {
			return 1
		}
		return -1
	}
	if p0.IsJailInfoElectable() != p1.IsJailInfoElectable() {
		if p0.IsJailInfoElectable() {
			return 1
		}
		return -1
	}
	return 0
}

func lessByPower(sc icmodule.StateContext, p0, p1 *PRep, cmp func(i, j *PRep, dsaMask int64) int) bool {
	var ret int
	if cmp != nil {
		if ret = cmp(p0, p1, sc.GetActiveDSAMask()); ret != 0 {
			return ret > 0
		}
	}
	br := sc.GetBondRequirement()
	if ret = p0.GetPower(br).Cmp(p1.GetPower(br)); ret != 0 {
		return ret > 0
	}
	if ret = p0.Delegated().Cmp(p1.Delegated()); ret != 0 {
		return ret > 0
	}
	return bytes.Compare(p0.Owner().Bytes(), p1.Owner().Bytes()) > 0
}

func (p *prepSetImpl) sortBeforeRevBTP2(sc icmodule.StateContext, cfg PRepCountConfig) {
	SortByPower(sc, p.preps)

	size := len(p.preps)

	mainPRepCount := icutils.Min(cfg.MainPReps(), size)
	electedPRepCount := icutils.Min(cfg.ElectedPReps(), size)
	extraMainPRepCount := icutils.Min(cfg.ExtraMainPReps(), electedPRepCount-mainPRepCount)

	if sc.RevisionValue() < icmodule.RevisionExtraMainPReps || extraMainPRepCount <= 0 {
		// No need to find extraMainPReps
		p.setPRepCounts(mainPRepCount, electedPRepCount)
		return
	}

	// Find extraMainPReps

	// Copy sub preps from preps to subPReps
	subPReps := p.preps[mainPRepCount:electedPRepCount]
	dupSubPReps := make([]*PRep, len(subPReps))
	copy(dupSubPReps, subPReps)

	// sort subPReps by LRU logic
	// Priority: low lastBH, high power, high address
	sortByLRU(sc, subPReps)

	// Find eligible extraMainPReps in subPReps
	br := sc.GetBondRequirement()
	extraMainPReps := chooseExtraMainPReps(subPReps, extraMainPRepCount, func(prep *PRep) bool {
		return prep.GetPower(br).Sign() > 0
	})
	copy(subPReps, extraMainPReps)
	extraMainPRepCount = len(extraMainPReps)

	// Move the extraMainPReps found above to the front of other subPReps,
	// filling excludePReps map with extraMainPReps
	excludePReps := make(map[string]bool)
	for _, prep := range extraMainPReps {
		excludePReps[icutils.ToKey(prep.Owner())] = true
	}

	// Append remaining subPReps excluding extraMainPReps
	// p.preps: | MainPReps | ExtraMainPReps | SubPReps |
	copyPReps(subPReps[extraMainPRepCount:], dupSubPReps, excludePReps)
	p.setPRepCounts(mainPRepCount+extraMainPRepCount, electedPRepCount)
}

func (p *prepSetImpl) sortAfterRevBTP2(sc icmodule.StateContext, cfg PRepCountConfig) {
	// All counts are configuration values; Default: 22, 78, 3
	SortByPower(sc, p.preps)

	// Count the number of electable PReps
	size := 0
	for _, prep := range p.preps {
		if prep.IsElectable(sc) {
			size++
		}
	}

	mainPRepCount := icutils.Min(cfg.MainPReps(), size)
	electedPRepCount := icutils.Min(cfg.ElectedPReps(), size)
	extraMainPRepCount := icutils.Min(cfg.ExtraMainPReps(), electedPRepCount-mainPRepCount)

	if extraMainPRepCount <= 0 {
		// No need to find extra MainPReps
		p.setPRepCounts(mainPRepCount, electedPRepCount)
		return
	}

	// Find extraMainPReps

	// Copy sub preps from preps to subPReps
	subPReps := p.preps[mainPRepCount:electedPRepCount]
	dupSubPReps := make([]*PRep, len(subPReps))
	copy(dupSubPReps, subPReps)

	// sort subPReps by LRU logic
	// Priority: low unjailRequestBH, low lastBH, high power, high address
	sortByLRU(sc, subPReps)

	// Add extra main preps to map
	excludePReps := make(map[string]bool)
	for i := 0; i < extraMainPRepCount; i++ {
		// Assume that subPReps are electable
		excludePReps[icutils.ToKey(subPReps[i].Owner())] = true
	}

	// Append remaining subPReps excluding extraMainPReps
	// p.preps: | MainPReps | ExtraMainPReps | SubPReps |
	copyPReps(subPReps[extraMainPRepCount:], dupSubPReps, excludePReps)
	p.setPRepCounts(mainPRepCount+extraMainPRepCount, electedPRepCount)
}

// setPRepCounts set mainPRepCount and electedPRepCount on PRepSorting finalization
// mainPReps: # of real mainPReps including extraMainPReps, not config value
// electedPReps: # of real electedPReps, not config value
func (p *prepSetImpl) setPRepCounts(mainPRepCount, electedPRepCount int) {
	// including extraMainPReps
	p.mainPRepCount = mainPRepCount
	p.electedPRepCount = electedPRepCount
}

func chooseExtraMainPReps(preps []*PRep, size int, isOk func(prep *PRep) bool) []*PRep {
	extras := make([]*PRep, 0, size)
	for _, prep := range preps {
		if len(extras) == size {
			break
		}
		if isOk == nil || isOk(prep) {
			extras = append(extras, prep)
		}
	}
	return extras
}

func copyPReps(dstPReps, srcPReps []*PRep, excludeMap map[string]bool) {
	i := 0
	for _, prep := range srcPReps {
		if excludeMap == nil || !excludeMap[icutils.ToKey(prep.Owner())] {
			dstPReps[i] = prep
			i++
		}
	}
}

func lessByLRU(sc icmodule.StateContext, p0, p1 *PRep) bool {
	if sc.TermIISSVersion() >= IISSVersion4 {
		if p0.IsUnjailing() != p1.IsUnjailing() {
			return p0.IsUnjailing()
		}
		if p0.IsUnjailing() {
			// If both of preps are unjailing, compare their unjailRequestHeight
			if p0.UnjailRequestHeight() != p1.UnjailRequestHeight() {
				return p0.UnjailRequestHeight() < p1.UnjailRequestHeight()
			}
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
	br := sc.GetBondRequirement()
	cmp := p0.GetPower(br).Cmp(p1.GetPower(br))
	if cmp == 0 {
		// Sort by address
		return bytes.Compare(p0.Owner().Bytes(), p1.Owner().Bytes()) > 0
	}
	return cmp > 0
}

// SortByPower sorts given preps to classify active PReps into 3 grades; main, sub, candidate
func SortByPower(sc icmodule.StateContext, preps []*PRep) {
	var cmp func(i, j *PRep, dsaMask int64) int
	if sc.RevisionValue() >= icmodule.RevisionBTP2 {
		cmp = cmpByValidatorElectable
	}
	// Priority: hasPubKey, JailInfoElectable, high power, high delegated, high address
	sort.Slice(preps, func(i, j int) bool {
		return lessByPower(sc, preps[i], preps[j], cmp)
	})
}

// sortByLRU sorts given preps to find extraMainPReps
func sortByLRU(sc icmodule.StateContext, preps []*PRep) {
	// sort subPReps by LRU logic
	// Priority: low unjailRequestBH, low lastBH, high power, high address
	sort.Slice(preps, func(i, j int) bool {
		return lessByLRU(sc, preps[i], preps[j])
	})
}

func NewPRepSet(sc icmodule.StateContext, preps []*PRep, cfg PRepCountConfig) PRepSet {
	prepSet := &prepSetImpl{
		preps: preps,
	}
	prepSet.init(sc, cfg)
	return prepSet
}
