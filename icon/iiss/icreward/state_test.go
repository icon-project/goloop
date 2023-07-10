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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstage"
)

func TestState_NewState(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)

	s := NewState(database, nil)

	addr1 := common.MustNewAddressFromString("hx1")
	iScore := NewIScore(big.NewInt(10))
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

func TestState_SetVoted(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)

	s := NewState(database, nil)

	addr1 := common.MustNewAddressFromString("hx1")
	voted1 := NewVoted()
	voted1.SetStatus(icstage.ESEnable)
	voted1.SetDelegated(big.NewInt(100))
	voted1.SetBondedDelegation(big.NewInt(100))
	err := s.SetVoted(addr1, voted1)
	assert.NoError(t, err)

	addr2 := common.MustNewAddressFromString("hx2")
	voted2 := NewVoted()
	voted2.SetStatus(icstage.ESDisablePermanent)
	voted2.SetDelegated(big.NewInt(200))
	voted2.SetBondedDelegation(big.NewInt(200))
	err = s.SetVoted(addr2, voted2)
	assert.NoError(t, err)

	v1, err := s.GetVoted(addr1)
	assert.NoError(t, err)
	assert.True(t, voted1.Equal(v1))
	v2, err := s.GetVoted(addr2)
	assert.NoError(t, err)
	assert.True(t, voted2.Equal(v2))

	ss := s.GetSnapshot()
	ss.Flush()

	prefix := VotedKey.Build()
	for iter := ss.Filter(prefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		assert.NoError(t, err)
		keySplit, err := containerdb.SplitKeys(key)
		assert.NoError(t, err)
		addr, err := common.NewAddress(keySplit[1])
		assert.NoError(t, err)
		obj := ToVoted(o)

		var v *Voted
		if addr.Equal(addr1) {
			v = voted1
		} else {
			v = voted2
		}
		assert.True(t, v.Equal(obj))
	}

	ss2 := NewSnapshot(database, ss.Bytes())
	for iter := ss2.Filter(prefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		assert.NoError(t, err)
		keySplit, err := containerdb.SplitKeys(key)
		assert.NoError(t, err)
		addr, err := common.NewAddress(keySplit[1])
		assert.NoError(t, err)
		obj := ToVoted(o)

		var v *Voted
		if addr.Equal(addr1) {
			v = voted1
		} else {
			v = voted2
		}
		assert.True(t, v.Equal(obj))
	}

	s2 := ss2.NewState()
	v1, err = s2.GetVoted(addr1)
	assert.NoError(t, err)
	assert.True(t, voted1.Equal(v1))
	v2, err = s2.GetVoted(addr2)
	assert.NoError(t, err)
	assert.True(t, voted2.Equal(v2))
}
