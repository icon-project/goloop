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
	"github.com/icon-project/goloop/service/contract"
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

type ExtensionSnapshotImpl struct {
	database db.Database

	c *calculation

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
	// TODO readonly?
	return &ExtensionStateImpl{
		database: s.database,
		c:        s.c,
		state:    icstate.NewStateFromSnapshot(s.state, readonly),
		Front:    icstage.NewStateFromSnapshot(s.front),
		Back:     icstage.NewStateFromSnapshot(s.back),
		Reward:   icreward.NewStateFromSnapshot(s.reward),
	}
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

	c *calculation

	state  *icstate.State
	Front  *icstage.State
	Back   *icstage.State
	Reward *icreward.State
}

func (s *ExtensionStateImpl) GetSnapshot() state.ExtensionSnapshot {
	return &ExtensionSnapshotImpl{
		database: s.database,
		c:        s.c,
		state:    s.state.GetSnapshot(),
		front:    s.Front.GetSnapshot(),
		back:     s.Back.GetSnapshot(),
		reward:   s.Reward.GetSnapshot(),
	}
}

func (s *ExtensionStateImpl) Reset(isnapshot state.ExtensionSnapshot) {
	snapshot := isnapshot.(*ExtensionSnapshotImpl)
	if err := s.state.Reset(snapshot.state); err != nil {
		panic(err)
	}
	s.Front.Reset(snapshot.front)
}

func (s *ExtensionStateImpl) ClearCache() {
	// TODO clear cached objects
	// It is called whenever executing a transaction is finish
}

func (s *ExtensionStateImpl) GetAccount(address module.Address) (*icstate.Account, error) {
	return s.state.GetAccount(address)
}

func (s *ExtensionStateImpl) GetUnstakingTimerState(height int64) (*icstate.TimerState, error) {
	return s.state.GetUnstakingTimerState(height)
}

func (s *ExtensionStateImpl) GetUnbondingTimerState(height int64) (*icstate.TimerState, error) {
	return s.state.GetUnbondingTimerState(height)
}

func (s *ExtensionStateImpl) CalculationBlockHeight() int64 {
	return s.c.currentBH
}

func (s *ExtensionStateImpl) PrevCalculationBlockHeight() int64 {
	return s.c.prevBH
}

// FIXME temp implementation
func calculationPeriod() int {
	return 3
}

func (s *ExtensionStateImpl) NewCalculationPeriod(blockHeight int64, calculator *Calculator) error {
	if blockHeight != s.c.currentBH+int64(calculationPeriod()) {
		return nil
	}

	if !s.c.isCalcDone(calculator) {
		return scoreresult.ErrTimeout
	}

	// set offsetLimit
	if err := s.Front.AddGlobal(int(blockHeight - s.CalculationBlockHeight())); err != nil {
		return err
	}

	// FIXME data for test
	if _, err := s.Front.AddEventPeriod(
		0,
		big.NewInt(YearBlock*IScoreICXRatio),
		big.NewInt(YearBlock*IScoreICXRatio),
	); err != nil {
		return err
	}
	// FIXME data for test

	s.Back = s.Front
	s.Front = icstage.NewState(s.database)
	if calculator.result != nil {
		s.Reward = calculator.result.NewState()
		s.c.start(calculator.result.Bytes(), blockHeight)
	} else {
		s.Reward = icreward.NewSnapshot(s.database, s.c.resultHash).NewState()
		s.c.start(s.c.resultHash, blockHeight)
	}

	return nil
}

type calculation struct {
	resultHash []byte
	currentBH  int64
	prevBH     int64
}

func (c *calculation) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(
		c.resultHash,
		c.currentBH,
		c.prevBH,
	)
}

func (c *calculation) RLPDecodeSelf(d codec.Decoder) error {
	return d.DecodeListOf(
		&c.resultHash,
		&c.currentBH,
		&c.prevBH,
	)
}

func (c *calculation) isCalcDone(calculator *Calculator) bool {
	if c.currentBH == 0 {
		return true
	}
	return calculator.blockHeight == c.currentBH && calculator.result != nil
}

func (c *calculation) start(resultHash []byte, blockHeight int64) {
	c.prevBH = c.currentBH
	c.currentBH = blockHeight
	c.resultHash = resultHash
}

func newCalculation() *calculation {
	return &calculation{nil, 0, 0}
}

func (s *ExtensionStateImpl) GetPRepsInJSON() map[string]interface{} {
	return s.state.GetPRepsInJSON()
}

func (s *ExtensionStateImpl) GetPRepInJSON(address module.Address) (map[string]interface{}, error) {
	return s.state.GetPRepInJSON(address)
}

func (s *ExtensionStateImpl) GetValidators() []module.Validator {
	return s.state.GetValidators()
}

func (s *ExtensionStateImpl) RegisterPRep(owner, node module.Address, params []string) error {
	return s.state.RegisterPRep(owner, node, params)
}

func (s *ExtensionStateImpl) SetDelegation(cc contract.CallContext, from module.Address, ds icstate.Delegations) error {
	var err error
	var account *icstate.Account

	err = s.state.SetDelegation(from, ds)
	if err != nil {
		return err
	}

	account, err = s.state.GetAccount(from)
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
	err := s.state.UnregisterPRep(owner)
	if err != nil {
		return err
	}

	_, err = s.Front.AddEventEnable(
		int(cc.BlockHeight()-s.CalculationBlockHeight()),
		owner,
		false,
	)
	return err
}

func (s *ExtensionStateImpl) SetPRep(from, node module.Address, params []string) error {
	return s.state.SetPRep(from, node, params)
}

func (s *ExtensionStateImpl) SetBond(cc contract.CallContext, from module.Address, bonds icstate.Bonds) error {
	var err error
	var account *icstate.Account
	blockHeight := cc.BlockHeight()

	err = s.state.SetBond(from, blockHeight, bonds)
	if err != nil {
		return err
	}

	account, err = s.state.GetAccount(from)
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
	return s.state.SetBonderList(from, bl)
}

func (s *ExtensionStateImpl) GetBonderList(address module.Address) ([]interface{}, error) {
	return s.state.GetBonderList(address)
}
