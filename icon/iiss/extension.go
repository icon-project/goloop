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

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

const (
	VarAccount = "account"
	VarPRep    = "prep"
)

type IconContext struct {
}

type iconContext struct {
	contract.CallContext
}

type extensionSnapshotImpl struct {
	database db.Database

	iissState *snapshotHolder
	//front *snapshotHolder
	//back *snapshotHolder
	//base *snapshotHolder
}

func (s *extensionSnapshotImpl) Bytes() []byte {
	// TODO add front, back and base
	return s.iissState.Bytes()
}

func (s *extensionSnapshotImpl) Flush() error {
	if ss, ok := s.iissState.state.(trie.Snapshot); ok {
		if err := ss.Flush(); err != nil {
			return err
		}
	}
	// TODO add front, back and base
	return nil
}

func (s *extensionSnapshotImpl) NewState(readonly bool) state.ExtensionState {
	// TODO readonly?
	es := &ExtensionStateImpl{
		database: s.database,
	}
	es.Reset(s)
	return es
}

func NewExtensionSnapshot(database db.Database, hash []byte) state.ExtensionSnapshot {
	s := &extensionSnapshotImpl{
		database: database,
	}

	// TODO parse hash and add front, back and base snapshot
	s.iissState = NewSnapshotHolder(database, hash)
	return s
}

type ExtensionStateImpl struct {
	database db.Database

	iissState *stateHolder
	//front *stateHolder
	//back *stateHolder
	//base *stateHolder
}

func (s *ExtensionStateImpl) GetSnapshot() state.ExtensionSnapshot {
	is := &snapshotHolder{
		database: s.database,
	}
	if s.iissState != nil {
		is.state = s.iissState.GetSnapshot()
		//if iissState.iissState.Empty() {
		//	iissState.iissState = nil
		//}
	}
	// TODO add front, back and base snapshot
	return &extensionSnapshotImpl{
		database:  s.database,
		iissState: is,
	}
}

func (s *ExtensionStateImpl) Reset(isnapshot state.ExtensionSnapshot) {
	snapshot, ok := isnapshot.(*extensionSnapshotImpl)
	if !ok {
		log.Panicf("It tries to Reset with invalid snapshot type=%T", s)
	}

	if snapshot.iissState == nil {
		s.iissState = nil
	} else if s.iissState == nil {
		s.iissState = NewStateHolder(snapshot.iissState.database, snapshot.iissState.state)
	} else {
		s.iissState.Reset(snapshot.iissState)
	}
}

func (s *ExtensionStateImpl) GetIISSAccountDB() *scoredb.DictDB {
	// TODO wrap DB API
	return scoredb.NewDictDB(s.iissState, VarAccount, 1)
}

func (s *ExtensionStateImpl) GetIISSAccount(database *scoredb.DictDB, address module.Address) (*Account, error) {
	as := NewAccount()
	if bs := database.Get(address); bs != nil {
		if err := as.SetBytes(bs.Bytes()); err != nil {
			return nil, err
		}
	}
	return as, nil
}

func (s *ExtensionStateImpl) SetStake(cc contract.CallContext, from module.Address, v *big.Int) error {
	aDB := s.GetIISSAccountDB()
	ia, err := s.GetIISSAccount(aDB, from)
	if err != nil {
		return err
	}

	if ia.getVotedPower().Cmp(v) == 1 {
		return errors.Errorf("Failed to stake: stake < votedPower")
	}

	stakeInc := new(big.Int).Sub(v, ia.GetStake())
	if stakeInc.Sign() == 0 {
		return nil
	}

	account := cc.GetAccountState(from.ID())
	balance := account.GetBalance()
	if balance.Cmp(v) == -1 {
		return errors.Errorf("Not enough balance")
	}

	expireHeight := s.calcUnstakeLockPeriod(cc.BlockHeight())
	if err := ia.UpdateUnstake(stakeInc, expireHeight); err != nil {
		return err
	}
	account.SetBalance(new(big.Int).Sub(balance, stakeInc))
	if err = ia.SetStake(v); err != nil {
		return err
	}
	return aDB.Set(from, ia.Bytes())
}

func (s *ExtensionStateImpl) calcUnstakeLockPeriod(blockHeight int64) int64 {
	// TODO implement me
	return blockHeight + 10
}

func (s *ExtensionStateImpl) GetStake(address module.Address) (map[string]interface{}, error) {
	aDB := s.GetIISSAccountDB()
	as, err := s.GetIISSAccount(aDB, address)
	if err != nil {
		return nil, err
	}
	return as.GetStakeInfo()
}

func (s *ExtensionStateImpl) SetDelegation(cc contract.CallContext, from module.Address, param []interface{}) error {
	aDB := s.GetIISSAccountDB()
	ia, err := s.GetIISSAccount(aDB, from)
	if err != nil {
		return err
	}

	if err = ia.SetDelegation(param); err != nil {
		return err
	}

	return aDB.Set(from, ia.Bytes())
}

func (s *ExtensionStateImpl) GetDelegation(address module.Address) (map[string]interface{}, error) {
	aDB := s.GetIISSAccountDB()
	as, err := s.GetIISSAccount(aDB, address)
	if err != nil {
		return nil, err
	}
	return as.GetDelegationInfo()
}

func (s *ExtensionStateImpl) GetIISSPRepDB() *scoredb.DictDB {
	return scoredb.NewDictDB(s.iissState, VarPRep, 1)
}

func (s *ExtensionStateImpl) GetIISSPRepState(database *scoredb.DictDB, address module.Address) (PRepState, error) {
	ps := NewPRepState()
	if bs := database.Get(address); bs != nil {
		if err := ps.SetBytes(bs.Bytes()); err != nil {
			return nil, err
		}
	}
	return ps, nil
}

func (s *ExtensionStateImpl) RegisterPRep(cc contract.CallContext, from module.Address, name string, email string,
	website string, country string, city string, details string, endpoint string, node module.Address,
) error {
	pDB := s.GetIISSPRepDB()
	ps, err := s.GetIISSPRepState(pDB, from)
	if err != nil {
		return err
	}
	if err = ps.SetPRep(name, email, website, country, city, details, endpoint, node); err != nil {
		return err
	}
	return pDB.Set(from, ps.Bytes())
}

func (s *ExtensionStateImpl) GetPRep(address module.Address) (map[string]interface{}, error) {
	pDB := s.GetIISSPRepDB()
	ps, err := s.GetIISSPRepState(pDB, address)
	if err != nil {
		return nil, err
	}
	return ps.GetPRep(), nil
}

func NewExtensionState(database db.Database, hash []byte) state.ExtensionState {
	s := &ExtensionStateImpl{
		database: database,
	}
	// TODO parse hash and make stateHolders
	return s
}

type snapshotHolder struct {
	database db.Database
	state    trie.Immutable
}

func (s *snapshotHolder) Bytes() []byte {
	return s.state.Hash()
}

func NewSnapshotHolder(database db.Database, hash []byte) *snapshotHolder {
	s := &snapshotHolder{
		database: database,
	}
	s.state = trie_manager.NewImmutable(database, hash)
	return s
}

type stateHolder struct {
	database db.Database
	state    trie.Mutable
}

func (s *stateHolder) GetSnapshot() trie.Snapshot {
	return s.state.GetSnapshot()
}

func (s *stateHolder) Reset(snapshot *snapshotHolder) {
	if snapshot.state == nil {
		s.state = nil
	} else if s.state == nil {
		s.state = trie_manager.NewMutableFromImmutable(snapshot.state)
	} else {
		if err := s.state.Reset(snapshot.state); err != nil {
			log.Panicf("Fail to make ExtensionStateImpl err=%v", err)
		}
	}
}

func (s *stateHolder) GetValue(key []byte) ([]byte, error) {
	if s.state == nil {
		return nil, nil
	}
	return s.state.Get(key)
}

func (s *stateHolder) SetValue(key []byte, value []byte) ([]byte, error) {
	if s.state == nil {
		s.state = trie_manager.NewMutable(s.database, nil)
	}
	return s.state.Set(key, value)
}

func (s *stateHolder) DeleteValue(key []byte) ([]byte, error) {
	return s.state.Delete(key)
}

func NewStateHolder(database db.Database, object trie.Immutable) *stateHolder {
	s := &stateHolder{
		database: database,
	}
	s.state = trie_manager.NewMutableFromImmutable(object)

	return s
}
