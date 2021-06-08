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

package iiss

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

type ExtensionSnapshotImpl struct {
	database db.Database

	state  *icstate.Snapshot
	front  *icstage.Snapshot
	back1  *icstage.Snapshot
	back2  *icstage.Snapshot
	reward *icreward.Snapshot
}

func (s *ExtensionSnapshotImpl) Back1() *icstage.Snapshot {
	return s.back1
}

func (s *ExtensionSnapshotImpl) Reward() *icreward.Snapshot {
	return s.reward
}

func (s *ExtensionSnapshotImpl) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(s)
}

func (s *ExtensionSnapshotImpl) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(
		s.state.Bytes(),
		s.front.Bytes(),
		s.back1.Bytes(),
		s.back2.Bytes(),
		s.reward.Bytes(),
	)
}

func (s *ExtensionSnapshotImpl) RLPDecodeSelf(d codec.Decoder) error {
	var stateHash, frontHash, back1Hash, back2Hash, rewardHash []byte
	if err := d.DecodeListOf(&stateHash, &frontHash, &back1Hash, &back2Hash, &rewardHash); err != nil {
		return err
	}
	s.state = icstate.NewSnapshot(s.database, stateHash)
	s.front = icstage.NewSnapshot(s.database, frontHash)
	s.back1 = icstage.NewSnapshot(s.database, back1Hash)
	s.back2 = icstage.NewSnapshot(s.database, back2Hash)
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
	if err := s.back1.Flush(); err != nil {
		return err
	}
	if err := s.back2.Flush(); err != nil {
		return err
	}
	if err := s.reward.Flush(); err != nil {
		return err
	}
	return nil
}

func (s *ExtensionSnapshotImpl) NewState(readonly bool) state.ExtensionState {
	logger := icutils.NewIconLogger(nil)

	es := &ExtensionStateImpl{
		database: s.database,
		logger:   logger,
		State:    icstate.NewStateFromSnapshot(s.state, readonly, logger),
		Front:    icstage.NewStateFromSnapshot(s.front),
		Back1:    icstage.NewStateFromSnapshot(s.back1),
		Back2:    icstage.NewStateFromSnapshot(s.back2),
		Reward:   icreward.NewStateFromSnapshot(s.reward),
	}

	pm := newPRepManager(es.State, logger)
	es.pm = pm
	return es
}

func NewExtensionSnapshot(database db.Database, hash []byte) state.ExtensionSnapshot {
	if hash == nil {
		return &ExtensionSnapshotImpl{
			database: database,
			state:    icstate.NewSnapshot(database, nil),
			front:    icstage.NewSnapshot(database, nil),
			back1:    icstage.NewSnapshot(database, nil),
			back2:    icstage.NewSnapshot(database, nil),
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

func NewExtensionSnapshotWithBuilder(builder merkle.Builder, raw []byte) state.ExtensionSnapshot {
	var hashes [5][]byte
	if _, err := codec.BC.UnmarshalFromBytes(raw, &hashes); err != nil {
		return nil
	}
	return &ExtensionSnapshotImpl{
		database: builder.Database(),
		state:    icstate.NewSnapshotWithBuilder(builder, hashes[0]),
		front:    icstage.NewSnapshotWithBuilder(builder, hashes[1]),
		back1:    icstage.NewSnapshotWithBuilder(builder, hashes[2]),
		back2:    icstage.NewSnapshotWithBuilder(builder, hashes[3]),
		reward:   icreward.NewSnapshotWithBuilder(builder, hashes[4]),
	}
}

type ExtensionStateImpl struct {
	database db.Database

	pm     *PRepManager
	logger log.Logger

	State  *icstate.State
	Front  *icstage.State
	Back1  *icstage.State
	Back2  *icstage.State
	Reward *icreward.State
}

func (s *ExtensionStateImpl) Logger() log.Logger {
	return s.logger
}

func (s *ExtensionStateImpl) SetLogger(logger log.Logger) {
	if logger != nil {
		s.logger = logger
	}
}

func (s *ExtensionStateImpl) GetSnapshot() state.ExtensionSnapshot {
	return &ExtensionSnapshotImpl{
		database: s.database,
		state:    s.State.GetSnapshot(),
		front:    s.Front.GetSnapshot(),
		back1:    s.Back1.GetSnapshot(),
		back2:    s.Back2.GetSnapshot(),
		reward:   s.Reward.GetSnapshot(),
	}
}

func (s *ExtensionStateImpl) Reset(isnapshot state.ExtensionSnapshot) {
	snapshot := isnapshot.(*ExtensionSnapshotImpl)
	if err := s.State.Reset(snapshot.state); err != nil {
		panic(err)
	}
	s.Front.Reset(snapshot.front)
	s.Back1.Reset(snapshot.back1)
	s.Back2.Reset(snapshot.back2)
	s.Reward.Reset(snapshot.reward)
}

// ClearCache clear cache. It's called before executing first transaction
// and also it could be called at the end of base transaction
func (s *ExtensionStateImpl) ClearCache() {
	s.State.ClearCache()
	s.Front.ClearCache()
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

func (s *ExtensionStateImpl) newCalculation() (err error) {
	term := s.State.GetTerm()

	// switch icstage values
	s.Back2 = s.Back1
	s.Back1 = s.Front
	s.Front = icstage.NewState(s.database)

	// write icstage.Global for new calculation to Front
	iissVersion := term.GetIISSVersion()
	switch iissVersion {
	case icstate.IISSVersion2:
		if err = s.Front.AddGlobalV1(
			term.Revision(),
			term.StartHeight(),
			int(term.Period()-1),
			term.Irep(),
			term.Rrep(),
			term.MainPRepCount(),
			term.GetElectedPRepCount(),
		); err != nil {
			return
		}
	case icstate.IISSVersion3:
		if err = s.Front.AddGlobalV2(
			term.Revision(),
			term.StartHeight(),
			int(term.Period()-1),
			term.Iglobal(),
			term.Iprep(),
			term.Ivoter(),
			term.Icps(),
			term.Irelay(),
			term.GetElectedPRepCount(),
			term.BondRequirement(),
		); err != nil {
			return
		}
	default:
		return errors.CriticalFormatError.Errorf(
			"InvalidIISSVersion(version=%d)", iissVersion)
	}
	return
}

func (s *ExtensionStateImpl) applyCalculationResult(calculator *Calculator) error {
	rc, err := s.State.GetRewardCalcInfo()
	rcInfo := rc.Clone()
	if err != nil {
		return err
	}

	if !calculator.IsCalcDone(rcInfo.StartHeight()) {
		if err = calculator.Error(); err != nil {
			return err
		}
		return icmodule.CalculationNotFinishedError.Errorf("Calculation is not finished %d", rcInfo.StartHeight())
	}

	var calcHash []byte
	if calculator.Result() != nil {
		s.Reward = calculator.Result().NewState()
		calcHash = calculator.Result().Bytes()
	}

	prevGlobal, err := s.Back2.GetGlobal()
	if err != nil {
		return err
	}
	reward := new(big.Int).Set(calculator.TotalReward())
	if prevGlobal != nil && icstate.IISSVersion3 == prevGlobal.GetIISSVersion() {
		pg := prevGlobal.GetV2()
		rewardCPS := new(big.Int).Mul(pg.GetIGlobal(), pg.GetICps())
		rewardCPS.Mul(rewardCPS, big.NewInt(10)) // 10 = IScoreICXRation / 100
		reward.Add(reward, rewardCPS)
		rewardRelay := new(big.Int).Mul(pg.GetIGlobal(), pg.GetIRelay())
		rewardRelay.Mul(rewardCPS, big.NewInt(10))
		reward.Add(reward, rewardRelay)
	}

	if err = s.UpdateIssueInfo(reward); err != nil {
		return err
	}

	// update rewardCalcInfo
	prevGlobal, err = s.Back1.GetGlobal()
	if err != nil {
		return err
	}
	if prevGlobal != nil {
		rcInfo.Update(prevGlobal.GetStartHeight(), reward, calcHash)
	}
	if err = s.State.SetRewardCalcInfo(rcInfo); err != nil {
		return err
	}

	return nil
}

func (s *ExtensionStateImpl) GetPRepInJSON(address module.Address, blockHeight int64) (map[string]interface{}, error) {
	prep := s.State.GetPRepByOwner(address)
	if prep == nil {
		return nil, errors.Errorf("PRep not found: %s", address)
	}
	return prep.ToJSON(blockHeight, s.GetBondRequirement()), nil
}

func (s *ExtensionStateImpl) GetBondRequirement() int64 {
	if s.State.GetIISSVersion() < icstate.IISSVersion3 {
		return 0
	}
	return s.State.GetBondRequirement()
}

func (s *ExtensionStateImpl) GetMainPRepsInJSON(blockHeight int64) (map[string]interface{}, error) {
	term := s.State.GetTerm()
	if term == nil {
		err := errors.Errorf("Term is nil")
		return nil, err
	}

	pssCount := term.GetPRepSnapshotCount()
	mainPRepCount := int(s.State.GetMainPRepCount())
	jso := make(map[string]interface{})
	preps := make([]interface{}, 0, mainPRepCount)
	sum := new(big.Int)

	for i := 0; i < pssCount; i++ {
		pss := term.GetPRepSnapshotByIndex(i)
		ps, _ := s.State.GetPRepStatusByOwner(pss.Owner(), false)

		if ps != nil && ps.Grade() == icstate.Main {
			preps = append(preps, pss.ToJSON())
			sum.Add(sum, pss.BondedDelegation())
			if len(preps) == mainPRepCount {
				break
			}
		}
	}

	jso["blockHeight"] = blockHeight
	jso["totalBondedDelegation"] = sum
	jso["totalDelegated"] = sum
	jso["preps"] = preps
	return jso, nil
}

func (s *ExtensionStateImpl) GetSubPRepsInJSON(blockHeight int64) (map[string]interface{}, error) {
	term := s.State.GetTerm()
	if term == nil {
		err := errors.Errorf("Term is nil")
		return nil, err
	}

	pssCount := term.GetPRepSnapshotCount()
	mainPRepCount := int(s.State.GetMainPRepCount())

	jso := make(map[string]interface{})
	preps := make([]interface{}, 0, mainPRepCount)
	sum := new(big.Int)

	for i := mainPRepCount; i < pssCount; i++ {
		pss := term.GetPRepSnapshotByIndex(i)
		ps, _ := s.State.GetPRepStatusByOwner(pss.Owner(), false)

		if ps != nil && ps.Grade() == icstate.Sub {
			preps = append(preps, pss.ToJSON())
			sum.Add(sum, pss.BondedDelegation())
		}
	}

	jso["blockHeight"] = blockHeight
	jso["totalBondedDelegation"] = sum
	jso["totalDelegated"] = sum
	jso["preps"] = preps
	return jso, nil
}

func (s *ExtensionStateImpl) SetDelegation(cc contract.CallContext, from module.Address, ds icstate.Delegations) error {
	var err error
	var account *icstate.AccountState
	var delta map[string]*big.Int

	account = s.State.GetAccountState(from)

	using := new(big.Int).Set(ds.GetDelegationAmount())
	using.Add(using, account.Unbond())
	using.Add(using, account.Bond())
	if account.Stake().Cmp(using) < 0 {
		return icmodule.IllegalArgumentError.Errorf("Not enough voting power")
	}
	delta, err = s.pm.ChangeDelegation(account.Delegations(), ds)
	if err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to change delegation")
	}

	if err = s.addEventDelegation(cc.BlockHeight(), from, delta); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventDelegation")
	}

	account.SetDelegation(ds)
	return nil
}

func deltaToVotes(delta map[string]*big.Int) (votes icstage.VoteList, err error) {
	size := len(delta)
	keys := make([]string, 0, size)
	for key := range delta {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	votes = make([]*icstage.Vote, size)
	for i, key := range keys {
		var addr *common.Address
		addr, err = common.NewAddress([]byte(key))
		if err != nil {
			return
		}
		votes[i] = icstage.NewVote(addr, delta[key])
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

func (s *ExtensionStateImpl) addEventEnable(blockHeight int64, from module.Address, flag icstage.EnableStatus) (err error) {
	term := s.State.GetTerm()
	_, err = s.Front.AddEventEnable(
		int(blockHeight-term.StartHeight()),
		from,
		flag,
	)
	return
}

func (s *ExtensionStateImpl) addBlockProduce(wc state.WorldContext) (err error) {
	var global icstage.Global
	var voters []module.Address

	global, err = s.Front.GetGlobal()
	if err != nil || global == nil {
		return
	}
	if global.GetIISSVersion() != icstate.IISSVersion2 {
		// Only IISS 2.0 support Block Produce Reward
		return
	}
	term := s.State.GetTerm()
	blockHeight := wc.BlockHeight()
	if blockHeight < term.GetVoteStartHeight() {
		return
	}

	csi := wc.ConsensusInfo()
	// if PrepManager is not ready, it returns immediately
	proposer := s.State.GetOwnerByNode(csi.Proposer())
	if proposer == nil {
		return
	}
	_, voters, err = CompileVoters(s.State, csi)
	if err != nil || voters == nil {
		return
	}
	if err = s.Front.AddBlockProduce(wc.BlockHeight(), proposer, voters); err != nil {
		return
	}
	return
}

func (s *ExtensionStateImpl) UnregisterPRep(cc contract.CallContext, owner module.Address) error {
	var err error

	if err = s.State.DisablePRep(owner, icstate.Unregistered); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(err, "Failed to unregister P-Rep %s", owner)
	}

	err = s.addEventEnable(cc.BlockHeight(), owner, icstage.ESDisablePermanent)
	if err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventEnable")
	}

	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepUnregistered(Address)")},
		[][]byte{owner.Bytes()},
	)

	return err
}

func (s *ExtensionStateImpl) DisqualifyPRep(owner module.Address) error {
	// TODO: add PRepDisqualified eventlog
	return s.State.DisablePRep(owner, icstate.Disqualified)
}

func (s *ExtensionStateImpl) SetBond(cc contract.CallContext, from module.Address, bonds icstate.Bonds) error {
	s.logger.Tracef("SetBond() start: from=%s bonds=%+v", from, bonds)

	var err error
	var account *icstate.AccountState
	blockHeight := cc.BlockHeight()

	account = s.State.GetAccountState(from)

	bondAmount := big.NewInt(0)
	for _, bond := range bonds {
		bondAmount.Add(bondAmount, bond.Amount())

		pb, _ := s.State.GetPRepBaseByOwner(bond.To(), false)
		if pb == nil {
			return scoreresult.InvalidParameterError.Errorf("PRep not found: %v", from)
		}
		if !pb.BonderList().Contains(from) {
			return scoreresult.InvalidParameterError.Errorf("%s is not in bonder List of %s", from, bond.To())
		}
	}
	if account.Stake().Cmp(new(big.Int).Add(bondAmount, account.Delegating())) == -1 {
		return icmodule.IllegalArgumentError.Errorf("Not enough voting power")
	}

	var delta map[string]*big.Int
	delta, err = s.pm.ChangeBond(account.Bonds(), bonds)
	if err != nil {
		return icmodule.IllegalArgumentError.Wrapf(err, "Failed to change bond")
	}

	account.SetBonds(bonds)
	unbondingHeight := s.State.GetUnbondingPeriodMultiplier()*s.State.GetTermPeriod() + blockHeight
	tl, err := account.UpdateUnbonds(delta, unbondingHeight)
	if err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to update unbonds")
	}
	unbondingCount := len(account.Unbonds())
	if unbondingCount > int(s.State.GetUnbondingMax()) {
		return icmodule.IllegalArgumentError.Errorf("Too many unbonds %d", unbondingCount)
	}
	if account.Stake().Cmp(account.UsingStake()) == -1 {
		return icmodule.IllegalArgumentError.Errorf("Not enough voting power")
	}
	for _, timerJobInfo := range tl {
		unbondingTimer := s.State.GetUnbondingTimerState(timerJobInfo.Height)
		if unbondingTimer == nil {
			panic(errors.Errorf("There is no timer"))
		}
		if err = icstate.ScheduleTimerJob(unbondingTimer, timerJobInfo, from); err != nil {
			return scoreresult.UnknownFailureError.Errorf("Error while scheduling Unbonding Timer Job")
		}
	}

	if err = s.AddEventBond(blockHeight, from, delta); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventBond")
	}

	s.logger.Tracef("SetBond() end")
	return nil
}

func (s *ExtensionStateImpl) AddEventBond(blockHeight int64, from module.Address, delta map[string]*big.Int) (err error) {
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
	s.logger.Tracef("SetBonderList() start: from=%s bl=%s", from, bl)

	pb, _ := s.State.GetPRepBaseByOwner(from, false)
	if pb == nil {
		return scoreresult.InvalidParameterError.Errorf("PRep not found: %v", from)
	}

	var account *icstate.AccountState
	for _, old := range pb.BonderList() {
		if !bl.Contains(old) {
			account = s.State.GetAccountState(old)
			if len(account.Bonds()) > 0 || len(account.Unbonds()) > 0 {
				return scoreresult.InvalidParameterError.Errorf("Bonding/Unbonding exist. bonds : %d, unbonds : %d",
					len(account.Bonds()), len(account.Unbonds()))
			}
		}
	}

	pb.SetBonderList(bl)
	s.logger.Tracef("SetBonderList() end")
	return nil
}

func (s *ExtensionStateImpl) GetBonderList(address module.Address) (map[string]interface{}, error) {
	pb, _ := s.State.GetPRepBaseByOwner(address, false)
	if pb == nil {
		return nil, errors.Errorf("PRep not found: %v", address)
	}
	jso := make(map[string]interface{})
	jso["bonderList"] = pb.GetBonderListInJSON()
	return jso, nil
}

func (s *ExtensionStateImpl) SetGovernanceVariables(from module.Address, irep *big.Int, blockHeight int64) error {
	pb, _ := s.State.GetPRepBaseByOwner(from, false)
	if pb == nil {
		return errors.Errorf("PRep not found: %v", from)
	}
	if err := s.ValidateIRep(pb.IRep(), irep, pb.IRepHeight()); err != nil {
		return err
	}

	pb.SetIrep(irep, blockHeight)
	return nil
}

const IrepInflationLimit = 14 // 14%

func (s *ExtensionStateImpl) ValidateIRep(oldIRep, newIRep *big.Int, prevSetIRepHeight int64) error {
	term := s.State.GetTerm()
	if prevSetIRepHeight >= term.StartHeight() {
		return errors.Errorf("IRep can be changed only once during a term")
	}
	if err := icutils.ValidateRange(oldIRep, newIRep, 20, 20); err != nil {
		return err
	}

	/* annual amount of beta1 + beta2 <= totalSupply * IrepInflationLimit / 100
	annual amount of beta1 + beta2
	= (1/2 * irep * MainPRepCount + 1/2 * irep * VotedRewardMultiplier) * MonthPerYear
	= irep * (MAIN_PREP_COUNT + VotedRewardMultiplier) * MonthPerBlock / 2
	<= totalSupply * IrepInflationLimit / 100
	irep <= totalSupply * IrepInflationLimit * 2 / (100 * MonthBlock * (MAIN_PREP_COUNT + PERCENTAGE_FOR_BETA_2))
	*/
	limit := new(big.Int).Mul(term.TotalSupply(), new(big.Int).SetInt64(IrepInflationLimit*2))
	divider := new(big.Int).SetInt64(int64(100 * MonthPerYear * (term.MainPRepCount() + VotedRewardMultiplier)))
	limit.Div(limit, divider)
	if newIRep.Cmp(limit) == 1 {
		return errors.Errorf("IRep is out of range: %v > %v", newIRep, limit)
	}
	return nil
}

func (s *ExtensionStateImpl) UpdateIssueInfo(reward *big.Int) error {
	is, err := s.State.GetIssue()
	issue := is.Clone()
	if err != nil {
		return err
	}

	RegulateIssueInfo(issue, reward)

	issue.ResetTotalIssued()

	if err = s.State.SetIssue(issue); err != nil {
		return err
	}
	return nil
}

func (s *ExtensionStateImpl) UpdateIssueInfoFee(fee *big.Int) error {
	is, err := s.State.GetIssue()
	if err != nil {
		return err
	}
	issue := is.Clone()
	issue.SetPrevBlockFee(new(big.Int).Add(issue.PrevBlockFee(), fee))
	if err = s.State.SetIssue(issue); err != nil {
		return err
	}
	return nil
}

func (s *ExtensionStateImpl) OnExecutionBegin(wc state.WorldContext) error {
	term := s.State.GetTerm()
	if term.IsDecentralized() {
		if err := s.addBlockProduce(wc); err != nil {
			return err
		}
	}
	if wc.BlockHeight() == term.StartHeight() {
		if err := s.newCalculation(); err != nil {
			return err
		}
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
	var isTermEnd bool

	switch blockHeight {
	case term.GetEndHeight():
		if err = s.onTermEnd(wc); err != nil {
			return err
		}
		isTermEnd = true
	case term.StartHeight():
		if err = s.applyCalculationResult(calculator); err != nil {
			return err
		}
	}

	if err = s.updateValidators(wc, isTermEnd); err != nil {
		return err
	}
	s.logger.Tracef("bh=%d", blockHeight)

	if err = s.Front.ResetEventSize(); err != nil {
		return err
	}
	return nil
}

func (s *ExtensionStateImpl) onTermEnd(wc state.WorldContext) error {
	var err error
	var totalSupply *big.Int
	var preps *icstate.PReps

	revision := wc.Revision().Value()
	mainPRepCount := int(s.State.GetMainPRepCount())
	subPRepCount := int(s.State.GetSubPRepCount())
	electedPRepCount := mainPRepCount + subPRepCount

	totalSupply, err = s.getTotalSupply(wc)
	if err != nil {
		return err
	}

	isDecentralized := s.IsDecentralized()
	if !isDecentralized {
		// After decentralization is finished, this code will not be reached
		if preps, err = s.State.GetOrderedPReps(); err != nil {
			return err
		}
		isDecentralized = s.State.IsDecentralizationConditionMet(revision, totalSupply, preps)
	}

	if isDecentralized {
		if preps == nil {
			if preps, err = s.State.GetOrderedPReps(); err != nil {
				return err
			}
		}
		// Reset the status of all active preps ordered by bondedDelegation
		if err = preps.ResetAllStatus(mainPRepCount, subPRepCount, wc.BlockHeight()); err != nil {
			return err
		}
	} else {
		preps = nil
	}

	return s.moveOnToNextTerm(preps, totalSupply, revision, electedPRepCount)
}

func (s *ExtensionStateImpl) moveOnToNextTerm(
	preps *icstate.PReps, totalSupply *big.Int, revision int, electedPRepCount int) error {

	// Create a new term
	nextTerm := icstate.NewNextTerm(s.State, totalSupply, revision)

	// Valid preps means that decentralization is activated
	if preps != nil {
		br := s.GetBondRequirement()
		mainPRepCount := preps.GetPRepSize(icstate.Main)
		pss := icstate.NewPRepSnapshots(preps, electedPRepCount, br)

		nextTerm.SetMainPRepCount(mainPRepCount)
		nextTerm.SetPRepSnapshots(pss)
		nextTerm.SetIsDecentralized(true)

		if irep := s.pm.CalculateIRep(preps, revision); irep != nil {
			nextTerm.SetIrep(irep)
		}

		// Record new validator list for the next term to State
		vss := icstate.NewValidatorsSnapshotWithPRepSnapshot(pss, s.State, mainPRepCount)
		if err := s.State.SetValidatorsSnapshot(vss); err != nil {
			return err
		}
	}

	rrep := s.pm.CalculateRRep(totalSupply, revision, s.State.GetTotalDelegation())
	if rrep != nil {
		nextTerm.SetRrep(rrep)
	}

	term := s.State.GetTerm()
	if !term.IsDecentralized() && nextTerm.IsDecentralized() {
		// reset sequence when network is decentralized
		nextTerm.ResetSequence()
	}

	s.logger.Debugf(nextTerm.String())
	return s.State.SetTerm(nextTerm)
}

func (s *ExtensionStateImpl) GenesisTerm(blockHeight int64, revision int) error {
	if revision >= icmodule.RevisionIISS && s.State.GetTerm() == nil {
		term := icstate.GenesisTerm(s.State, blockHeight+1, revision)
		if err := s.State.SetTerm(term); err != nil {
			return err
		}
	}
	return nil
}

// updateValidators set a new validator set to world context
func (s *ExtensionStateImpl) updateValidators(wc state.WorldContext, isTermEnd bool) error {
	vss := s.State.GetValidatorsSnapshot()
	if vss == nil {
		return nil
	}

	if !isTermEnd {
		hash := wc.GetValidatorState().GetSnapshot().Hash()
		if bytes.Compare(vss.Hash(), hash) == 0 {
			// ValidatorList is not changed during a term
			return nil
		}
	}

	newValidators := vss.NewValidatorSet()
	err := wc.GetValidatorState().Set(newValidators)
	s.logNewValidators(wc.BlockHeight(), newValidators)
	return err
}

func (s *ExtensionStateImpl) logNewValidators(blockHeight int64, vs []module.Validator) {
	var b strings.Builder
	b.WriteString("New validators: ")
	b.WriteString(fmt.Sprintf("bh=%d cnt=%d", blockHeight, len(vs)))

	for _, v := range vs {
		b.WriteString(fmt.Sprintf(" %s", v.Address()))
	}
	s.logger.Debugf(b.String())
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
	term := s.State.GetTerm()
	return term != nil && term.IsDecentralized()
}
