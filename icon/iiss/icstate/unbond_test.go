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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/icon/icmodule"
)

func TestUnbonds(t *testing.T) {
	addr1 := "hx1"
	addr2 := "hx2"
	v1 := int64(1)
	v2 := int64(2)
	ub1 := NewUnbond(common.MustNewAddressFromString(addr1), big.NewInt(v1), 0)

	ub2 := NewUnbond(common.MustNewAddressFromString(addr2), big.NewInt(v2), 0)
	ubl1 := Unbonds{ub1, ub2}

	ubl2 := ubl1.Clone()

	assert.True(t, !ubl1.IsEmpty())
	assert.True(t, ubl1.Equal(ubl2))
	assert.Equal(t, v1+v2, ubl2.GetUnbondAmount().Int64())
}

func TestUnbonds_Slash(t *testing.T) {
	addr1 := common.MustNewAddressFromString("hx1")
	addr2 := common.MustNewAddressFromString("hx2")
	ub1 := NewUnbond(addr1, big.NewInt(100), 100)
	ub2 := NewUnbond(addr2, big.NewInt(200), 200)
	ubl1 := Unbonds{ub1, ub2}

	type values struct {
		target *common.Address
		rate   icmodule.Rate
	}

	type wants struct {
		slashAmount int64
		length      int
		expire      int64
	}

	tests := []struct {
		name string
		in   values
		out  wants
	}{
		{
			"Invalid address",
			values{
				common.MustNewAddressFromString("hx321"),
				icmodule.ToRate(10),
			},
			wants{
				0,
				2,
				-1,
			},
		},
		{
			"slash 10%",
			values{
				addr1,
				icmodule.ToRate(10),
			},
			wants{
				int64(10),
				2,
				-1,
			},
		},
		{
			"slash 100%",
			values{
				addr1,
				icmodule.ToRate(100),
			},
			wants{
				int64(90),
				1,
				100,
			},
		},
		{
			"slash 10% last entry",
			values{
				addr2,
				icmodule.ToRate(10),
			},
			wants{
				int64(20),
				1,
				-1,
			},
		},
		{
			"slash 100% last entry",
			values{
				addr2,
				icmodule.ToRate(100),
			},
			wants{
				int64(180),
				0,
				200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			out := tt.out
			newUbs, slashAmount, expire := ubl1.Slash(in.target, in.rate)
			ubl1 = newUbs

			assert.Equal(t, out.slashAmount, slashAmount.Int64())
			assert.Equal(t, out.expire, expire)
			assert.Equal(t, out.length, len(ubl1))
		})
	}
}
