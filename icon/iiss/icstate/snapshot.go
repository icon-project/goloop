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
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type Snapshot struct {
	store *icobject.ObjectStoreSnapshot
}

func (ss *Snapshot) Bytes() []byte {
	return ss.store.Hash()
}

func (ss *Snapshot) Flush() error {
	if s, ok := ss.store.ImmutableForObject.(trie.SnapshotForObject); ok {
		return s.Flush()
	}
	return nil
}

func (ss *Snapshot) GetValue(key []byte) ([]byte, error) {
	var value []byte
	o, err := ss.store.Get(key)
	if o != nil {
		value = o.Bytes()
	}

	if err != nil {
		return value, err
	}

	return value, nil
}

func NewSnapshot(dbase db.Database, h []byte) *Snapshot {
	dbase = icobject.AttachObjectFactory(dbase, NewObjectImpl)
	t := trie_manager.NewImmutableForObject(dbase, h, icobject.ObjectType)
	return newSnapshotFromImmutableForObject(t)
}

func NewSnapshotWithBuilder(builder merkle.Builder, h []byte) *Snapshot {
	dbase := icobject.AttachObjectFactory(builder.Database(), NewObjectImpl)
	t := trie_manager.NewImmutableForObject(dbase, h, icobject.ObjectType)
	t.Resolve(builder)
	return newSnapshotFromImmutableForObject(t)
}

func newSnapshotFromImmutableForObject(t trie.ImmutableForObject) *Snapshot {
	return &Snapshot{icobject.NewObjectStoreSnapshot(t)}
}
