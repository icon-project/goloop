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

package icstate

import (
	"fmt"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBond(t *testing.T) {
	b1 := newBond()
	b1.Address.SetString("hx1")
	b1.Value.SetInt64(100)

	b2 := b1.Clone()

	assert.True(t, b1.Equal(b2))
	assert.True(t, b1.Address.Equal(b2.Address))
	assert.Equal(t, 0, b1.Value.Cmp(b2.Value.Value()))
}

func TestBonds(t *testing.T) {
	addr1 := "hx1"
	addr2 := "hx2"
	v1 := int64(1)
	v2 := int64(2)
	b1 := Bond{
		Address: common.NewAddressFromString(addr1),
		Value:   common.NewHexInt(v1),
	}
	b2 := Bond{
		Address: common.NewAddressFromString(addr2),
		Value:   common.NewHexInt(v2),
	}
	bl1 := Bonds{
		&b1, &b2,
	}

	bl2 := bl1.Clone()

	assert.True(t, bl1.Has())
	assert.True(t, bl1.Equal(bl2))
	assert.Equal(t, v1+v2, bl2.GetBondAmount().Int64())
}

func TestNewBonds(t *testing.T) {
	setMaxBondCount(2)

	v1 := 1
	v2 := 2
	tests := []struct {
		name      string
		param     []interface{}
		err       bool
		totalBond int
	}{
		{"Nil param", nil, false, 0},
		{"Empty param", []interface{}{}, false, 0},
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
			v1 + v2,
		},
		{
			"Duplicated Address Address",
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
		},
		{
			"Too many delegations",
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delegations, err := NewBonds(tt.param)
			if tt.err {
				assert.Error(t, err, "NewBonds() was not failed for %v.", tt.param)
			} else {
				assert.NoError(t, err, "NewBonds() was failed for %v. err=%v", tt.param, err)

				got := delegations.ToJSON(module.JSONVersion3)
				if len(tt.param) != len(got) {
					t.Errorf("NewBonds() = %v, want %v", got, tt.param)
				}
				if int64(tt.totalBond) != delegations.GetBondAmount().Int64() {
					t.Errorf("NewBonds() = %v, want %v", got, tt.param)
				}
			}
		})
	}
}
