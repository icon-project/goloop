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
	"github.com/icon-project/goloop/icon/iiss/icstate"
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
	d1 := icstate.Delegation{
		Address: common.NewAddressFromString(addr1),
		Value: common.NewHexInt(v1),
	}


	t1 := newEventDelegation(icobject.MakeTag(type_, version))
	t1.From = common.NewAddressFromString(addr1)
	t1.Delegations = icstate.Delegations{&d1}

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

	t2 := ToEventDelegation(o2)
	assert.Equal(t, true, t1.Equal(t2))
	assert.Equal(t, true, t1.From.Equal(t2.From))
	assert.Equal(t, true, t1.Delegations.Equal(t2.Delegations))
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

func TestEvent_Period(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	type_ := TypeEventPeriod
	version := 0
	irep := int64(1000)
	rrep := int64(2000)


	t1 := newEventPeriod(icobject.MakeTag(type_, version))
	t1.Irep = big.NewInt(irep)
	t1.Rrep = big.NewInt(rrep)

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

	t2 := ToEventPeriod(o2)
	assert.Equal(t, true, t1.Equal(t2))
	assert.Equal(t, 0, t1.Irep.Cmp(t2.Irep))
	assert.Equal(t, 0, t1.Rrep.Cmp(t2.Rrep))
}

func TestEvent_Validator(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	type_ := TypeEventValidator
	version := 0
	validators := []*common.Address{
		common.NewAddressFromString("hx1"),
		common.NewAddressFromString("hx2"),
	}


	t1 := newEventValidator(icobject.MakeTag(type_, version))
	t1.validators = validators

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

	t2 := ToEventValidator(o2)
	assert.Equal(t, true, t1.Equal(t2))
	assert.Equal(t, len(t1.validators), len(t2.validators))
	for i, v := range t1.validators {
		assert.True(t, v.Equal(t2.validators[i]))
	}
}