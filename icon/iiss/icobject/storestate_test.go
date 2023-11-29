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

package icobject

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/trie/trie_manager"
)

func testFactory(_ Tag) (Impl, error) {
	return nil, errors.New("Unsupported")
}

func TestNewObjectStoreState(t *testing.T) {
	database := db.NewMapDB()
	database = AttachObjectFactory(database, testFactory)
	tree := trie_manager.NewMutableForObject(database, nil, ObjectType)

	oss := NewObjectStoreState(tree)
	key := containerdb.ToKey(containerdb.PrefixedHashBuilder, []byte{0x04}, "test")
	array := containerdb.NewArrayDB(oss, key)

	assert.Zero(t, array.Size())
	assert.NoError(t, array.Put("Test"))
	assert.NoError(t, array.Put(1))

	// check stored value.
	assert.Equal(t, 2, array.Size())
	assert.Equal(t, "Test", array.Get(0).String())
	assert.Equal(t, int64(1), array.Get(1).Int64())

	ss := tree.GetSnapshot()
	assert.NoError(t, ss.Flush())

	// check stored value with recovered mutable
	tree2 := trie_manager.NewMutableForObject(database, ss.Hash(), ObjectType)
	oss2 := NewObjectStoreState(tree2)
	array2 := containerdb.NewArrayDB(oss2, key)
	assert.Equal(t, 2, array2.Size())
	assert.Equal(t, "Test", array2.Get(0).String())
	assert.Equal(t, int64(1), array2.Get(1).Int64())

	// check stored value with snapshot
	oss3 := NewObjectStoreSnapshot(ss)
	array3 := containerdb.NewArrayDB(oss3, key)
	assert.Equal(t, 2, array3.Size())
	assert.Equal(t, "Test", array3.Get(0).String())
	assert.Equal(t, int64(1), array3.Get(1).Int64())
}
