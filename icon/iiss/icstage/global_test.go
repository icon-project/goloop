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

package icstage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestGlobalV1(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)

	type_ := TypeGlobal
	version := GlobalVersion1
	offsetLimit := 10

	g, err := NewGlobal(version)
	assert.NoError(t, err)
	assert.Equal(t, version, g.Version())
	g1 := g.GetV1()
	assert.NotNil(t, g1)
	g1.offsetLimit = offsetLimit

	o1 := icobject.New(type_, g)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes. %v", err)
		return
	}

	assert.Equal(t, serialized, o2.Bytes())
	assert.Equal(t, type_, o2.Tag().Type())
	assert.Equal(t, version, o2.Tag().Version())

	global := ToGlobal(o2)
	g2 := global.GetV1()
	assert.Equal(t, true, g1.Equal(g2))
	assert.Equal(t, offsetLimit, g2.GetOffsetLimit())
}

func TestGlobalV2(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)

	type_ := TypeGlobal
	version := GlobalVersion2
	offsetLimit := 10

	g, err := NewGlobal(version)
	assert.NoError(t, err)
	assert.Equal(t, version, g.Version())
	g1 := g.GetV2()
	assert.NotNil(t, g1)
	g1.offsetLimit = offsetLimit

	o1 := icobject.New(type_, g)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes. %v", err)
		return
	}

	assert.Equal(t, serialized, o2.Bytes())
	assert.Equal(t, type_, o2.Tag().Type())
	assert.Equal(t, version, o2.Tag().Version())

	global := ToGlobal(o2)
	g2 := global.GetV2()
	assert.Equal(t, true, g1.Equal(g2))
	assert.Equal(t, offsetLimit, g2.GetOffsetLimit())
}
