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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestTimerSnapshot_Bytes(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	t1 := newTimer()
	t1.Add(common.NewAccountAddress([]byte("1")))
	t1.Add(common.NewAccountAddress([]byte("2")))
	t1.Add(common.NewAccountAddress([]byte("3")))

	ts1 := t1.GetSnapshot()
	o1 := icobject.New(TypeTimer, ts1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}
	assert.True(t, o2.Equal(o1))
}

func TestTimer_Add(t *testing.T) {
	timer := newTimer()
	tc1 := []*common.Address {
		common.NewAccountAddress([]byte("1")),
		common.NewAccountAddress([]byte("2")),
		common.NewAccountAddress([]byte("3")),
	}
	for _, a := range tc1 {
		timer.Add(a)
	}

	for _, a := range tc1 {
		assert.True(t, timer.Contains(a))
	}

	var ret1 []*common.Address
	for itr := timer.Iterator(); itr.Has() ; itr.Next() {
		a, ok := itr.Get()
		assert.True(t, ok)
		ret1 = append(ret1, common.AddressToPtr(a))
	}
	assert.Equal(t, tc1, ret1)
}

func TestTimer_Delete(t *testing.T) {
	timer := newTimer()
	tc1 := []*common.Address {
		common.NewAccountAddress([]byte("1")),
		common.NewAccountAddress([]byte("2")),
		common.NewAccountAddress([]byte("3")),
		common.NewAccountAddress([]byte("4")),
	}
	for _, a := range tc1 {
		timer.Add(a)
	}

	for _, a := range tc1 {
		assert.True(t, timer.Contains(a))
	}

	timer.Delete(tc1[1])
	assert.False(t, timer.Contains(tc1[1]))

	timer.Delete(tc1[0])
	assert.False(t, timer.Contains(tc1[0]))

	timer.Delete(tc1[2])
	assert.False(t, timer.Contains(tc1[2]))

	timer.Delete(tc1[3])
	assert.False(t, timer.Contains(tc1[3]))
}
