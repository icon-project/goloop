/*
 * Copyright 2023 ICON Foundation
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
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/icmodule"
)

func TestNewCommissionInfo(t *testing.T) {
	args := []struct{
		rate,
		maxRate,
		maxChangeRate icmodule.Rate
		ok bool
	}{
		{-100, 0, 0, false},
		{0, -200, 0, false},
		{0, 0, -10, false},
		{0, 0, 0, true},
		{1000, 2000, 100, true},
		{1000, 10000, 1000, true},
		{10000, 10000, 10000, true},
		{3000, 5000, 50, true},
		{20000, 10000, 10000, false},
		{20000, 20000, 10000, false},
		{2000, 10001, 100, false},
		{0, 10000, 10001, false},
		{5000, 4000, 10001, false},
		{0, 10000, 0, false},
		{500, 500, 0, true},
	}
	for i, arg := range args {
		name := fmt.Sprintf("test-%02d", i)
		t.Run(name, func(t *testing.T) {
			ci, err := NewCommissionInfo(arg.rate, arg.maxRate, arg.maxChangeRate)
			if arg.ok {
				assert.NoError(t, err)
				assert.Equal(t, arg.rate, ci.Rate())
				assert.Equal(t, arg.maxRate, ci.MaxRate())
				assert.Equal(t, arg.maxChangeRate, ci.MaxChangeRate())
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCommissionInfo_RLPEncodeSelf(t *testing.T) {
	const (
		rate = icmodule.Rate(1000)
		maxRate = icmodule.Rate(2000)
		maxChangeRate = icmodule.Rate(100)
	)
	ci, err := NewCommissionInfo(rate, maxRate, maxChangeRate)
	assert.NoError(t, err)

	buf := bytes.NewBuffer(nil)
	e := codec.BC.NewEncoder(buf)

	err = ci.RLPEncodeSelf(e)
	assert.NoError(t, err)

	err = e.Close()
	assert.NoError(t, err)

	d := codec.BC.NewDecoder(bytes.NewBuffer(buf.Bytes()))
	ci2 := NewEmptyCommissionInfo()
	err = ci2.RLPDecodeSelf(d)
	assert.NoError(t, err)

	assert.True(t, ci.Equal(ci2))
	assert.Equal(t, rate, ci2.Rate())
	assert.Equal(t, maxRate, ci2.MaxRate())
	assert.Equal(t, maxChangeRate, ci2.MaxChangeRate())
}

func TestCommissionInfo_SetRate(t *testing.T) {
	const (
		Rate = icmodule.Rate(1000)
		MaxRate = icmodule.Rate(2000)
		MaxChangeRate = icmodule.Rate(100)
	)
	ci, err := NewCommissionInfo(Rate, MaxRate, MaxChangeRate)
	assert.NoError(t, err)

	rate := Rate + MaxChangeRate
	err = ci.SetRate(rate)
	assert.NoError(t, err)
	assert.Equal(t, rate, ci.Rate())
	assert.Equal(t, MaxRate, ci.MaxRate())
	assert.Equal(t, MaxChangeRate, ci.maxChangeRate)
}