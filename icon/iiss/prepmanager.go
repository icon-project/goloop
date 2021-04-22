package iiss

import (
	"bytes"
	"fmt"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"math"
	"math/big"
	"sort"
)

const (
	InitialIRep = 50_000 // in icx, not loop
	MinIRep     = 10_000
)

var BigIntInitialIRep = new(big.Int).Mul(new(big.Int).SetInt64(InitialIRep), icutils.BigIntICX)
var BigIntMinIRep = new(big.Int).Mul(new(big.Int).SetInt64(MinIRep), icutils.BigIntICX)

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

func (r *RegInfo) String() string {
	return fmt.Sprintf(
		"city=%s country=%s details=%s email=%s name=%s p2p=%s website=%s owner=%s",
		r.city, r.country, r.details, r.email, r.name, r.p2pEndpoint, r.website, r.owner,
	)
}

func (r *RegInfo) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(
				f,
				"RegInfo{city=%s country=%s details=%s email=%s p2p=%s website=%s owner=%s}",
				r.city, r.country, r.details, r.email, r.p2pEndpoint, r.website, r.owner)
		} else {
			fmt.Fprintf(f, "RegInfo{%s %s %s %s %s %s %s}",
				r.city, r.country, r.details, r.email, r.p2pEndpoint, r.website, r.owner)
		}
	case 's':
		fmt.Fprint(f, r.String())
	}
}

func (r *RegInfo) UpdateRegInfo(prepInfo *icstate.PRepBase) {
	if len(r.city) == 0 {
		r.city = prepInfo.City()
	}

	if len(r.country) == 0 {
		r.country = prepInfo.Country()
	}

	if len(r.details) == 0 {
		r.details = prepInfo.Details()
	}

	if len(r.email) == 0 {
		r.email = prepInfo.Email()
	}

	if len(r.name) == 0 {
		r.name = prepInfo.Name()
	}

	if len(r.p2pEndpoint) == 0 {
		r.p2pEndpoint = prepInfo.P2pEndpoint()
	}

	if len(r.website) == 0 {
		r.website = prepInfo.Website()
	}

	if r.node == nil {
		r.node = prepInfo.Node()
	}
}

func (r *RegInfo) Validate(revision int) error {
	if err := icutils.ValidateEndpoint(r.p2pEndpoint); err != nil {
		return err
	}

	if err := icutils.ValidateURL(r.website); err != nil {
		return err
	}

	if err := icutils.ValidateURL(r.details); err != nil {
		return err
	}

	if err := icutils.ValidateEmail(r.email, revision); err != nil {
		return err
	}

	return nil
}

func NewRegInfo(city, country, details, email, name, p2pEndpoint, website string, node, owner module.Address) *RegInfo {
	if node == nil {
		node = owner
	}

	regInfo := &RegInfo{
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

	return regInfo
}

type PRep struct {
	owner module.Address

	*icstate.PRepBase
	*icstate.PRepStatus
}

func (p *PRep) Owner() module.Address {
	return p.owner
}

func (p *PRep) GetNode() module.Address {
	if p.Node() != nil {
		return p.Node()
	}
	return p.owner
}

func (p *PRep) ToJSON(blockHeight int64, bondRequirement int64) map[string]interface{} {
	jso := icutils.MergeMaps(p.PRepBase.ToJSON(), p.PRepStatus.ToJSON(blockHeight, bondRequirement))
	jso["address"] = p.owner
	return jso
}

func (p *PRep) Clone() *PRep {
	return newPRep(p.owner, p.PRepBase.Clone(), p.PRepStatus.Clone())
}

func newPRep(owner module.Address, pb *icstate.PRepBase, ps *icstate.PRepStatus) *PRep {
	ps.SetOwner(owner)
	return &PRep{owner: owner, PRepBase: pb, PRepStatus: ps}
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
	logger log.Logger
	state  *icstate.State

	totalBonded    *big.Int
	totalDelegated *big.Int // total delegated amount of all active P-Reps

	sorted       bool
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
			pm.logger.Warnf("Failed to load PRep: %s", owner)
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
	pm.prepMap[icutils.ToKey(p.owner)] = p
	if p.PRepStatus.Status() == icstate.Active {
		pm.orderedPReps = append(pm.orderedPReps, p)
		pm.totalBonded.Add(pm.totalBonded, p.Bonded())
		pm.totalDelegated.Add(pm.totalDelegated, p.Delegated())
		pm.adjustPRepSize(p.Grade(), true)
	}
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
	if pm.sorted {
		return
	}
	pm.sort()
	pm.sorted = true
}

func (pm *PRepManager) sort() {
	br := pm.getBondRequirement()
	sort.Slice(pm.orderedPReps, func(i, j int) bool {
		ret := pm.orderedPReps[i].GetBondedDelegation(br).Cmp(pm.orderedPReps[j].GetBondedDelegation(br))
		if ret > 0 {
			return true
		} else if ret < 0 {
			return false
		}

		ret = pm.orderedPReps[i].Delegated().Cmp(pm.orderedPReps[j].Delegated())
		if ret > 0 {
			return true
		} else if ret < 0 {
			return false
		}

		return bytes.Compare(pm.orderedPReps[i].owner.Bytes(), pm.orderedPReps[j].owner.Bytes()) > 0
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

func (pm *PRepManager) RegisterPRep(regInfo *RegInfo, irep *big.Int) error {
	if regInfo == nil {
		return errors.Errorf("Invalid argument: regInfo")
	}

	node := regInfo.node
	owner := regInfo.owner

	if pm.contains(owner) {
		return errors.Errorf("PRep already exists: %s", owner)
	}
	ps := pm.state.GetPRepStatus(owner, false)
	if ps != nil && ps.Status() != icstate.NotReady {
		return errors.Errorf("Already in use: addr=%s status=%s", owner, ps.Status())
	}

	pb := pm.state.GetPRepBase(owner, true)
	err := setPRep(pb, regInfo)
	if err != nil {
		return err
	}
	pb.SetIrep(irep, 0)

	if ps == nil {
		ps = pm.state.GetPRepStatus(owner, true)
	}
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
		if owner.Equal(pm.orderedPReps[i].owner) {
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
			ps.Delegated().Add(ps.Delegated(), value)
			if !ps.IsActive() {
				delegatedToInactiveNode.Add(delegatedToInactiveNode, value)
			}
		}
	}

	totalDelegated := pm.totalDelegated
	totalDelegated.Add(totalDelegated, nd.GetDelegationAmount())
	totalDelegated.Sub(totalDelegated, od.GetDelegationAmount())
	// Ignore the delegated amount to Inactive P-Rep
	totalDelegated.Sub(totalDelegated, delegatedToInactiveNode)

	pm.sort()
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
				// this code is not reachable, because there is no case of bonding to not-registered PRep
				bondedToInactiveNode.Add(bondedToInactiveNode, value)
			}
		}
	}

	totalBonded := pm.totalBonded
	totalBonded.Add(totalBonded, nBonds.GetBondAmount())
	totalBonded.Sub(totalBonded, oBonds.GetBondAmount())
	// Ignore the bonded amount to inactive P-Rep
	totalBonded.Sub(totalBonded, bondedToInactiveNode)

	pm.sort()
	return delta, nil
}

func (pm *PRepManager) OnTermEnd(mainPRepCount, subPRepCount int, blockHeight int64) error {
	pm.Sort()
	pm.mainPReps = 0
	pm.subPReps = 0
	electedPRepCount := mainPRepCount + subPRepCount

	for i, prep := range pm.orderedPReps {
		ls := prep.LastState()

		if i < mainPRepCount {
			prep.SetGrade(icstate.Main)
		} else if i < electedPRepCount {
			prep.SetGrade(icstate.Sub)
		} else {
			prep.SetGrade(icstate.Candidate)
		}
		pm.adjustPRepSize(prep.Grade(), true)

		if prep.Grade() == icstate.Main {
			if ls == icstate.None {
				prep.SetLastState(icstate.Ready)
				prep.SetLastHeight(blockHeight)
			}
		} else {
			if ls != icstate.None {
				if err := prep.SyncBlockVoteStats(blockHeight); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (pm *PRepManager) ShiftVPenaltyMaskByNode(node module.Address) error {
	prep := pm.GetPRepByNode(node)
	if prep == nil {
		return errors.Errorf("PRep not found: node=%s", node)
	}

	prep.ShiftVPenaltyMask(buildPenaltyMask(pm.state.GetConsistentValidationPenaltyMask()))
	return nil
}

// UpdateBlockVoteStats updates PRepLastState based on ConsensusInfo
func (pm *PRepManager) UpdateBlockVoteStats(owner module.Address, voted bool, blockHeight int64) error {
	prep := pm.GetPRepByOwner(owner)
	if prep == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}
	err := prep.UpdateBlockVoteStats(blockHeight, voted)
	//pm.logger.Debugf("UpdateBlockVoteStats: bh=%d %s", blockHeight, prep.PRepStatus)
	return err
}

// Grade change, LastState to icstate.None
func (pm *PRepManager) ImposePenalty(owner module.Address, blockHeight int64) error {
	var err error
	prep := pm.GetPRepByOwner(owner)
	if prep == nil {
		return errors.Errorf("PRep not found: %v", owner)
	}

	pm.logger.Debugf("ImposePenalty() start: bh=%d %s", blockHeight, prep.PRepStatus)

	if err = pm.ChangeGrade(owner, icstate.Candidate); err != nil {
		return err
	}
	err = prep.OnPenaltyImposed(blockHeight)

	pm.logger.Debugf("ImposePenalty() end: bh=%d %s", blockHeight, prep.PRepStatus)
	return err
}

// Slash handles to reduce PRepStatus.bonded and PRepManager.totalBonded
// Do not change PRep grade here
// Caution: amount should not include the amount from unbonded
func (pm *PRepManager) Slash(owner module.Address, amount *big.Int) error {
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

	pm.logger.Debugf(
		"Slash: addr=%s amount=%s tb=%s",
		owner, amount, pm.totalBonded,
	)
	pm.sorted = false
	return nil
}

func (pm *PRepManager) GetPRepStatsInJSON(blockHeight int64) (map[string]interface{}, error) {
	size := pm.Size()
	jso := make(map[string]interface{})
	preps := make([]interface{}, size, size)

	for i, prep := range pm.orderedPReps {
		preps[i] = prep.GetStatsInJSON(blockHeight)
	}

	jso["blockHeight"] = blockHeight
	jso["preps"] = preps
	return jso, nil
}

func (pm *PRepManager) CalculateIRep(revision int) *big.Int {
	irep := new(big.Int)
	if revision < icmodule.RevisionDecentralize ||
		revision >= icmodule.RevisionICON2 {
		return irep
	}
	if revision >= icmodule.Revision9 {
		// set IRep via network proposal
		return nil
	}
	size := pm.GetPRepSize(icstate.Main)
	totalDelegated := new(big.Int)
	totalWeightedIrep := new(big.Int)
	for i := 0; i < size; i++ {
		prep := pm.orderedPReps[i]

		totalWeightedIrep.Add(totalWeightedIrep, new(big.Int).Mul(prep.IRep(), prep.Delegated()))
		totalDelegated.Add(totalDelegated, prep.Delegated())
	}

	if totalDelegated.Sign() == 0 {
		return irep
	}

	irep.Div(totalWeightedIrep, totalDelegated)
	if irep.Cmp(BigIntMinIRep) == -1 {
		irep.Set(BigIntMinIRep)
	}
	return irep
}

func (pm *PRepManager) CalculateRRep(totalSupply *big.Int, revision int) *big.Int {
	if revision < icmodule.RevisionIISS || revision >= icmodule.RevisionICON2 {
		// rrep is disabled
		return new(big.Int)
	}
	return calculateRRep(totalSupply, pm.totalDelegated)
}

const (
	rrepMin        = 200   // 2%
	rrepMax        = 1_200 // 12%
	rrepPoint      = 7_000 // 70%
	rrepMultiplier = 10_000
)

func calculateRRep(totalSupply, totalDelegated *big.Int) *big.Int {
	ts := new(big.Float).SetInt(totalSupply)
	td := new(big.Float).SetInt(totalDelegated)
	delegatePercentage := new(big.Float).Quo(td, ts)
	delegatePercentage.Mul(delegatePercentage, new(big.Float).SetInt64(rrepMultiplier))
	dp, _ := delegatePercentage.Float64()
	if dp >= rrepPoint {
		return new(big.Int).SetInt64(rrepMin)
	}

	firstOperand := (rrepMax - rrepMin) / math.Pow(rrepPoint, 2)
	secondOperand := math.Pow(dp-rrepPoint, 2)
	return new(big.Int).SetInt64(int64(firstOperand*secondOperand + rrepMin))
}

func newPRepManager(state *icstate.State, logger log.Logger) *PRepManager {
	if logger == nil {
		logger = log.WithFields(log.Fields{
			log.FieldKeyModule: "ICON",
		})
	}

	pm := &PRepManager{
		logger:         logger,
		state:          state,
		totalDelegated: big.NewInt(0),
		totalBonded:    big.NewInt(0),
		prepMap:        make(map[string]*PRep),
	}

	pm.init()
	pm.sort()
	return pm
}
