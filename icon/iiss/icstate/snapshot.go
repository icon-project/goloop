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

package icstate

import (
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

type Snapshot struct {
	trie trie.ImmutableForObject
}

var (
	accountPrefix    = scoredb.ToKey(scoredb.DictDBPrefix, "account_db")
	prepPrefix       = scoredb.ToKey(scoredb.DictDBPrefix, "prep")
	prepStatusPrefix = scoredb.ToKey(scoredb.DictDBPrefix, "prep_status")
)

func (ss *Snapshot) Bytes() []byte {
	return ss.trie.Hash()
}

func (ss *Snapshot) Flush() error {
	if s, ok := ss.trie.(trie.SnapshotForObject); ok {
		return s.Flush()
	}
	return nil
}

func (ss *Snapshot) GetAccountSnapshot(addr module.Address) (*AccountSnapshot, error) {
	key := crypto.SHA3Sum256(scoredb.AppendKeys(accountPrefix, addr))
	obj, err := icobject.GetFromImmutableForObject(ss.trie, key)
	if err != nil {
		return nil, err
	}
	return ToAccountSnapshot(obj), nil
}

func (ss *Snapshot) GetPRepSnapshot(addr module.Address) (*PRepSnapshot, error) {
	key := crypto.SHA3Sum256(scoredb.AppendKeys(prepPrefix, addr))
	obj, err := icobject.GetFromImmutableForObject(ss.trie, key)
	if err != nil {
		return nil, err
	}
	return ToPRepSnapshot(obj), nil
}

func (ss *Snapshot) GetPRepStatusSnapshot(addr module.Address) (*PRepStatusSnapshot, error) {
	key := crypto.SHA3Sum256(scoredb.AppendKeys(prepStatusPrefix, addr))
	obj, err := icobject.GetFromImmutableForObject(ss.trie, key)
	if err != nil {
		return nil, err
	}
	return ToPRepStatusSnapshot(obj), nil
}

func (ss *Snapshot) NewState() *State {
	return NewStateFromSnapshot(ss)
}

func NewSnapshot(dbase db.Database, h []byte) *Snapshot {
	dbase = icobject.AttachObjectFactory(dbase, newObjectImpl)
	return &Snapshot{
		trie: trie_manager.NewImmutableForObject(dbase, h, icobject.ObjectType),
	}
}
