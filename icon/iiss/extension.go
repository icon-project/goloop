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
	if err := s.c.flush(s.database); err != nil {
		return err
	}
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
		state:    icstate.NewStateFromSnapshot(s.state),
		Front:    icstage.NewStateFromSnapshot(s.front),
		back:     icstage.NewStateFromSnapshot(s.back),
		Reward:   icreward.NewStateFromSnapshot(s.reward),
	}
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

	c := newCalculation()
	c.load(s.database)
	s.c = c

	return s
}

type ExtensionStateImpl struct {
	database db.Database

	c *calculation

	state  *icstate.State
	Front  *icstage.State
	back   *icstage.State
	Reward *icreward.State
}

func (s *ExtensionStateImpl) State() *icstate.State {
	return s.state
}

func (s *ExtensionStateImpl) GetSnapshot() state.ExtensionSnapshot {
	return &ExtensionSnapshotImpl{
		database: s.database,
		c:        s.c,
		state:    s.state.GetSnapshot(),
		front:    s.Front.GetSnapshot(),
		back:     s.back.GetSnapshot(),
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
	// It is called whenever executing a transaction is done
}

func (s *ExtensionStateImpl) GetAccountState(address module.Address) (*icstate.AccountState, error) {
	return s.state.GetAccountState(address)
}

func (s *ExtensionStateImpl) GetPRepState(address module.Address) (*icstate.PRepState, error) {
	return s.state.GetPRepState(address)
}

func (s *ExtensionStateImpl) GetPRepStatusState(address module.Address) (*icstate.PRepStatusState, error) {
	return s.state.GetPRepStatusState(address)
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

func (s *ExtensionStateImpl) NewCalculationPeriod(blockHeight int64) error {
	if blockHeight != s.c.currentBH+int64(calculationPeriod()) {
		return nil
	}

	if !s.c.isCalcDone() {
		return scoreresult.ErrTimeout
	}

	// set offsetLimit
	if err := s.Front.AddGlobal(calculationPeriod()); err != nil {
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

	s.back = s.Front
	s.Front = icstage.NewState(s.database)
	s.Reward = icreward.NewSnapshot(s.database, s.c.resultHash).NewState()
	s.c.start(blockHeight)

	return nil
}

const keyCalculation = "iiss.calculation"

type calculation struct {
	run        bool
	resultHash []byte
	currentBH  int64
	prevBH     int64
}

func (c *calculation) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(
		c.run,
		c.resultHash,
		c.currentBH,
		c.prevBH,
	)
}

func (c *calculation) RLPDecodeSelf(d codec.Decoder) error {
	return d.DecodeListOf(
		&c.run,
		&c.resultHash,
		&c.currentBH,
		&c.prevBH,
	)
}

func (c *calculation) isCalcDone() bool {
	return c.currentBH == 0 || c.currentBH == c.prevBH
}

func (c *calculation) start(blockHeight int64) {
	c.currentBH = blockHeight
	c.run = true
}

func (c *calculation) stop() {
	c.run = false
}

func (c *calculation) done(hash []byte) {
	c.prevBH = c.currentBH
	c.resultHash = hash
}

func (c *calculation) load(dbase db.Database) error {
	bk, err := dbase.GetBucket(db.ChainProperty)
	if err != nil {
		return err
	}
	value, err := bk.Get([]byte(keyCalculation))
	if err != nil {
		return err
	}
	if len(value) == 0 {
		return nil
	}
	_, err = codec.BC.UnmarshalFromBytes(value, c)
	if err != nil {
		return err
	}
	return nil
}

func (c *calculation) flush(dbase db.Database) error {
	bk, err := dbase.GetBucket(db.ChainProperty)
	if err != nil {
		return err
	}
	data, err := codec.BC.MarshalToBytes(c)
	if err != nil {
		return err
	}
	if err = bk.Set([]byte(keyCalculation), data); err != nil {
		return err
	}
	return nil
}

func newCalculation() *calculation {
	return &calculation{false, nil, 0, 0}
}
