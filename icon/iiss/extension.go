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
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icstate"
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

	state *icstate.Snapshot
	//front *snapshotHolder
	//back *snapshotHolder
	//base *snapshotHolder
}

func (s *extensionSnapshotImpl) Bytes() []byte {
	// TODO add front, back and base
	return s.state.Bytes()
}

func (s *extensionSnapshotImpl) Flush() error {
	if err := s.state.Flush(); err != nil {
		return err
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
	s.state = icstate.NewSnapshot(database, hash)
	return s
}

type ExtensionStateImpl struct {
	database db.Database

	state *icstate.State
}

func (s *ExtensionStateImpl) GetSnapshot() state.ExtensionSnapshot {
	var is *icstate.Snapshot
	if s.state != nil {
		is = s.state.GetSnapshot()
	}
	// TODO add front, back and base snapshot
	return &extensionSnapshotImpl{
		database: s.database,
		state:    is,
	}
}

func (s *ExtensionStateImpl) Reset(isnapshot state.ExtensionSnapshot) {
	snapshot, ok := isnapshot.(*extensionSnapshotImpl)
	if !ok {
		log.Panicf("It tries to Reset with invalid snapshot type=%T", s)
	}

	if snapshot.state == nil {
		s.state = nil
	} else if s.state == nil {
		s.state = icstate.NewStateFromSnapshot(snapshot.state)
	} else {
		if err := s.state.Reset(snapshot.state); err != nil {
			log.Panicf("It tries to Reset with invalid snapshot type=%T", s)
		}
	}
}

func (s *ExtensionStateImpl) GetIISSPRepDB() *scoredb.DictDB {
	//return scoredb.NewDictDB(s.state, VarPRep, 1)
	return nil
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
