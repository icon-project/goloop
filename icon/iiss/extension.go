/*
 * Copyright 2020 ICON Foundation
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package iiss

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

type ExtensionSnapshotImpl struct {
	database db.Database

	pm *PRepManager

	state  *icstate.Snapshot
	front  *icstage.Snapshot
	back   *icstage.Snapshot
	reward *icreward.Snapshot
}

func (s *ExtensionSnapshotImpl) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(s)
}

func (s *ExtensionSnapshotImpl) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(
		s.state.Bytes(),
		s.front.Bytes(),
		s.back.Bytes(),
		s.reward.Bytes(),
	)
}

func (s *ExtensionSnapshotImpl) RLPDecodeSelf(d codec.Decoder) error {
	var stateHash, frontHash, backHash, rewardHash []byte
	if err := d.DecodeListOf(&stateHash, &frontHash, &backHash, &rewardHash); err != nil {
		return err
	}
	s.state = icstate.NewSnapshot(s.database, stateHash)
	s.front = icstage.NewSnapshot(s.database, frontHash)
	s.back = icstage.NewSnapshot(s.database, backHash)
	s.reward = icreward.NewSnapshot(s.database, rewardHash)
	return nil
}

func (s *ExtensionSnapshotImpl) Flush() error {
	if err := s.state.Flush(); err != nil {
		return err
	}
	if err := s.front.Flush(); err != nil {
		return err
	}
	if err := s.back.Flush(); err != nil {
		return err
	}
	if err := s.reward.Flush(); err != nil {
		return err
	}
	return nil
}

func (s *ExtensionSnapshotImpl) NewState(readonly bool) state.ExtensionState {
	es := &ExtensionStateImpl{
		database: s.database,
		State:    icstate.NewStateFromSnapshot(s.state, readonly),
		Front:    icstage.NewStateFromSnapshot(s.front),
		Back:     icstage.NewStateFromSnapshot(s.back),
		Reward:   icreward.NewStateFromSnapshot(s.reward),
	}

	// TODO: Need to get totalStake from State
	totalStake := big.NewInt(0)
	es.pm = newPRepManager(es.State, totalStake)
	return es
}

func NewExtensionSnapshot(database db.Database, hash []byte) state.ExtensionSnapshot {
	if hash == nil {
		return &ExtensionSnapshotImpl{
			database: database,
			state:    icstate.NewSnapshot(database, nil),
			front:    icstage.NewSnapshot(database, nil),
			back:     icstage.NewSnapshot(database, nil),
			reward:   icreward.NewSnapshot(database, nil),
		}
	}
	s := &ExtensionSnapshotImpl{
		database: database,
	}
	if _, err := codec.BC.UnmarshalFromBytes(hash, s); err != nil {
		return nil
	}
	return s
}

type ExtensionStateImpl struct {
	database db.Database

	pm *PRepManager

	State  *icstate.State
	Front  *icstage.State
	Back   *icstage.State
	Reward *icreward.State

	// Memory only -----
	// If this flag is on, new validators will be set in OnExecutionEnd()
	// It is valid until a transition is finished
	updateValidator bool
}

func (s *ExtensionStateImpl) GetSnapshot() state.ExtensionSnapshot {
	return &ExtensionSnapshotImpl{
		database: s.database,
		state:    s.State.GetSnapshot(),
		front:    s.Front.GetSnapshot(),
		back:     s.Back.GetSnapshot(),
		reward:   s.Reward.GetSnapshot(),
	}
}

func (s *ExtensionStateImpl) Reset(isnapshot state.ExtensionSnapshot) {
	snapshot := isnapshot.(*ExtensionSnapshotImpl)
	if err := s.State.Reset(snapshot.state); err != nil {
		panic(err)
	}
	s.Front.Reset(snapshot.front)
	s.Back.Reset(snapshot.back)
	s.Reward.Reset(snapshot.reward)
}

func (s *ExtensionStateImpl) ClearCache() {
	// TODO clear cached objects
	// It is called whenever executing a transaction is finish
}

func (s *ExtensionStateImpl) GetAccount(address module.Address) *icstate.Account {
	return s.State.GetAccount(address)
}


func (s *ExtensionStateImpl) GetUnstakingTimerState(height int64, createIfNotExist bool) *icstate.Timer {
	return s.State.GetUnstakingTimer(height, createIfNotExist)
}

func (s *ExtensionStateImpl) GetUnbondingTimerState(height int64, createIfNotExist bool) *icstate.Timer {
	return s.State.GetUnbondingTimer(height, createIfNotExist)
}

func (s *ExtensionStateImpl) CalculationBlockHeight() int64 {
	rcInfo, err := s.State.GetRewardCalcInfo()
	if err != nil || rcInfo == nil {
		return 0
	}
	return rcInfo.StartHeight()
}

func (s *ExtensionStateImpl) PrevCalculationBlockHeight() int64 {
	rcInfo, err := s.State.GetRewardCalcInfo()
	if err != nil || rcInfo == nil {
		return 0
	}
	return rcInfo.PrevHeight()
}

func (s *ExtensionStateImpl) NewCalculation(term *icstate.Term, calculator *Calculator) error {
	rc, err := s.State.GetRewardCalcInfo()
	rcInfo := rc.Clone()
	if err != nil {
		return err
	}
	if !calculator.IsCalcDone(rcInfo.StartHeight()) {
		err = CriticalCalculatorError.Errorf("Reward calculation is not finished (%d)", rcInfo.StartHeight())
		return err
	}

	// apply calculation result
	if calculator.Result() != nil {
		s.Reward = calculator.Result().NewState()
		if err = RegulateIssueInfo(s, calculator.TotalReward()); err != nil {
			return err
		}
	}
	// switch icstage and write global
	s.Back = s.Front
	s.Front = icstage.NewState(s.database)
	version := s.State.GetIISSVersion()
	switch version {
	case icstate.IISSVersion1:
		if err = s.Back.AddGlobalV1(
			term.StartHeight(),
			int(term.Period()),
			term.Irep(),
			term.Rrep(),
			term.MainPRepCount(),
			term.ElectedPRepCount(),
		); err != nil {
			return err
		}
	case icstate.IISSVersion2:
		if err = s.Back.AddGlobalV2(
			term.StartHeight(),
			int(term.Period()),
			term.Iglobal(),
			term.Iprep(),
			term.Ivoter(),
			term.ElectedPRepCount(),
			term.BondRequirement(),
		); err != nil {
			return err
		}
	default:
		return errors.CriticalFormatError.Errorf(
			"InvalidIISSVersion(version=%d)", version)
	}

	// update rewardCalcInfo
	rcInfo.Start(term.StartHeight())
	if err = s.State.SetRewardCalcInfo(rcInfo); err != nil {
		return err
	}

	return nil
}

func (s *ExtensionStateImpl) GetPRepManagerInJSON() map[string]interface{} {
	return s.pm.ToJSON()
}

func (s *ExtensionStateImpl) GetPRepsInJSON(blockHeight int64, start, end int) map[string]interface{} {
	return s.pm.GetPRepsInJSON(blockHeight, start, end)
}

func (s *ExtensionStateImpl) GetPRepInJSON(address module.Address, blockHeight int64) (map[string]interface{}, error) {
	prep := s.pm.GetPRepByOwner(address)
	if prep == nil {
		return nil, errors.Errorf("PRep not found: %s", address)
	}

	return prep.ToJSON(blockHeight, s.State.GetBondRequirement()), nil
}

func (s *ExtensionStateImpl) GetTotalDelegated() *big.Int {
	return s.pm.TotalDelegated()
}

func (s *ExtensionStateImpl) RegisterPRep(owner, node module.Address, params []string) error {
	return s.pm.RegisterPRep(owner, node, params)
}

func (s *ExtensionStateImpl) SetDelegation(cc contract.CallContext, from module.Address, ds icstate.Delegations) error {
	var err error
	var account *icstate.Account
	var delta map[string]*big.Int

	account = s.State.GetAccount(from)

	if account.Stake().Cmp(new(big.Int).Add(ds.GetDelegationAmount(), account.Bond())) == -1 {
		return errors.Errorf("Not enough voting power")
	}
	delta, err = s.pm.ChangeDelegation(account.Delegations(), ds)
	if err != nil {
		return err
	}

	if err = s.addEventDelegation(cc.BlockHeight(), from, delta); err != nil {
		return err
	}

	account.SetDelegation(ds)
	return nil
}

func deltaToVotes(delta map[string]*big.Int) (votes icstage.VoteList, err error) {
	votes = make([]*icstage.Vote, 0, len(delta))
	for key, value := range delta {
		var addr *common.Address
		vote := icstage.NewVote()

		addr, err = common.NewAddress([]byte(key))
		if err != nil {
			return
		}
		vote.Address = addr
		vote.Value.Set(value)
		votes = append(votes, vote)
	}
	return
}

func (s *ExtensionStateImpl) addEventDelegation(blockHeight int64, from module.Address, delta map[string]*big.Int) (err error) {
	votes, err := deltaToVotes(delta)
	if err != nil {
		return
	}
	term := s.State.GetTerm()
	_, err = s.Front.AddEventDelegation(
		int(blockHeight-term.StartHeight()),
		from,
		votes,
	)
	return
}

func (s *ExtensionStateImpl) UnregisterPRep(cc contract.CallContext, owner module.Address) error {
	prep := s.pm.GetPRepByOwner(owner)
	if prep == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}

	grade := prep.Grade()
	if grade != icstate.Candidate {
		term := s.State.GetTerm()
		if err := term.RemovePRepSnapshot(owner); err != nil {
			return err
		}
		s.updateValidator = grade == icstate.Main
	}

	err := s.pm.UnregisterPRep(owner)
	if err != nil {
		return err
	}

	term := s.State.GetTerm()
	_, err = s.Front.AddEventEnable(
		int(cc.BlockHeight()-term.StartHeight()),
		owner,
		false,
	)

	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepUnRegistered(Address)")},
		[][]byte{owner.Bytes()},
	)

	return err
}

func (s *ExtensionStateImpl) SetPRep(from, node module.Address, params []string) error {
	if node != nil {
		prep := s.pm.GetPRepByOwner(from)
		if prep != nil && prep.Grade() == icstate.Main && !prep.GetNode().Equal(node) {
			s.updateValidator = true
		}
	}
	return s.pm.SetPRep(from, node, params)
}

func (s *ExtensionStateImpl) SetBond(cc contract.CallContext, from module.Address, bonds icstate.Bonds) error {
	var err error
	var account *icstate.Account
	blockHeight := cc.BlockHeight()

	account = s.GetAccount(from)

	bondAmount := big.NewInt(0)
	for _, bond := range bonds {
		bondAmount.Add(bondAmount, bond.Amount())

		prep := s.pm.GetPRepByOwner(bond.To())
		if prep == nil {
			return errors.Errorf("PRep not found: %v", from)
		}
		if !prep.BonderList().Contains(from) {
			return errors.Errorf("%s is not in bonder List of %s", from.String(), bond.To().String())
		}
	}
	if account.Stake().Cmp(new(big.Int).Add(bondAmount, account.Delegating())) == -1 {
		return errors.Errorf("Not enough voting power")
	}

	unbondingHeight := blockHeight + icstate.UnbondingPeriod
	ubToAdd, ubToMod, ubDiff := account.GetUnbondingInfo(bonds, unbondingHeight)
	votingAmount := new(big.Int).Add(account.Delegating(), bondAmount)
	votingAmount.Sub(votingAmount, account.Bond())
	unbondingAmount := new(big.Int).Add(account.Unbonds().GetUnbondAmount(), ubDiff)
	if account.Stake().Cmp(new(big.Int).Add(votingAmount, unbondingAmount)) == -1 {
		return errors.Errorf("Not enough voting power")
	}

	var delta map[string]*big.Int
	delta, err = s.pm.ChangeBond(account.Bonds(), bonds)
	if err != nil {
		return err
	}

	account.SetBonds(bonds)
	tl := account.UpdateUnbonds(ubToAdd, ubToMod)
	for _, t := range tl {
		ts := s.State.GetUnbondingTimer(t.Height, true)
		if err = icstate.ScheduleTimerJob(ts, t, from); err != nil {
			return errors.Errorf("Error while scheduling Unbonding Timer Job")
		}
	}

	if err = s.addEventBond(blockHeight, from, delta); err != nil {
		return err
	}

	return nil
}

func (s *ExtensionStateImpl) addEventBond(blockHeight int64, from module.Address, delta map[string]*big.Int) (err error) {
	votes, err := deltaToVotes(delta)
	if err != nil {
		return
	}
	term := s.State.GetTerm()
	_, err = s.Front.AddEventBond(
		int(blockHeight-term.StartHeight()),
		from,
		votes,
	)
	return
}

func (s *ExtensionStateImpl) SetBonderList(from module.Address, bl icstate.BonderList) error {
	pb := s.State.GetPRepBase(from, false)
	if pb == nil {
		return errors.Errorf("PRep not found: %v", from)
	}

	var account *icstate.Account
	for _, old := range pb.BonderList() {
		if !bl.Contains(old) {
			account = s.GetAccount(old)
			if len(account.Bonds()) > 0 || len(account.Unbonds()) > 0 {
				return errors.Errorf("Bonding/Unbonding exist. bonds : %d, unbonds : %d", len(account.Bonds()), len(account.Unbonds()))
			}
		}
	}

	pb.SetBonderList(bl)
	return nil
}

func (s *ExtensionStateImpl) GetBonderList(address module.Address) ([]interface{}, error) {
	pb := s.State.GetPRepBase(address, false)
	if pb == nil {
		return nil, errors.Errorf("PRep not found: %v", address)
	}
	return pb.GetBonderListInJSON(), nil
}

func (s *ExtensionStateImpl) UpdateIssueInfo(fee *big.Int) error {
	is, err := s.State.GetIssue()
	if err != nil {
		return err
	}
	issue := is.Clone()
	issue.PrevBlockFee.Add(issue.PrevBlockFee, fee)
	if err = s.State.SetIssue(issue); err != nil {
		return err
	}
	return nil
}

func (s *ExtensionStateImpl) OnExecutionEnd(wc state.WorldContext, calculator *Calculator) error {
	var err error
	term := s.State.GetTerm()
	if term == nil {
		return nil
	}

	blockHeight := wc.BlockHeight()
	if blockHeight == term.GetEndBlockHeight() {
		if err = s.NewCalculation(term, calculator); err != nil {
			return err
		}

		err = s.onTermEnd(wc)
		if err != nil {
			panic(err)
		}
	}

	if s.updateValidator {
		err = s.setValidators(wc)
		if err != nil {
			panic(err)
		}
		s.updateValidator = false
	}

	return nil
}

func (s *ExtensionStateImpl) onTermEnd(wc state.WorldContext) error {
	var err error
	var totalSupply *big.Int

	if err = s.pm.OnTermEnd(); err != nil {
		return err
	}

	totalSupply, err = s.getTotalSupply(wc)
	if err != nil {
		return err
	}
	if err = s.moveOnToNextTerm(totalSupply); err != nil {
		return err
	}

	if s.IsDecentralized() {
		s.updateValidator = true
	}
	return nil
}

func (s *ExtensionStateImpl) moveOnToNextTerm(totalSupply *big.Int) error {
	term := s.State.GetTerm()
	rf := s.State.GetRewardFund()
	nextTerm := icstate.NewNextTerm(
		term,
		s.State.GetTermPeriod(),
		s.State.GetIRep(),
		s.State.GetRRep(),
		totalSupply,
		s.pm.TotalDelegated(),
		rf.Iglobl,
		rf.Iprep,
		rf.Ivoter,
		int(s.State.GetBondRequirement()),
		s.State.GetIISSVersion(),
	)

	size := 0
	mainPRepCount := int(s.State.GetMainPRepCount())
	activePRepCount := s.pm.Size()

	if term.IsDecentralized() || activePRepCount >= mainPRepCount {
		prepCount := int(s.State.GetPRepCount())
		size = icutils.Min(activePRepCount, prepCount)
	}

	if size > 0 {
		prepSnapshots := make(icstate.PRepSnapshots, size, size)
		br := s.State.GetBondRequirement()
		for i := 0; i < size; i++ {
			prep := s.pm.GetPRepByIndex(i)
			prepSnapshots[i] = icstate.NewPRepSnapshotFromPRepStatus(prep.PRepStatus, br)
		}

		nextTerm.SetPRepSnapshots(prepSnapshots)
		s.updateValidator = true
	}

	log.Debugf("#### %s", nextTerm)
	return s.State.SetTerm(nextTerm)
}

func (s *ExtensionStateImpl) setValidators(wc state.WorldContext) error {
	blockHeight := wc.BlockHeight()
	validators := s.GetValidators()
	size := len(validators)

	if size > 0 {
		// shift validation penalty mask
		for _, v := range validators {
			// TODO IC2-35 When creating a validator with a validation penalty, only the newly added P-Rep is modified.
			pRepStatus := s.pm.GetPRepByNode(v.Address())
			pRepStatus.ShiftVPenaltyMask(ConsistentValidationPenaltyMask)
		}

		// TODO: Remove the comment below when testing with multiple nodes
		//return wc.GetValidatorState().Set(validators)
		for i := 0; i < size; i++ {
			log.Debugf("Validator %d: %s", i, validators[i].Address())
		}
	} else {
		log.Infof("Not enough PReps height=%d, size=%d", blockHeight, size)
	}
	return nil
}

func (s *ExtensionStateImpl) GetValidators() []module.Validator {
	mainPRepCount := int(s.State.GetMainPRepCount())

	term := s.State.GetTerm()
	prepSnapshotCount := term.GetPRepSnapshotCount()
	if prepSnapshotCount < mainPRepCount {
		log.Warnf("Not enough PReps: %d < %d", prepSnapshotCount, mainPRepCount)
	}

	var err error
	size := icutils.Min(mainPRepCount, prepSnapshotCount)
	validators := make([]module.Validator, size, size)

	for i := 0; i < size; i++ {
		prepSnapshot := term.GetPRepSnapshotByIndex(i)
		prep := s.pm.GetPRepByOwner(prepSnapshot.Owner())
		node := prep.GetNode()
		validators[i], err = state.ValidatorFromAddress(node)
		if err != nil {
			log.Errorf("Failed to run GetValidators(): %s", node.String())
		}
	}

	return validators
}

func (s *ExtensionStateImpl) GetPRepTermInJSON() (map[string]interface{}, error) {
	term := s.State.GetTerm()
	if term == nil {
		err := errors.Errorf("Term is nil")
		return nil, err
	}
	return term.ToJSON(), nil
}

func (s *ExtensionStateImpl) GetNetworkValueInJSON() (map[string]interface{}, error) {
	return icstate.NetworkValueToJSON(s.State), nil
}

func (s *ExtensionStateImpl) getTotalSupply(wc state.WorldContext) (*big.Int, error) {
	ass := wc.GetAccountState(state.SystemID).GetSnapshot()
	as := scoredb.NewStateStoreWith(ass)
	tsVar := scoredb.NewVarDB(as, state.VarTotalSupply)
	if ts := tsVar.BigInt(); ts != nil {
		return ts, nil
	}
	return big.NewInt(0), nil
}

func (s *ExtensionStateImpl) IsDecentralized() bool {
	return s.State.GetTerm().IsDecentralized()
}
