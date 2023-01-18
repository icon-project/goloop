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

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestIScore(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)

	type_ := TypeIScore
	version := 0
	v1 := int64(100)

	t1 := NewIScore(big.NewInt(v1))

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

	t2 := ToIScore(o2)
	assert.Equal(t, true, t1.Equal(t2))
	assert.Equal(t, 0, t1.Value().Cmp(t2.Value()))
}

func TestIScore_Equal(t *testing.T) {
	value := big.NewInt(100)
	is := NewIScore(value)
	is2 := NewIScore(value)
	assert.True(t, is.Equal(is2))
	assert.True(t, is2.Equal(is))

	is2 = NewIScore(big.NewInt(200))
	assert.False(t, is.Equal(is2))
	assert.False(t, is2.Equal(is))

	assert.False(t, is.Equal(nil))
}

func TestIScore_Value(t *testing.T) {
	value := big.NewInt(100)
	iscore := NewIScore(value)
	assert.Zero(t, iscore.Value().Cmp(value))
}

func TestIScore_IsEmpty(t *testing.T) {
	amount := big.NewInt(0)
	iscore := NewIScore(amount)
	assert.True(t, iscore.IsEmpty())

	iscore = newIScore(icobject.Tag(TypeIScore))
	assert.True(t, iscore.IsEmpty())
}

func TestIScore_Clear(t *testing.T) {
	value := int64(100)
	amount := big.NewInt(value)
	iscore := NewIScore(amount)
	assert.Equal(t, value, iscore.Value().Int64())

	iscore.Clear()
	assert.Zero(t, iscore.Value().Sign())
}

func TestIScore_Added(t *testing.T) {
	var is *IScore
	value := int64(100)
	amount := big.NewInt(value)
	is2 := is.Added(amount)
	assert.Equal(t, value, is2.Value().Int64())

	is3 := is2.Added(amount)
	assert.Equal(t, value*2, is3.Value().Int64())
	assert.Equal(t, value, is2.Value().Int64())
}

func TestIScore_Subtracted(t *testing.T) {
	var is *IScore
	value := int64(100)
	values := []int64{-value, 0, value}

	for _, v := range values {
		is = nil
		is2 := is.Subtracted(big.NewInt(v))
		assert.Equal(t, -v, is2.Value().Int64())
	}

	for _, v := range values {
		is = NewIScore(new(big.Int))
		is2 := is.Subtracted(big.NewInt(v))
		assert.Equal(t, -v, is2.Value().Int64())
	}
}

func TestIScore_Clone(t *testing.T) {
	value := int64(100)
	amount := big.NewInt(value)
	is := NewIScore(amount)
	is2 := is.Clone()
	assert.True(t, is.Equal(is2))
	assert.True(t, is2.Equal(is))

	is2.Clear()
	assert.False(t, is.Equal(is2))
	assert.False(t, is2.Equal(is))
}
