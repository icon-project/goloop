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
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/service/scoredb"
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

type ExtensionSnapshotImpl struct {
	database db.Database

	// TODO move to icstate?
	c  *calculation
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
		s.c,
		s.state.Bytes(),
		s.front.Bytes(),
		s.back.Bytes(),
		s.reward.Bytes(),
	)
}

func (s *ExtensionSnapshotImpl) RLPDecodeSelf(d codec.Decoder) error {
	var stateHash, frontHash, backHash, rewardHash []byte
	if err := d.DecodeListOf(&s.c, &stateHash, &frontHash, &backHash, &rewardHash); err != nil {
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
		c:        s.c,
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

func (s *ExtensionSnapshotImpl) State() *icstate.Snapshot {
	return s.state
}

func NewExtensionSnapshot(database db.Database, hash []byte) state.ExtensionSnapshot {
	if hash == nil {
		return &ExtensionSnapshotImpl{
			database: database,
			c:        newCalculation(),
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

	c  *calculation
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
		c:        s.c,
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
}

func (s *ExtensionStateImpl) ClearCache() {
	// TODO clear cached objects
	// It is called whenever executing a transaction is finish
}

func (s *ExtensionStateImpl) GetAccount(address module.Address) (*icstate.Account, error) {
	return s.State.GetAccount(address)
}

func (s *ExtensionStateImpl) GetUnstakingTimerState(height int64) (*icstate.Timer, error) {
	return s.State.GetUnstakingTimer(height)
}

func (s *ExtensionStateImpl) GetUnbondingTimerState(height int64) (*icstate.Timer, error) {
	return s.State.GetUnbondingTimer(height)
}

func (s *ExtensionStateImpl) AddUnbondingTimerToState(height int64) *icstate.Timer {
	return s.State.AddUnbondingTimerToCache(height)
}

func (s *ExtensionStateImpl) AddUnstakingTimerToState(height int64) *icstate.Timer {
	return s.State.AddUnstakingTimerToCache(height)
}

func (s *ExtensionStateImpl) CalculationBlockHeight() int64 {
	return s.c.currentBH
}

func (s *ExtensionStateImpl) PrevCalculationBlockHeight() int64 {
	return s.c.prevBH
}

func (s *ExtensionStateImpl) NewCalculationPeriod(blockHeight int64, calculator *Calculator) error {
	if blockHeight != s.c.currentBH+icstate.GetCalculatePeriod(s.State) {
		return nil
	}

	if !s.c.isCalcDone(calculator) {
		return scoreresult.ErrTimeout
	}

	// set offsetLimit
	if err := s.Front.AddGlobal(int(blockHeight - s.CalculationBlockHeight())); err != nil {
		return err
	}

	if _, err := s.Front.AddEventPeriod(
		0,
		icstate.GetIRep(s.State),
		icstate.GetRRep(s.State),
		icstate.GetMainPRepCount(s.State),
		icstate.GetPRepCount(s.State),
	); err != nil {
		return err
	}

	s.Back = s.Front
	s.Front = icstage.NewState(s.database)
	if calculator.result != nil {
		s.Reward = calculator.result.NewState()
		s.c.start(calculator.stats.totalReward(), blockHeight)
	}

	RegulateIssueInfo(s, s.c.rewardAmount)

	return nil
}

type calculation struct {
	currentBH    int64
	prevBH       int64
	rewardAmount *big.Int
}

func (c *calculation) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(
		c.currentBH,
		c.prevBH,
		c.rewardAmount,
	)
}

func (c *calculation) RLPDecodeSelf(d codec.Decoder) error {
	return d.DecodeListOf(
		&c.currentBH,
		&c.prevBH,
		&c.rewardAmount,
	)
}

func (c *calculation) isCalcDone(calculator *Calculator) bool {
	if c.currentBH == 0 {
		return true
	}
	return calculator.blockHeight == c.currentBH && calculator.result != nil
}

func (c *calculation) start(reward *big.Int, blockHeight int64) {
	c.prevBH = c.currentBH
	c.currentBH = blockHeight
	c.rewardAmount = reward
}

func newCalculation() *calculation {
	return &calculation{0, 0, nil}
}

func (s *ExtensionStateImpl) GetPRepsInJSON(blockHeight int64) map[string]interface{} {
	return s.pm.GetPRepsInJSON(blockHeight)
}

func (s *ExtensionStateImpl) GetPRepInJSON(address module.Address, blockHeight int64) (map[string]interface{}, error) {
	prep := s.pm.GetPRepByOwner(address)
	if prep == nil {
		return nil, errors.Errorf("PRep not found: %s", address)
	}

	return prep.ToJSON(blockHeight, s.pm.state.GetBondRequirement()), nil
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

	account, err = s.State.GetAccount(from)
	if err != nil {
		return err
	}
	if account.Stake().Cmp(new(big.Int).Add(ds.GetDelegationAmount(), account.Bond())) == -1 {
		return errors.Errorf("Not enough voting power")
	}
	account.SetDelegation(ds)
	err = s.pm.ChangeDelegation(account.Delegations(), ds)
	if err != nil {
		return err
	}

	bonds := account.Bonds()
	event := make([]*icstate.Delegation, 0, len(ds)+len(bonds))
	for _, d := range ds {
		event = append(event, d)
	}
	for _, b := range bonds {
		d := new(icstate.Delegation)
		d.Address = b.Address
		d.Value = b.Value
		event = append(event, d)
	}
	_, err = s.Front.AddEventDelegation(
		int(cc.BlockHeight()-s.CalculationBlockHeight()),
		from,
		event,
	)
	return nil
}

func (s *ExtensionStateImpl) UnregisterPRep(cc contract.CallContext, owner module.Address) error {
	prep := s.pm.GetPRepByOwner(owner)
	if prep == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}

	grade := prep.Grade()
	if grade != icstate.Candidate {
		term := s.State.Term()
		if err := term.RemovePRepSnapshot(owner); err != nil {
			return err
		}
		s.updateValidator = grade == icstate.Main
	}

	err := s.pm.UnregisterPRep(owner)
	if err != nil {
		return err
	}

	_, err = s.Front.AddEventEnable(
		int(cc.BlockHeight()-s.CalculationBlockHeight()),
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

	account, err = s.GetAccount(from)
	if err != nil {
		return err
	}

	bondAmount := big.NewInt(0)
	for _, bond := range bonds {
		bondAmount.Add(bondAmount, bond.Amount())

		prep := s.pm.GetPRepByOwner(bond.To())
		if prep == nil {
			return errors.Errorf("PRep not found: %v", from)
		}
		if !prep.BonderList().Contains(from) {
			return errors.Errorf("%s is not in bonder List of %s", from.String(), bond.Address.String())
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

	err = s.pm.ChangeBond(account.Bonds(), bonds)
	if err != nil {
		return err
	}

	account.SetBonds(bonds)
	tl := account.UpdateUnbonds(ubToAdd, ubToMod)
	for _, t := range tl {
		ts, e := s.State.GetUnbondingTimer(t.Height)
		if e != nil {
			return errors.Errorf("Error while getting unbonding Timer")
		} else if ts == nil {
			ts = s.State.AddUnbondingTimerToCache(t.Height)
		}
		if err = icstate.ScheduleTimerJob(ts, t, from); err != nil {
			return errors.Errorf("Error while scheduling Unbonding Timer Job")
		}
	}

	ds := account.Delegations()
	event := make([]*icstate.Delegation, 0, len(ds)+len(bonds))
	for _, d := range ds {
		event = append(event, d)
	}
	for _, b := range bonds {
		d := new(icstate.Delegation)
		d.Address = b.Address
		d.Value = b.Value
		event = append(event, d)
	}
	_, err = s.Front.AddEventDelegation(
		int(blockHeight-s.CalculationBlockHeight()),
		from,
		event,
	)
	return nil
}

func (s *ExtensionStateImpl) SetBonderList(from module.Address, bl icstate.BonderList) error {
	pb := s.State.GetPRepBase(from)
	if pb == nil {
		return errors.Errorf("PRep not found: %v", from)
	}

	var account *icstate.Account
	var err error
	for _, old := range pb.BonderList() {
		if !bl.Contains(old) {
			account, err = s.GetAccount(old)
			if err != nil {
				return err
			}
			if len(account.Bonds()) > 0 || len(account.Unbonds()) > 0 {
				return errors.Errorf("Bonding/Unbonding exist. bonds : %d, unbonds : %d", len(account.Bonds()), len(account.Unbonds()))
			}
		}
	}

	pb.SetBonderList(bl)
	return nil
}

func (s *ExtensionStateImpl) GetBonderList(address module.Address) ([]interface{}, error) {
	pb := s.State.GetPRepBase(address)
	if pb == nil {
		return nil, errors.Errorf("PRep not found: %v", address)
	}
	return pb.GetBonderListInJSON(), nil
}

func (s *ExtensionStateImpl) OnExecutionEnd(wc state.WorldContext) error {
	var err error
	term := s.State.Term()
	if term == nil {
		return nil
	}

	blockHeight := wc.BlockHeight()
	if blockHeight == term.GetEndBlockHeight() {
		err = s.onTermEnd(wc)
		if err != nil {
			panic(err)
		}

		// NextTerm
		term = s.State.Term()
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
	nextTerm, err := term.NewNextTerm(totalSupply, s.pm.totalDelegated)
	if err != nil {
		return err
	}

	size := 0
	mainPRepCount := int(icstate.GetMainPRepCount(s.State))
	activePRepCount := s.pm.Size()

	if term.IsDecentralized() || activePRepCount >= mainPRepCount {
		prepCount := int(icstate.GetPRepCount(s.State))
		size = icutils.Min(activePRepCount, prepCount)
	}

	if size > 0 {
		prepSnapshots := make(icstate.PRepSnapshots, size, size)
		for i := 0; i < size; i++ {
			prep := s.pm.GetPRepByIndex(i)
			prepSnapshots[i] = icstate.NewPRepSnapshotFromPRepStatus(prep.PRepStatus)
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
	mainPRepCount := s.State.GetMainPRepCount()

	term := s.State.GetTerm()
	prepSnapshotCount := term.GetPRepSnapshotCount()
	if prepSnapshotCount < mainPRepCount {
		log.Warnf("Not enough PReps: %d < %d", prepSnapshotCount, mainPRepCount)
	}

	var err error
	size := icutils.Min(mainPRepCount, prepSnapshotCount)
	validators := make([]module.Validator, size, size)

	for i := 0; i < size; i++ {
		prepSnapshot := term.GetPRepSnapshot(i)
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
	term := s.State.Term()
	if term == nil {
		err := errors.Errorf("Term is nil")
		return nil, err
	}
	return term.ToJSON(), nil
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
	return s.State.Term().IsDecentralized()
}
