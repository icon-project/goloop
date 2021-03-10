package iiss

import (
	"bytes"
	"github.com/icon-project/goloop/common/log"
	"math/big"
	"sort"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

type RegInfo struct {
	city        string
	country     string
	details     string
	email       string
	name        string
	p2pEndpoint string
	website     string
	owner       module.Address
	node        module.Address
}

func NewRegInfo(city, country, details, email, name, p2pEndpoint, website string, node, owner module.Address) *RegInfo {
	if node == nil {
		node = owner
	}
	return &RegInfo{
		city:        city,
		country:     country,
		details:     details,
		email:       email,
		name:        name,
		p2pEndpoint: p2pEndpoint,
		website:     website,
		node:        node,
		owner:       owner,
	}
}

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
	return newPRep(p.Owner(), p.PRepBase.Clone(), p.PRepStatus.Clone())
}

func newPRep(owner module.Address, pb *icstate.PRepBase, ps *icstate.PRepStatus) *PRep {
	pb.SetOwner(owner)
	ps.SetOwner(owner)
	return &PRep{PRepBase: pb, PRepStatus: ps}
}

func setPRep(pb *icstate.PRepBase, regInfo *RegInfo) error {
	return pb.SetPRep(
		regInfo.name,
		regInfo.email,
		regInfo.website,
		regInfo.country,
		regInfo.city,
		regInfo.details,
		regInfo.p2pEndpoint,
		regInfo.node,
	)
}

// Manage PRepBase, PRepStatus and ActivePRep
type PRepManager struct {
	state *icstate.State

	totalBonded    *big.Int
	totalDelegated *big.Int // total delegated amount of all active P-Reps

	mainPReps    int
	subPReps     int
	orderedPReps []*PRep
	prepMap      map[string]*PRep
}

func (pm *PRepManager) init() {
	size := pm.state.GetActivePRepSize()

	for i := 0; i < size; i++ {
		owner := pm.state.GetActivePRep(i)
		prep := pm.getPRepFromState(owner)
		if prep == nil {
			log.Warnf("Failed to load PRep: %s", owner)
		} else {
			pm.appendPRep(prep)
		}
	}
}

func (pm *PRepManager) GetPRepSize(grade icstate.Grade) int {
	switch grade {
	case icstate.Main:
		return pm.mainPReps
	case icstate.Sub:
		return pm.subPReps
	case icstate.Candidate:
		return pm.Size() - pm.mainPReps - pm.subPReps
	default:
		panic(errors.Errorf("Invalid grade: %d", grade))
	}
}

func (pm *PRepManager) getBondRequirement() int64 {
	return pm.state.GetBondRequirement()
}

func (pm *PRepManager) getPRepFromState(owner module.Address) *PRep {
	pb := pm.state.GetPRepBase(owner, false)
	if pb == nil {
		return nil
	}

	ps := pm.state.GetPRepStatus(owner, false)
	if ps == nil {
		panic(errors.Errorf("PRepStatus not found: %s", owner))
	}

	return newPRep(owner, pb, ps)
}

func (pm *PRepManager) appendPRep(p *PRep) {
	pm.orderedPReps = append(pm.orderedPReps, p)
	pm.prepMap[icutils.ToKey(p.Owner())] = p
	pm.totalBonded.Add(pm.totalBonded, p.Bonded())
	pm.totalDelegated.Add(pm.totalDelegated, p.Delegated())
	pm.adjustPRepSize(p.Grade(), true)
}

func (pm *PRepManager) adjustPRepSize(grade icstate.Grade, increment bool) {
	delta := 1
	if !increment {
		delta = -1
	}

	switch grade {
	case icstate.Main:
		pm.mainPReps += delta
	case icstate.Sub:
		pm.subPReps += delta
	case icstate.Candidate:
		// Nothing to do
	default:
		panic(errors.Errorf("Invalid grade: %d", grade))
	}
}

// Sort preps in descending order by bonded delegation
func (pm *PRepManager) Sort() {
	br := pm.getBondRequirement()
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

func (pm *PRepManager) ChangeGrade(owner module.Address, grade icstate.Grade) error {
	prep := pm.GetPRepByOwner(owner)
	if prep == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}

	oldGrade := prep.Grade()
	if oldGrade != grade {
		prep.SetGrade(grade)
		pm.adjustPRepSize(oldGrade, false)
		pm.adjustPRepSize(grade, true)
	}
	return nil
}

func (pm *PRepManager) GetPRepByOwner(owner module.Address) *PRep {
	return pm.prepMap[icutils.ToKey(owner)]
}

func (pm *PRepManager) GetPRepByNode(node module.Address) *PRep {
	owner := pm.GetOwnerByNode(node)
	return pm.GetPRepByOwner(owner)
}

func (pm *PRepManager) GetOwnerByNode(node module.Address) module.Address {
	owner := pm.state.GetOwnerByNode(node)
	if owner == nil {
		owner = node
	}
	return owner
}

func (pm *PRepManager) GetNodeByOwner(owner module.Address) module.Address {
	prep := pm.GetPRepByOwner(owner)
	if prep == nil {
		return nil
	}
	return prep.GetNode()
}

func (pm *PRepManager) GetPRepByIndex(i int) *PRep {
	if i < 0 || i >= len(pm.orderedPReps) {
		return nil
	}
	return pm.orderedPReps[i]
}

// TotalBonded returns the sum of PRep.Bonded()
func (pm *PRepManager) TotalBonded() *big.Int {
	return pm.totalBonded
}

// TotalDelegated returns the sum of PRep.Delegated()
func (pm *PRepManager) TotalDelegated() *big.Int {
	return pm.totalDelegated
}

func (pm *PRepManager) GetTotalBondedDelegation(br int64) *big.Int {
	total := new(big.Int)
	for _, prep := range pm.orderedPReps {
		total.Add(total, prep.GetBondedDelegation(br))
	}
	return total
}

func (pm *PRepManager) GetValidators(term *icstate.Term) ([]module.Validator, error) {
	mainPRepCount := pm.mainPReps
	size := term.GetPRepSnapshotCount()
	if size < mainPRepCount {
		return nil, errors.Errorf("Not enough PReps: %d", size)
	}

	vs := make([]module.Validator, 0)

	for i := 0; i < size; i++ {
		pss := term.GetPRepSnapshotByIndex(i)
		prep := pm.GetPRepByOwner(pss.Owner())
		if prep == nil {
			// Some PReps can be disabled
			continue
		}
		if prep.Grade() != icstate.Main {
			continue
		}

		v, err := state.ValidatorFromAddress(prep.GetNode())
		if err != nil {
			return nil, err
		}

		vs = append(vs, v)
		if len(vs) == mainPRepCount {
			break
		}
	}

	return vs, nil
}

func (pm *PRepManager) ToJSON(totalStake *big.Int) map[string]interface{} {
	br := pm.getBondRequirement()
	jso := make(map[string]interface{})
	jso["totalStake"] = totalStake
	jso["totalBonded"] = pm.totalBonded
	jso["totalDelegated"] = pm.totalDelegated
	jso["totalBondedDelegation"] = pm.GetTotalBondedDelegation(br)
	jso["preps"] = pm.Size()
	return jso
}

func (pm *PRepManager) GetPRepsInJSON(blockHeight int64, start, end int) (map[string]interface{}, error) {
	if start < 0 {
		return nil, errors.IllegalArgumentError.Errorf("start(%d) < 0", start)
	}
	if end < 0 {
		return nil, errors.IllegalArgumentError.Errorf("end(%d) < 0", end)
	}

	size := len(pm.orderedPReps)
	if start > end {
		return nil, errors.IllegalArgumentError.Errorf("start(%d) > end(%d)", start, end)
	}
	if start > size {
		return nil, errors.IllegalArgumentError.Errorf("start(%d) > # of preps(%d)", start, size)
	}
	if start == 0 {
		start = 1
	}
	if end == 0 || end > size {
		end = size
	}

	jso := make(map[string]interface{})
	prepList := make([]interface{}, 0, end)
	br := pm.getBondRequirement()

	for i := start - 1; i < end; i++ {
		prepList = append(prepList, pm.orderedPReps[i].ToJSON(blockHeight, br))
	}

	jso["startRanking"] = start
	jso["preps"] = prepList
	jso["totalDelegated"] = pm.TotalDelegated()
	return jso, nil
}

func (pm *PRepManager) contains(owner module.Address) bool {
	_, ok := pm.prepMap[icutils.ToKey(owner)]
	return ok
}

func (pm *PRepManager) RegisterPRep(regInfo *RegInfo) error {
	if regInfo == nil {
		return errors.Errorf("Invalid argument: regInfo")
	}

	node := regInfo.node
	owner := regInfo.owner

	if pm.contains(owner) {
		return errors.Errorf("PRep already exists: %s", owner)
	}

	pb := pm.state.GetPRepBase(owner, true)
	err := setPRep(pb, regInfo)
	if err != nil {
		return err
	}

	ps := pm.state.GetPRepStatus(owner, true)
	ps.SetStatus(icstate.Active)

	pm.state.AddActivePRep(owner)
	if err = pm.addNodeToOwner(node, owner); err != nil {
		return err
	}

	// Do not share pb and ps with pm.state
	prep := newPRep(owner, pb, ps)
	pm.appendPRep(prep)
	pm.Sort()

	return nil
}

func (pm *PRepManager) SetPRep(regInfo *RegInfo) error {
	owner := regInfo.owner

	pb := pm.state.GetPRepBase(owner, false)
	if pb == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}
	return setPRep(pb, regInfo)
}

func (pm *PRepManager) UnregisterPRep(owner module.Address) error {
	return pm.disablePRep(owner, icstate.Unregistered)
}

func (pm *PRepManager) DisqualifyPRep(owner module.Address) error {
	return pm.disablePRep(owner, icstate.Disqualified)
}

// Case: Penalty, UnregisterPRep and DisqualifyPRep
func (pm *PRepManager) disablePRep(owner module.Address, status icstate.Status) error {
	prep := pm.prepMap[icutils.ToKey(owner)]
	if prep == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}
	if err := pm.removePRep(owner); err != nil {
		return err
	}

	pm.totalDelegated.Sub(pm.totalDelegated, prep.Delegated())
	pm.totalBonded.Sub(pm.totalBonded, prep.Bonded())
	pm.adjustPRepSize(prep.Grade(), false)
	prep.SetGrade(icstate.Candidate)
	prep.SetStatus(status)
	return nil
}

func (pm *PRepManager) removePRep(owner module.Address) error {
	var err error
	if err = pm.state.RemoveActivePRep(owner); err != nil {
		return err
	}
	if err = pm.removeFromPRepMap(owner); err != nil {
		return err
	}
	return pm.removeFromOrderedPReps(owner)
}

func (pm *PRepManager) removeFromPRepMap(owner module.Address) error {
	key := icutils.ToKey(owner)
	if _, ok := pm.prepMap[key]; !ok {
		return errors.Errorf("PRep not found in prepMap: %s", owner)
	}
	delete(pm.prepMap, key)
	return nil
}

func (pm *PRepManager) removeFromOrderedPReps(owner module.Address) error {
	var i int
	size := len(pm.orderedPReps)

	for i = 0; i < size; i++ {
		if owner.Equal(pm.orderedPReps[i].Owner()) {
			break
		}
	}

	if i < 0 {
		return errors.Errorf("PRep not found in orderedPRep: %s", owner)
	}

	for ; i < size-1; i++ {
		pm.orderedPReps[i] = pm.orderedPReps[i+1]
	}
	pm.orderedPReps = pm.orderedPReps[:size-1]
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
		return errors.Errorf("Node address in use: %s", node)
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
			if ps.IsActive() {
				ps.Delegated().Add(ps.Delegated(), value)
			} else {
				delegatedToInactiveNode.Add(delegatedToInactiveNode, value)
			}
		}
	}

	totalDelegated := pm.totalDelegated
	totalDelegated.Add(totalDelegated, nd.GetDelegationAmount())
	totalDelegated.Sub(totalDelegated, od.GetDelegationAmount())
	// Ignore the delegated amount to Inactive P-Rep
	totalDelegated.Sub(totalDelegated, delegatedToInactiveNode)

	pm.Sort()
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
			ps := pm.state.GetPRepStatus(owner, false)
			if ps == nil {
				return nil, errors.Errorf("Failed to set bonded value to PRepStatus")
			}

			if ps.IsActive() {
				ps.Bonded().Add(ps.Bonded(), value)
			} else {
				bondedToInactiveNode.Add(bondedToInactiveNode, value)
			}
		}
	}

	totalBonded := pm.totalBonded
	totalBonded.Add(totalBonded, nBonds.GetBondAmount())
	totalBonded.Sub(totalBonded, oBonds.GetBondAmount())
	// Ignore the bonded amount to inactive P-Rep
	totalBonded.Sub(totalBonded, bondedToInactiveNode)

	pm.Sort()
	return delta, nil
}

func (pm *PRepManager) OnTermEnd(mainPRepCount, subPRepCount int) error {
	pm.mainPReps = 0
	pm.subPReps = 0
	electedPRepCount := mainPRepCount + subPRepCount

	for i, prep := range pm.orderedPReps {
		if i < mainPRepCount {
			prep.SetGrade(icstate.Main)
		} else if i < electedPRepCount {
			prep.SetGrade(icstate.Sub)
		} else {
			prep.SetGrade(icstate.Candidate)
		}
		pm.adjustPRepSize(prep.Grade(), true)
	}

	return nil
}

func (pm *PRepManager) ShiftVPenaltyMaskByNode(node module.Address) error {
	prep := pm.GetPRepByNode(node)
	if prep == nil {
		return errors.Errorf("PRep not found: node=%s", node)
	}

	prep.ShiftVPenaltyMask(ConsistentValidationPenaltyMask)
	return nil
}

// UpdateBlockVoteStats updates PRepLastState based on ConsensusInfo
func (pm *PRepManager) UpdateBlockVoteStats(owner module.Address, voted bool, blockHeight int64) error {
	prep := pm.GetPRepByOwner(owner)
	if prep == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}

	vs := icstate.Success
	if !voted {
		vs = icstate.Failure
	}

	ps := prep.PRepStatus
	ls := ps.LastState()
	if ls == icstate.None {
		if vs == icstate.Failure {
			ps.SetVFail(ps.VFail() + 1)
		}
		ps.SetVTotal(ps.VTotal() + 1)
		ps.SetLastHeight(blockHeight)
	} else {
		if vs != ls {
			diff := blockHeight - ps.LastHeight()
			ps.SetVTotal(ps.VTotal() + diff)
			if vs == icstate.Success {
				ps.SetVFail(ps.VFail() + diff - 1)
			} else {
				ps.SetVFail(ps.VFail() + 1)
			}
			ps.SetLastState(vs)
			ps.SetLastHeight(blockHeight)
		}
	}

	return nil
}

func (pm *PRepManager) SyncBlockVoteStats(owner module.Address, blockHeight int64) error {
	prep := pm.GetPRepByOwner(owner)
	if prep == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}

	lh := prep.LastHeight()
	if blockHeight < lh {
		return errors.Errorf("blockHeight(%d) < lastHeight(%d)", blockHeight, lh)
	}
	if prep.LastState() == icstate.None {
		return nil
	}

	if blockHeight == prep.LastHeight() {
		// Already done by other reasons
		return nil
	}

	prep.SetVFail(prep.GetVFail(blockHeight))
	prep.SetVTotal(prep.GetVTotal(blockHeight))
	prep.SetLastHeight(blockHeight)
	prep.SetLastState(icstate.None)
	return nil
}

// Grade change, LastState to icstate.None
func (pm *PRepManager) ImposePenalty(owner module.Address, blockHeight int64) error {
	prep := pm.GetPRepByOwner(owner)
	if prep == nil {
		return errors.Errorf("PRep not found: %v", owner)
	}

	if err := pm.ChangeGrade(owner, icstate.Candidate); err != nil {
		return err
	}
	if err := pm.SyncBlockVoteStats(owner, blockHeight); err != nil {
		return err
	}
	prep.IncrementVPenalty()
	return nil
}

// Slash handles to reduce PRepStatus.bonded and PRepManager.totalBonded
// Do not change PRep grade here
// Caution: amount should not include the amount from unbonded
func (pm *PRepManager) Slash(owner module.Address, amount *big.Int, sort bool) error {
	if owner == nil {
		return errors.Errorf("Owner is nil")
	}
	if amount == nil {
		return errors.Errorf("Amount is nil")
	}
	if amount.Sign() < 0 {
		return errors.Errorf("Amount is less than zero: %v", amount)
	}
	if amount.Sign() == 0 {
		return nil
	}

	prep := pm.GetPRepByOwner(owner)
	if prep == nil {
		return errors.Errorf("PRep not found: %v", owner)
	}

	bonded := new(big.Int).Set(prep.Bonded())
	if bonded.Cmp(amount) < 0 {
		return errors.Errorf("bonded=%v < slash=%v", bonded, amount)
	}
	prep.SetBonded(bonded.Sub(bonded, amount))
	pm.totalBonded.Sub(pm.totalBonded, amount)

	if sort {
		pm.Sort()
	}
	return nil
}

func (pm *PRepManager) GetPRepStatsInJSON(blockHeight int64) (map[string]interface{}, error) {
	size := pm.GetPRepSize(icstate.Main)
	jso := make(map[string]interface{})
	preps := make([]interface{}, size, size)

	for i, prep := range pm.orderedPReps {
		preps[i] = prep.GetStatsInJSON(blockHeight)
	}

	jso["blockHeight"] = blockHeight
	jso["preps"] = preps
	return jso, nil
}

func newPRepManager(state *icstate.State) *PRepManager {
	pm := &PRepManager{
		state:          state,
		totalDelegated: big.NewInt(0),
		totalBonded:    big.NewInt(0),
		prepMap:        make(map[string]*PRep),
	}

	pm.init()
	pm.Sort()
	return pm
}
