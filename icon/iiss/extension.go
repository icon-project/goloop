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
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/service/state"
)

const (
	VarAccount = "account"
	VarPRep = "prep"
)

type extensionSnapshotImpl struct {
	database db.Database

	state trie.Immutable	// TODO rename?
	//front trie.Immutable
	//back trie.Immutable
	//base trie.Immutable
}

func (s *extensionSnapshotImpl) Bytes() []byte {
	// TODO add front, back and base
	return s.state.Hash()
}

func (s *extensionSnapshotImpl) Flush() error {
	if ss, ok := s.state.(trie.Snapshot); ok {
		if err := ss.Flush(); err != nil {
			return err
		}
	}
	// TODO add front, back and base
	return nil
}

func (s *extensionSnapshotImpl) NewState(readonly bool) state.ExtensionState {
	es := &ExtensionStateImpl{
		Database: s.database,
	}
	es.Reset(s)
	return es
}

func NewExtensionSnapshot(database db.Database, hash []byte) state.ExtensionSnapshot {
	s := &extensionSnapshotImpl{
		database: database,
	}

	s.state = trie_manager.NewImmutable(database, hash)
	return s
}

type ExtensionStateImpl struct {
	Database db.Database

	state		trie.Mutable	// TODO rename?
	//front trie.Mutable
	//back trie.Mutable
	//base trie.Mutable
}

func (s *ExtensionStateImpl) GetSnapshot() state.ExtensionSnapshot {
	var iissState trie.Immutable
	if s.state != nil {
		iissState = s.state.GetSnapshot()
		//if iissState.Empty() {
		//	iissState = nil
		//}
	}
	return &extensionSnapshotImpl{
		database: s.Database,
		state: iissState,
	}
}

func (s *ExtensionStateImpl) Reset(isnapshot state.ExtensionSnapshot) {
	log.Debugf("ExtensionStateImpl.Reset() called with %v", isnapshot)
	snapshot, ok := isnapshot.(*extensionSnapshotImpl)
	if !ok {
		log.Panicf("It tries to Reset with invalid snapshot type=%T", s)
	}

	if snapshot.state == nil {
		s.state = nil
	} else if s.state == nil {
		s.state = trie_manager.NewMutableFromImmutable(snapshot.state)
	} else {
		if err := s.state.Reset(snapshot.state); err != nil {
			log.Panicf("Fail to make ExtensionStateImpl err=%v", err)
		}
	}
	log.Debugf("ExtensionStateImpl.Reset() make state %v", s)
}

func addressIDToKey(id []byte) []byte {
	if id == nil {
		return []byte("genesis")
	}
	return crypto.SHA3Sum256(id)
}

func (s *ExtensionStateImpl) GetAccountState(id []byte) AccountState {
	//key := addressIDToKey(id)
	//bs, err := s.state.Get(key)
	//if err != nil {
	//	log.Errorf("Fail to get account for %x err=%+v", key, err)
	//	return nil
	//}
	//var as *AccountStateImpl
	//if bs != nil {
	//	as = bs.(*AccountStateImpl)
	//}
	//ac := newAccountState(ws.Database, as, key, ws.nodeCacheEnabled)
	//return ac
	return nil
}

func (s *ExtensionStateImpl) GetValue(key []byte) ([]byte, error) {
	if s.state == nil {
		return nil, nil
	}
	return s.state.Get(key)
}

func (s *ExtensionStateImpl) SetValue(key []byte, value []byte) ([]byte, error) {
	if s.state == nil {
		s.state = trie_manager.NewMutable(s.Database, nil)
	}
	return s.state.Set(key, value)
}

func (s *ExtensionStateImpl) DeleteValue(key []byte) ([]byte, error) {
	return s.state.Delete(key)
}

func NewExtensionState(database db.Database, hash []byte) state.ExtensionState {
	s := &ExtensionStateImpl{
		Database: database,
	}
	s.state = trie_manager.NewMutable(database, hash)
	return nil
}