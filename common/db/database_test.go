/*
 * Copyright 2021 ICON Foundation
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

package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testDatabase_GetSetDelete(t *testing.T, creator dbCreator) {
	dir := t.TempDir()
	testDB, _ := creator("test", dir)
	defer testDB.Close()

	key := []byte("hello")
	key2 := []byte("hell")
	value := []byte("world")

	bucket, _ := testDB.GetBucket("hello")

	// check it has nothing before set
	has, err := bucket.Has(key)
	assert.NoError(t, err)
	assert.False(t, has)

	// SET valid value
	err = bucket.Set(key, value)
	assert.NoError(t, err)

	// GET returns same value
	result, _ := bucket.Get(key)
	assert.Equal(t, value, result)

	// HAS returns true
	has, err = bucket.Has(key)
	assert.NoError(t, err)
	assert.True(t, has)

	// DELETE value
	err = bucket.Delete(key)
	assert.NoError(t, err)

	// GET returns nothing
	result, err = bucket.Get(key)
	assert.NoError(t, err)
	assert.Nil(t, result)

	// HAS returns false
	has, err = bucket.Has(key)
	assert.NoError(t, err)
	assert.False(t, has)

	// SET with empty bytes
	err = bucket.Set(key2, []byte{})
	assert.NoError(t, err)

	// HAS returns true
	has, err = bucket.Has(key2)
	assert.NoError(t, err)
	assert.True(t, has)

	// GET returns non-nil(empty)
	value, err = bucket.Get(key2)
	assert.NoError(t, err)
	assert.True(t, value != nil)
	assert.Zero(t, len(value))

	var emptyKeys = [][]byte{nil, {}}
	for _, k := range emptyKeys {
		// HAS return false before store it
		has, err = bucket.Has(k)
		assert.NoError(t, err)

		// GET returns nil before store it
		value, err = bucket.Get(k)
		assert.NoError(t, err)
		assert.Zero(t, value)

		// DELETE succeeds although it's not exists
		err = bucket.Delete(k)
		assert.NoError(t, err)

		// SET value with empty key succeeds
		value1 := []byte("TEST")
		err = bucket.Set(k, value1)
		assert.NoError(t, err)

		// check stored value for empty key
		for _, k2 := range emptyKeys {
			has, err = bucket.Has(k)
			assert.NoError(t, err)
			assert.True(t, has)

			value, err = bucket.Get(k2)
			assert.NoError(t, err)
			assert.Equal(t, value1, value)
		}

		// DELETE for next case
		err = bucket.Delete(k)
		assert.NoError(t, err)
	}
}

func TestDatabase_GetSetDelete(t *testing.T) {
	for name, be := range backends {
		t.Run(string(name), func(t *testing.T) {
			testDatabase_GetSetDelete(t, be)
		})
	}
	t.Run("layerdb", func(t *testing.T) {
		var creator dbCreator = func(name string, dir string) (Database, error) {
			origin := NewMapDB()
			return NewLayerDB(origin), nil
		}
		testDatabase_GetSetDelete(t, creator)
	})
}

func testDatabase_SetReopenGet(t *testing.T, creator dbCreator) {
	dir := t.TempDir()
	key := []byte("hello")
	key2 := []byte("hell")
	value := []byte("world")

	buckets := []BucketID{"hello", MerkleTrie, BytesByHash}
	testDB, err := creator("test", dir)
	assert.NoError(t, err)
	defer func() {
		if testDB != nil {
			testDB.Close()
		}
	}()

	for _, id := range buckets {
		bucket, err := testDB.GetBucket(id)
		assert.NoError(t, err)
		err = bucket.Set(key, value)
		assert.NoError(t, err)
	}
	err = testDB.Close()
	testDB = nil
	assert.NoError(t, err)

	testDB, err = creator("test", dir)
	assert.NoError(t, err)

	for _, id := range buckets {
		bucket, err := testDB.GetBucket(id)
		assert.NoError(t, err)
		stored, err := bucket.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, value, stored)

		stored, err = bucket.Get(key2)
		assert.NoError(t, err)
		assert.Nil(t, stored)

		has, err := bucket.Has(key2)
		assert.NoError(t, err)
		assert.False(t, has)
	}
}

func TestDatabase_SetReopenGet(t *testing.T) {
	for name, creator := range backends {
		if name == MapDBBackend {
			continue
		}
		t.Run(string(name), func(t *testing.T) {
			testDatabase_SetReopenGet(t, creator)
		})
	}
}

func testDatabase_OperationAfterClose(t *testing.T, creator dbCreator) {
	dir := t.TempDir()
	key := []byte("hello")
	value := []byte("world")
	value2 := []byte("world2")

	testDB, err := creator("test", dir)
	assert.NoError(t, err)

	bucket, err := testDB.GetBucket(MerkleTrie)
	assert.NoError(t, err)

	err = bucket.Set(key, value)
	assert.NoError(t, err)

	err = testDB.Close()
	assert.NoError(t, err)

	// all operations after close should fail.
	err = testDB.Close()
	assert.Error(t, err)

	bucket2, err := testDB.GetBucket(MerkleTrie)
	assert.Error(t, err)
	assert.Nil(t, bucket2)

	bs, err := bucket.Get(key)
	assert.Error(t, err)
	assert.Nil(t, bs)

	err = bucket.Delete(key)
	assert.Error(t, err)

	err = bucket.Set(key, value2)
	assert.Error(t, err)
}

func TestDatabase_OperationAfterClose(t *testing.T) {
	for name, creator := range backends {
		if name == MapDBBackend {
			continue
		}
		t.Run(string(name), func(t *testing.T) {
			testDatabase_OperationAfterClose(t, creator)
		})
	}
}
