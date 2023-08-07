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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstate"
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
	assert.Nil(t, global.GetV2())
	assert.Nil(t, global.GetV3())
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
	assert.Nil(t, global.GetV1())
	assert.Nil(t, global.GetV3())
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
	iprep := icmodule.Rate(7700)
	iwage := icmodule.Rate(1300)
	icps := icmodule.Rate(1000)
	irelay := icmodule.Rate(0)
	minBond := big.NewInt(10000)

	g, err := newGlobal(tag)
	assert.NoError(t, err)
	assert.Equal(t, version, g.Version())
	g1 := g.GetV3()
	assert.NotNil(t, g1)
	g1.offsetLimit = offsetLimit
	rFund, err := icstate.NewSafeRewardFundV2(iglobal, iprep, iwage, icps, irelay)
	assert.NoError(t, err)
	g1.rFund = rFund
	g1.minBond = minBond

	o1 := icobject.New(type_, g)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err = o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes. %v", err)
		return
	}

	assert.Equal(t, serialized, o2.Bytes())
	assert.Equal(t, type_, o2.Tag().Type())
	assert.Equal(t, version, o2.Tag().Version())

	global := ToGlobal(o2)
	assert.Nil(t, global.GetV1())
	assert.Nil(t, global.GetV2())
	g2 := global.GetV3()
	assert.True(t, g1.Equal(g2))
	assert.Equal(t, offsetLimit, g2.GetOffsetLimit())
	assert.Equal(t, 0, g2.GetIGlobal().Cmp(iglobal))
	assert.Equal(t, iprep, g2.GetIPRep())
	assert.Equal(t, iprep, g2.GetRewardFundRateByKey(icstate.KeyIprep))
	assert.Equal(t, iwage, g2.GetIWage())
	assert.Equal(t, iwage, g2.GetRewardFundRateByKey(icstate.KeyIwage))
	assert.Equal(t, icps, g2.GetICps())
	assert.Equal(t, icps, g2.GetRewardFundRateByKey(icstate.KeyIcps))
	assert.Equal(t, irelay, g2.GetIRelay())
	assert.Equal(t, irelay, g2.GetRewardFundRateByKey(icstate.KeyIrelay))
	assert.Equal(t, 0, g2.MinBond().Cmp(minBond))
}
