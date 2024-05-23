/*
 * Copyright 2024 ICON Foundation
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
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewBondRequirementInfo(t *testing.T) {
	rate := icmodule.ToRate(5)
	nextRate := icmodule.ToRate(2)
	info := NewBondRequirementInfo(rate, nextRate)
	assert.Zero(t, info.Version())
	assert.Equal(t, rate, info.Rate())
	assert.Equal(t, nextRate, info.NextRate())

	bs := info.Bytes()
	assert.True(t, len(bs) > 0)

	info2 := NewBondRequirementInfoFromByte(bs)
	assert.True(t, info != info2)
	assert.True(t, info.Equal(info2))
	assert.True(t, info2.Equal(info))
	assert.Equal(t, rate, info2.Rate())
	assert.Equal(t, nextRate, info2.NextRate())

	newRate := icmodule.ToRate(10)
	newNextRate := icmodule.ToRate(11)
	info3 := NewBondRequirementInfo(newRate, newNextRate)
	assert.Equal(t, newRate, info3.Rate())
	assert.Equal(t, newNextRate, info3.NextRate())

	assert.False(t, info3.Equal(info))
	assert.False(t, info3.Equal(info2))
	assert.True(t, info3.Equal(info3))
}
