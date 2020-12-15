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

package icreward

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestState_NewState(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	s := NewState(database)

	addr1 := common.NewAddressFromString("hx1")
	iScore := NewIScore()
	iScore.Value.SetInt64(int64(10))
	s.SetIScore(addr1, iScore)

	is, err := s.GetIScore(addr1)
	assert.NoError(t, err)
	assert.NotNil(t, is)
	assert.True(t, iScore.Equal(is))

	ss := s.GetSnapshot()
	prefix := IScoreKey.Append(addr1).Build()
	for iter := ss.Filter(prefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		assert.NoError(t, err)
		assert.NotNil(t, o)
		assert.NotNil(t, key)

		obj := ToIScore(o)
		assert.NotNil(t, obj)
		assert.True(t, iScore.Equal(obj))
	}

	newState := ss.NewState()
	is, err = newState.GetIScore(addr1)
	assert.NoError(t, err)
	assert.NotNil(t, is)
	assert.True(t, iScore.Equal(is))
}

func TestState_SetDelegated(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	s := NewState(database)

	addr1 := common.NewAddressFromString("hx1")
	delegated1 := NewDelegated()
	delegated1.Enable = true
	delegated1.Current.SetInt64(100)
	delegated1.Snapshot.SetInt64(100)
	err := s.SetDelegated(addr1, delegated1)
	assert.NoError(t, err)

	addr2 := common.NewAddressFromString("hx2")
	delegated2 := NewDelegated()
	delegated2.Enable = false
	delegated2.Current.SetInt64(200)
	delegated2.Snapshot.SetInt64(200)
	err = s.SetDelegated(addr2, delegated2)
	assert.NoError(t, err)

	d1, err := s.GetDelegated(addr1)
	assert.NoError(t, err)
	assert.True(t, delegated1.Equal(d1))
	d2, err := s.GetDelegated(addr2)
	assert.NoError(t, err)
	assert.True(t, delegated2.Equal(d2))

	ss := s.GetSnapshot()
	ss.Flush()

	prefix := DelegatedKey.Build()
	for iter := ss.Filter(prefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		assert.NoError(t, err)
		keySplit, err := containerdb.SplitKeys(key)
		assert.NoError(t, err)
		addr, err := common.NewAddress(keySplit[1])
		assert.NoError(t, err)
		obj := ToDelegated(o)

		var d *Delegated
		if addr.Equal(addr1) {
			d = delegated1
		} else {
			d = delegated2
		}
		assert.True(t, d.Equal(obj))
	}

	ss2 := NewSnapshot(database, ss.Bytes())
	for iter := ss2.Filter(prefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		assert.NoError(t, err)
		keySplit, err := containerdb.SplitKeys(key)
		assert.NoError(t, err)
		addr, err := common.NewAddress(keySplit[1])
		assert.NoError(t, err)
		obj := ToDelegated(o)

		var d *Delegated
		if addr.Equal(addr1) {
			d = delegated1
		} else {
			d = delegated2
		}
		assert.True(t, d.Equal(obj))
	}

	s2 := ss2.NewState()
	d1, err = s2.GetDelegated(addr1)
	assert.NoError(t, err)
	assert.True(t, delegated1.Equal(d1))
	d2, err = s2.GetDelegated(addr2)
	assert.NoError(t, err)
	assert.True(t, delegated2.Equal(d2))
}
