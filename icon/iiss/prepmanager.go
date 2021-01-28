package iiss

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
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
}

func (p *PRep) Owner() module.Address {
	return p.PRepBase.Owner()
}

func (p *PRep) ToJSON(blockHeight int64, bondRequirement int64) map[string]interface{} {
	return icutils.MergeMaps(p.PRepBase.ToJSON(), p.PRepStatus.ToJSON(blockHeight, bondRequirement))
}

func (p *PRep) Clone() *PRep {
	return newPRep(p.Owner(), p.PRepBase, p.PRepStatus)
}

func newPRep(owner module.Address, base *icstate.PRepBase, status *icstate.PRepStatus) *PRep {
	base = base.Clone()
	base.SetOwner(owner)

	status = status.Clone()
	status.SetOwner(owner)

	return &PRep{PRepBase: base, PRepStatus: status}
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
	state *icstate.State

	totalDelegated *big.Int // total delegated amount of all active P-Reps
	totalStake     *big.Int // total stake of all ICONist

	orderedPReps preps
	prepMap      map[string]*PRep
}

type preps []*PRep

func (pm *PRepManager) init() {
	size := pm.state.GetActivePRepSize()

	for i := 0; i < size; i++ {
		owner := pm.state.GetActivePRep(i)
		prep := pm.getPRep(owner, true)
		pm.Add(prep)
	}

	pm.sort()
}

func (pm *PRepManager) getMainPRepCount() int {
	return int(pm.state.GetMainPRepCount())
}

func (pm *PRepManager) getSubPRepCount() int {
	return int(pm.state.GetSubPRepCount())
}

func (pm *PRepManager) getPRep(owner module.Address, createIfNotExist bool) *PRep {
	base := pm.state.GetPRepBase(owner, createIfNotExist)
	if base == nil {
		return nil
	}

	status := pm.state.GetPRepStatus(owner, createIfNotExist)
	return newPRep(owner, base, status)
}

func (pm *PRepManager) Add(p *PRep) {
	pm.orderedPReps = append(pm.orderedPReps, p)
	pm.prepMap[icutils.ToKey(p.Owner())] = p
	pm.totalDelegated.Add(pm.totalDelegated, p.GetVoted())
}

// sort preps in descending order by bonded delegation
func (pm *PRepManager) sort() {
	//sort.Sort(sort.Reverse(pm.orderedPReps))
	br := pm.state.GetBondRequirement()
	sort.Slice(pm.orderedPReps, func(i, j int) bool {
		ret := pm.orderedPReps[i].GetBondedDelegation(br).Cmp(pm.orderedPReps[j].GetBondedDelegation(br))
		if ret > 0 {
			return true
		} else if ret < 0 {
			return false
		}

		ret = pm.orderedPReps[i].Delegated().Cmp(pm.orderedPReps[i].Delegated())
		if ret > 0 {
			return true
		} else if ret < 0 {
			return false
		}

		return bytes.Compare(pm.orderedPReps[i].Owner().Bytes(), pm.orderedPReps[j].Owner().Bytes()) > 0
	})
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

func (pm *PRepManager) GetTotalBondedDelegation() *big.Int {
	total := new(big.Int)
	br := pm.state.GetBondRequirement()
	for _, prep := range pm.orderedPReps {
		total.Add(total, prep.GetBondedDelegation(br))
	}
	return total
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

func (pm *PRepManager) ToJSON() map[string]interface{} {
	ret := make(map[string]interface{})
	ret["totalStake"] = intconv.FormatBigInt(pm.totalStake)
	ret["totalDelegated"] = intconv.FormatBigInt(pm.totalDelegated)
	ret["totalBondedDelegation"] = intconv.FormatBigInt(pm.GetTotalBondedDelegation())

	return ret
}

func (pm *PRepManager) GetPRepsInJSON(blockHeight int64) map[string]interface{} {
	size := len(pm.orderedPReps)
	ret := make(map[string]interface{})
	prepList := make([]map[string]interface{}, size, size)
	ret["preps"] = prepList
	br := pm.state.GetBondRequirement()
	for i, prep := range pm.orderedPReps {
		prepList[i] = prep.ToJSON(blockHeight, br)
	}

	return ret
}

func (pm *PRepManager) contains(owner module.Address) bool {
	pb := pm.state.GetPRepBase(owner, false)
	if pb == nil {
		return false
	}
	return !pb.IsEmpty()
}

func (pm *PRepManager) RegisterPRep(owner, node module.Address, params []string) error {
	if pm.contains(owner) {
		return errors.Errorf("PRep already exists: %s", owner)
	}

	pb := pm.state.GetPRepBase(owner, true)
	err := setPRep(pb, node, params)
	if err != nil {
		return err
	}

	ps := pm.state.GetPRepStatus(owner, true)
	ps.SetStatus(icstate.Active)

	pm.state.AddActivePRep(owner)
	if err = pm.addNodeToOwner(node, owner); err != nil {
		return err
	}

	return nil
}

func (pm *PRepManager) SetPRep(owner, node module.Address, params []string) error {
	pb := pm.state.GetPRepBase(owner, false)
	if pb == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}
	return setPRep(pb, node, params)
}

func (pm *PRepManager) UnregisterPRep(owner module.Address) error {
	p := pm.getPRep(owner, false)
	if p == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}

	pm.totalDelegated.Sub(pm.totalDelegated, p.GetVoted())

	pb := pm.state.GetPRepBase(owner, false)
	if pb != nil {
		pb.Clear()
	}

	ps := pm.state.GetPRepStatus(owner, false)
	if ps != nil {
		ps.Clear()
	}
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

func (pm *PRepManager) ChangeDelegation(od, nd icstate.Delegations) (map[string]*big.Int, error) {
	delta := make(map[string]*big.Int)

	for _, d := range od {
		key := icutils.ToKey(d.To())
		delta[key] = new(big.Int).Neg(d.Value.Value())
	}
	for _, d := range nd {
		key := icutils.ToKey(d.To())
		if delta[key] == nil {
			delta[key] = new(big.Int)
		}
		delta[key].Add(delta[key], d.Value.Value())
	}

	delegatedToInactiveNode := big.NewInt(0)
	for key, value := range delta {
		owner, err := common.NewAddress([]byte(key))
		if err != nil {
			return nil, err
		}
		if value.Sign() != 0 {
			ps := pm.state.GetPRepStatus(owner, true)
			if !ps.IsActive() {
				delegatedToInactiveNode.Add(delegatedToInactiveNode, value)
			}
			ps.Delegated().Add(ps.Delegated(), value)
		}
	}

	totalDelegated := pm.totalDelegated
	totalDelegated.Add(totalDelegated, nd.GetDelegationAmount())
	totalDelegated.Sub(totalDelegated, od.GetDelegationAmount())
	// Ignore the delegated amount to Inactive P-Rep
	totalDelegated.Sub(totalDelegated, delegatedToInactiveNode)
	return delta, nil
}

func (pm *PRepManager) ChangeBond(oBonds, nBonds icstate.Bonds) (map[string]*big.Int, error) {
	delta := make(map[string]*big.Int)

	for _, bond := range oBonds {
		key := icutils.ToKey(bond.To())
		delta[key] = new(big.Int).Neg(bond.Amount())
	}
	for _, bond := range nBonds {
		key := icutils.ToKey(bond.To())
		if delta[key] == nil {
			delta[key] = new(big.Int)
		}
		delta[key].Add(delta[key], bond.Amount())
	}

	bondedToInactiveNode := big.NewInt(0)
	for key, value := range delta {
		owner, err := common.NewAddress([]byte(key))
		if err != nil {
			return nil, err
		}

		if value.Sign() != 0 {
			ps := pm.state.GetPRepStatus(owner, true)
			if !ps.IsActive() {
				bondedToInactiveNode.Add(bondedToInactiveNode, value)
			}
			ps.Bonded().Add(ps.Bonded(), value)
		}
	}
	totalDelegated := pm.totalDelegated
	totalDelegated.Add(totalDelegated, oBonds.GetBondAmount())
	totalDelegated.Sub(totalDelegated, nBonds.GetBondAmount())
	// Ignore the bonded amount to inactive P-Rep
	totalDelegated.Sub(totalDelegated, bondedToInactiveNode)
	return delta, nil
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
				ps := pm.state.GetPRepStatus(prep.Owner(), true)
				ps.Set(prep.PRepStatus)
				ps.SetGrade(icstate.Main)
			}
		} else if i < electedPRepCount {
			if prep.Grade() != icstate.Sub {
				ps := pm.state.GetPRepStatus(prep.Owner(), true)
				ps.Set(prep.PRepStatus)
				ps.SetGrade(icstate.Sub)
			}
		} else {
			if prep.Grade() != icstate.Candidate {
				ps := pm.state.GetPRepStatus(prep.Owner(), true)
				ps.Set(prep.PRepStatus)
				ps.SetGrade(icstate.Candidate)
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
		prepMap:        make(map[string]*PRep),
	}

	pm.init()
	return pm
}
