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
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTimerSnapshot_Bytes(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	t1 := newTimerWithTag(icobject.MakeTag(TypeTimer, timerVersion))
	t1.Height = 10

	al1 := make(addresses, 3)
	a1 := common.NewAccountAddress([]byte("1"))
	a2 := common.NewAccountAddress([]byte("2"))
	a3 := common.NewAccountAddress([]byte("3"))
	al1 = append(al1, a1)
	al1 = append(al1, a2)
	al1 = append(al1, a3)

	t1.Addresses = al1

	o1 := icobject.New(TypeTimer, t1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}
	assert.True(t, o2.Equal(o1))
}

func TestTimer_Add(t *testing.T) {
	timer := newTimerWithTag(icobject.MakeTag(TypeTimer, timerVersion))
	timer.Height = 10

	a1 := common.NewAccountAddress([]byte("1"))
	a2 := common.NewAccountAddress([]byte("2"))
	a3 := common.NewAccountAddress([]byte("3"))
	timer.Add(a1)
	timer.Add(a2)
	timer.Add(a3)
	assert.Contains(t, timer.Addresses, a1)
	assert.Contains(t, timer.Addresses, a2)
	assert.Contains(t, timer.Addresses, a3)
}

func TestTimer_Delete(t *testing.T) {
	timer := newTimerWithTag(icobject.MakeTag(TypeTimer, timerVersion))
	timer.Height = 10

	a1 := common.NewAccountAddress([]byte("1"))
	a2 := common.NewAccountAddress([]byte("2"))
	a3 := common.NewAccountAddress([]byte("3"))
	a4 := common.NewAccountAddress([]byte("4"))
	timer.Add(a1)
	timer.Add(a2)
	timer.Add(a3)
	timer.Add(a4)
	assert.Contains(t, timer.Addresses, a1)
	assert.Contains(t, timer.Addresses, a2)
	assert.Contains(t, timer.Addresses, a3)
	assert.Contains(t, timer.Addresses, a4)
	length := len(timer.Addresses)

	err := timer.Delete(a2)
	assert.NoError(t, err)
	assert.NotContains(t, timer.Addresses, a2)
	assert.Equal(t, length-1, len(timer.Addresses))
	length -= 1

	err = timer.Delete(a2)
	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), errors.Errorf("%s not in timer", a2.String()).Error())
	}

	err = timer.Delete(a4)
	assert.NoError(t, err)
	assert.NotContains(t, timer.Addresses, a4)
	assert.Equal(t, length-1, len(timer.Addresses))
	length -= 1

	err = timer.Delete(a1)
	assert.NoError(t, err)
	assert.NotContains(t, timer.Addresses, a1)
	assert.Equal(t, length-1, len(timer.Addresses))
	length -= 1

	err = timer.Delete(a3)
	assert.NoError(t, err)
	assert.NotContains(t, timer.Addresses, a3)
	assert.Equal(t, length-1, len(timer.Addresses))
}

func Test_ScheduleTimerJob(t *testing.T) {
	timer := newTimerWithTag(icobject.MakeTag(TypeTimer, timerVersion))

	a1 := common.NewAccountAddress([]byte("1"))
	a2 := common.NewAccountAddress([]byte("2"))
	a3 := common.NewAccountAddress([]byte("3"))
	j1 := &TimerJobInfo{JobTypeAdd, 1}
	length := 0

	assert.NotContains(t, timer.Addresses, a1)
	assert.NotContains(t, timer.Addresses, a2)
	assert.NotContains(t, timer.Addresses, a3)
	assert.Equal(t, length, len(timer.Addresses))

	err := ScheduleTimerJob(timer, *j1, a1)
	assert.NoError(t, err)
	assert.Contains(t, timer.Addresses, a1)
	length += 1
	assert.Equal(t, length, len(timer.Addresses))

	err = ScheduleTimerJob(timer, *j1, a2)
	assert.NoError(t, err)
	assert.Contains(t, timer.Addresses, a2)
	length += 1
	assert.Equal(t, length, len(timer.Addresses))

	err = ScheduleTimerJob(timer, *j1, a3)
	assert.NoError(t, err)
	assert.Contains(t, timer.Addresses, a3)
	length += 1
	assert.Equal(t, length, len(timer.Addresses))

	j2 := &TimerJobInfo{JobTypeRemove, 1}

	err = ScheduleTimerJob(timer, *j2, a2)
	assert.NoError(t, err)
	assert.NotContains(t, timer.Addresses, a2)
	length -= 1
	assert.Equal(t, length, len(timer.Addresses))

	err = ScheduleTimerJob(timer, *j2, a2)
	assert.Error(t, err)

	err = ScheduleTimerJob(timer, *j2, a3)
	assert.NoError(t, err)
	assert.NotContains(t, timer.Addresses, a3)
	length -= 1
	assert.Equal(t, length, len(timer.Addresses))

	err = ScheduleTimerJob(timer, *j2, a1)
	assert.NoError(t, err)
	assert.NotContains(t, timer.Addresses, a1)
	length -= 1
	assert.Equal(t, length, len(timer.Addresses))
}
