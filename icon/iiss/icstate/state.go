/*
 * Copyright 2020 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package icstate

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/ompt"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/iccache"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
)

var (
	IssueKey          = containerdb.ToKey(containerdb.HashBuilder, "issue_icx").Build()
	RewardCalcInfoKey = containerdb.ToKey(containerdb.HashBuilder, "reward_calc_info").Build()
	ValidatorsKey     = containerdb.ToKey(
		containerdb.HashBuilder, scoredb.VarDBPrefix, "validators",
	)
	UnstakeSlotMaxKey = containerdb.ToKey(
		containerdb.HashBuilder, scoredb.VarDBPrefix, "unstake_slot_max",
	)
	TotalDelegationKey = containerdb.ToKey(
		containerdb.HashBuilder, scoredb.VarDBPrefix, "total_delegation",
	)
	TotalBondKey = containerdb.ToKey(
		containerdb.HashBuilder, scoredb.VarDBPrefix, "total_bond",
	)
	LastBlockVotersKey = containerdb.ToKey(
		containerdb.HashBuilder, scoredb.VarDBPrefix, "lastBlockVoters",
	)
	termKey = containerdb.ToKey(containerdb.HashBuilder, scoredb.VarDBPrefix, "term")

	pRepIllegalDelegatedKey = containerdb.ToKey(containerdb.HashBuilder, scoredb.DictDBPrefix, "prep_illegal_delegated")
)

type State struct {
	readonly               bool
	accountCache           *AccountCache
	allPRepCache           *AllPRepCache
	nodeOwnerCache         *NodeOwnerCache
	prepBaseCache          *PRepBaseCache
	prepStatusCache        *PRepStatusCache
	unstakingTimerCache    *TimerCache
	unbondingTimerCache    *TimerCache
	networkScoreTimerCache *TimerCache
	logger                 log.Logger

	store                *icobject.ObjectStoreState
	totalDelegationVarDB *containerdb.VarDB
	totalBondVarDB       *containerdb.VarDB
	validatorsVarDB      *containerdb.VarDB
	lastBlockVotersVarDB *containerdb.VarDB
	termVarDB            *containerdb.VarDB

	pRepIllegalDelegatedDB *containerdb.DictDB
}

func (s *State) Reset(ss *Snapshot) error {
	s.store.Reset(ss.store.ImmutableForObject)
	s.accountCache.Reset()
	s.nodeOwnerCache.Reset()
	s.prepBaseCache.Reset()
	s.prepStatusCache.Reset()
	s.unstakingTimerCache.Reset()
	s.unbondingTimerCache.Reset()
	s.networkScoreTimerCache.Reset()
	return nil
}

func (s *State) Flush() error {
	s.accountCache.Flush()
	s.nodeOwnerCache.Flush()
	s.prepBaseCache.Flush()
	s.prepStatusCache.Flush()
	s.unstakingTimerCache.Flush()
	s.unbondingTimerCache.Flush()
	s.networkScoreTimerCache.Flush()
	return nil
}

func (s *State) GetSnapshot() *Snapshot {
	if err := s.Flush(); err != nil {
		panic(err)
	}
	return newSnapshotFromImmutableForObject(s.store.GetSnapshot())
}

func (s *State) GetAccountState(addr module.Address) *AccountState {
	a := s.accountCache.Get(addr, true)
	return a
}

func (s *State) GetAccountSnapshot(addr module.Address) *AccountSnapshot {
	return s.accountCache.GetSnapshot(addr)
}

func (s *State) GetUnstakingTimerState(height int64) *TimerState {
	timer := s.unstakingTimerCache.Get(height)
	return timer
}

func (s *State) GetUnstakingTimerSnapshot(height int64) *TimerSnapshot {
	return s.unstakingTimerCache.GetSnapshot(height)
}

func (s *State) GetUnbondingTimerState(height int64) *TimerState {
	timer := s.unbondingTimerCache.Get(height)
	return timer
}

func (s *State) GetUnbondingTimerSnapshot(height int64) *TimerSnapshot {
	timer := s.unbondingTimerCache.GetSnapshot(height)
	return timer
}

func (s *State) GetNetworkScoreTimerState(height int64) *TimerState {
	return s.networkScoreTimerCache.Get(height)
}

func (s *State) GetNetworkScoreTimerSnapshot(height int64) *TimerSnapshot {
	return s.networkScoreTimerCache.GetSnapshot(height)
}

func (s *State) GetPRepBaseByOwner(owner module.Address, createIfNotExist bool) *PRepBaseState {
	return s.prepBaseCache.Get(owner, createIfNotExist)
}

func (s *State) GetPRepStatusByOwner(owner module.Address, createIfNotExist bool) *PRepStatusState {
	return s.prepStatusCache.Get(owner, createIfNotExist)
}

func (s *State) GetPRepByOwner(owner module.Address) *PRep {
	return NewPRep(owner, s)
}

func NewStateFromSnapshot(ss *Snapshot, readonly bool, logger log.Logger) *State {
	t := trie_manager.NewMutableFromImmutableForObject(ss.store.ImmutableForObject)
	if c := iccache.StateNodeCacheOf(t.Database()); c != nil && !readonly {
		ompt.SetCacheOfMutableForObject(t, c)
	}
	return NewStateFromTrie(t, readonly, logger)
}

func NewStateFromTrie(t trie.MutableForObject, readonly bool, logger log.Logger) *State {
	store := icobject.NewObjectStoreState(t)
	tdVarDB := containerdb.NewVarDB(store, TotalDelegationKey)
	tbVarDB := containerdb.NewVarDB(store, TotalBondKey)
	validatorsVarDB := containerdb.NewVarDB(store, ValidatorsKey)
	lastBlockVotersVarDB := containerdb.NewVarDB(store, LastBlockVotersKey)
	termVarDB := containerdb.NewVarDB(store, termKey)
	pRepIllegalDelegatedDB := containerdb.NewDictDB(store, 1, pRepIllegalDelegatedKey)

	return &State{
		readonly:               readonly,
		accountCache:           newAccountCache(store),
		allPRepCache:           NewAllPRepCache(store),
		nodeOwnerCache:         newNodeOwnerCache(store),
		prepBaseCache:          newPRepBaseCache(store),
		prepStatusCache:        newPRepStatusCache(store),
		unstakingTimerCache:    newTimerCache(store, unstakingTimerDictPrefix),
		unbondingTimerCache:    newTimerCache(store, unbondingTimerDictPrefix),
		networkScoreTimerCache: newTimerCache(store, networkScoreTimerDictPrefix),
		logger:                 logger,

		store:                store,
		totalDelegationVarDB: tdVarDB,
		totalBondVarDB:       tbVarDB,
		validatorsVarDB:      validatorsVarDB,
		lastBlockVotersVarDB: lastBlockVotersVarDB,
		termVarDB:            termVarDB,

		pRepIllegalDelegatedDB: pRepIllegalDelegatedDB,
	}
}

// addNodeToOwner adds alias from node to owner
// If the alias already exists, then it silently ignores
// If node is already used by others, then it returns errors.
func (s *State) addNodeToOwner(node, owner module.Address) error {
	if !node.Equal(owner) {
		ps := s.GetPRepStatusByOwner(node, false)
		if ps != nil && ps.Status() != NotReady {
			return errors.InvalidStateError.Errorf("AlreadyUsedByPRep(node=%s)", node)
		}
	}
	// nodeOwner map stores the entry only if node is different from owner
	return s.nodeOwnerCache.Add(node, owner)
}

func (s *State) SetIssue(issue *Issue) error {
	_, err := s.store.Set(IssueKey, icobject.New(TypeIssue, issue))
	if err != nil {
		return err
	}
	return nil
}

func (s *State) GetIssue() (*Issue, error) {
	obj, err := s.store.Get(IssueKey)
	if err != nil {
		return nil, err
	}
	issue := ToIssue(obj)
	if issue == nil {
		issue = NewIssue()
	}
	return issue, nil
}

func (s *State) GetTermSnapshot() *TermSnapshot {
	return ToTerm(s.termVarDB.Object())
}

func (s *State) SetTermSnapshot(term *TermSnapshot) error {
	return s.termVarDB.Set(icobject.New(TypeTerm, term))
}

func (s *State) SetRewardCalcInfo(rc *RewardCalcInfo) error {
	_, err := s.store.Set(RewardCalcInfoKey, icobject.New(TypeRewardCalcInfo, rc))
	if err != nil {
		return err
	}
	return nil
}

func (s *State) GetRewardCalcInfo() (*RewardCalcInfo, error) {
	obj, err := s.store.Get(RewardCalcInfoKey)
	if err != nil {
		return nil, err
	}
	rc := ToRewardCalcInfo(obj)
	if rc == nil {
		rc = NewRewardCalcInfo()
	}
	return rc, nil
}

func (s *State) SetUnstakeSlotMax(v int64) error {
	db := containerdb.NewVarDB(s.store, UnstakeSlotMaxKey)
	err := db.Set(v)
	return err
}

func (s *State) GetUnstakeSlotMax() int64 {
	db := containerdb.NewVarDB(s.store, UnstakeSlotMaxKey)
	return db.Int64()
}

func (s *State) ClearCache() {
	s.accountCache.Clear()
	s.unstakingTimerCache.Clear()
	s.unbondingTimerCache.Clear()
	s.nodeOwnerCache.Clear()
	s.prepBaseCache.Clear()
	s.prepStatusCache.Clear()
	// TODO clear other caches
	s.store.ClearCache()
}

func (s *State) updatePRepInfoOf(owner module.Address, pb *PRepBaseState, info *PRepInfo) error {
	pb.UpdateInfo(info)
	node := info.GetNode(owner)
	if err := s.addNodeToOwner(node, owner); err != nil {
		return err
	}
	return nil
}

func (s *State) RegisterPRep(owner module.Address, ri *PRepInfo, irep *big.Int, irepHeight int64) error {
	if ri == nil {
		return errors.Errorf("Invalid argument: ri")
	}

	ps := s.GetPRepStatusByOwner(owner, true)
	if err := ps.Activate(); err != nil {
		return errors.Wrapf(err, "ActivationFail(addr=%s)", owner)
	}
	if err := s.allPRepCache.Add(owner); err != nil {
		return err
	}
	pb := s.GetPRepBaseByOwner(owner, true)
	if !pb.IsEmpty() {
		return errors.Errorf("Already in use: addr=%s %+v", owner, pb)
	}
	if err := s.updatePRepInfoOf(owner, pb, ri); err != nil {
		return err
	}
	pb.SetIrep(irep, irepHeight)

	if ps.Delegated().Sign() == 1 {
		if err := s.SetTotalDelegation(new(big.Int).Add(s.GetTotalDelegation(), ps.Delegated())); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) SetPRep(blockHeight int64, owner module.Address, info *PRepInfo) (bool, error) {
	pb := s.GetPRepBaseByOwner(owner, false)
	if pb == nil {
		return false, errors.Errorf("PRep not found: %s", owner)
	}

	oldNode := pb.GetNode(owner)
	oldP2P := pb.P2PEndpoint()
	if info.Node != nil && info.Node.Equal(oldNode) {
		return false, errors.Errorf("SameAsOld(%s)", info.Node)
	}
	if err := s.updatePRepInfoOf(owner, pb, info); err != nil {
		return false, err
	}
	newNode := pb.GetNode(owner)
	nodeUpdate := pb.P2PEndpoint() != oldP2P || !newNode.Equal(oldNode)

	if !oldNode.Equal(newNode) {
		ps := s.GetPRepStatusByOwner(owner, false)
		if ps.Grade() == GradeMain {
			return nodeUpdate, s.changeValidatorNodeAddress(blockHeight, owner, oldNode, newNode)
		}
	}
	return nodeUpdate, nil
}

func (s *State) SetTotalDelegation(value *big.Int) error {
	return s.totalDelegationVarDB.Set(value)
}

func (s *State) GetTotalDelegation() *big.Int {
	ret := s.totalDelegationVarDB.BigInt()
	if ret == nil {
		ret = new(big.Int)
	}
	return ret
}

func (s *State) SetTotalBond(value *big.Int) error {
	return s.totalBondVarDB.Set(value)
}

func (s *State) GetTotalBond() *big.Int {
	ret := s.totalBondVarDB.BigInt()
	if ret == nil {
		ret = new(big.Int)
	}
	return ret
}

func (s *State) GetOwnerByNode(node module.Address) module.Address {
	return s.nodeOwnerCache.Get(node)
}

func (s *State) GetNodeByOwner(owner module.Address) module.Address {
	if owner == nil {
		return nil
	}
	pb := s.GetPRepBaseByOwner(owner, false)
	if pb == nil {
		return nil
	}
	return pb.GetNode(owner)
}

func (s *State) OnBlockVote(sc icmodule.StateContext, owner module.Address, voted bool) error {
	blockHeight := sc.BlockHeight()
	if !voted {
		s.logger.Debugf("Nil vote: bh=%d owner=%s", blockHeight, owner)
	}
	ps := s.GetPRepStatusByOwner(owner, false)
	if ps == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}
	err := ps.NotifyEvent(sc, icmodule.PRepEventBlockVote, voted)
	s.logger.Tracef("OnBlockVote() bh=%d voted=%t owner=%v %+v", blockHeight, voted, owner, ps)
	return err
}

func (s *State) OnMainPRepReplaced(sc icmodule.StateContext, oldOwner, newOwner module.Address) error {
	blockHeight := sc.BlockHeight()
	s.logger.Tracef("OnMainPRepReplaced() start: bh=%d old=%v new=%v", blockHeight, oldOwner, newOwner)
	if newOwner == nil {
		// No sub prep remains
		return nil
	}

	ps := s.GetPRepStatusByOwner(newOwner, false)
	if ps == nil {
		return errors.Errorf("PRep not found: %s", newOwner)
	}
	err := ps.NotifyEvent(sc, icmodule.PRepEventMainIn, s.GetConsistentValidationPenaltyMask())
	s.logger.Tracef("OnMainPRepReplaced()   end: bh=%d old=%v new=%v %+v", blockHeight, oldOwner, newOwner, ps)
	return err
}

func (s *State) OnValidatorOut(sc icmodule.StateContext, owner module.Address) error {
	ps := s.GetPRepStatusByOwner(owner, false)
	if ps == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}
	err := ps.NotifyEvent(sc, icmodule.PRepEventValidatorOut)
	s.logger.Tracef("OnValidatorOut(): bh=%d owner=%v %+v", sc.BlockHeight(), owner, ps)

	return err
}

// GetPRepStatsList returns PRepStatus list ordered by bonded delegation
func (s *State) GetPRepStatsList() ([]*PRepStats, error) {
	br := s.GetBondRequirement()

	size := s.allPRepCache.Size()
	statsList := make([]*PRepStats, 0)

	for i := 0; i < size; i++ {
		owner := s.allPRepCache.Get(i)
		ps := s.GetPRepStatusByOwner(owner, false)
		if ps.Status() == Active {
			stats := NewPRepStats(owner, ps)
			statsList = append(statsList, stats)
		}
	}

	sortPRepStatsList(statsList, br)
	return statsList, nil
}

func sortPRepStatsList(statsList []*PRepStats, br icmodule.Rate) {
	sort.Slice(statsList, func(i, j int) bool {
		ret := statsList[i].GetBondedDelegation(br).Cmp(statsList[j].GetBondedDelegation(br))
		if ret > 0 {
			return true
		} else if ret < 0 {
			return false
		}

		ret = statsList[i].Delegated().Cmp(statsList[j].Delegated())
		if ret > 0 {
			return true
		} else if ret < 0 {
			return false
		}

		return bytes.Compare(statsList[i].Owner().Bytes(), statsList[j].Owner().Bytes()) > 0
	})
}

// ImposePenalty changes grade and set LastState to icstate.None
func (s *State) ImposePenalty(
	sc icmodule.StateContext, pt icmodule.PenaltyType, owner module.Address, ps *PRepStatusState) error {
	var err error
	blockHeight := sc.BlockHeight()

	// Update status of the penalized main prep
	s.logger.Debugf("OnPenaltyImposed() start: owner=%v bh=%d %+v", owner, blockHeight, ps)

	// Update the state of PRepStatus
	oldGrade := ps.Grade()
	err = ps.NotifyEvent(sc, icmodule.PRepEventImposePenalty, pt)
	s.logger.Debugf("OnPenaltyImposed() end: owner=%v bh=%d %+v", owner, blockHeight, ps)
	if err != nil {
		return err
	}

	// If a penalized prep is a main prep, choose a new validator from prep snapshots
	if oldGrade == GradeMain {
		err = s.replaceMainPRepByOwner(sc, owner)
	}
	return err
}

// ReducePRepBonded handles to reduce PRepStatus.bonded
// Do not change PRep grade here
// Caution: amount should not include the amount from unbonded
func (s *State) ReducePRepBonded(owner module.Address, amount *big.Int) error {
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

	ps := s.GetPRepStatusByOwner(owner, false)
	if ps == nil {
		return errors.Errorf("PRep not found: %v", owner)
	}

	bonded := ps.Bonded()
	if bonded.Cmp(amount) < 0 {
		return errors.Errorf("bonded=%v < slash=%v", bonded, amount)
	}
	ps.SetBonded(new(big.Int).Sub(bonded, amount))
	return s.SetTotalBond(new(big.Int).Sub(s.GetTotalBond(), amount))
}

func (s *State) DisablePRep(sc icmodule.StateContext, owner module.Address, status Status) error {
	ps := s.GetPRepStatusByOwner(owner, false)
	if ps == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}

	if status == Unregistered && ps.Bonded().Sign() > 0 {
		return errors.Errorf("A P-Rep that has a bond can't unregister")
	}

	oldStatus := ps.Status()
	oldGrade, err := ps.DisableAs(status)
	if err != nil {
		return err
	}
	if oldGrade == GradeMain {
		if err = s.replaceMainPRepByOwner(sc, owner); err != nil {
			return err
		}
	}
	if oldStatus == Active && oldStatus != status {
		if err = s.SetTotalDelegation(new(big.Int).Sub(s.GetTotalDelegation(), ps.Delegated())); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) GetValidatorsSnapshot() *ValidatorsSnapshot {
	return ToValidators(s.validatorsVarDB.Object())
}

func (s *State) SetValidatorsSnapshot(vss *ValidatorsSnapshot) error {
	o := icobject.New(TypeValidators, vss)
	return s.validatorsVarDB.Set(o)
}

func (s *State) GetLastBlockVotersSnapshot() *BlockVotersSnapshot {
	return ToBlockVoters(s.lastBlockVotersVarDB.Object())
}

func (s *State) SetLastBlockVotersSnapshot(value *BlockVotersSnapshot) error {
	o := icobject.New(TypeBlockVoters, value)
	return s.lastBlockVotersVarDB.Set(o)
}

func (s *State) IsDecentralizationConditionMet(revision int, totalSupply *big.Int, preps PRepSet) bool {
	predefinedMainPRepCount := int(s.GetMainPRepCount())
	br := s.GetBondRequirement()

	if revision >= icmodule.RevisionDecentralize && preps.Size() >= predefinedMainPRepCount {
		prep := preps.GetByIndex(predefinedMainPRepCount - 1)
		if prep == nil {
			return false
		}
		return totalSupply.Cmp(new(big.Int).Mul(prep.GetBondedDelegation(br), big.NewInt(500))) <= 0
	}
	return false
}

func (s *State) GetPRepSet() PRepSet {
	preps := s.GetPReps(true)
	return NewPRepSet(preps)
}

func (s *State) GetPReps(activeOnly bool) []*PRep {
	size := s.allPRepCache.Size()
	preps := make([]*PRep, 0)

	for i := 0; i < size; i++ {
		owner := s.allPRepCache.Get(i)
		prep := s.GetPRepByOwner(owner)
		if activeOnly && !prep.IsActive() {
			continue
		}
		if prep != nil {
			preps = append(preps, prep)
		}
	}
	return preps
}

func (s *State) GetPRepStatsInJSON(rev int, blockHeight int64) (map[string]interface{}, error) {
	statsList, err := s.GetPRepStatsList()
	if err != nil {
		return nil, err
	}

	size := len(statsList)
	preps := make([]interface{}, size)
	for i := 0; i < size; i++ {
		stats := statsList[i]
		preps[i] = stats.ToJSON(rev, blockHeight)
	}

	return map[string]interface{}{
		"blockHeight": blockHeight,
		"preps":       preps,
	}, nil
}

func (s *State) GetPRepStatsOfInJSON(
	rev int, blockHeight int64, address module.Address) (map[string]interface{}, error) {
	if address == nil {
		return nil, scoreresult.InvalidParameterError.New("InvalidAddress")
	}

	ps := s.GetPRepStatusByOwner(address, false)
	if ps == nil {
		return nil, scoreresult.InvalidParameterError.Errorf("PRepStatusNotFound(address=%s)", address)
	}

	stats := NewPRepStats(address, ps)
	return map[string]interface{}{
		"blockHeight": blockHeight,
		"preps": []interface{}{
			stats.ToJSON(rev, blockHeight),
		},
	}, nil
}

func (s *State) GetPRepsInJSON(sc icmodule.StateContext, start, end int) (map[string]interface{}, error) {
	prepSet := s.GetPRepSet()
	prepSet.SortForQuery(sc)

	if start < 0 {
		return nil, errors.IllegalArgumentError.Errorf("start(%d) < 0", start)
	}
	if end < 0 {
		return nil, errors.IllegalArgumentError.Errorf("end(%d) < 0", end)
	}

	size := prepSet.Size()
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

	for i := start - 1; i < end; i++ {
		prep := prepSet.GetByIndex(i)
		prepJSO := prep.ToJSON(sc)
		prepList = append(prepList, prepJSO)
	}

	jso["startRanking"] = start
	jso["blockHeight"] = sc.BlockHeight()
	jso["totalStake"] = s.GetTotalStake()
	jso["totalDelegated"] = prepSet.TotalDelegated()
	jso["preps"] = prepList
	return jso, nil
}

func (s *State) CheckValidationPenalty(ps *PRepStatusState, blockHeight int64) bool {
	condition := s.GetValidationPenaltyCondition()
	return checkValidationPenalty(ps, blockHeight, condition)
}

func checkValidationPenalty(ps *PRepStatusState, blockHeight, condition int64) bool {
	return !ps.IsAlreadyPenalized() && ps.GetVFailCont(blockHeight) >= condition
}

func (s *State) CheckConsistentValidationPenalty(revision int, ps *PRepStatusState) bool {
	if revision < icmodule.RevisionEnableIISS3 {
		return false
	}
	condition := int(s.GetConsistentValidationPenaltyCondition())
	return checkConsistentValidationPenalty(ps, condition)
}

func checkConsistentValidationPenalty(ps *PRepStatusState, condition int) bool {
	return ps.GetVPenaltyCount() >= condition
}

func (s *State) GetUnstakeLockPeriod(revision int, totalSupply *big.Int) int64 {
	totalStake := s.GetTotalStake()
	termPeriod := new(big.Int)
	if revision < icmodule.RevisionStopICON1Support {
		termPeriod.SetInt64(icmodule.InitialTermPeriod)
	} else {
		termPeriod.SetInt64(s.GetTermPeriod())
	}
	lMin := new(big.Int).Mul(s.GetLockMinMultiplier(), termPeriod)
	lMax := new(big.Int).Mul(s.GetLockMaxMultiplier(), termPeriod)
	return CalcUnstakeLockPeriod(lMin, lMax, totalStake, totalSupply)
}

func (s *State) SetIllegalDelegation(id *IllegalDelegation) error {
	dict := containerdb.NewDictDB(s.store, 1, IllegalDelegationPrefix)
	o := icobject.New(TypeIllegalDelegation, id)
	return dict.Set(id.Address(), o)
}

func (s *State) DeleteIllegalDelegation(addr module.Address) error {
	dict := containerdb.NewDictDB(s.store, 1, IllegalDelegationPrefix)
	return dict.Delete(addr)
}

func (s *State) GetIllegalDelegation(addr module.Address) *IllegalDelegation {
	dict := containerdb.NewDictDB(s.store, 1, IllegalDelegationPrefix)
	obj := dict.Get(addr)
	if obj == nil {
		return nil
	}
	return ToIllegalDelegation(obj.Object())
}

func (s *State) GetPRepIllegalDelegated(address module.Address) *big.Int {
	value := s.pRepIllegalDelegatedDB.Get(address)
	if value == nil {
		return new(big.Int)
	} else {
		return value.BigInt()
	}
}

func (s *State) SetPRepIllegalDelegated(address module.Address, value *big.Int) error {
	if value.Sign() == 0 {
		return s.pRepIllegalDelegatedDB.Delete(address)
	} else {
		return s.pRepIllegalDelegatedDB.Set(address, value)
	}
}

func (s *State) InitCommissionInfo(owner module.Address, ci *CommissionInfo) error {
	if owner == nil {
		return scoreresult.InvalidParameterError.Errorf("InvalidOwner(%s)", owner)
	}
	if ci == nil {
		return scoreresult.InvalidParameterError.New("InvalidCommissionInfo")
	}
	pb := s.GetPRepBaseByOwner(owner, false)
	if pb == nil {
		return icmodule.NotFoundError.Errorf("PRepBaseNotFound(%s)", owner)
	}
	ps := s.GetPRepStatusByOwner(owner, false)
	if ps == nil {
		return icmodule.NotFoundError.Errorf("PRepStatusNotFound(%s)", owner)
	}
	if !ps.IsActive() {
		return icmodule.NotReadyError.Errorf("PRepNotActive(%s)", owner)
	}
	return pb.InitCommissionInfo(ci)
}