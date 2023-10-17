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
	"math"
	"math/big"
	"sort"

	"github.com/icon-project/goloop/btp/ntm"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
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

func (s *ExtensionSnapshotImpl) DB() db.Database {
	return s.database
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

	return &ExtensionStateImpl{
		database: s.database,
		logger:   logger,
		State:    icstate.NewStateFromSnapshot(s.state, readonly, logger),
		Front:    icstage.NewStateFromSnapshot(s.front),
		Back1:    icstage.NewStateFromSnapshot(s.back1),
		Back2:    icstage.NewStateFromSnapshot(s.back2),
		Reward:   icreward.NewStateFromSnapshot(s.reward),
	}
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

	logger           log.Logger
	log              []ExtensionLog
	illegalDelegated map[string]*icstate.PRepStatusState
	claimed          map[string]*Claimed

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
	case icstate.IISSVersion4:
		if err = es.Front.AddGlobalV3(
			term.StartHeight(),
			term.Revision(),
			int(term.Period()-1),
			term.GetElectedPRepCount(),
			term.BondRequirement(),
			term.RewardFund(),
			term.MinimumBond(),
		); err != nil {
			return
		}
	default:
		return errors.CriticalFormatError.Errorf(
			"InvalidIISSVersion(version=%d)", iissVersion)
	}
	return
}

func (es *ExtensionStateImpl) GetPRep(address module.Address) *icstate.PRep {
	return es.State.GetPRepByOwner(address)
}

func (es *ExtensionStateImpl) GetPRepInJSON(cc icmodule.CallContext, address module.Address) (map[string]interface{}, error) {
	prep := es.State.GetPRepByOwner(address)
	if prep == nil {
		return nil, errors.Errorf("PRep not found: %s", address)
	}
	sc := NewStateContext(cc, es)
	return prep.ToJSON(sc), nil
}

func (es *ExtensionStateImpl) GetPRepsInJSON(cc icmodule.CallContext, start, end int) (map[string]interface{}, error) {
	sc := NewStateContext(cc, es)
	return es.State.GetPRepsInJSON(sc, start, end)
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
		ps := es.State.GetPRepStatusByOwner(pss.Owner(), false)
		pb := es.State.GetPRepBaseByOwner(pss.Owner(), false)

		if ps != nil && ps.Grade() == icstate.GradeMain {
			pj := pss.ToJSON()
			pj["name"] = pb.Name()
			preps = append(preps, pj)
			sum.Add(sum, pss.Power())
			if len(preps) == mainPRepCount {
				break
			}
		}
	}

	jso["blockHeight"] = blockHeight
	jso["totalPower"] = sum
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
		ps := es.State.GetPRepStatusByOwner(pss.Owner(), false)
		pb := es.State.GetPRepBaseByOwner(pss.Owner(), false)

		if ps != nil && ps.Grade() == icstate.GradeSub {
			pj := pss.ToJSON()
			pj["name"] = pb.Name()
			preps = append(preps, pj)
			sum.Add(sum, pss.Power())
		}
	}

	jso["blockHeight"] = blockHeight
	jso["totalPower"] = sum
	jso["totalDelegated"] = sum
	jso["preps"] = preps
	return jso, nil
}

func (es *ExtensionStateImpl) SetDelegation(cc icmodule.CallContext, ds icstate.Delegations) error {

	var account *icstate.AccountState

	from := cc.From()
	blockHeight := cc.BlockHeight()
	account = es.State.GetAccountState(from)
	revision := cc.Revision().Value()
	replayPRepIllegalDelegated := revision >= icmodule.RevisionSystemSCORE && revision < icmodule.RevisionFixIllegalDelegation

	using := new(big.Int).Set(ds.GetDelegationAmount())
	using.Add(using, account.Unbond())
	using.Add(using, account.Bond())
	if account.Stake().Cmp(using) < 0 {
		return icmodule.IllegalArgumentError.Errorf("Not enough voting power")
	}

	delta := account.Delegations().Delta(ds)

	// apply delta to P-Rep and update total delegation
	nTotal := new(big.Int).Set(es.State.GetTotalDelegation())
	for key, value := range delta {
		owner, err := common.NewAddress([]byte(key))
		if err != nil {
			return scoreresult.UnknownFailureError.Wrapf(err, "Failed to update P-Rep delegation")
		}
		ps := es.State.GetPRepStatusByOwner(owner, true)
		oDelegated := ps.Delegated()
		if value.Sign() != 0 {
			ps.SetDelegated(new(big.Int).Add(oDelegated, value))
			if ps.IsActive() {
				nTotal.Add(nTotal, value)
			}
		}
		if replayPRepIllegalDelegated {
			// to replay PRep illegal delegated bug
			ps.SetEffectiveDelegated(new(big.Int).Add(oDelegated, value))
			if es.illegalDelegated == nil {
				es.illegalDelegated = make(map[string]*icstate.PRepStatusState)
			}
			es.illegalDelegated[key] = ps
		}
	}
	if err := es.State.SetTotalDelegation(nTotal); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to update total delegation")
	}

	var err error
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
		// update IllegalDelegation
		nId := id.Clone()
		nId.SetDelegations(ds)
		if err = es.State.SetIllegalDelegation(nId); err != nil {
			return scoreresult.UnknownFailureError.Wrapf(err, "Failed to set IllegalDelegation")
		}
	}

	if revision < icmodule.RevisionFixSetDelegation {
		dLog := newDelegationLog(from, offset, idx, obj, ds)
		es.AppendExtensionLog(dLog)
	}

	account.SetDelegation(ds)
	if icmodule.RevisionMultipleUnstakes <= revision && revision < icmodule.RevisionFixInvalidUnstake {
		migrate.ReproduceUnstakeBugForDelegation(cc, es.logger)
	}

	EmitDelegationSetEvent(cc, ds)

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

func (es *ExtensionStateImpl) addEventDelegated(blockHeight int64, delta map[string]*big.Int) error {
	votes, err := deltaToVotes(delta)
	if err != nil {
		return err
	}
	if len(votes) == 0 {
		return nil
	}
	term := es.State.GetTermSnapshot()
	offset := int(blockHeight - term.StartHeight())
	_, _, err = es.Front.AddEventDelegated(offset, state.ZeroAddress, votes)
	return err
}

func (es *ExtensionStateImpl) AddEventEnable(blockHeight int64, owner module.Address, status icmodule.EnableStatus) (err error) {
	term := es.State.GetTermSnapshot()
	_, err = es.Front.AddEventEnable(
		int(blockHeight-term.StartHeight()),
		owner,
		status,
	)
	return
}

func (es *ExtensionStateImpl) addBlockProduce(wc icmodule.WorldContext) (err error) {
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
	if csi == nil {
		return
	}
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
	sc := NewStateContext(cc, es)

	if err = es.State.DisablePRep(sc, owner, icstate.Unregistered); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(err, "Failed to unregister P-Rep %s", owner)
	}
	if err = es.AddEventEnable(blockHeight, owner, icmodule.ESDisablePermanent); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventEnable")
	}

	EmitPRepUnregisteredEvent(cc)
	return nil
}

func (es *ExtensionStateImpl) DisqualifyPRep(cc icmodule.CallContext, address module.Address) error {
	blockHeight := cc.BlockHeight()
	sc := NewStateContext(cc, es)

	if err := es.State.DisablePRep(sc, address, icstate.Disqualified); err != nil {
		return err
	}
	if err := es.AddEventEnable(blockHeight, address, icmodule.ESDisablePermanent); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventEnable")
	}
	pt := icmodule.PenaltyPRepDisqualification
	ps := es.State.GetPRepStatusByOwner(address, false)
	// Record PenaltyImposed eventlog
	EmitPenaltyImposedEvent(cc, ps, pt)
	rate, _ := es.State.GetSlashingRate(cc.Revision().Value(), pt)
	return es.slash(cc, address, rate)
}

func (es *ExtensionStateImpl) PenalizeNonVoters(cc icmodule.CallContext, address module.Address) error {
	// Record PenaltyImposed eventlog
	ps := es.State.GetPRepStatusByOwner(address, false)
	if ps == nil {
		return icmodule.NotFoundError.Errorf("PRepNotFound(%s)", address)
	}
	pt := icmodule.PenaltyMissedNetworkProposalVote
	EmitPenaltyImposedEvent(cc, ps, pt)
	rate, _ := es.State.GetSlashingRate(cc.Revision().Value(), pt)
	return es.slash(cc, address, rate)
}

func (es *ExtensionStateImpl) SetBond(cc icmodule.CallContext, bonds icstate.Bonds) error {
	blockHeight := cc.BlockHeight()
	from := cc.From()
	es.logger.Tracef("SetBond() start: from=%s bonds=%+v", from, bonds)

	var account *icstate.AccountState
	account = es.State.GetAccountState(from)

	bondAmount := big.NewInt(0)
	for _, bond := range bonds {
		bondAmount.Add(bondAmount, bond.Amount())

		pb := es.State.GetPRepBaseByOwner(bond.To(), false)
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

	delta := account.Bonds().Delta(bonds)

	// apply delta to P-Rep and update total bond
	nTotal := new(big.Int).Set(es.State.GetTotalBond())
	for key, value := range delta {
		owner, err := common.NewAddress([]byte(key))
		if err != nil {
			return scoreresult.UnknownFailureError.Wrapf(err, "Failed to update P-Rep bond")
		}
		if value.Sign() != 0 {
			ps := es.State.GetPRepStatusByOwner(owner, true)
			ps.SetBonded(new(big.Int).Add(ps.Bonded(), value))
			if ps.IsActive() {
				nTotal.Add(nTotal, value)
			}
		}
	}
	if err := es.State.SetTotalBond(nTotal); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to update total bond")
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
		icstate.ScheduleTimerJob(unbondingTimer, timerJobInfo, from)
	}

	if err = es.AddEventBond(blockHeight, from, delta); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(err, "Failed to add EventBond")
	}

	EmitBondSetEvent(cc, bonds)

	es.logger.Tracef("SetBond() end")
	return nil
}

func (es *ExtensionStateImpl) GetBond(address module.Address) (map[string]interface{}, error) {
	a := es.State.GetAccountSnapshot(address)
	if a == nil {
		a = icstate.GetEmptyAccountSnapshot()
	}
	return a.GetBondInJSON(), nil
}

func (es *ExtensionStateImpl) AddEventBond(blockHeight int64, from module.Address, delta map[string]*big.Int) (err error) {
	votes, err := deltaToVotes(delta)
	if err != nil {
		return
	}
	term := es.State.GetTermSnapshot()
	_, _, err = es.Front.AddEventBond(
		int(blockHeight-term.StartHeight()),
		from,
		votes,
	)
	return
}

func (es *ExtensionStateImpl) SetBonderList(from module.Address, bl icstate.BonderList) error {
	es.logger.Tracef("SetBonderList() start: from=%s bl=%s", from, bl)

	pb := es.State.GetPRepBaseByOwner(from, false)
	if pb == nil {
		return scoreresult.InvalidParameterError.Errorf("PRep not found: %v", from)
	}
	ps := es.State.GetPRepStatusByOwner(from, false)
	if ps == nil || !ps.IsActive() {
		return scoreresult.InvalidParameterError.Errorf("Inactive PRep can't set bonder list: %v", from)
	}

	var account *icstate.AccountState
	for _, old := range pb.BonderList() {
		if !bl.Contains(old) {
			account = es.State.GetAccountState(old)
			if account.Bonds().Contains(from) || account.Unbonds().Contains(from) {
				return scoreresult.InvalidParameterError.Errorf("Can't remove bonder(%s) who bonded to you", old)
			}
		}
	}

	pb.SetBonderList(bl)
	es.logger.Tracef("SetBonderList() end")
	return nil
}

func (es *ExtensionStateImpl) GetBonderList(address module.Address) (map[string]interface{}, error) {
	pb := es.State.GetPRepBaseByOwner(address, false)
	if pb == nil {
		return nil, errors.Errorf("PRep not found: %v", address)
	}
	jso := make(map[string]interface{})
	jso["bonderList"] = pb.GetBonderListInJSON()
	return jso, nil
}

func (es *ExtensionStateImpl) SetGovernanceVariables(from module.Address, irep *big.Int, blockHeight int64) error {
	pb := es.State.GetPRepBaseByOwner(from, false)
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
	divider := new(big.Int).SetInt64(int64(100 * icmodule.MonthPerYear * (term.MainPRepCount() + icmodule.VotedRewardMultiplier)))
	limit.Div(limit, divider)
	if newIRep.Cmp(limit) == 1 {
		return scoreresult.InvalidParameterError.Errorf("IRep is out of range: %v > %v", newIRep, limit)
	}
	return nil
}

func (es *ExtensionStateImpl) ValidateRewardFund(iglobal *big.Int, totalSupply *big.Int, revision int) error {
	rf := es.State.GetRewardFund(revision)
	return validateRewardFund(iglobal, rf.IGlobal(), totalSupply)
}

const inflationLimit = 15

func validateRewardFund(iglobal *big.Int, currentIglobal *big.Int, totalSupply *big.Int) error {
	min := new(big.Int).Mul(currentIglobal, big.NewInt(3))
	min.Div(min, big.NewInt(4))
	max := new(big.Int).Mul(currentIglobal, big.NewInt(5))
	max.Div(max, big.NewInt(4))
	if (iglobal.Cmp(min) <= 0) || (iglobal.Cmp(max) >= 0) {
		return scoreresult.InvalidParameterError.Errorf(
			"Failed to validate Iglobal: Out of range. Min:%d, Max: %d, Iglobal=%d", min, max, iglobal,
		)
	}
	rewardPerYear := new(big.Int).Mul(iglobal, big.NewInt(12))
	maxRewardPerYear := new(big.Int).Mul(totalSupply, big.NewInt(100+inflationLimit))
	maxRewardPerYear.Div(maxRewardPerYear, big.NewInt(100))

	if rewardPerYear.Cmp(maxRewardPerYear) == 1 {
		return scoreresult.InvalidParameterError.Errorf(
			"Failed to validate Iglobal: too much inflation %d > %d", rewardPerYear, maxRewardPerYear,
		)
	}
	return nil
}

func (es *ExtensionStateImpl) OnExecutionBegin(wc icmodule.WorldContext) error {
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

func (es *ExtensionStateImpl) OnExecutionEnd(wc icmodule.WorldContext, totalFee *big.Int, calculator Calculator) error {
	var err error
	if err = es.handleTimerJob(wc); err != nil {
		return err
	}
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
		if err = es.checkCalculationDone(calculator); err != nil {
			return err
		}
		if err = es.regulateIssue(calculator.TotalReward()); err != nil {
			return err
		}
	case term.GetEndHeight():
		if err = es.onTermEnd(wc); err != nil {
			return err
		}
		isTermEnd = true

		nTerm := es.State.GetTermSnapshot()
		if term.IsDecentralized() {
			if err = es.resetIssueTotalReward(); err != nil {
				return err
			}
		} else if nTerm.IsDecentralized() {
			// last centralized block
			if err = es.setIssuePrevBlockFee(totalFee); err != nil {
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
	es.claimed = nil
	return nil
}

func (es *ExtensionStateImpl) checkCalculationDone(calculator Calculator) error {
	// Called at the end block of Term and effected to base TX issue amount in ICON1
	rcInfo, err := es.State.GetRewardCalcInfo()
	if err != nil {
		return err
	}

	if err := calculator.WaitResult(rcInfo.StartHeight()); err != nil {
		return err
	}
	return nil
}

func (es *ExtensionStateImpl) regulateIssue(iScore *big.Int) error {
	// Update Issue with calculation result from 2nd Term of decentralization
	term := es.State.GetTermSnapshot()
	if !term.IsDecentralized() || term.Sequence() == 0 {
		return nil
	}

	prevGlobal, err := es.Back2.GetGlobal()
	if err != nil {
		return err
	}
	reward := new(big.Int).Set(iScore)
	if prevGlobal != nil && icstate.IISSVersion3 == prevGlobal.GetIISSVersion() {
		pg := prevGlobal.GetV2()
		multiplier := big.NewInt(int64(prevGlobal.GetTermPeriod() * icmodule.IScoreICXRatio))
		divider := big.NewInt(icmodule.MonthBlock * icmodule.DenomInRate)
		rewardCPS := new(big.Int).Mul(pg.GetIGlobal(), pg.GetICps().NumBigInt())
		rewardCPS.Mul(rewardCPS, multiplier)
		rewardCPS.Div(rewardCPS, divider)
		reward.Add(reward, rewardCPS)
		rewardRelay := new(big.Int).Mul(pg.GetIGlobal(), pg.GetIRelay().NumBigInt())
		rewardRelay.Mul(rewardRelay, multiplier)
		rewardRelay.Div(rewardRelay, divider)
		reward.Add(reward, rewardRelay)
		es.logger.Tracef("regulateIssue with cps: %d, relay: %d", rewardCPS, rewardRelay)
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

func (es *ExtensionStateImpl) onTermEnd(wc icmodule.WorldContext) error {
	var err error

	revision := wc.Revision().Value()
	pcCfg := es.State.GetPRepCountConfig(revision)

	totalSupply := wc.GetTotalSupply()
	isDecentralized := es.IsDecentralized()
	sc := NewStateContext(wc, es)

	prepSet := icstate.NewPRepSet(sc, es.State.GetPReps(true), pcCfg)
	if !isDecentralized {
		// After decentralization is finished, this code will not be reached
		isDecentralized = es.State.IsDecentralizationConditionMet(revision, totalSupply, prepSet)
	}

	if isDecentralized {
		// Reset the status of all active preps ordered by power
		limit := es.State.GetConsistentValidationPenaltyMask()
		if err = prepSet.OnTermEnd(sc, limit); err != nil {
			return err
		}
	} else {
		prepSet = nil
	}

	return es.moveOnToNextTerm(sc, prepSet, totalSupply)
}

func (es *ExtensionStateImpl) moveOnToNextTerm(
	sc icmodule.StateContext, preps icstate.PRepSet, totalSupply *big.Int) error {

	// Create a new term
	revision := sc.Revision()
	nextTerm := icstate.NewNextTerm(es.State, totalSupply, revision)

	// Valid preps means that decentralization is activated
	if preps != nil {
		br := sc.GetBondRequirement()
		mainPRepCount := preps.GetPRepSize(icstate.GradeMain)
		pss := preps.ToPRepSnapshots(br)

		nextTerm.SetMainPRepCount(mainPRepCount)
		nextTerm.SetPRepSnapshots(pss)
		nextTerm.SetIsDecentralized(true)
		es.setIrepToTerm(revision, preps, nextTerm)

		// Record new validator list for the next term to State
		vss := icstate.NewValidatorsSnapshotWithPRepSnapshot(pss, es.State, mainPRepCount)
		if err := es.State.SetValidatorsSnapshot(vss); err != nil {
			return err
		}
	}

	es.setRrepToTerm(revision, totalSupply, nextTerm)

	term := es.State.GetTermSnapshot()
	if !term.IsDecentralized() && nextTerm.IsDecentralized() {
		// reset sequence when network is decentralized
		nextTerm.ResetSequence()
	}

	es.logger.Debugf(nextTerm.String())
	return es.State.SetTermSnapshot(nextTerm.GetSnapshot())
}

func (es *ExtensionStateImpl) setIrepToTerm(revision int, preps icstate.PRepSet, term *icstate.TermState) {
	var irep *big.Int
	if revision < icmodule.RevisionDecentralize || revision >= icmodule.RevisionEnableIISS3 {
		// disable IRep
		irep = new(big.Int)
	} else if revision >= icmodule.RevisionSetIRepViaNetworkProposal {
		// use network value IRep
		irep = new(big.Int).Set(es.State.GetIRep())
	} else {
		irep = calculateIRep(preps)
	}
	term.SetIrep(irep)
}

func (es *ExtensionStateImpl) setRrepToTerm(revision int, totalSupply *big.Int, term *icstate.TermState) {
	var rrep *big.Int
	if revision < icmodule.RevisionIISS || revision >= icmodule.RevisionEnableIISS3 {
		// disable Rrep
		rrep = new(big.Int)
	} else {
		rrep = calculateRRep(totalSupply, es.State.GetTotalDelegation())
	}
	term.SetRrep(rrep)
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

func (es *ExtensionStateImpl) applyCalculationResult(calculator Calculator, blockHeight int64) error {
	var resultHash []byte
	result := calculator.Result()
	reward := calculator.TotalReward()

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
			// 0.1 = IScoreICXRation / 10000
			divider := big.NewInt(10)
			rewardCPS := new(big.Int).Mul(pg.GetIGlobal(), pg.GetICps().NumBigInt())
			rewardCPS.Div(rewardCPS, divider)
			reward.Add(reward, rewardCPS)
			rewardRelay := new(big.Int).Mul(pg.GetIGlobal(), pg.GetIRelay().NumBigInt())
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

	// switch icstage back
	es.Back2 = es.Back1
	es.Back1 = icstage.NewState(es.database) // ss.Byte() nil 확인
	return nil
}

func (es *ExtensionStateImpl) GenesisTerm(blockHeight int64, revision int) error {
	// Start genesis term according to the period information if it's not started.
	if revision >= icmodule.RevisionIISS && es.State.GetTermSnapshot() == nil {
		term := icstate.GenesisTerm(es.State, blockHeight+1, revision)
		if err := es.State.SetTermSnapshot(term.GetSnapshot()); err != nil {
			return err
		}
	}
	return nil
}

// updateValidators set a new validator set to world context
func (es *ExtensionStateImpl) updateValidators(wc icmodule.WorldContext, isTermEnd bool) error {
	var err error
	vss := es.State.GetValidatorsSnapshot()
	if vss == nil {
		return nil
	}

	blockHeight := wc.BlockHeight()
	if isTermEnd || vss.IsUpdated(blockHeight) {
		newValidators := vss.NewValidatorSet()
		err = wc.SetValidators(newValidators)
		es.logger.Debugf("New validators: bh=%d vss=%+v", blockHeight, vss)
	}
	return err
}

func (es *ExtensionStateImpl) GetPRepTermInJSON(cc icmodule.CallContext) (map[string]interface{}, error) {
	term := es.State.GetTermSnapshot()
	if term == nil {
		err := errors.Errorf("Term is nil")
		return nil, err
	}
	sc := NewStateContext(cc, es)
	jso := term.ToJSON(sc, es.State)
	jso["blockHeight"] = cc.BlockHeight()
	return jso, nil
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

func (es *ExtensionStateImpl) handleExtensionLog() error {
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
	if stakeInc.Sign() == 0 && revision >= icmodule.RevisionStopICON1Support {
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

	// update IISS account
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
		icstate.ScheduleTimerJob(ts, t, from)
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
		if err = cc.Withdraw(from, diff, module.Stake); err != nil {
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
	err = cc.Withdraw(state.SystemAddress, icmodule.BigIntRegPRepFee, module.RegPRep)
	if err != nil {
		return err
	}
	// Burn regPRepFee
	if err = cc.HandleBurn(from, icmodule.BigIntRegPRepFee); err != nil {
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

	if es.IsDecentralized() {
		term := es.State.GetTermSnapshot()
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

	if err = es.AddEventEnable(blockHeight, from, icmodule.ESEnable); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(
			err, "Failed to add EventEnable: from=%v", from,
		)
	}

	EmitPRepRegisteredEvent(cc)
	return nil
}

func (es *ExtensionStateImpl) SetPRep(cc icmodule.CallContext, info *icstate.PRepInfo, fromBTP bool) error {
	var err error
	var nodeUpdate bool
	from := cc.From()
	blockHeight := cc.BlockHeight()
	revision := cc.Revision().Value()

	if !fromBTP && revision >= icmodule.RevisionBTP2 && info.Node != nil {
		prep := es.GetPRep(from)
		if !info.Node.Equal(prep.NodeAddress()) {
			return scoreresult.InvalidParameterError.Errorf(
				"Can't modify node address by setPRep method of chain SCORE")
		}
	}

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
	EmitPRepSetEvent(cc)

	if icmodule.Revision8 <= revision && revision < icmodule.RevisionStopICON1Support && nodeUpdate {
		// ICON1 update term when main P-Rep modify p2p endpoint or node address
		// Thus reward calculator segment VotedReward period
		ps := es.State.GetPRepStatusByOwner(from, false)
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

func (es *ExtensionStateImpl) GetIScore(from module.Address, revision int, txID []byte) (*big.Int, error) {
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
	for i, stage := range stages {
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
		// replay ICON1's queryIScore behavior
		if revision < icmodule.RevisionFixClaimIScore && i == 0 && len(txID) != 0 {
			if claimed, ok := es.claimed[icutils.ToKey(from)]; ok {
				if bytes.Compare(claimed.ID(), txID) == 0 {
					// Subtract claimed amount only when claimIScore and queryIScore are in the same TX
					// Reverted claimIScore works the same
					iScore.Sub(iScore, claimed.Amount())
				}
			} else {
				if claim != nil {
					iScore.Sub(iScore, claim.Value())
				}
			}
		} else {
			if claim != nil {
				iScore.Sub(iScore, claim.Value())
			}
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
		EmitIScoreClaimEvent(cc, icmodule.BigIntZero, icmodule.BigIntZero)
		return nil
	}

	icx, remains := new(big.Int).DivMod(iScore, icmodule.BigIntIScoreICXRatio, new(big.Int))
	claim := new(big.Int).Sub(iScore, remains)

	if err = cc.Transfer(cc.Treasury(), from, icx, module.Claim); err != nil {
		return scoreresult.InvalidInstanceError.Errorf(
			"Failed to transfer: from=%v to=%v amount=%v",
			cc.Treasury(), from, icx,
		)
	}

	// write claim data to front
	// IISS 2.x : do not burn iScore < 1000
	// IISS 3.x : burn iScore < 1000. To burn remains, set full iScore
	var ic *icstage.IScoreClaim
	revision := cc.Revision().Value()
	if revision < icmodule.RevisionEnableIISS3 {
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
		if es.claimed == nil {
			es.claimed = make(map[string]*Claimed)
		}
		es.claimed[icutils.ToKey(from)] = newClaimed(cc.TransactionID(), claim)
	}
	EmitIScoreClaimEvent(cc, claim, icx)
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

func calculateIRep(prepSet icstate.PRepSet) *big.Int {
	irep := new(big.Int)
	mainPRepCount := prepSet.GetPRepSize(icstate.GradeMain)
	totalDelegated := new(big.Int)
	totalWeightedIrep := new(big.Int)
	value := new(big.Int)

	for i := 0; i < mainPRepCount; i++ {
		prep := prepSet.GetByIndex(i)
		totalWeightedIrep.Add(totalWeightedIrep, value.Mul(prep.IRep(), prep.Delegated()))
		totalDelegated.Add(totalDelegated, prep.Delegated())
	}

	if totalDelegated.Sign() == 0 {
		return irep
	}

	irep.Div(totalWeightedIrep, totalDelegated)
	if irep.Cmp(icmodule.BigIntMinIRep) == -1 {
		irep.Set(icmodule.BigIntMinIRep)
	}
	return irep
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

func (es *ExtensionStateImpl) handlePRepIllegalDelegated(blockHeight int64, txSuccess bool) error {
	delta := make(map[string]*big.Int)
	nTotal := new(big.Int).Set(es.State.GetTotalDelegation())
	for key, ps := range es.illegalDelegated {
		owner := common.MustNewAddress([]byte(key))
		es.logger.Tracef("handlePRepIllegalDelegated %v %s: %+v", txSuccess, owner, ps)
		eDelegated := ps.EffectiveDelegated()
		delegated := ps.Delegated()
		nDiff := new(big.Int).Sub(eDelegated, delegated)
		oDiff := es.State.GetPRepIllegalDelegated(owner)
		if txSuccess {
			if err := es.State.SetPRepIllegalDelegated(owner, nDiff); err != nil {
				return err
			}
			delta[key] = new(big.Int).Sub(nDiff, oDiff)
			if ps.IsActive() {
				nTotal.Add(nTotal, delta[key])
			}
			if oDiff.Sign() != 0 && nDiff.Sign() == 0 {
				ps.SetEffectiveDelegated(nil)
				es.logger.Warnf("PRepIllegalDelegated was cleared at %d. %s: %d", blockHeight, owner, oDiff)
			}
			if oDiff.Sign() == 0 && nDiff.Sign() != 0 {
				es.logger.Warnf("PRepIllegalDelegated was occurred at %d. %s: %d", blockHeight, owner, nDiff)
			}
		} else {
			// revert effectiveDelegated
			ps.SetEffectiveDelegated(new(big.Int).Add(delegated, oDiff))
		}
	}
	if len(delta) > 0 {
		if err := es.addEventDelegated(blockHeight, delta); err != nil {
			return err
		}
	}
	if err := es.State.SetTotalDelegation(nTotal); err != nil {
		return err
	}
	es.illegalDelegated = nil
	return nil
}

func (es *ExtensionStateImpl) ClearPRepIllegalDelegated() error {
	preps := es.State.GetPReps(false)
	for _, prep := range preps {
		if prep.EffectiveDelegated() != nil {
			if es.illegalDelegated == nil {
				es.illegalDelegated = make(map[string]*icstate.PRepStatusState)
			}
			ps := prep.PRepStatusState
			ps.SetEffectiveDelegated(ps.Delegated())
			key := icutils.ToKey(prep.Owner())
			es.illegalDelegated[key] = ps
		}
	}
	// will be cleared at OnTransactionEnd()
	return nil
}

func (es *ExtensionStateImpl) OnTransactionEnd(blockHeight int64, success bool) error {
	if err := es.handlePRepIllegalDelegated(blockHeight, success); err != nil {
		return err
	}
	return es.handleExtensionLog()
}

type Claimed struct {
	id     []byte
	amount *big.Int
}

func (c *Claimed) ID() []byte {
	return c.id
}

func (c *Claimed) Amount() *big.Int {
	return c.amount
}

func newClaimed(id []byte, amount *big.Int) *Claimed {
	return &Claimed{
		id:     id,
		amount: amount,
	}
}

func (es *ExtensionStateImpl) transferRewardFund(cc icmodule.CallContext) error {
	term := es.State.GetTermSnapshot()
	if term == nil || !term.IsDecentralized() || term.GetIISSVersion() < icstate.IISSVersion3 ||
		term.GetEndHeight() != cc.BlockHeight() {
		return nil
	}

	rf := term.RewardFund()
	if rf.IGlobal().Sign() != 1 {
		return nil
	}
	fs := []struct {
		key  string
		rate icmodule.Rate
	}{
		{icstate.CPSKey, rf.ICps()},
		{icstate.RelayKey, rf.IRelay()},
	}
	ns := es.State.GetNetworkScores(cc)
	div := big.NewInt(icmodule.DenomInRate * icmodule.MonthBlock)
	base := new(big.Int).Mul(rf.IGlobal(), new(big.Int).SetInt64(term.Period()))
	from := cc.Treasury()
	for _, k := range fs {
		if k.rate == 0 {
			es.logger.Warnf("There is no reward fund for %s", k.key)
			continue
		}
		if !k.rate.IsValid() {
			es.logger.Panicf("InvalidInflationRate(key=%s,rate=%d)", k.key, k.rate)
		}
		to, ok := ns[k.key]
		amount := new(big.Int).Mul(base, k.rate.NumBigInt())
		amount.Div(amount, div)
		if ok {
			if err := cc.Transfer(from, to, amount, module.Reward); err != nil {
				return err
			}
			EmitRewardFundTransferredEvent(cc, k.key, from, to, amount)
		} else {
			if cc.Revision().Value() >= icmodule.RevisionFixTransferRewardFund {
				if err := cc.Withdraw(from, amount, module.Burn); err != nil {
					return scoreresult.InvalidParameterError.Errorf(
						"Not enough balance: from=%v value=%v", from, amount,
					)
				}
			}
			if err := cc.HandleBurn(from, amount); err != nil {
				return err
			}
			EmitRewardFundBurnedEvent(cc, k.key, from, amount)
			es.logger.Warnf("Burn %s for %s", amount, k.key)
		}
	}
	return nil
}

func (es *ExtensionStateImpl) OnOpenBTPNetwork(cc icmodule.CallContext, bc state.BTPContext, ntName string) error {
	dsaName := ntm.ForUID(ntName).DSA()
	ntids, err := bc.GetNetworkTypeIDs()
	if err != nil {
		return err
	}

	dsaActivated := true
	for _, ntid := range ntids {
		if nt, err := bc.GetNetworkType(ntid); err != nil {
			return err
		} else {
			if nt.UID() == ntName {
				continue
			}
			if ntm.ForUID(nt.UID()).DSA() == dsaName {
				dsaActivated = false
				break
			}
		}
	}

	if dsaActivated {
		term := es.State.GetTermSnapshot()
		return es.Front.AddBTPDSA(
			int(cc.BlockHeight()-term.StartHeight()),
			int(cc.TransactionInfo().Index),
			bc.GetDSAIndex(dsaName),
		)
	}
	return nil
}

func (es *ExtensionStateImpl) OnSetPublicKey(cc icmodule.CallContext, from module.Address, dsaIndex int) error {
	term := es.State.GetTermSnapshot()
	return es.Front.AddBTPPublicKey(
		int(cc.BlockHeight()-term.StartHeight()),
		int(cc.TransactionInfo().Index),
		from,
		dsaIndex,
	)
}

func (es *ExtensionStateImpl) SetSlashingRates(cc icmodule.CallContext, values map[string]icmodule.Rate) error {
	var pt icmodule.PenaltyType
	rates := make(map[icmodule.PenaltyType]icmodule.Rate)

	for name, rate := range values {
		pt = icmodule.ToPenaltyType(name)
		if !pt.IsValid() {
			return scoreresult.InvalidParameterError.Errorf("InvalidPenaltyName(%s)", name)
		}
		rates[pt] = rate
	}

	revision := cc.Revision().Value()
	for _, pt = range icmodule.GetPenaltyTypes() {
		if rate, ok := rates[pt]; ok {
			oldRate, err := es.State.GetSlashingRate(revision, pt)
			if err != nil {
				return err
			}
			if oldRate != rate {
				if err = es.State.SetSlashingRate(revision, pt, rate); err != nil {
					return err
				}
				// Record slashingRateChangedV2 eventLogs in PenaltyType order
				EmitSlashingRateSetEvent(cc, pt, rate)
			}
		}
	}
	return nil
}

func (es *ExtensionStateImpl) GetSlashingRates(
	cc icmodule.CallContext, penaltyTypes []icmodule.PenaltyType) (map[string]interface{}, error) {
	if len(penaltyTypes) == 0 {
		penaltyTypes = icmodule.GetPenaltyTypes()
	}

	revision := cc.Revision().Value()
	jso := make(map[string]interface{})
	for _, pt := range penaltyTypes {
		if rate, err := es.State.GetSlashingRate(revision, pt); err == nil {
			jso[pt.String()] = rate.NumInt64()
		} else {
			return nil, err
		}
	}
	return jso, nil
}

func (es *ExtensionStateImpl) InitCommissionInfo(
	cc icmodule.CallContext, rate, maxRate, maxChangeRate icmodule.Rate) error {
	ci, err := icstate.NewCommissionInfo(rate, maxRate, maxChangeRate)
	if err != nil {
		return err
	}
	owner := cc.From()
	if err = es.State.InitCommissionInfo(owner, ci); err != nil {
		return err
	}
	if err = es.Front.AddCommissionRate(owner, rate); err != nil {
		return err
	}
	EmitCommissionRateInitializedEvent(cc, rate, maxRate, maxChangeRate)
	return nil
}

func (es *ExtensionStateImpl) SetCommissionRate(cc icmodule.CallContext, rate icmodule.Rate) error {
	owner := cc.From()
	if owner == nil {
		return scoreresult.InvalidParameterError.Errorf("InvalidOwner(%s)", owner)
	}
	if !rate.IsValid() {
		return scoreresult.InvalidParameterError.Errorf("InvalidCommissionRate(%d)", rate)
	}
	pb := es.State.GetPRepBaseByOwner(owner, false)
	if pb == nil {
		return icmodule.NotFoundError.Errorf("PRepBaseNotFound(%s)", owner)
	}
	if !pb.CommissionInfoExists() {
		return icmodule.NotFoundError.Errorf("CommissionInfoNotFound(%s)", owner)
	}
	ps := es.State.GetPRepStatusByOwner(owner, false)
	if ps == nil {
		return icmodule.NotFoundError.Errorf("PRepStatusNotFound(%s)", owner)
	}
	if !ps.IsActive() {
		return icmodule.NotReadyError.Errorf("PRepNotActive(%s)", owner)
	}

	if rate == pb.CommissionRate() {
		// Nothing to do
		return nil
	}
	oldRate, err := es.getOldCommissionRate(owner)
	if err != nil {
		return err
	}
	changeRate := rate - oldRate
	maxChangeRate := pb.MaxCommissionChangeRate()
	if changeRate > maxChangeRate {
		return icmodule.IllegalArgumentError.Errorf(
			"CommissionRateTooHigh(oldRate=%d,newRate=%d,maxChangeRate=%d)",
			oldRate, rate, maxChangeRate)
	}
	if err = pb.SetCommissionRate(rate); err != nil {
		return err
	}
	if err = es.Front.AddCommissionRate(owner, rate); err != nil {
		return err
	}
	EmitCommissionRateSetEvent(cc, rate)
	return nil
}

func (es *ExtensionStateImpl) getOldCommissionRate(owner module.Address) (icmodule.Rate, error) {
	// Find owner's old commissionRate in Back1 and Back2
	for _, back := range []*icstage.State{es.Back1, es.Back2} {
		cr, err := back.GetCommissionRate(owner)
		if err != nil {
			return icmodule.Rate(0), err
		}
		if cr != nil {
			return cr.Value(), nil
		}
	}

	// If there is no commission rate for a given owner in Back1 and Back2, look for it in Base.
	voted, err := es.Reward.GetVoted(owner)
	if err != nil {
		return icmodule.Rate(0), err
	}
	if voted != nil {
		return voted.CommissionRate(), nil
	}
	// It is not allowed to call setCommissionRate() during the term when commissionInfo is initialized.
	return icmodule.Rate(0), icmodule.NotFoundError.Errorf("OldCommissionRateNotFound(%s)", owner)
}

func (es *ExtensionStateImpl) RequestUnjail(cc icmodule.CallContext) error {
	owner := cc.From()
	if owner == nil {
		return scoreresult.InvalidParameterError.Errorf("InvalidOwner(%s)", owner)
	}
	ps := es.State.GetPRepStatusByOwner(owner, false)
	if ps == nil {
		return icmodule.NotFoundError.Errorf("PRepStatusNotFound(%s)", owner)
	}
	if !ps.IsActive() {
		return icmodule.NotReadyError.Errorf("PRepNotActive(%s)", owner)
	}

	return ps.OnEvent(NewStateContext(cc, es), icmodule.PRepEventRequestUnjail)
}

func (es *ExtensionStateImpl) GetPRepStats(cc icmodule.CallContext) (map[string]interface{}, error) {
	sc := NewStateContext(cc, es)
	return es.State.GetPRepStatsInJSON(sc)
}

func (es *ExtensionStateImpl) GetPRepStatsOf(
	cc icmodule.CallContext, address module.Address) (map[string]interface{}, error) {
	sc := NewStateContext(cc, es)
	return es.State.GetPRepStatsOfInJSON(sc, address)
}

// HandleDoubleSignReport handles DoubleSignReport from system
// signer: Node address of a validator that committed double signing
func (es *ExtensionStateImpl) HandleDoubleSignReport(
	cc icmodule.CallContext, dsType string, dsBlockHeight int64, signer module.Address) error {
	es.Logger().Tracef("HandleDoubleSignReport(dsType=%s,dsBlockHeight=%d,signer=%s) start",
		dsType, dsBlockHeight, signer)
	defer es.Logger().Tracef("HandleDoubleSignReport(dsType=%s,dsBlockHeight=%d,signer=%s) end",
		dsType, dsBlockHeight, signer)

	if dsType != module.DSTProposal && dsType != module.DSTVote {
		return scoreresult.InvalidParameterError.Errorf("UnknownDoubleSignType(%s)", dsType)
	}
	if dsBlockHeight < int64(1) || dsBlockHeight >= cc.BlockHeight() {
		return scoreresult.InvalidParameterError.Errorf("InvalidDoubleSignBlockHeight(%d)", dsBlockHeight)
	}

	sc := NewStateContext(cc, es)
	if sc.TermIISSVersion() < icstate.IISSVersion4 {
		// Ignore DoubleSignReports silently
		//return icmodule.NotReadyError.New("IISS4NotReady")
		return nil
	}

	owner := es.State.GetOwnerByNode(signer)
	ps := es.State.GetPRepStatusByOwner(owner, false)
	if ps == nil {
		return icmodule.NotFoundError.Errorf("PRepStatusNotFound(%s)", owner)
	}
	if !ps.IsDoubleSignReportable(sc, dsBlockHeight) {
		// Ignore DoubleSignReports silently
		return nil
	}
	EmitDoubleSignReportedEvent(cc, owner, dsBlockHeight, dsType)

	const pt = icmodule.PenaltyDoubleSign
	if err := es.State.ImposePenalty(sc, pt, ps); err != nil {
		return err
	}
	EmitPenaltyImposedEvent(cc, ps, pt)

	rate, err := es.State.GetSlashingRate(sc.Revision(), pt)
	if err != nil {
		return err
	}
	return es.slash(cc, owner, rate)
}

func (es *ExtensionStateImpl) SetPRepCountConfig(cc icmodule.CallContext, counts map[string]int64) error {
	var main, sub, extra int64
	oldCfg := es.State.GetPRepCountConfig(cc.Revision().Value())
	updated := 0

	for k, v := range counts {
		if v < int64(0) || v > int64(1000) {
			return scoreresult.InvalidParameterError.Errorf("CountOutOfRange(%s=%d)", k, v)
		}
		switch k {
		case "main":
			main = v
			updated |= 1
		case "sub":
			sub = v
			updated |= 2
		case "extra":
			extra = v
			updated |= 4
		default:
			return scoreresult.InvalidParameterError.Errorf("InvalidName(%s)", k)
		}
	}

	if updated > 0 {
		if updated&1 == 0 {
			main = int64(oldCfg.MainPReps(icstate.MainPRepNormal))
		}
		if updated&2 == 0 {
			sub = int64(oldCfg.SubPReps())
		}
		if updated&4 == 0 {
			extra = int64(oldCfg.MainPReps(icstate.MainPRepExtra))
		}

		newCfg := icstate.NewPRepCountConfig(int(main), int(sub), int(extra))
		if !icstate.IsPRepCountConfigValid(newCfg) {
			return scoreresult.InvalidParameterError.Errorf(
				"InvalidPRepCount(main=%d,sub=%d,extra=%d)", main, sub, extra)
		}

		if oldCfg != newCfg {
			args := []struct {
				fn    func(value int64) error
				value int64
			}{
				{es.State.SetMainPRepCount, main},
				{es.State.SetSubPRepCount, sub},
				{es.State.SetExtraMainPRepCount, extra},
			}
			for i, arg := range args {
				if updated&(1<<i) > 0 {
					if err := arg.fn(arg.value); err != nil {
						return err
					}
				}
			}
			EmitPRepCountConfigSetEvent(cc, newCfg)
		}
	}
	return nil
}

func (es *ExtensionStateImpl) GetPRepCountConfig(names []string) (map[string]interface{}, error) {
	jso := make(map[string]interface{})

	if len(names) == 0 {
		jso["main"] = es.State.GetMainPRepCount()
		jso["sub"] = es.State.GetSubPRepCount()
		jso["extra"] = es.State.GetExtraMainPRepCount()
	} else {
		for _, name := range names {
			if _, ok := jso[name]; ok {
				return nil, icmodule.DuplicateError.Errorf("DuplicateName(%s)", name)
			}

			switch name {
			case "main":
				jso[name] = es.State.GetMainPRepCount()
			case "sub":
				jso[name] = es.State.GetSubPRepCount()
			case "extra":
				jso[name] = es.State.GetExtraMainPRepCount()
			default:
				return nil, scoreresult.InvalidParameterError.Errorf("UnknownName(%s)", name)
			}
		}
	}

	return jso, nil
}