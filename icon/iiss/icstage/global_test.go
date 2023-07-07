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
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestGlobalV1(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)

	type_ := TypeGlobal
	version := GlobalVersion1
	tag := icobject.MakeTag(type_, version)
	offsetLimit := 10

	g, err := newGlobal(tag)
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
	tag := icobject.MakeTag(type_, version)
	offsetLimit := 10

	g, err := newGlobal(tag)
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

func TestGlobalV3(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)

	type_ := TypeGlobal
	version := GlobalVersion3
	tag := icobject.MakeTag(type_, version)
	offsetLimit := 10
	iglobal := big.NewInt(3000000000)
	iprep := big.NewInt(7000)
	iwage := big.NewInt(3000)
	minBond := big.NewInt(10000)

	g, err := newGlobal(tag)
	assert.NoError(t, err)
	assert.Equal(t, version, g.Version())
	g1 := g.GetV3()
	assert.NotNil(t, g1)
	g1.offsetLimit = offsetLimit
	g1.rFund.Set(keyIglobal, iglobal)
	g1.rFund.Set(keyIprep, iprep)
	g1.rFund.Set(keyIwage, iwage)
	g1.minBond = minBond

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
	g2 := global.GetV3()
	fmt.Printf("%+v\n", g2)
	assert.Equal(t, offsetLimit, g2.GetOffsetLimit())
	assert.Equal(t, 0, g2.GetIGlobal().Cmp(iglobal))
	assert.Equal(t, 0, g2.GetIprep().Cmp(iprep))
	assert.Equal(t, 0, g2.GetRewardFundByKey(keyIwage).Cmp(iwage))
	assert.Equal(t, 0, g2.GetMinBond().Cmp(minBond))
}
