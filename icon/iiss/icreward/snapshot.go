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

package icreward

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

type Snapshot struct {
	store *icobject.ObjectStoreSnapshot
}

func (ss *Snapshot) Flush() error {
	if sso, ok := ss.store.ImmutableForObject.(trie.SnapshotForObject); ok {
		return sso.Flush()
	}
	return nil
}

func (ss *Snapshot) Bytes() []byte {
	return ss.store.Hash()
}

func (ss *Snapshot) GetDSA() (*DSA, error) {
	if sso, ok := ss.store.ImmutableForObject.(trie.SnapshotForObject); ok {
		obj, err := sso.Get(DSAKey)
		if err != nil {
			return nil, err
		}
		if obj == nil {
			return NewDSA(), nil
		}
		return ToDSA(obj), nil
	}
	return nil, nil
}

func (ss *Snapshot) GetPublicKey(addr module.Address) (*PublicKey, error) {
	if sso, ok := ss.store.ImmutableForObject.(trie.SnapshotForObject); ok {
		key := PubKeyKey.Append(addr).Build()
		obj, err := sso.Get(key)
		if err != nil {
			return nil, err
		}
		if obj == nil {
			return NewPublicKey(), nil
		}
		return ToPublicKey(obj), nil
	}
	return nil, nil
}

func (ss *Snapshot) Filter(prefix []byte) trie.IteratorForObject {
	return ss.store.Filter(prefix)
}

func (ss *Snapshot) NewState() *State {
	return NewStateFromSnapshot(ss)
}

func NewSnapshot(database db.Database, hash []byte) *Snapshot {
	database = icobject.AttachObjectFactory(database, NewObjectImpl)
	t := trie_manager.NewImmutableForObject(database, hash, icobject.ObjectType)
	return &Snapshot{
		store: icobject.NewObjectStoreSnapshot(t),
	}
}

func NewSnapshotWithBuilder(builder merkle.Builder, hash []byte) *Snapshot {
	database := icobject.AttachObjectFactory(builder.Database(), NewObjectImpl)
	t := trie_manager.NewImmutableForObject(database, hash, icobject.ObjectType)
	t.Resolve(builder)
	return &Snapshot{
		store: icobject.NewObjectStoreSnapshot(t),
	}
}
