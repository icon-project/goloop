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
	"math/big"
	"sort"

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

func (s *ExtensionSnapshotImpl) Back2() *icstage.Snapshot {
	return s.back2
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

func (es *ExtensionStateImpl) Logger() log.Logger {
	return es.logger
}

func (es *ExtensionStateImpl) SetLogger(logger log.Logger) {
	if logger != nil {
		es.logger = logger
	}
}

func (es *ExtensionStateImpl) GetSnapshot() state.ExtensionSnapshot {
	return &ExtensionSnapshotImpl{
		database: es.database,
		state:    es.State.GetSnapshot(),
		front:    es.Front.GetSnapshot(),
		back1:    es.Back1.GetSnapshot(),
		back2:    es.Back2.GetSnapshot(),
		reward:   es.Reward.GetSnapshot(),
	}
}

func (es *ExtensionStateImpl) Reset(isnapshot state.ExtensionSnapshot) {
	snapshot := isnapshot.(*ExtensionSnapshotImpl)
	if err := es.State.Reset(snapshot.state); err != nil {
		panic(err)
	}
	es.Front.Reset(snapshot.front)
	es.Back1.Reset(snapshot.back1)
	es.Back2.Reset(snapshot.back2)
	es.Reward.Reset(snapshot.reward)
}

// ClearCache clear cache. It's called before executing first transaction
// and also it could be called at the end of base transaction
func (es *ExtensionStateImpl) ClearCache() {
	es.State.ClearCache()
	es.Front.ClearCache()
}

func (es *ExtensionStateImpl) CalculationBlockHeight() int64 {
	rcInfo, err := es.State.GetRewardCalcInfo()
	if err != nil || rcInfo == nil {
		return 0
	}
	return rcInfo.StartHeight()
}

func (es *ExtensionStateImpl) setNewFront() (err error) {
	term := es.State.GetTermSnapshot()

	// switch icstage values
	es.Back1 = es.Front
	es.Front = icstage.NewState(es.database)

	// write icstage.Global to Front
	iissVersion := term.GetIISSVersion()
	switch iissVersion {
	case icstate.IISSVersion2:
		if err = es.Front.AddGlobalV1(
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
		if err = es.Front.AddGlobalV2(
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

func (es *ExtensionStateImpl) GetPRepInJSON(address module.Address, blockHeight int64) (map[string]interface{}, error) {
	prep := es.State.GetPRepByOwner(address)
	if prep == nil {
		return nil, errors.Errorf("PRep not found: %s", address)
	}
	return prep.ToJSON(blockHeight, es.State.GetBondRequirement()), nil
}

func (es *ExtensionStateImpl) GetMainPRepsInJSON(blockHeight int64) (map[string]interface{}, error) {
	term := es.State.GetTermSnapshot()
	if term == nil {
		err := errors.Errorf("Term is nil")
		return nil, err
	}

	pssCount := term.GetPRepSnapshotCount()
	mainPRepCount := term.MainPRepCount()
	jso := make(map[string]interface{})
	preps := make([]interface{}, 0, mainPRepCount)
	sum := new(big.Int)

	for i := 0; i < pssCount; i++ {
		pss := term.GetPRepSnapshotByIndex(i)
		ps, _ := es.State.GetPRepStatusByOwner(pss.Owner(), false)
		pb, _ := es.State.GetPRepBaseByOwner(pss.Owner(), false)

		if ps != nil && ps.Grade() == icstate.GradeMain {
			pj := pss.ToJSON()
			pj["name"] = pb.Name()
			preps = append(preps, pj)
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

func (es *ExtensionStateImpl) GetSubPRepsInJSON(blockHeight int64) (map[string]interface{}, error) {
	term := es.State.GetTermSnapshot()
	if term == nil {
		err := errors.Errorf("Term is nil")
		return nil, err
	}

	pssCount := term.GetPRepSnapshotCount()
	mainPRepCount := term.MainPRepCount()
	subPRepCount := term.GetElectedPRepCount() - mainPRepCount

	jso := make(map[string]interface{})
	preps := make([]interface{}, 0, subPRepCount)
	sum := new(big.Int)

	for i := mainPRepCount; i < pssCount; i++ {
		pss := term.GetPRepSnapshotByIndex(i)
		ps, _ := es.State.GetPRepStatusByOwner(pss.Owner(), false)
		pb, _ := es.State.GetPRepBaseByOwner(pss.Owner(), false)

		if ps != nil && ps.Grade() == icstate.GradeSub {
			pj := pss.ToJSON()
			pj["name"] = pb.Name()
			preps = append(preps, pj)
			sum.Add(sum, pss.BondedDelegation())
		}
	}

	jso["blockHeight"] = blockHeight
	jso["totalBondedDelegation"] = sum
	jso["totalDelegated"] = sum
	jso["preps"] = preps
	return jso, nil
}

func (es *ExtensionStateImpl) SetDelegation(blockHeight int64, from module.Address, ds icstate.Delegations) error {
	var err error
	var account *icstate.AccountState
	var delta map[string]*big.Int

	account = es.State.GetAccountState(from)

	using := new(big.Int).Set(ds.GetDelegationAmount())
	using.Add(using, account.Unbond())
	using.Add(using, account.Bond())
	if account.Stake().Cmp(using) < 0 {
		return icmodule.IllegalArgumentError.Errorf("Not enough voting power")
	}
	delta, err = es.pm.ChangeDelegation(account.Delegations(), ds)
	if err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to change delegation")
	}

	if err = es.addEventDelegation(blockHeight, from, delta); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventDelegation")
	}

	account.SetDelegation(ds)
	return nil
}

func deltaToVotes(delta map[string]*big.Int) (votes icstage.VoteList, err error) {
	keys := make([]string, 0)
	for key, value := range delta {
		if value.Sign() == 0 {
			// skip zero-valued
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	size := len(keys)
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

func (es *ExtensionStateImpl) addEventDelegation(blockHeight int64, from module.Address, delta map[string]*big.Int) (err error) {
	votes, err := deltaToVotes(delta)
	if err != nil {
		return
	}
	term := es.State.GetTermSnapshot()
	_, err = es.Front.AddEventDelegation(
		int(blockHeight-term.StartHeight()),
		from,
		votes,
	)
	return
}

func (es *ExtensionStateImpl) addEventEnable(blockHeight int64, from module.Address, flag icstage.EnableStatus) (err error) {
	term := es.State.GetTermSnapshot()
	_, err = es.Front.AddEventEnable(
		int(blockHeight-term.StartHeight()),
		from,
		flag,
	)
	return
}

func (es *ExtensionStateImpl) addBlockProduce(wc state.WorldContext) (err error) {
	var global icstage.Global
	var voters []module.Address

	global, err = es.Front.GetGlobal()
	if err != nil || global == nil {
		return
	}
	if global.GetIISSVersion() != icstate.IISSVersion2 {
		// Only IISS 2.0 support Block Produce Reward
		return
	}
	term := es.State.GetTermSnapshot()
	blockHeight := wc.BlockHeight()
	if blockHeight < term.GetVoteStartHeight() {
		return
	}

	csi := wc.ConsensusInfo()
	// if PrepManager is not ready, it returns immediately
	proposer := es.State.GetOwnerByNode(csi.Proposer())
	if proposer == nil {
		return
	}
	_, voters, err = CompileVoters(es.State, csi)
	if err != nil || voters == nil {
		return
	}
	if err = es.Front.AddBlockProduce(wc.BlockHeight(), proposer, voters); err != nil {
		return
	}
	return
}

func (es *ExtensionStateImpl) UnregisterPRep(blockHeight int64, owner module.Address) error {
	var err error
	if err = es.State.DisablePRep(owner, icstate.Unregistered, blockHeight); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(err, "Failed to unregister P-Rep %s", owner)
	}
	if err = es.addEventEnable(blockHeight, owner, icstage.ESDisablePermanent); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventEnable")
	}
	return nil
}

func (es *ExtensionStateImpl) DisqualifyPRep(blockHeight int64, owner module.Address) error {
	if err := es.State.DisablePRep(owner, icstate.Disqualified, blockHeight); err != nil {
		return err
	}
	if err := es.addEventEnable(blockHeight, owner, icstage.ESDisablePermanent); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventEnable")
	}
	return nil
}

func (es *ExtensionStateImpl) SetBond(blockHeight int64, from module.Address, bonds icstate.Bonds) error {
	es.logger.Tracef("SetBond() start: from=%s bonds=%+v", from, bonds)

	var err error
	var account *icstate.AccountState
	account = es.State.GetAccountState(from)

	bondAmount := big.NewInt(0)
	for _, bond := range bonds {
		bondAmount.Add(bondAmount, bond.Amount())

		pb, _ := es.State.GetPRepBaseByOwner(bond.To(), false)
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
	delta, err = es.pm.ChangeBond(account.Bonds(), bonds)
	if err != nil {
		return icmodule.IllegalArgumentError.Wrapf(err, "Failed to change bond")
	}

	account.SetBonds(bonds)
	unbondingHeight := es.State.GetUnbondingPeriodMultiplier()*es.State.GetTermPeriod() + blockHeight
	tl, err := account.UpdateUnbonds(delta, unbondingHeight)
	if err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to update unbonds")
	}
	unbondingCount := len(account.Unbonds())
	if unbondingCount > int(es.State.GetUnbondingMax()) {
		return icmodule.IllegalArgumentError.Errorf("Too many unbonds %d", unbondingCount)
	}
	if account.Stake().Cmp(account.UsingStake()) == -1 {
		return icmodule.IllegalArgumentError.Errorf("Not enough voting power")
	}
	for _, timerJobInfo := range tl {
		unbondingTimer := es.State.GetUnbondingTimerState(timerJobInfo.Height)
		if unbondingTimer == nil {
			panic(errors.Errorf("There is no timer"))
		}
		if err = icstate.ScheduleTimerJob(unbondingTimer, timerJobInfo, from); err != nil {
			return scoreresult.UnknownFailureError.Errorf("Error while scheduling Unbonding Timer Job")
		}
	}

	if err = es.AddEventBond(blockHeight, from, delta); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventBond")
	}

	es.logger.Tracef("SetBond() end")
	return nil
}

func (es *ExtensionStateImpl) AddEventBond(blockHeight int64, from module.Address, delta map[string]*big.Int) (err error) {
	votes, err := deltaToVotes(delta)
	if err != nil {
		return
	}
	term := es.State.GetTermSnapshot()
	_, err = es.Front.AddEventBond(
		int(blockHeight-term.StartHeight()),
		from,
		votes,
	)
	return
}

func (es *ExtensionStateImpl) SetBonderList(from module.Address, bl icstate.BonderList) error {
	es.logger.Tracef("SetBonderList() start: from=%s bl=%s", from, bl)

	pb, _ := es.State.GetPRepBaseByOwner(from, false)
	if pb == nil {
		return scoreresult.InvalidParameterError.Errorf("PRep not found: %v", from)
	}

	var account *icstate.AccountState
	for _, old := range pb.BonderList() {
		if !bl.Contains(old) {
			account = es.State.GetAccountState(old)
			if len(account.Bonds()) > 0 || len(account.Unbonds()) > 0 {
				return scoreresult.InvalidParameterError.Errorf("Bonding/Unbonding exist. bonds : %d, unbonds : %d",
					len(account.Bonds()), len(account.Unbonds()))
			}
		}
	}

	pb.SetBonderList(bl)
	es.logger.Tracef("SetBonderList() end")
	return nil
}

func (es *ExtensionStateImpl) GetBonderList(address module.Address) (map[string]interface{}, error) {
	pb, _ := es.State.GetPRepBaseByOwner(address, false)
	if pb == nil {
		return nil, errors.Errorf("PRep not found: %v", address)
	}
	jso := make(map[string]interface{})
	jso["bonderList"] = pb.GetBonderListInJSON()
	return jso, nil
}

func (es *ExtensionStateImpl) SetGovernanceVariables(from module.Address, irep *big.Int, blockHeight int64) error {
	pb, _ := es.State.GetPRepBaseByOwner(from, false)
	if pb == nil {
		return scoreresult.InvalidParameterError.Errorf("PRep not found: %v", from)
	}
	if err := es.ValidateIRep(pb.IRep(), irep, pb.IRepHeight()); err != nil {
		return err
	}

	pb.SetIrep(irep, blockHeight)
	return nil
}

const IrepInflationLimit = 14 // 14%

func (es *ExtensionStateImpl) ValidateIRep(oldIRep, newIRep *big.Int, prevSetIRepHeight int64) error {
	term := es.State.GetTermSnapshot()
	if prevSetIRepHeight >= term.StartHeight() {
		return scoreresult.IllegalFormatError.Errorf("IRep can be changed only once during a term")
	}
	if newIRep.Cmp(icmodule.BigIntMinIRep) == -1 {
		return scoreresult.InvalidParameterError.Errorf("IRep is out of range. %d < %d", newIRep, icmodule.BigIntMinIRep)
	}
	if err := icutils.ValidateRange(oldIRep, newIRep, 20, 20); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(err, "IRep is out of range")
	}

	/* annual amount of beta1 + beta2 <= totalSupply * IrepInflationLimit / 100
	annual amount of beta1 + beta2
	= (1/2 * irep * MainPRepCount + 1/2 * irep * VotedRewardMultiplier) * MonthPerYear
	= irep * (MAIN_PREP_COUNT + VotedRewardMultiplier) * MonthPerBlock / 2
	<= totalSupply * IrepInflationLimit / 100
	irep <= totalSupply * IrepInflationLimit * 2 / (100 * MonthBlock * (MAIN_PREP_COUNT + PERCENTAGE_FOR_BETA_2))
	*/
	limit := new(big.Int).Mul(term.TotalSupply(), new(big.Int).SetInt64(IrepInflationLimit*2))
	divider := new(big.Int).SetInt64(int64(100 * MonthPerYear * (term.MainPRepCount() + icmodule.VotedRewardMultiplier)))
	limit.Div(limit, divider)
	if newIRep.Cmp(limit) == 1 {
		return scoreresult.InvalidParameterError.Errorf("IRep is out of range: %v > %v", newIRep, limit)
	}
	return nil
}

func (es *ExtensionStateImpl) OnExecutionBegin(wc state.WorldContext) error {
	term := es.State.GetTermSnapshot()
	if term.IsDecentralized() {
		if err := es.addBlockProduce(wc); err != nil {
			return err
		}
	}
	if wc.BlockHeight() == term.StartHeight() {
		if err := es.setNewFront(); err != nil {
			return err
		}
	}
	return nil
}

func (es *ExtensionStateImpl) OnExecutionEnd(wc state.WorldContext, totalFee *big.Int, calculator *Calculator) error {
	var err error
	term := es.State.GetTermSnapshot()
	if term == nil {
		return nil
	}

	if term.IsDecentralized() {
		if err = es.setIssuePrevBlockFee(totalFee); err != nil {
			return err
		}
	}

	blockHeight := wc.BlockHeight()
	var isTermEnd bool

	switch blockHeight {
	case term.GetEndHeight() - 1:
		if err := es.checkCalculationDone(calculator); err != nil {
			return err
		}
		if err := es.regulateIssue(calculator.TotalReward()); err != nil {
			return err
		}
	case term.GetEndHeight():
		if err = es.onTermEnd(wc); err != nil {
			return err
		}
		isTermEnd = true

		nTerm := es.State.GetTermSnapshot()
		if term.IsDecentralized() {
			if err := es.resetIssueTotalReward(); err != nil {
				return err
			}
		} else if nTerm.IsDecentralized() {
			// last centralized block
			if err := es.setIssuePrevBlockFee(totalFee); err != nil {
				return err
			}
		}
	case term.StartHeight():
		if err = es.applyCalculationResult(calculator, blockHeight); err != nil {
			return err
		}
	}

	if err = es.updateValidators(wc, isTermEnd); err != nil {
		return err
	}
	es.logger.Tracef("bh=%d", blockHeight)

	if err = es.Front.ResetEventSize(); err != nil {
		return err
	}
	return nil
}

func (es *ExtensionStateImpl) checkCalculationDone(calculator *Calculator) error {
	// Called at the end block of Term and effected to base TX issue amount in ICON1
	rcInfo, err := es.State.GetRewardCalcInfo()
	if err != nil {
		return err
	}

	if !calculator.IsCalcDone(rcInfo.StartHeight()) {
		if err = calculator.Error(); err != nil {
			return err
		}
		return icmodule.CalculationNotFinishedError.Errorf("Calculation is not finished %d, %d",
			calculator.startHeight, rcInfo.StartHeight())
	}
	return nil
}

func (es *ExtensionStateImpl) regulateIssue(iScore *big.Int) error {
	// Update Issue with calculation result from 2nd Term of decentralization
	term := es.State.GetTermSnapshot()
	if !term.IsDecentralized() || term.Sequence() == 0 {
		return nil
	}

	prevGlobal, err := es.Back1.GetGlobal()
	if err != nil {
		return err
	}
	reward := new(big.Int).Set(iScore)
	if prevGlobal != nil && icstate.IISSVersion3 == prevGlobal.GetIISSVersion() {
		pg := prevGlobal.GetV2()
		rewardCPS := new(big.Int).Mul(pg.GetIGlobal(), pg.GetICps())
		rewardCPS.Mul(rewardCPS, big.NewInt(10)) // 10 = IScoreICXRation / 100
		reward.Add(reward, rewardCPS)
		rewardRelay := new(big.Int).Mul(pg.GetIGlobal(), pg.GetIRelay())
		rewardRelay.Mul(rewardCPS, big.NewInt(10))
		reward.Add(reward, rewardRelay)
	}

	is, err := es.State.GetIssue()
	issue := is.Clone()
	if err != nil {
		return err
	}

	RegulateIssueInfo(issue, reward)

	if err = es.State.SetIssue(issue); err != nil {
		return err
	}

	return nil
}

func (es *ExtensionStateImpl) onTermEnd(wc state.WorldContext) error {
	var err error
	var totalSupply *big.Int
	var preps icstate.PRepSet

	revision := wc.Revision().Value()
	mainPRepCount := int(es.State.GetMainPRepCount())
	subPRepCount := int(es.State.GetSubPRepCount())
	electedPRepCount := mainPRepCount + subPRepCount

	totalSupply, err = es.getTotalSupply(wc)
	if err != nil {
		return err
	}

	isDecentralized := es.IsDecentralized()
	if !isDecentralized {
		// After decentralization is finished, this code will not be reached
		if preps, err = es.State.GetPRepsOnTermEnd(revision); err != nil {
			return err
		}
		isDecentralized = es.State.IsDecentralizationConditionMet(revision, totalSupply, preps)
	}

	if isDecentralized {
		if preps == nil {
			if preps, err = es.State.GetPRepsOnTermEnd(revision); err != nil {
				return err
			}
		}
		// Reset the status of all active preps ordered by bondedDelegation
		limit := es.State.GetConsistentValidationPenaltyMask()
		if err = preps.OnTermEnd(mainPRepCount, subPRepCount, limit); err != nil {
			return err
		}
	} else {
		preps = nil
	}

	return es.moveOnToNextTerm(preps, totalSupply, revision, electedPRepCount)
}

func (es *ExtensionStateImpl) moveOnToNextTerm(
	preps icstate.PRepSet, totalSupply *big.Int, revision int, electedPRepCount int) error {

	// Create a new term
	nextTerm := icstate.NewNextTerm(es.State, totalSupply, revision)

	// Valid preps means that decentralization is activated
	if preps != nil {
		br := es.State.GetBondRequirement()
		mainPRepCount := preps.GetPRepSize(icstate.GradeMain)
		pss := preps.ToPRepSnapshots(electedPRepCount, br)

		nextTerm.SetMainPRepCount(mainPRepCount)
		nextTerm.SetPRepSnapshots(pss)
		nextTerm.SetIsDecentralized(true)

		if irep := CalculateIRep(preps, revision); irep != nil {
			nextTerm.SetIrep(irep)
		}

		// Record new validator list for the next term to State
		vss := icstate.NewValidatorsSnapshotWithPRepSnapshot(pss, es.State, mainPRepCount)
		if err := es.State.SetValidatorsSnapshot(vss); err != nil {
			return err
		}
	}

	rrep := CalculateRRep(totalSupply, revision, es.State.GetTotalDelegation())
	if rrep != nil {
		nextTerm.SetRrep(rrep)
	}

	term := es.State.GetTermSnapshot()
	if !term.IsDecentralized() && nextTerm.IsDecentralized() {
		// reset sequence when network is decentralized
		nextTerm.ResetSequence()
	}

	es.logger.Debugf(nextTerm.String())
	return es.State.SetTermSnapshot(nextTerm.GetSnapshot())
}

func (es *ExtensionStateImpl) resetIssueTotalReward() error {
	is, err := es.State.GetIssue()
	if err != nil {
		return err
	}
	issue := is.Clone()
	issue.ResetTotalReward()
	if err = es.State.SetIssue(issue); err != nil {
		return err
	}
	return nil
}

func (es *ExtensionStateImpl) setIssuePrevBlockFee(fee *big.Int) error {
	is, err := es.State.GetIssue()
	if err != nil {
		return err
	}
	issue := is.Clone()
	issue.SetPrevBlockFee(fee)
	if err = es.State.SetIssue(issue); err != nil {
		return err
	}
	return nil
}

func (es *ExtensionStateImpl) applyCalculationResult(calculator *Calculator, blockHeight int64) error {
	var resultHash []byte
	result := calculator.Result()
	reward := new(big.Int).Set(calculator.TotalReward())

	rc, err := es.State.GetRewardCalcInfo()
	rcInfo := rc.Clone()
	if err != nil {
		return err
	}

	if result != nil {
		g2, err := es.Back2.GetGlobal()
		if err != nil {
			return err
		}

		if icstate.IISSVersion3 == g2.GetIISSVersion() {
			pg := g2.GetV2()
			rewardCPS := new(big.Int).Mul(pg.GetIGlobal(), pg.GetICps())
			rewardCPS.Mul(rewardCPS, big.NewInt(10)) // 10 = IScoreICXRation / 100
			reward.Add(reward, rewardCPS)
			rewardRelay := new(big.Int).Mul(pg.GetIGlobal(), pg.GetIRelay())
			rewardRelay.Mul(rewardCPS, big.NewInt(10))
			reward.Add(reward, rewardRelay)
		}
		resultHash = result.Bytes()

		// set new reward
		es.Reward = result.NewState()
	}

	es.logger.Tracef("applyCalculationResult %d", blockHeight)
	g1, err := es.Back1.GetGlobal()
	if err != nil {
		return err
	}
	if g1 == nil {
		rcInfo.Update(blockHeight, reward, resultHash)
	} else {
		rcInfo.Update(g1.GetStartHeight(), reward, resultHash)
	}
	if err = es.State.SetRewardCalcInfo(rcInfo); err != nil {
		return err
	}

	//switch icstage back
	es.Back2 = es.Back1
	es.Back1 = icstage.NewState(es.database) // ss.Byte() nil 확인
	return nil
}

func (es *ExtensionStateImpl) GenesisTerm(blockHeight int64, revision int) error {
	if revision >= icmodule.RevisionIISS && es.State.GetTermSnapshot() == nil {
		term := icstate.GenesisTerm(es.State, blockHeight+1, revision)
		if err := es.State.SetTermSnapshot(term.GetSnapshot()); err != nil {
			return err
		}
	}
	return nil
}

// updateValidators set a new validator set to world context
func (es *ExtensionStateImpl) updateValidators(wc state.WorldContext, isTermEnd bool) error {
	var err error
	vss := es.State.GetValidatorsSnapshot()
	if vss == nil {
		return nil
	}

	blockHeight := wc.BlockHeight()
	if isTermEnd || vss.IsUpdated(blockHeight) {
		newValidators := vss.NewValidatorSet()
		err = wc.GetValidatorState().Set(newValidators)
		es.logger.Debugf("New validators: bh=%d vss=%+v", blockHeight, vss)
	}
	return err
}

func (es *ExtensionStateImpl) GetPRepTermInJSON() (map[string]interface{}, error) {
	term := es.State.GetTermSnapshot()
	if term == nil {
		err := errors.Errorf("Term is nil")
		return nil, err
	}
	return term.ToJSON(), nil
}

func (es *ExtensionStateImpl) getTotalSupply(wc state.WorldContext) (*big.Int, error) {
	ass := wc.GetAccountState(state.SystemID).GetSnapshot()
	as := scoredb.NewStateStoreWith(ass)
	tsVar := scoredb.NewVarDB(as, state.VarTotalSupply)
	if ts := tsVar.BigInt(); ts != nil {
		return ts, nil
	}
	return icmodule.BigIntZero, nil
}

func (es *ExtensionStateImpl) IsDecentralized() bool {
	term := es.State.GetTermSnapshot()
	return term != nil && term.IsDecentralized()
}
