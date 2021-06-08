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
)

type State struct {
	readonly            bool
	accountCache        *AccountCache
	activePRepCache     *ActivePRepCache
	nodeOwnerCache      *NodeOwnerCache
	prepBaseCache       *PRepBaseCache
	prepStatusCache     *PRepStatusCache
	unstakingTimerCache *TimerCache
	unbondingTimerCache *TimerCache
	termCache           *termCache
	logger              log.Logger

	store                *icobject.ObjectStoreState
	totalDelegationVarDB *containerdb.VarDB
	totalBondVarDB       *containerdb.VarDB
	validatorsVarDB      *containerdb.VarDB
}

func (s *State) Reset(ss *Snapshot) error {
	var err error
	s.store.Reset(ss.store.ImmutableForObject)
	s.accountCache.Reset()
	s.activePRepCache.Reset()
	s.nodeOwnerCache.Reset()
	s.prepBaseCache.Reset()
	s.prepStatusCache.Reset()
	s.unstakingTimerCache.Reset()
	s.unbondingTimerCache.Reset()
	if err = s.termCache.Reset(); err != nil {
		return err
	}
	return nil
}

func (s *State) Flush() error {
	s.accountCache.Flush()
	s.activePRepCache.Flush()
	s.nodeOwnerCache.Flush()
	s.prepBaseCache.Flush()
	s.prepStatusCache.Flush()
	s.unstakingTimerCache.Flush()
	s.unbondingTimerCache.Flush()
	return s.termCache.Flush()
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

func (s *State) addActivePRep(owner module.Address) {
	s.activePRepCache.Add(owner)
}

func (s *State) RemoveActivePRep(owner module.Address) error {
	return s.activePRepCache.Remove(owner)
}

func (s *State) GetActivePRepSize() int {
	return s.activePRepCache.Size()
}

func (s *State) getActivePRepOwner(i int) module.Address {
	return s.activePRepCache.Get(i)
}

func (s *State) GetPRepBaseByOwner(owner module.Address, createIfNotExist bool) (*PRepBase, bool) {
	return s.prepBaseCache.Get(owner, createIfNotExist)
}

func (s *State) GetPRepBaseByNode(node module.Address) *PRepBase {
	pb, _ := s.GetPRepBaseByOwner(s.GetOwnerByNode(node), false)
	return pb
}

func (s *State) GetPRepStatusByOwner(owner module.Address, createIfNotExist bool) (*PRepStatus, bool) {
	return s.prepStatusCache.Get(owner, createIfNotExist)
}

func (s *State) GetPRepByOwner(owner module.Address) *PRep {
	pb, _ := s.GetPRepBaseByOwner(owner, false)
	if pb == nil {
		return nil
	}
	ps, _ := s.GetPRepStatusByOwner(owner, false)
	if ps == nil {
		panic(errors.Errorf("PRepStatus not found: %s", owner))
	}
	return newPRep(owner, pb, ps)
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

	return &State{
		readonly:            readonly,
		accountCache:        newAccountCache(store),
		activePRepCache:     newActivePRepCache(store),
		nodeOwnerCache:      newNodeOwnerCache(store),
		prepBaseCache:       newPRepBaseCache(store),
		prepStatusCache:     newPRepStatusCache(store),
		unstakingTimerCache: newTimerCache(store, unstakingTimerDictPrefix),
		unbondingTimerCache: newTimerCache(store, unbondingTimerDictPrefix),
		termCache:           newTermCache(store),
		logger:              logger,

		store:                store,
		totalDelegationVarDB: tdVarDB,
		totalBondVarDB:       tbVarDB,
		validatorsVarDB:      validatorsVarDB,
	}
}

func (s *State) addNodeToOwner(node, owner module.Address) error {
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

func (s *State) GetTerm() *Term {
	return s.termCache.Get()
}

func (s *State) SetTerm(term *Term) error {
	return s.termCache.Set(term)
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
	// TODO clear other caches
	s.store.ClearCache()
}

func (s *State) RegisterPRep(owner module.Address, ri *RegInfo, irep *big.Int) error {
	if ri == nil {
		return errors.Errorf("Invalid argument: ri")
	}

	var err error
	ps, _ := s.GetPRepStatusByOwner(owner, true)
	if ps.Status() != NotReady {
		return errors.Errorf("Already in use: addr=%s %+v", owner, ps)
	}

	s.addActivePRep(owner)

	// Update PRepBase
	pb, created := s.GetPRepBaseByOwner(owner, true)
	if !created {
		return errors.Errorf("Already in use: addr=%s %+v", owner, pb)
	}
	if err = pb.SetRegInfo(ri); err != nil {
		return err
	}
	pb.SetIrep(irep, 0)
	ps.SetStatus(Active)

	// Register a node address
	node := ri.Node()
	if node == nil || owner.Equal(node) {
		return nil
	}
	if pb, _ = s.GetPRepBaseByOwner(node, false); pb != nil {
		return errors.Errorf("Node address in use: %s", ri.node)
	}
	return s.addNodeToOwner(node, owner)
}

func (s *State) SetPRep(owner module.Address, ri *RegInfo) error {
	// owner -> node
	// node1 -> node2
	// node -> owner

	pb, _ := s.GetPRepBaseByOwner(owner, false)
	if pb == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}

	oldNode := pb.GetNode(owner)
	if err := pb.fillEmptyRegInfo(ri); err != nil {
		return nil
	}

	// If node address is changed and the node is a main prep, validator set should be changed too.
	newNode := ri.Node()
	if newNode == nil || oldNode.Equal(newNode) {
		return nil
	}

	if !owner.Equal(newNode) {
		// Forbidden to use other node's owner address as a node address
		if pb, _ = s.GetPRepBaseByOwner(newNode, false); pb != nil {
			return errors.Errorf("Node address in use: %s", ri.node)
		}
	}

	if err := s.addNodeToOwner(newNode, owner); err != nil {
		return err
	}

	return s.changeValidatorNodeAddress(owner, oldNode, newNode)
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

func (s *State) ShiftVPenaltyMaskByNode(node module.Address) error {
	owner := s.GetOwnerByNode(node)
	ps, _ := s.GetPRepStatusByOwner(owner, false)
	if ps == nil {
		return errors.Errorf("PRep not found: node=%v owner=%v", node, owner)
	}

	ps.ShiftVPenaltyMask(buildPenaltyMask(s.GetConsistentValidationPenaltyMask()))
	return nil
}

func (s *State) GetOwnerByNode(node module.Address) module.Address {
	return s.nodeOwnerCache.Get(node)
}

func (s *State) GetNodeByOwner(owner module.Address) module.Address {
	pb, _ := s.GetPRepBaseByOwner(owner, false)
	if pb == nil {
		return nil
	}
	return pb.GetNode(owner)
}

func buildPenaltyMask(input int) (res uint32) {
	for i := 0; i < input; i++ {
		res = (res << 1) | uint32(1)
	}
	return
}

func (s *State) UpdateBlockVoteStats(owner module.Address, voted bool, blockHeight int64) error {
	if !voted {
		s.logger.Debugf("Nil vote: bh=%d addr=%s", blockHeight, owner)
	}
	ps, _ := s.GetPRepStatusByOwner(owner, false)
	if ps == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}
	err := ps.UpdateBlockVoteStats(blockHeight, voted)
	s.logger.Debugf("voted=%t %+v", voted, ps)
	return err
}

// GetPRepStatuses returns PRepStatus list ordered by bonded delegation
func (s *State) GetPRepStatuses() ([]*PRepStatus, error) {
	br := s.GetBondRequirement()

	size := s.activePRepCache.Size()
	owners := make([]module.Address, 0)
	pss := make([]*PRepStatus, 0)

	for i := 0; i < size; i++ {
		owner := s.getActivePRepOwner(i)
		ps, _ := s.GetPRepStatusByOwner(owner, false)
		if ps.Status() == Active {
			owners = append(owners, owner)
			pss = append(pss, ps)
		}
	}

	sortPRepStatuses(owners, pss, br)
	return pss, nil
}

func sortPRepStatuses(owners []module.Address, pss []*PRepStatus, br int64) {
	sort.Slice(pss, func(i, j int) bool {
		ret := pss[i].GetBondedDelegation(br).Cmp(pss[j].GetBondedDelegation(br))
		if ret > 0 {
			return true
		} else if ret < 0 {
			return false
		}

		ret = pss[i].Delegated().Cmp(pss[j].Delegated())
		if ret > 0 {
			return true
		} else if ret < 0 {
			return false
		}

		return bytes.Compare(owners[i].Bytes(), owners[j].Bytes()) > 0
	})
}

// ImposePenalty changes grade change and set LastState to icstate.None
func (s *State) ImposePenalty(owner module.Address, ps *PRepStatus, blockHeight int64) error {
	var err error

	// Update status of the penalized main prep
	s.logger.Debugf("ImposePenalty() start: bh=%d %+v", blockHeight, ps)

	oldGrade := ps.Grade()
	err = ps.OnPenaltyImposed(blockHeight)

	s.logger.Debugf("ImposePenalty() end: bh=%d %+v", blockHeight, ps)

	// If a penalized prep is a main prep, choose a new validator from prep snapshots
	if err == nil && oldGrade == Main {
		err = s.replaceValidatorByOwner(owner)
	}
	return err
}

// Slash handles to reduce PRepStatus.bonded and PRepManager.totalBonded
// Do not change PRep grade here
// Caution: amount should not include the amount from unbonded
func (s *State) Slash(owner module.Address, amount *big.Int) error {
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

	ps, _ := s.GetPRepStatusByOwner(owner, false)
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

func (s *State) DisablePRep(owner module.Address, status Status) error {
	ps, _ := s.GetPRepStatusByOwner(owner, false)
	if ps == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}

	oldGrade := ps.Grade()
	ps.SetGrade(Candidate)
	ps.SetStatus(status)

	if oldGrade == Main {
		if err := s.replaceValidatorByOwner(owner); err != nil {
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

func (s *State) IsDecentralizationConditionMet(revision int, totalSupply *big.Int, preps *PReps) bool {
	predefinedMainPRepCount := int(s.GetMainPRepCount())
	br := s.GetBondRequirement()

	if revision >= icmodule.RevisionDecentralize && s.GetActivePRepSize() >= predefinedMainPRepCount {
		prep := preps.GetPRepByIndex(predefinedMainPRepCount - 1)
		return totalSupply.Cmp(new(big.Int).Mul(prep.GetBondedDelegation(br), big.NewInt(500))) <= 0
	}
	return false
}

func (s *State) GetOrderedPReps() (*PReps, error) {
	size := s.GetActivePRepSize()
	prepList := make([]*PRep, size)

	for i := 0; i < size; i++ {
		owner := s.getActivePRepOwner(i)
		prep := s.GetPRepByOwner(owner)
		if prep != nil {
			prepList[i] = prep
		}
	}

	return newPReps(prepList, s.GetBondRequirement()), nil
}

func (s *State) GetPRepStatsInJSON(blockHeight int64) (map[string]interface{}, error) {
	pss, err := s.GetPRepStatuses()
	if err != nil {
		return nil, err
	}

	size := len(pss)
	jso := make(map[string]interface{})
	psList := make([]interface{}, size)

	for i := 0; i < size; i++ {
		ps := pss[i]
		psList[i] = ps.GetStatsInJSON(blockHeight)
	}

	jso["blockHeight"] = blockHeight
	jso["preps"] = psList
	return jso, nil
}

func (s *State) GetPRepsInJSON(blockHeight int64, start, end int) (map[string]interface{}, error) {
	preps, err := s.GetOrderedPReps()
	if err != nil {
		return nil, err
	}

	if start < 0 {
		return nil, errors.IllegalArgumentError.Errorf("start(%d) < 0", start)
	}
	if end < 0 {
		return nil, errors.IllegalArgumentError.Errorf("end(%d) < 0", end)
	}

	size := preps.Size()
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
	br := s.GetBondRequirement()

	for i := start - 1; i < end; i++ {
		prep := preps.GetPRepByIndex(i)
		prepList = append(prepList, prep.ToJSON(blockHeight, br))
	}

	jso["startRanking"] = start
	jso["blockHeight"] = blockHeight
	jso["totalStake"] = s.GetTotalStake()
	jso["totalDelegated"] = preps.TotalDelegated()
	jso["preps"] = prepList
	return jso, nil
}

func (s *State) GetPRepManagerInJSON() map[string]interface{} {
	br := s.GetBondRequirement()
	preps, _ := s.GetOrderedPReps()
	if preps == nil {
		return nil
	}

	return map[string]interface{}{
		"totalStake": s.GetTotalStake(),
		"totalBonded": preps.TotalBonded(),
		"totalDelegated": preps.TotalDelegated(),
		"totalBondedDelegation": preps.GetTotalBondedDelegation(br),
		"preps": preps.Size(),
	}
}
