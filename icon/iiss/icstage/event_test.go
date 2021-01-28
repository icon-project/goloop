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

package icstage

import (
	"github.com/icon-project/goloop/common"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestEvent_Delegation(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	type_ := TypeEventDelegation
	version := 0
	addr1 := "hx1"
	v1 := int64(1)
	vote1 := Vote{
		Address: common.NewAddressFromString(addr1),
		Value: big.NewInt(v1),
	}

	t1 := newEventVote(icobject.MakeTag(type_, version))
	t1.From = common.NewAddressFromString(addr1)
	t1.Votes = VoteList{&vote1}

	o1 := icobject.New(type_, t1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())
	assert.Equal(t, type_, o2.Tag().Type())
	assert.Equal(t, version, o2.Tag().Version())

	t2 := ToEventVote(o2)
	assert.Equal(t, true, t1.Equal(t2))
	assert.Equal(t, true, t1.From.Equal(t2.From))
	assert.Equal(t, true, t1.Votes.Equal(t2.Votes))
}

func TestEvent_Bond(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	type_ := TypeEventBond
	version := 0
	addr1 := "hx1"
	v1 := int64(1)
	vote1 := Vote{
		Address: common.NewAddressFromString(addr1),
		Value: big.NewInt(v1),
	}


	t1 := newEventVote(icobject.MakeTag(type_, version))
	t1.From = common.NewAddressFromString(addr1)
	t1.Votes = VoteList{&vote1}

	o1 := icobject.New(type_, t1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())
	assert.Equal(t, type_, o2.Tag().Type())
	assert.Equal(t, version, o2.Tag().Version())

	t2 := ToEventVote(o2)
	assert.Equal(t, true, t1.Equal(t2))
	assert.Equal(t, true, t1.From.Equal(t2.From))
	assert.Equal(t, true, t1.Votes.Equal(t2.Votes))
}

func TestEvent_Enable(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	type_ := TypeEventEnable
	version := 0
	addr1 := "hx1"
	enable := false


	t1 := newEventEnable(icobject.MakeTag(type_, version))
	t1.Target = common.NewAddressFromString(addr1)
	t1.Enable = enable

	o1 := icobject.New(type_, t1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())
	assert.Equal(t, type_, o2.Tag().Type())
	assert.Equal(t, version, o2.Tag().Version())

	t2 := ToEventEnable(o2)
	assert.Equal(t, true, t1.Equal(t2))
	assert.Equal(t, true, t1.Target.Equal(t2.Target))
	assert.Equal(t, t1.Enable, t2.Enable)
}
