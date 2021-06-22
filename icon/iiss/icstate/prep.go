package icstate

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

type PRep struct {
	owner module.Address

	*PRepBaseState
	*PRepStatusState
}

func (p *PRep) Owner() module.Address {
	return p.owner
}

func (p *PRep) ToJSON(blockHeight int64, bondRequirement int64) map[string]interface{} {
	jso := icutils.MergeMaps(p.PRepBaseState.ToJSON(), p.PRepStatusState.ToJSON(blockHeight, bondRequirement))
	jso["address"] = p.owner
	return jso
}

func newPRep(owner module.Address, pb *PRepBaseState, ps *PRepStatusState) *PRep {
	return &PRep{owner: owner, PRepBaseState: pb, PRepStatusState: ps}
}

type PReps struct {
	totalBonded    *big.Int
	totalDelegated *big.Int // total delegated amount of all active P-Reps
	mainPReps      int
	subPReps       int
	orderedPReps   []*PRep
	prepMap        map[string]*PRep
}

func (p *PReps) appendPRep(owner module.Address, prep *PRep) {
	p.prepMap[icutils.ToKey(owner)] = prep
	if prep.PRepStatusState.Status() == Active {
		p.orderedPReps = append(p.orderedPReps, prep)
		p.totalBonded.Add(p.totalBonded, prep.Bonded())
		p.totalDelegated.Add(p.totalDelegated, prep.Delegated())
		p.adjustPRepSize(prep.Grade(), true)
	}
}

func (p *PReps) adjustPRepSize(grade Grade, increment bool) {
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

func (p *PReps) sort(br int64) {
	sort.Slice(p.orderedPReps, func(i, j int) bool {
		ret := p.orderedPReps[i].GetBondedDelegation(br).Cmp(p.orderedPReps[j].GetBondedDelegation(br))
		if ret > 0 {
			return true
		} else if ret < 0 {
			return false
		}

		ret = p.orderedPReps[i].Delegated().Cmp(p.orderedPReps[j].Delegated())
		if ret > 0 {
			return true
		} else if ret < 0 {
			return false
		}

		return bytes.Compare(p.orderedPReps[i].owner.Bytes(), p.orderedPReps[j].owner.Bytes()) > 0
	})
}

// OnTermEnd initializes all prep status including grade on term end
func (p *PReps) OnTermEnd(blockHeight int64, mainPRepCount, subPRepCount, limit int) error {
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

		// Reset VFailContOffset if this prep got penalized during this term
		if err := prep.OnTermEnd(newGrade, limit); err != nil {
			return err
		}
	}

	p.mainPReps = mainPReps
	p.subPReps = subPReps
	return nil
}

func (p *PReps) GetPRepSize(grade Grade) int {
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

func (p *PReps) Size() int {
	return len(p.orderedPReps)
}

func (p *PReps) TotalBonded() *big.Int {
	return p.totalBonded
}

func (p *PReps) TotalDelegated() *big.Int {
	return p.totalDelegated
}

func (p *PReps) GetTotalBondedDelegation(br int64) *big.Int {
	tbd := new(big.Int)
	for _, prep := range p.orderedPReps {
		tbd.Add(tbd, prep.GetBondedDelegation(br))
	}
	return tbd
}

func (p *PReps) GetPRepByIndex(i int) *PRep {
	if i < 0 || i >= len(p.orderedPReps) {
		return nil
	}
	return p.orderedPReps[i]
}

func newPReps(prepList []*PRep, br int64) *PReps {
	preps := newEmptyPReps()

	for _, prep := range prepList {
		preps.appendPRep(prep.Owner(), prep)
	}
	preps.sort(br)
	return preps
}

func newEmptyPReps() *PReps {
	return &PReps{
		totalDelegated: new(big.Int),
		totalBonded:    new(big.Int),
		prepMap:        make(map[string]*PRep),
	}
}
