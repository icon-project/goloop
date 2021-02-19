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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

func TestDelegation(t *testing.T) {
	d1 := NewDelegation()
	d1.Address.SetString("hx1")
	d1.Value.SetInt64(100)

	d2 := d1.Clone()

	assert.True(t, d1.Equal(d2))
	assert.True(t, d1.Address.Equal(d2.Address))
	assert.Equal(t, 0, d1.Value.Cmp(d2.Value.Value()))
}

func TestDelegations(t *testing.T) {
	addr1 := "hx1"
	addr2 := "hx2"
	v1 := int64(1)
	v2 := int64(2)
	d1 := Delegation{
		Address: common.NewAddressFromString(addr1),
		Value:   common.NewHexInt(v1),
	}
	d2 := Delegation{
		Address: common.NewAddressFromString(addr2),
		Value:   common.NewHexInt(v2),
	}
	ds1 := Delegations{
		&d1, &d2,
	}

	ds2 := ds1.Clone()

	assert.True(t, ds1.Has())
	assert.True(t, ds1.Equal(ds2))
	assert.Equal(t, v1+v2, ds2.GetDelegationAmount().Int64())
}

func TestDelegations_Delete(t *testing.T) {
	addr1 := "hx1"
	addr2 := "hx2"
	addr3 := "hx3"
	v1 := int64(1)
	v2 := int64(2)
	v3 := int64(3)
	d1 := Delegation{
		Address: common.NewAddressFromString(addr1),
		Value:   common.NewHexInt(v1),
	}
	d2 := Delegation{
		Address: common.NewAddressFromString(addr2),
		Value:   common.NewHexInt(v2),
	}
	d3 := Delegation{
		Address: common.NewAddressFromString(addr3),
		Value:   common.NewHexInt(v3),
	}
	ds := Delegations{&d1, &d2, &d3}

	tests := []struct {
		name  string
		index int
		err   bool
		want  Delegations
	}{
		{"Delete first item", 0, false, Delegations{&d2, &d3}},
		{"Delete middle item", 1, false, Delegations{&d1, &d3}},
		{"Delete last item", 2, false, Delegations{&d1, &d2}},
		{"Negative index", -1, true, Delegations{}},
		{"Too big index", 100, true, Delegations{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := ds.Clone()
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
		ds1 := Delegations{&d1}
		err := ds1.Delete(0)
		assert.NoError(t, err)
		assert.False(t, ds1.Has())
	})
}

func TestNewDelegations(t *testing.T) {
	setMaxDelegationCount(2)
	defer setMaxDelegationCount(0)

	v1 := 1
	v2 := 2
	tests := []struct {
		name          string
		param         []interface{}
		err           bool
		totalDelegate int
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
		{
			"negative delegation",
			[]interface{}{
				map[string]interface{}{
					"Address": "hx1",
					"Value":   fmt.Sprintf("-0x%x", v1),
				},
			},
			true,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delegations, err := NewDelegations(tt.param)
			if tt.err {
				assert.Error(t, err, "NewDelegations() was not failed for %v.", tt.param)
			} else {
				assert.NoError(t, err, "NewDelegations() was failed for %v. err=%v", tt.param, err)

				got := delegations.ToJSON(module.JSONVersion3)
				if len(tt.param) != len(got) {
					t.Errorf("NewDelegations() = %v, want %v", got, tt.param)
				}
				if int64(tt.totalDelegate) != delegations.GetDelegationAmount().Int64() {
					t.Errorf("NewDelegations() = %v, want %v", got, tt.param)
				}
			}
		})
	}
}
