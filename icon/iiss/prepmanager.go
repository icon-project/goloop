package iiss

import (
	"bytes"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"math/big"
	"sort"
)

type RegInfoIdx int

func (_ RegInfoIdx) Size() int {
	return idxSize
}

const (
	IdxCity RegInfoIdx = iota
	IdxCountry
	IdxDetails
	IdxEmail
	IdxName
	IdxP2pEndpoint
	IdxWebsite

	idxSize = iota - 1
)

type PRep struct {
	*icstate.PRepBase
	*icstate.PRepStatus
	*icstate.State
}

func (p *PRep) Owner() module.Address {
	return p.PRepBase.Owner()
}

func (p *PRep) ToJSON() map[string]interface{} {
	br := icstate.GetBondRequirement(p.State)
	return icutils.MergeMaps(p.PRepBase.ToJSON(), p.PRepStatus.ToJSON(br))
}

func (p *PRep) Clone() *PRep {
	return newPRep(p.Owner(), p.PRepBase, p.PRepStatus, p.State)
}

func newPRep(owner module.Address, base *icstate.PRepBase, status *icstate.PRepStatus, state *icstate.State) *PRep {
	base = base.Clone()
	base.SetOwner(owner)

	status = status.Clone()
	status.SetOwner(owner)

	return &PRep{PRepBase: base, PRepStatus: status, State: state}
}

func setPRep(pb *icstate.PRepBase, node module.Address, params []string) error {
	return pb.SetPRep(
		params[IdxName],
		params[IdxEmail],
		params[IdxWebsite],
		params[IdxCountry],
		params[IdxCity],
		params[IdxDetails],
		params[IdxP2pEndpoint],
		node,
	)
}

// Manage PRepBase, PRepStatus and ActivePRep
type PRepManager struct {
	state          *icstate.State
	totalDelegated *big.Int
	totalStake     *big.Int

	orderedPReps preps
	prepMap      map[string]*PRep
}

type preps []*PRep

func (p preps) Len() int      { return len(p) }
func (p preps) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p preps) Less(i, j int) bool {
	br := icstate.GetBondRequirement(p[i].State)
	ret := p[i].GetBondedDelegation(br).Cmp(p[j].GetBondedDelegation(br))
	if ret < 0 {
		return true
	} else if ret > 0 {
		return false
	}

	ret = p[i].Delegated().Cmp(p[j].Delegated())
	if ret < 0 {
		return true
	} else if ret > 0 {
		return false
	}

	return bytes.Compare(p[i].Owner().Bytes(), p[j].Owner().Bytes()) < 0
}

func (pm *PRepManager) init() {
	size := pm.state.GetActivePRepSize()

	for i := 0; i < size; i++ {
		owner := pm.state.GetActivePRep(i)
		prep := pm.getPRep(owner)
		pm.Add(prep)
	}

	pm.sort()
}

func (pm *PRepManager) getMainPRepCount() int {
	return int(icstate.GetMainPRepCount(pm.state))
}

func (pm *PRepManager) getSubPRepCount() int {
	return int(icstate.GetSubPRepCount(pm.state))
}

func (pm *PRepManager) getPRep(owner module.Address) *PRep {
	base := pm.state.GetPRepBase(owner)
	if base == nil {
		return nil
	}

	status := pm.state.GetPRepStatus(owner)
	return newPRep(owner, base, status, pm.state)
}

func (pm *PRepManager) Add(p *PRep) {
	pm.orderedPReps = append(pm.orderedPReps, p)
	pm.prepMap[icutils.ToKey(p.Owner())] = p
	pm.totalDelegated.Add(pm.totalDelegated, p.Delegated())
}

// sort preps in descending order by bonded delegation
func (pm *PRepManager) sort() {
	sort.Sort(sort.Reverse(pm.orderedPReps))
}

func (pm *PRepManager) Size() int {
	return len(pm.orderedPReps)
}

func (pm *PRepManager) GetPRepByOwner(owner module.Address) *PRep {
	return pm.prepMap[icutils.ToKey(owner)]
}

func (pm *PRepManager) GetPRepByNode(node module.Address) *PRep {
	owner := pm.state.GetOwnerByNode(node)
	if owner == nil {
		owner = node
	}

	return pm.GetPRepByOwner(owner)
}

func (pm *PRepManager) GetPRepByIndex(i int) *PRep {
	return pm.orderedPReps[i]
}

func (pm *PRepManager) TotalDelegated() *big.Int {
	return pm.totalDelegated
}

func (pm *PRepManager) TotalStake() *big.Int {
	// TODO: Not implemented
	return pm.totalStake
}

func (pm *PRepManager) GetValidators() []module.Validator {
	size := len(pm.orderedPReps)
	mainPRepCount := pm.getMainPRepCount()

	if size < mainPRepCount {
		log.Errorf("Not enough PReps: %d < %d", size, mainPRepCount)
	}

	var err error
	var address module.Address
	validators := make([]module.Validator, size)
	for i := 0; i < size; i++ {
		address = pm.orderedPReps[i].GetNode()
		validators[i], err = state.ValidatorFromAddress(address)
		if err != nil {
			log.Errorf("Failed to run GetValidators(): %s", address.String())
		}
	}

	return validators
}

func (pm *PRepManager) GetPRepsInJSON() map[string]interface{} {
	size := len(pm.orderedPReps)
	ret := make(map[string]interface{})
	prepList := make([]map[string]interface{}, size, size)
	ret["preps"] = prepList

	for i, prep := range pm.orderedPReps {
		prepList[i] = prep.ToJSON()
	}

	return ret
}

func (pm *PRepManager) contains(owner module.Address) bool {
	pb := pm.state.GetPRepBase(owner)
	return !pb.IsEmpty()
}

func (pm *PRepManager) RegisterPRep(owner, node module.Address, params []string) error {
	if pm.contains(owner) {
		return errors.Errorf("PRep already exists: %s", owner)
	}

	pb := icstate.NewPRepBase(owner)
	err := setPRep(pb, node, params)
	if err != nil {
		return err
	}

	ps := pm.state.GetPRepStatus(owner)
	if ps == nil {
		ps = icstate.NewPRepStatus(owner)
		pm.state.AddPRepStatus(ps)
	} else {
		// NotReady -> Active
		ps.SetStatus(icstate.Active)
	}

	pm.state.AddPRepBase(pb)
	pm.state.AddActivePRep(owner)
	if err = pm.addNodeToOwner(node, owner); err != nil {
		return err
	}

	return nil
}

func (pm *PRepManager) SetPRep(owner, node module.Address, params []string) error {
	pb := pm.state.GetPRepBase(owner)
	if pb == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}
	return setPRep(pb, node, params)
}

func (pm *PRepManager) UnregisterPRep(owner module.Address) error {
	var err error
	p := pm.getPRep(owner)
	if p == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}

	err = pm.state.RemovePRepBase(owner)
	if err != nil {
		return err
	}
	err = pm.state.RemovePRepStatus(owner)
	if err != nil {
		return err
	}
	pm.totalDelegated.Sub(pm.totalDelegated, p.Delegated())
	return nil
}

func (pm *PRepManager) addNodeToOwner(node, owner module.Address) error {
	if node == nil {
		return nil
	}
	if owner.Equal(node) {
		return nil
	}
	if pm.contains(node) {
		return errors.Errorf("Node must not be owner of other")
	}
	return pm.state.AddNodeToOwner(node, owner)
}

func (pm *PRepManager) ChangeDelegation(od, nd icstate.Delegations) error {
	delta := make(map[string]*big.Int)

	for _, d := range od {
		key := icutils.ToKey(d.Address)
		delta[key] = new(big.Int).Neg(d.Value.Value())
	}
	for _, d := range nd {
		key := icutils.ToKey(d.Address)
		if delta[key] == nil {
			delta[key] = new(big.Int)
		}
		delta[key].Add(delta[key], d.Value.Value())
	}

	delegatedToNotReadyNode := big.NewInt(0)
	var newPs *icstate.PRepStatus
	for k, v := range delta {
		owner, err := common.NewAddress([]byte(k))
		if err != nil {
			return err
		}

		key := icutils.ToKey(owner)
		if delta[key].Cmp(icstate.BigIntZero) != 0 {
			ps := pm.state.GetPRepStatus(owner)
			if ps == nil {
				// Someone tries to set delegation to a PRep which has not been registered
				newPs = icstate.NewPRepStatus(owner)
				newPs.SetStatus(icstate.NotReady)
				delegatedToNotReadyNode.Add(delegatedToNotReadyNode, delta[key])
			} else {
				newPs = ps.Clone()
			}

			newPs.Delegated().Add(newPs.Delegated(), v)

			if newPs.Status() == icstate.NotReady && newPs.Delegated().Cmp(icstate.BigIntZero) == 0 {
				err = pm.state.RemovePRepStatus(owner)
				if err != nil {
					panic(errors.Errorf("PRepStatusCache is broken: %s", owner))
				}
			} else {
				pm.state.AddPRepStatus(newPs)
			}
		}
	}

	totalDelegated := pm.totalDelegated
	totalDelegated.Add(totalDelegated, nd.GetDelegationAmount())
	totalDelegated.Sub(totalDelegated, od.GetDelegationAmount())
	// Ignore the delegation to NotReady PReps
	totalDelegated.Sub(totalDelegated, delegatedToNotReadyNode)
	return nil
}

func (pm *PRepManager) OnTermEnd() error {
	mainPRepCount := pm.getMainPRepCount()
	subPRepCount := pm.getSubPRepCount()

	if len(pm.orderedPReps) < mainPRepCount {
		return nil
	}

	// Main PRep
	electedPRepCount := mainPRepCount + subPRepCount

	for i, prep := range pm.orderedPReps {
		if i < mainPRepCount {
			if prep.Grade() != icstate.Main {
				prep.SetGrade(icstate.Main)
				pm.state.AddPRepStatus(prep.PRepStatus)
			}
		} else if i < electedPRepCount {
			if prep.Grade() != icstate.Sub {
				prep.SetGrade(icstate.Sub)
				pm.state.AddPRepStatus(prep.PRepStatus)
			}
		} else {
			if prep.Grade() != icstate.Candidate {
				prep.SetGrade(icstate.Candidate)
				pm.state.AddPRepStatus(prep.PRepStatus)
			}
		}
	}

	return nil
}

func newPRepManager(state *icstate.State, totalStake *big.Int) *PRepManager {
	pm := &PRepManager{
		state:          state,
		totalDelegated: big.NewInt(0),
		totalStake:     totalStake,

		prepMap: make(map[string]*PRep),
	}

	pm.init()
	return pm
}
