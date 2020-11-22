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
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

type extensionSnapshotImpl struct {
	database db.Database

	state *icstate.Snapshot
	front *icstage.Snapshot
	back  *icstage.Snapshot
	//base *icstate.Snapshot
}

func (s *extensionSnapshotImpl) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(s)
}

func (s *extensionSnapshotImpl) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(
		s.state.Bytes(),
		s.front.Bytes(),
		s.back.Bytes(),
	)
}

func (s *extensionSnapshotImpl) RLPDecodeSelf(d codec.Decoder) error {
	var stateHash, frontHash, backHash []byte
	if err := d.DecodeListOf(&stateHash, &frontHash, &backHash); err != nil {
		return err
	}
	s.state = icstate.NewSnapshot(s.database, stateHash)
	return nil
}

func (s *extensionSnapshotImpl) Flush() error {
	if err := s.state.Flush(); err != nil {
		return err
	}
	if err := s.front.Flush(); err != nil {
		return err
	}
	if err := s.back.Flush(); err != nil {
		return err
	}
	return nil
}

func (s *extensionSnapshotImpl) NewState(readonly bool) state.ExtensionState {
	// TODO readonly?
	return &ExtensionStateImpl{
		database: s.database,
		state:    icstate.NewStateFromSnapshot(s.state),
		front:    icstage.NewStateFromSnapshot(s.front),
		back:     icstage.NewStateFromSnapshot(s.back),
	}
}

func NewExtensionSnapshot(database db.Database, hash []byte) state.ExtensionSnapshot {
	if hash == nil {
		return &extensionSnapshotImpl{
			database: database,
			state:    icstate.NewSnapshot(database, nil),
		}
	}
	s := &extensionSnapshotImpl{
		database: database,
	}
	if _, err := codec.BC.UnmarshalFromBytes(hash, s); err != nil {
		return nil
	}
	return s
}

type ExtensionStateImpl struct {
	database db.Database
	state    *icstate.State
	front    *icstage.State
	back     *icstage.State
}

func (s *ExtensionStateImpl) GetSnapshot() state.ExtensionSnapshot {
	// TODO add front, back and base snapshot
	return &extensionSnapshotImpl{
		database: s.database,
		state:    s.state.GetSnapshot(),
		front:    s.front.GetSnapshot(),
		back:     s.back.GetSnapshot(),
	}
}

func (s *ExtensionStateImpl) Reset(isnapshot state.ExtensionSnapshot) {
	snapshot := isnapshot.(*extensionSnapshotImpl)
	if err := s.state.Reset(snapshot.state); err != nil {
		panic(err)
	}
}

func (s *ExtensionStateImpl) ClearCache() {
	// TODO clear cached objects
	// It is called whenever executing a transaction is done
}

func NewExtensionState(database db.Database, hash []byte) state.ExtensionState {
	s := &ExtensionStateImpl{
		database: database,
	}
	// TODO parse hash and make stateHolders
	return s
}

func (s *ExtensionStateImpl) GetAccountState(address module.Address) (*icstate.AccountState, error) {
	return s.state.GetAccountState(address)
}

func (s *ExtensionStateImpl) GetPRepState(address module.Address) (*icstate.PRepState, error) {
	return s.state.GetPRepState(address)
}
