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
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icstate/migrate"
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
	log    []ExtensionLog

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

func (es *ExtensionStateImpl) SetDelegation(
	cc icmodule.CallContext, ds icstate.Delegations) error {

	var err error
	var account *icstate.AccountState
	var delta map[string]*big.Int

	from := cc.From()
	blockHeight := cc.BlockHeight()
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

	var offset int
	var idx int64
	var obj *icobject.Object
	id := es.State.GetIllegalDelegation(from)
	if id == nil {
		offset, idx, obj, err = es.addEventDelegation(blockHeight, from, delta)
		if err != nil {
			return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventDelegation")
		}
	} else {
		delegatingDelta := id.Delegations().Delta(ds)
		offset, idx, obj, err = es.addEventDelegationV2(blockHeight, from, delta, delegatingDelta)
		if err != nil {
			return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventDelegation")
		}
	}

	revision := cc.Revision().Value()
	if revision < icmodule.RevisionFixSetDelegation {
		dLog := newDelegationLog(from, offset, idx, obj, ds)
		es.AppendExtensionLog(dLog)
	}

	account.SetDelegation(ds)
	if icmodule.RevisionMultipleUnstakes <= revision && revision < icmodule.RevisionFixInvalidUnstake {
		migrate.ReproduceUnstakeBugForDelegation(cc, es.logger)
	}
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

func (es *ExtensionStateImpl) addEventDelegation(blockHeight int64, from module.Address, delta map[string]*big.Int,
) (offset int, idx int64, obj *icobject.Object, err error) {
	votes, err := deltaToVotes(delta)
	if err != nil {
		return
	}
	term := es.State.GetTermSnapshot()
	offset = int(blockHeight - term.StartHeight())
	idx, obj, err = es.Front.AddEventDelegation(offset, from, votes)
	return
}

func (es *ExtensionStateImpl) addEventDelegationV2(
	blockHeight int64, from module.Address, delegatedDelta map[string]*big.Int, delegatingDelta map[string]*big.Int,
) (offset int, idx int64, obj *icobject.Object, err error) {
	delegated, err := deltaToVotes(delegatedDelta)
	if err != nil {
		return
	}

	delegating, err := deltaToVotes(delegatingDelta)
	if err != nil {
		return
	}
	term := es.State.GetTermSnapshot()
	offset = int(blockHeight - term.StartHeight())
	idx, obj, err = es.Front.AddEventDelegationV2(offset, from, delegated, delegating)
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

func (es *ExtensionStateImpl) UnregisterPRep(cc icmodule.CallContext) error {
	var err error
	blockHeight := cc.BlockHeight()
	owner := cc.From()

	if err = es.State.DisablePRep(owner, icstate.Unregistered, blockHeight); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(err, "Failed to unregister P-Rep %s", owner)
	}
	if err = es.addEventEnable(blockHeight, owner, icstage.ESDisablePermanent); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventEnable")
	}

	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepUnregistered(Address)")},
		[][]byte{owner.Bytes()},
	)
	return nil
}

func (es *ExtensionStateImpl) DisqualifyPRep(cc icmodule.CallContext, address module.Address) error {
	blockHeight := cc.BlockHeight()
	if err := es.State.DisablePRep(address, icstate.Disqualified, blockHeight); err != nil {
		return err
	}
	if err := es.addEventEnable(blockHeight, address, icstage.ESDisablePermanent); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventEnable")
	}
	ps, _ := es.State.GetPRepStatusByOwner(address, false)
	// Record PenaltyImposed eventlog
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PenaltyImposed(Address,int,int)"), address.Bytes()},
		[][]byte{
			intconv.Int64ToBytes(int64(ps.Status())),
			intconv.Int64ToBytes(PRepDisqualification),
		},
	)
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
		if err = es.checkCalculationDone(calculator); err != nil {
			return err
		}
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
		} else {
			irep = new(big.Int).Set(es.State.GetIRep())
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
	return term.ToJSON(es.State), nil
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

func (es *ExtensionStateImpl) AppendExtensionLog(el ExtensionLog) {
	es.Logger().Tracef("Append ExtensionLog %+v", el)
	if es.log == nil {
		es.log = make([]ExtensionLog, 0)
	}
	es.log = append(es.log, el)
}

func (es *ExtensionStateImpl) HandleExtensionLog() error {
	for _, el := range es.log {
		es.Logger().Tracef("Handle ExtensionLog %+v", el)
		if err := el.Handle(es); err != nil {
			return err
		}
	}
	es.log = nil
	return nil
}

func (es *ExtensionStateImpl) SetStake(cc icmodule.CallContext, v *big.Int) (err error) {
	from := cc.From()
	ia := es.State.GetAccountState(from)

	usingStake := ia.UsingStake()
	if v.Cmp(usingStake) < 0 {
		return scoreresult.InvalidParameterError.Errorf(
			"Failed to set stake: newStake=%v < usingStake=%v from=%v",
			v, usingStake, from,
		)
	}

	revision := cc.Revision().Value()
	stakeInc := new(big.Int).Sub(v, ia.Stake())
	// ICON1 update unstakes when stakeInc == 0
	if stakeInc.Sign() == 0 && revision >= icmodule.RevisionICON2 {
		return nil
	}

	balance := cc.GetBalance(from)
	maxStake := new(big.Int).Add(balance, ia.GetTotalStake())
	if revision < icmodule.RevisionSystemSCORE {
		maxStake.Sub(maxStake, new(big.Int).Mul(cc.SumOfStepUsed(), cc.StepPrice()))
	}
	if maxStake.Cmp(v) == -1 {
		return scoreresult.OutOfBalanceError.Errorf("Not enough balance")
	}

	tStake := es.State.GetTotalStake()
	tSupply := cc.GetTotalSupply()
	oldTotalStake := ia.GetTotalStake()

	//update IISS account
	expireHeight := cc.BlockHeight() + es.State.GetUnstakeLockPeriod(revision, tSupply)
	var tl []icstate.TimerJobInfo
	switch stakeInc.Sign() {
	case 0, 1:
		// Condition: stakeInc >= 0
		tl, err = ia.DecreaseUnstake(stakeInc, expireHeight, revision)
	case -1:
		slotMax := int(es.State.GetUnstakeSlotMax())
		tl, err = ia.IncreaseUnstake(new(big.Int).Abs(stakeInc), expireHeight, slotMax, revision)
	}
	if err != nil {
		return scoreresult.UnknownFailureError.Wrapf(
			err,
			"Error while updating unstakes: from=%v",
			from,
		)
	}

	for _, t := range tl {
		ts := es.State.GetUnstakingTimerState(t.Height)
		if err = icstate.ScheduleTimerJob(ts, t, from); err != nil {
			return scoreresult.UnknownFailureError.Wrapf(
				err,
				"Error while scheduling UnStaking Timer Job: from=%v",
				from,
			)
		}
	}
	if err = ia.SetStake(v); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err,
			"Failed to set stake: from=%v stake=%v",
			from,
			v,
		)
	}
	if err = es.State.SetTotalStake(new(big.Int).Add(tStake, stakeInc)); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(
			err,
			"Failed to set totalStake: from=%v totalStake=%v stakeInc=%v",
			from,
			tStake,
			stakeInc,
		)
	}

	// Update the balance
	totalStake := ia.GetTotalStake()
	diff := new(big.Int).Sub(totalStake, oldTotalStake)
	sign := diff.Sign()
	if sign < 0 {
		es.Logger().Panicf(
			"Failed to setStake: oldTotalStake=%v > newTotalStake=%v from=%v",
			totalStake, oldTotalStake, from,
		)
	} else if sign > 0 {
		if err = cc.Withdraw(from, diff); err != nil {
			return err
		}
	}
	if icmodule.RevisionMultipleUnstakes <= revision && revision < icmodule.RevisionFixInvalidUnstake {
		migrate.ReproduceUnstakeBugForStake(cc, es.logger)
	}
	return
}

func (es *ExtensionStateImpl) RegisterPRep(cc icmodule.CallContext, info *icstate.PRepInfo) error {
	var err error
	from := cc.From()

	if err = info.Validate(cc.Revision().Value(), true); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err, "Failed to validate regInfo: from=%v", from,
		)
	}

	// Subtract RegPRepFee from SystemAddress
	err = cc.Withdraw(state.SystemAddress, icmodule.BigIntRegPRepFee)
	if err != nil {
		return err
	}
	// Burn regPRepFee
	if err = cc.Burn(state.SystemAddress, icmodule.BigIntRegPRepFee); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(
			err,
			"Failed to burn regPRepFee: from=%v fee=%v",
			from,
			icmodule.BigIntRegPRepFee,
		)
	}

	var irep *big.Int
	irepHeight := int64(0)
	blockHeight := cc.BlockHeight()
	term := es.State.GetTermSnapshot()

	if es.IsDecentralized() {
		irep = term.Irep()
		irepHeight = blockHeight
	} else {
		irep = icmodule.BigIntInitialIRep
	}

	if err = es.State.RegisterPRep(from, info, irep, irepHeight); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err, "Failed to register PRep: from=%v", from,
		)
	}

	_, err = es.Front.AddEventEnable(
		int(blockHeight-term.StartHeight()),
		from,
		icstage.ESEnable,
	)
	if err != nil {
		return scoreresult.UnknownFailureError.Wrapf(
			err, "Failed to add EventEnable: from=%v", from,
		)
	}

	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepRegistered(Address)")},
		[][]byte{from.Bytes()},
	)
	return nil
}

func (es *ExtensionStateImpl) SetPRep(cc icmodule.CallContext, info *icstate.PRepInfo) error {
	var err error
	var nodeUpdate bool
	from := cc.From()
	blockHeight := cc.BlockHeight()
	revision := cc.Revision().Value()

	if err = info.Validate(revision, false); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err, "Failed to validate regInfo: from=%v", from,
		)
	}
	if err = validateEndpoint(cc, info.P2PEndpoint); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err, "Failed to validate regInfo: from=%v", from,
		)
	}

	nodeUpdate, err = es.State.SetPRep(blockHeight, from, info)
	if err != nil {
		return scoreresult.InvalidParameterError.Wrapf(err, "Failed to set PRep: from=%v", from)
	}
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepSet(Address)")},
		[][]byte{from.Bytes()},
	)

	if icmodule.Revision8 <= revision && revision < icmodule.RevisionICON2 && nodeUpdate {
		// ICON1 update term when main P-Rep modify p2p endpoint or node address
		// Thus reward calculator segment VotedReward period
		ps, _ := es.State.GetPRepStatusByOwner(from, false)
		if ps.Grade() == icstate.GradeMain {
			term := es.State.GetTermSnapshot()
			if _, err = es.Front.AddEventVotedReward(int(blockHeight - term.StartHeight())); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateEndpoint(cc icmodule.CallContext, p2pEndpoint *string) error {
	revision := cc.Revision().Value()
	if p2pEndpoint == nil || revision < icmodule.RevisionPreventDuplicatedEndpoint {
		return nil
	}

	txID := cc.TransactionID()
	switch string(txID) {
	case "\x52\x9c\x33\xba\x49\x5f\x85\x88\x83\xd1\x31\x39\x5a\x97\x24\x8b\x37\x36\x99\xa4\x4f\x1a\xbe\x49\x60\xd7\x50\x1b\x0a\x53\x07\x4e":
		return errors.Errorf("Duplicated endpoint")
	}
	return nil
}

func (es *ExtensionStateImpl) GetIScore(from module.Address) (*big.Int, error) {
	iScore := new(big.Int)
	if es.Reward == nil {
		return iScore, nil
	}
	is, err := es.Reward.GetIScore(from)
	if err != nil {
		return nil, scoreresult.UnknownFailureError.Wrapf(
			err,
			"Failed to get IScore data: from=%v",
			from,
		)
	}
	if is == nil {
		return iScore, nil
	}

	iScore.Set(is.Value())
	stages := []*icstage.State{es.Front, es.Back1, es.Back2}
	for _, stage := range stages {
		if stage == nil {
			continue
		}
		claim, err := stage.GetIScoreClaim(from)
		if err != nil {
			return nil, scoreresult.UnknownFailureError.Wrapf(
				err,
				"Failed to get claim data from back: from=%v",
				from,
			)
		}
		if claim != nil {
			iScore.Sub(iScore, claim.Value())
		}
	}
	return iScore, nil
}

func (es *ExtensionStateImpl) ClaimIScore(cc icmodule.CallContext) error {
	from := cc.From()

	iScore, err := es.getIScore(from)
	if err != nil {
		return err
	}
	if iScore.Sign() == 0 {
		// there is no IScore to claim
		ClaimEventLog(cc, from, new(big.Int), new(big.Int))
		return nil
	}

	icx, remains := new(big.Int).DivMod(iScore, icmodule.BigIntIScoreICXRatio, new(big.Int))
	claim := new(big.Int).Sub(iScore, remains)

	if err = cc.Transfer(cc.Treasury(), from ,icx); err != nil {
		return scoreresult.InvalidInstanceError.Errorf(
			"Failed to transfer: from=%v to=%v amount=%v",
			cc.Treasury(), from, icx,
		)
	}

	// write claim data to front
	// IISS 2.0 : do not burn iScore < 1000
	// IISS 3.1 : burn iScore < 1000. To burn remains, set full iScore
	var ic *icstage.IScoreClaim
	revision := cc.Revision().Value()
	if revision < icmodule.RevisionICON2 {
		ic, err = es.Front.AddIScoreClaim(from, claim)
	} else {
		ic, err = es.Front.AddIScoreClaim(from, iScore)
	}
	if err != nil {
		return scoreresult.UnknownFailureError.Wrapf(
			err,
			"Failed to add IScore claim event: from=%v",
			from,
		)
	}
	if revision < icmodule.RevisionFixClaimIScore {
		cl := NewClaimIScoreLog(from, claim, ic)
		es.AppendExtensionLog(cl)
	}
	ClaimEventLog(cc, from, claim, icx)
	return nil
}

func (es *ExtensionStateImpl) getIScore(from module.Address) (*big.Int, error) {
	iScore := new(big.Int)
	if es.Reward == nil {
		return iScore, nil
	}
	is, err := es.Reward.GetIScore(from)
	if err != nil {
		return nil, scoreresult.UnknownFailureError.Wrapf(
			err,
			"Failed to get IScore data: from=%v",
			from,
		)
	}
	if is == nil {
		return iScore, nil
	}

	iScore.Set(is.Value())
	stages := []*icstage.State{es.Front, es.Back1, es.Back2}
	for _, stage := range stages {
		if stage == nil {
			continue
		}
		claim, err := stage.GetIScoreClaim(from)
		if err != nil {
			return nil, scoreresult.UnknownFailureError.Wrapf(
				err,
				"Failed to get claim data from back: from=%v",
				from,
			)
		}
		if claim != nil {
			iScore.Sub(iScore, claim.Value())
		}
	}
	return iScore, nil
}

func ClaimEventLog(cc icmodule.CallContext, address module.Address, claim *big.Int, icx *big.Int) {
	revision := cc.Revision().Value()
	if revision < icmodule.Revision9 {
		cc.OnEvent(state.SystemAddress,
			[][]byte{
				[]byte("IScoreClaimed(int,int)"),
			},
			[][]byte{
				intconv.BigIntToBytes(claim),
				intconv.BigIntToBytes(icx),
			},
		)
	} else {
		cc.OnEvent(state.SystemAddress,
			[][]byte{
				[]byte("IScoreClaimedV2(Address,int,int)"),
				address.Bytes(),
			},
			[][]byte{
				intconv.BigIntToBytes(claim),
				intconv.BigIntToBytes(icx),
			},
		)
	}
}

func (es *ExtensionStateImpl) Burn(cc icmodule.CallContext, amount *big.Int) error {
	from := cc.From()
	if err := cc.Withdraw(state.SystemAddress, amount); err != nil {
		return scoreresult.InvalidParameterError.Errorf(
			"Not enough value: from=%v value=%v", from, amount,
		)
	}
	if err := cc.Burn(from, amount); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err, "Failed to burn: from=%v value=%v", from, amount,
		)
	}
	return nil
}
