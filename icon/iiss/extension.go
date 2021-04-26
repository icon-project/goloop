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
	"github.com/icon-project/goloop/service/state"
)

type ExtensionSnapshotImpl struct {
	database db.Database

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
	esLogger := log.WithFields(log.Fields{
		log.FieldKeyModule: "ICON",
	})

	es := &ExtensionStateImpl{
		database: s.database,
		logger:   esLogger,
		State:    icstate.NewStateFromSnapshot(s.state, readonly),
		Front:    icstage.NewStateFromSnapshot(s.front),
		Back:     icstage.NewStateFromSnapshot(s.back),
		Reward:   icreward.NewStateFromSnapshot(s.reward),
	}

	pm := newPRepManager(es.State, esLogger)
	term := es.State.GetTerm()

	vm := NewValidatorManager()
	if err := vm.Init(pm, term); err != nil {
		log.Errorf(err.Error())
	}

	es.pm = pm
	es.vm = vm
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

func NewExtensionSnapshotWithBuilder(builder merkle.Builder, raw []byte) state.ExtensionSnapshot {
	var hashes [4][]byte
	if _, err := codec.BC.UnmarshalFromBytes(raw, &hashes); err != nil {
		return nil
	}
	return &ExtensionSnapshotImpl{
		database: builder.Database(),
		state:    icstate.NewSnapshotWithBuilder(builder, hashes[0]),
		front:    icstage.NewSnapshotWithBuilder(builder, hashes[1]),
		back:     icstage.NewSnapshotWithBuilder(builder, hashes[2]),
		reward:   icreward.NewSnapshotWithBuilder(builder, hashes[3]),
	}
}

type ExtensionStateImpl struct {
	database db.Database

	pm     *PRepManager
	vm     *ValidatorManager
	logger log.Logger

	State  *icstate.State
	Front  *icstage.State
	Back   *icstage.State
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
	}

	if err = s.UpdateIssueInfo(calculator.TotalReward(), rcInfo.IsDecentralized(), rcInfo.AdditionalReward()); err != nil {
		return err
	}

	// switch icstage and write global
	s.Back = s.Front
	s.Front = icstage.NewState(s.database)
	iissVersion := s.State.GetIISSVersion()
	switch iissVersion {
	case icstate.IISSVersion1:
		if err = s.Back.AddGlobalV1(
			term.Revision(),
			term.StartHeight(),
			int(term.Period()-1),
			term.Irep(),
			term.Rrep(),
			term.MainPRepCount(),
			term.GetElectedPRepCount(),
		); err != nil {
			return err
		}
	case icstate.IISSVersion2:
		if err = s.Back.AddGlobalV2(
			term.Revision(),
			term.StartHeight(),
			int(term.Period()-1),
			term.Iglobal(),
			term.Iprep(),
			term.Ivoter(),
			term.GetElectedPRepCount(),
			term.BondRequirement(),
		); err != nil {
			return err
		}
	default:
		return errors.CriticalFormatError.Errorf(
			"InvalidIISSVersion(version=%d)", iissVersion)
	}

	// update rewardCalcInfo
	additionalReward := new(big.Int)
	if iissVersion == icstate.IISSVersion2 {
		rewardCPS := new(big.Int).Mul(term.Iglobal(), term.Icps())
		rewardCPS.Div(rewardCPS, big.NewInt(100))
		rewardRelay := new(big.Int).Mul(term.Iglobal(), term.Irelay())
		rewardRelay.Div(rewardCPS, big.NewInt(100))
		additionalReward.Add(rewardCPS, rewardRelay)
	}
	rcInfo.Start(term.StartHeight(), term.Period(), term.IsDecentralized(), calculator.TotalReward(), additionalReward)
	if err = s.State.SetRewardCalcInfo(rcInfo); err != nil {
		return err
	}

	return nil
}

func (s *ExtensionStateImpl) GetPRepManagerInJSON() map[string]interface{} {
	totalStake := s.State.GetTotalStake()
	return s.pm.ToJSON(totalStake)
}

func (s *ExtensionStateImpl) GetPRepsInJSON(blockHeight int64, start, end int) (map[string]interface{}, error) {
	jso, err := s.pm.GetPRepsInJSON(blockHeight, start, end)
	if err != nil {
		return nil, err
	}

	jso["totalStake"] = s.State.GetTotalStake()
	jso["blockHeight"] = blockHeight
	return jso, nil
}

func (s *ExtensionStateImpl) GetPRepInJSON(address module.Address, blockHeight int64) (map[string]interface{}, error) {
	prep := s.pm.GetPRepByOwner(address)
	if prep == nil {
		return nil, errors.Errorf("PRep not found: %s", address)
	}

	return prep.ToJSON(blockHeight, s.GetBondRequirement()), nil
}

func (s *ExtensionStateImpl) GetBondRequirement() int64 {
	if s.State.GetIISSVersion() < icstate.IISSVersion2 {
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
	mainPRepCount := s.pm.GetPRepSize(icstate.Main)
	jso := make(map[string]interface{})
	preps := make([]interface{}, 0, mainPRepCount)
	sum := new(big.Int)

	for i := 0; i < pssCount; i++ {
		pss := term.GetPRepSnapshotByIndex(i)
		prep := s.pm.GetPRepByOwner(pss.Owner())

		if prep != nil && prep.Grade() == icstate.Main {
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
	mainPRepCount := s.pm.GetPRepSize(icstate.Main)

	jso := make(map[string]interface{}, 2)
	preps := make([]interface{}, 0, mainPRepCount)
	sum := new(big.Int)

	for i := mainPRepCount; i < pssCount; i++ {
		pss := term.GetPRepSnapshotByIndex(i)
		prep := s.pm.GetPRepByOwner(pss.Owner())

		if prep != nil && prep.Grade() == icstate.Sub {
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

func (s *ExtensionStateImpl) GetTotalDelegated() *big.Int {
	return s.pm.TotalDelegated()
}

func (s *ExtensionStateImpl) RegisterPRep(regInfo *RegInfo, irep *big.Int) error {
	s.logger.Tracef("RegisterPRep() start: regInfo=[%s] irep=%s", regInfo, irep)
	err := s.pm.RegisterPRep(regInfo, irep)
	if err != nil {
		s.logger.Infof(err.Error())
	}
	s.logger.Tracef("RegisterPRep() end")
	return err
}

func (s *ExtensionStateImpl) SetDelegation(cc contract.CallContext, from module.Address, ds icstate.Delegations) error {
	var err error
	var account *icstate.Account
	var delta map[string]*big.Int

	account = s.State.GetAccount(from)

	using := ds.GetDelegationAmount()
	using.Add(using, account.Unbond())
	using.Add(using, account.Bond())
	if account.Stake().Cmp(using) < 0 {
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
	size := len(delta)
	keys := make([]string, 0, size)
	for key := range delta {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	votes = make([]*icstage.Vote, size, size)
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

func (s *ExtensionStateImpl) addEventEnable(blockHeight int64, from module.Address, flag icstage.EnableFlag) (err error) {
	term := s.State.GetTerm()
	_, err = s.Front.AddEventEnable(
		int(blockHeight-term.StartHeight()),
		from,
		flag,
	)
	return
}

func (s *ExtensionStateImpl) UnregisterPRep(cc contract.CallContext, owner module.Address) error {
	var err error
	prep := s.pm.GetPRepByOwner(owner)
	if prep == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}

	if err = s.pm.UnregisterPRep(owner); err != nil {
		return err
	}

	if s.IsDecentralized() && prep.Grade() == icstate.Main {
		if err = s.replaceValidator(owner); err != nil {
			return err
		}
	}

	s.addEventEnable(cc.BlockHeight(), owner, icstage.EfDisablePermanent)

	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepUnregistered(Address)")},
		[][]byte{owner.Bytes()},
	)

	return err
}

func (s *ExtensionStateImpl) DisqualifyPRep(owner module.Address) error {
	return s.pm.DisqualifyPRep(owner)
}

// replaceValidator replaces a penalized or inactive validator with a new validator from subPReps
func (s *ExtensionStateImpl) replaceValidator(owner module.Address) error {
	var err error
	prep := s.pm.GetPRepByOwner(owner)
	if prep == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}
	if err = s.vm.Remove(prep.GetNode()); err != nil {
		return err
	}
	return s.selectNewValidator()
}

// selectNewValidator chooses a new validator from sub PReps ordered by BondedDelegation
func (s *ExtensionStateImpl) selectNewValidator() error {
	var err error
	var prep *PRep
	term := s.State.GetTerm()
	pssCount := term.GetPRepSnapshotCount()

	loop := true
	i := s.vm.PRepSnapshotIndex()
	for ; loop && i < pssCount; i++ {
		pss := term.GetPRepSnapshotByIndex(i)
		prep = s.pm.GetPRepByOwner(pss.Owner())
		if prep != nil {
			switch prep.Grade() {
			case icstate.Main:
				panic(errors.Errorf("Invalid validator management"))
			case icstate.Sub:
				if err = s.pm.ChangeGrade(pss.Owner(), icstate.Main); err != nil {
					return err
				}
				if err = s.vm.Add(prep.GetNode()); err != nil {
					return err
				}
				// Break the loop
				loop = false
			}
		}
	}
	s.vm.SetPRepSnapshotIndex(i + 1)
	return nil
}

func (s *ExtensionStateImpl) SetPRep(regInfo *RegInfo) error {
	owner := regInfo.owner
	node := regInfo.node

	if node != nil {
		if prep := s.pm.GetPRepByOwner(owner); prep != nil {
			if err := s.vm.Replace(prep.GetNode(), node); err != nil {
				s.logger.Debugf(err.Error())
			}
		}
	}

	prepInfo := s.State.GetPRepBase(owner, false)
	if prepInfo == nil {
		return errors.Errorf("PRep Not Found: %v", owner)
	}

	regInfo.UpdateRegInfo(prepInfo)

	return s.pm.SetPRep(regInfo)
}

func (s *ExtensionStateImpl) SetBond(cc contract.CallContext, from module.Address, bonds icstate.Bonds) error {
	s.logger.Tracef("SetBond() start: from=%s bonds=%+v", from, bonds)

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
			return errors.Errorf("%s is not in bonder List of %s", from, bond.To())
		}
	}
	if account.Stake().Cmp(new(big.Int).Add(bondAmount, account.Delegating())) == -1 {
		return errors.Errorf("Not enough voting power")
	}

	var delta map[string]*big.Int
	delta, err = s.pm.ChangeBond(account.Bonds(), bonds)
	if err != nil {
		return err
	}

	account.SetBonds(bonds)
	unbondingHeight := s.State.GetUnbondingPeriodMultiplier()*s.State.GetTermPeriod() + blockHeight
	tl, err := account.UpdateUnbonds(delta, unbondingHeight)
	if err != nil {
		return err
	}
	unbondingCount := len(account.Unbonds())
	if unbondingCount > int(s.State.GetUnbondingMax().Int64()) {
		return errors.Errorf("Too many unbonds %d", unbondingCount)
	}
	if account.Stake().Cmp(account.UsingStake()) == -1 {
		return errors.Errorf("Not enough voting power")
	}
	for _, timerJobInfo := range tl {
		creatIfNotExist := true
		if timerJobInfo.Type == icstate.JobTypeRemove {
			creatIfNotExist = false
		}
		unbondingTimer := s.State.GetUnbondingTimer(timerJobInfo.Height, creatIfNotExist)
		if unbondingTimer == nil {
			panic(errors.Errorf("There is no timer"))
		}
		if err = icstate.ScheduleTimerJob(unbondingTimer, timerJobInfo, from); err != nil {
			return errors.Errorf("Error while scheduling Unbonding Timer Job")
		}
	}

	if err = s.AddEventBond(blockHeight, from, delta); err != nil {
		return err
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
	s.logger.Tracef("SetBonderList() end")
	return nil
}

func (s *ExtensionStateImpl) GetBonderList(address module.Address) (map[string]interface{}, error) {
	pb := s.State.GetPRepBase(address, false)
	if pb == nil {
		return nil, errors.Errorf("PRep not found: %v", address)
	}
	jso := make(map[string]interface{})
	jso["bonderList"] = pb.GetBonderListInJSON()
	return jso, nil
}

func (s *ExtensionStateImpl) SetGovernanceVariables(from module.Address, irep *big.Int, blockHeight int64) error {
	pb := s.State.GetPRepBase(from, false)
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
		return errors.Errorf("IRep is out of range: %s > %s", newIRep.String(), limit.String())
	}
	return nil
}

func (s *ExtensionStateImpl) UpdateIssueInfo(reward *big.Int, isDecentralized bool, additionalReward *big.Int) error {
	is, err := s.State.GetIssue()
	issue := is.Clone()
	if err != nil {
		return err
	}

	RegulateIssueInfo(issue, reward, additionalReward)

	issue.ResetTotalIssued()

	if err = s.State.SetIssue(issue); err != nil {
		return err
	}
	return nil
}

func (s *ExtensionStateImpl) UpdateIssueInfoFee(fee *big.Int) error {
	if !s.IsDecentralized() {
		return nil
	}
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
		if err = s.onTermEnd(wc); err != nil {
			return err
		}
	}

	if err = s.updateValidators(wc); err != nil {
		return err
	}
	s.logger.Tracef("bh=%d %s", blockHeight, s.vm)

	if err = s.Front.ResetEventSize(); err != nil {
		return err
	}
	return nil
}

func (s *ExtensionStateImpl) onTermEnd(wc state.WorldContext) error {
	var err error
	var totalSupply *big.Int
	mainPRepCount := int(s.State.GetMainPRepCount())
	subPRepCount := int(s.State.GetSubPRepCount())
	isDecentralized := s.IsDecentralized() || s.IsDecentralizationTriggered(wc.Revision().Value())

	if isDecentralized {
		// Assign grades to PReps ordered by bondedDelegation
		if err = s.pm.OnTermEnd(mainPRepCount, subPRepCount, wc.BlockHeight()); err != nil {
			return err
		}
	}

	totalSupply, err = s.getTotalSupply(wc)
	if err != nil {
		return err
	}
	if err = s.moveOnToNextTerm(totalSupply, wc.Revision().Value()); err != nil {
		return err
	}

	if isDecentralized {
		if err = s.vm.Clear(); err != nil {
			return err
		}
		if err = s.vm.Load(s.pm, s.State.GetTerm()); err != nil {
			return err
		}
		s.vm.SetUpdated(true)
	}
	return nil
}

func (s *ExtensionStateImpl) moveOnToNextTerm(totalSupply *big.Int, revision int) error {
	term := s.State.GetTerm()
	nextTerm := icstate.NewNextTerm(
		term,
		s.State.GetTermPeriod(),
		s.State.GetIRep(),
		s.State.GetRRep(),
		totalSupply,
		s.pm.TotalDelegated(),
		s.State.GetRewardFund(),
		int(s.GetBondRequirement()),
		revision,
	)

	// Take prep snapshots only if mainPReps exist
	if revision >= icmodule.RevisionDecentralize {
		mainPRepCount := 0
		if s.pm.GetPRepSize(icstate.Main) > 0 {
			size := icutils.Min(s.pm.Size(), int(s.State.GetPRepCount()))
			if size > 0 {
				prepSnapshots := make(icstate.PRepSnapshots, size, size)
				br := s.GetBondRequirement()
				for i := 0; i < size; i++ {
					prep := s.pm.GetPRepByIndex(i)
					prepSnapshots[i] = icstate.NewPRepSnapshotFromPRepStatus(prep.PRepStatus, br)
					if prep.Grade() == icstate.Main {
						mainPRepCount++
					}
				}

				nextTerm.SetMainPRepCount(mainPRepCount)
				nextTerm.SetPRepSnapshots(prepSnapshots)
			}
		}
		irep := s.pm.CalculateIRep(revision)
		if irep != nil {
			nextTerm.SetIrep(irep)
		}
	}
	rrep := s.pm.CalculateRRep(totalSupply, revision)
	if rrep != nil {
		nextTerm.SetRrep(rrep)
	}
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

func (s *ExtensionStateImpl) updateValidators(wc state.WorldContext) error {
	vm := s.vm
	if !vm.IsUpdated() {
		return nil
	}

	vs, err := vm.GetValidators()
	if err != nil {
		return err
	}

	for it := vm.Iterator(); it.Has(); it.Next() {
		vi, _ := it.Get()
		if vi == nil {
			break
		}
		if vi.IsAdded() {
			if err = s.pm.ShiftVPenaltyMaskByNode(vi.Address()); err != nil {
				return nil
			}
			vi.ResetFlags()
		}
	}

	if err = wc.GetValidatorState().Set(vs); err != nil {
		return err
	}

	s.logNewValidators(wc.BlockHeight(), vs)
	vm.ResetUpdated()
	return nil
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
	if term == nil {
		return false
	}
	return term.IsDecentralized()
}

func (s *ExtensionStateImpl) IsDecentralizationTriggered(revision int) bool {
	if revision >= icmodule.RevisionDecentralize && s.pm.Size() >= int(s.State.GetMainPRepCount()) {
		// TODO check delegated amount of 22nd P-Reps
		return true
	}
	return false
}

func (s *ExtensionStateImpl) GetPRepStatsInJSON(blockHeight int64) (map[string]interface{}, error) {
	return s.pm.GetPRepStatsInJSON(blockHeight)
}
