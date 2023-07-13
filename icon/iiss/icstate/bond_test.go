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
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
)

func TestBond(t *testing.T) {
	b1 := NewBond(common.MustNewAddressFromString("hx1"), big.NewInt(100))
	b2 := b1.Clone()

	assert.True(t, b1.Equal(b2))
	assert.True(t, b1.To().Equal(b2.To()))
	assert.Equal(t, 0, b1.Amount().Cmp(b2.Amount()))
}

func TestBonds(t *testing.T) {
	addr1 := "hx1"
	addr2 := "hx2"
	v1 := int64(1)
	v2 := int64(2)
	b1 := NewBond(common.MustNewAddressFromString(addr1), big.NewInt(v1))
	b2 := NewBond(common.MustNewAddressFromString(addr2), big.NewInt(v2))
	bl1 := Bonds{b1, b2}

	bl2 := bl1.Clone()

	assert.True(t, bl1.Has())
	assert.True(t, bl1.Equal(bl2))
	assert.Equal(t, v1+v2, bl2.GetBondAmount().Int64())
}

func TestBonds_Delete(t *testing.T) {
	addr1 := "hx1"
	addr2 := "hx2"
	addr3 := "hx3"
	v1 := int64(1)
	v2 := int64(2)
	v3 := int64(3)
	bond1 := NewBond(common.MustNewAddressFromString(addr1), big.NewInt(v1))
	bond2 := NewBond(common.MustNewAddressFromString(addr2), big.NewInt(v2))
	bond3 := NewBond(common.MustNewAddressFromString(addr3), big.NewInt(v3))
	bonds := Bonds{bond1, bond2, bond3}

	tests := []struct {
		name  string
		index int
		err   bool
		want  Bonds
	}{
		{"Delete first item", 0, false, Bonds{bond2, bond3}},
		{"Delete middle item", 1, false, Bonds{bond1, bond3}},
		{"Delete last item", 2, false, Bonds{bond1, bond2}},
		{"Negative index", -1, true, Bonds{}},
		{"Too big index", 100, true, Bonds{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := bonds.Clone()
			err := test.Delete(tt.index)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, tt.want.Equal(test))
			}
		})
	}

	t.Run("Delete and empty", func(t *testing.T) {
		bonds1 := Bonds{bond1}
		err := bonds1.Delete(0)
		assert.NoError(t, err)
		assert.False(t, bonds1.Has())
	})
}

func TestNewBonds(t *testing.T) {
	setMaxBondCount(2)

	v1 := 1
	v2 := 2
	tests := []struct {
		name      string
		param     []interface{}
		err       bool
		len       int
		totalBond int
	}{
		{"Nil param", nil, false, 0, 0},
		{"Empty param", []interface{}{}, false, 0, 0},
		{
			"Success",
			[]interface{}{
				map[string]interface{}{
					"Address": "hx1",
					"Value":   fmt.Sprintf("0x%x", v1),
				},
				map[string]interface{}{
					"Address": "hx2",
					"Value":   fmt.Sprintf("0x%x", v2),
				},
			},
			false,
			2,
			v1 + v2,
		},
		{
			"Bond zero value",
			[]interface{}{
				map[string]interface{}{
					"Address": "hx1",
					"Value":   fmt.Sprintf("0x%x", v1),
				},
				map[string]interface{}{
					"Address": "hx2",
					"Value":   "0x0",
				},
			},
			false,
			1,
			v1,
		},
		{
			"Duplicated Address",
			[]interface{}{
				map[string]interface{}{
					"Address": "hx1",
					"Value":   fmt.Sprintf("0x%x", v1),
				},
				map[string]interface{}{
					"Address": "hx1",
					"Value":   fmt.Sprintf("0x%x", v2),
				},
			},
			true,
			0,
			0,
		},
		{
			"Too many bonds",
			[]interface{}{
				map[string]interface{}{
					"Address": "hx1",
					"Value":   fmt.Sprintf("0x%x", v1),
				},
				map[string]interface{}{
					"Address": "hx2",
					"Value":   fmt.Sprintf("0x%x", v2),
				},
				map[string]interface{}{
					"Address": "hx3",
					"Value":   fmt.Sprintf("0x%x", v2),
				},
			},
			true,
			0,
			0,
		},
		{
			"negative bond",
			[]interface{}{
				map[string]interface{}{
					"Address": "hx1",
					"Value":   fmt.Sprintf("-0x%x", v1),
				},
			},
			true,
			0,
			0,
		},
	}

	revisions := []int{icmodule.Revision13, icmodule.Revision14}
	for _, revision := range revisions {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s Rev%d", tt.name, revision), func(t *testing.T) {
				bonds, err := NewBonds(tt.param, revision)
				if tt.err {
					assert.Error(t, err, "NewBonds() was not failed for %v.", tt.param)
				} else {
					assert.NoError(t, err, "NewBonds() was failed for %v. err=%v", tt.param, err)

					got := bonds.ToJSON(module.JSONVersion3)
					if tt.len != len(got) {
						if revision > icmodule.Revision13 || tt.len+1 != len(got) {
							t.Errorf("Invalid bonds length %d. want %d", len(got), tt.len)
						}
					}
					assert.Equal(t, int64(tt.totalBond), bonds.GetBondAmount().Int64())
				}
			})
		}
	}
}

func TestBonds_Slash(t *testing.T) {
	addr1 := common.MustNewAddressFromString("hx1")
	addr2 := common.MustNewAddressFromString("hx2")
	b1 := NewBond(addr1, big.NewInt(100))
	b2 := NewBond(addr2, big.NewInt(200))
	bl1 := Bonds{b1, b2}

	type values struct {
		target *common.Address
		ratio  icmodule.Rate
	}

	type wants struct {
		slashAmount int64
		length      int
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			out := tt.out
			newBl, slashAmount := bl1.Slash(in.target, in.ratio)
			bl1 = newBl

			assert.Equal(t, out.slashAmount, slashAmount.Int64())
			assert.Equal(t, out.length, len(bl1))
		})
	}
}
