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

package containerdb

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
)

func TestSplitKeys(t *testing.T) {
	type args struct {
		key []byte
	}
	tests := []struct {
		name    string
		args    args
		want    [][]byte
		wantErr bool
	}{
		{
			name: "Empty",
			args: args{key: []byte{}},
		},
		{
			name: "OneByte",
			args: args{key: []byte{0x7f}},
			want: [][]byte{{0x7f}},
		},
		{
			name: "Short",
			args: args{key: []byte{0x82, 0x12, 0x34}},
			want: [][]byte{{0x12, 0x34}},
		},
		{
			name: "Shortx2",
			args: args{key: []byte{0x82, 0x12, 0x34, 0x82, 0x56, 0x78}},
			want: [][]byte{{0x12, 0x34}, {0x56, 0x78}},
		},
		{
			name:    "List",
			args:    args{key: []byte{0xC1, 0x12}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitKeys(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitKeys() got = %v, want %v", got, tt.want)
			}
		})
	}
}

type testValueImpl struct {
	Value
	value []byte
}

func (v *testValueImpl) Bytes() []byte {
	return v.value
}

func TestToBytes(t *testing.T) {
	type args struct {
		v interface{}
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"string", args{"TEST"}, []byte("TEST")},
		{"address", args{common.MustNewAddressFromString("cx4444444444444444444444444444444444444444")}, []byte{0x01, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44}},
		{"bool#1", args{true}, []byte{0x01}},
		{"bool#2", args{false}, []byte{0x00}},
		{"[]byte", args{[]byte{0x78, 0xab}}, []byte{0x78, 0xab}},
		{"byte", args{byte(0x7d)}, []byte{0x7d}},
		{"Value", args{&testValueImpl{value: []byte{0xf9}}}, []byte{0xf9}},
		{"HexInt", args{common.NewHexInt(0x87ac)}, []byte{0x00, 0x87, 0xac}},
		{"big.Int", args{big.NewInt(0x87ac)}, []byte{0x00, 0x87, 0xac}},
		{"int64", args{int64(0x87ac)}, []byte{0x00, 0x87, 0xac}},
		{"int32", args{int32(0x87ac)}, []byte{0x00, 0x87, 0xac}},
		{"int16", args{int16(0x77ac)}, []byte{0x77, 0xac}},
		{"int", args{int(0x87ac)}, []byte{0x00, 0x87, 0xac}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, ToBytes(tt.args.v), "ToBytes(%v)", tt.args.v)
		})
	}
}

func TestAppendRawKeys(t *testing.T) {
	type args struct {
		key  []byte
		keys []interface{}
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"Single", args{[]byte("TEST"), nil}, []byte("TEST")},
		{"AppendEmpty", args{[]byte("TEST"), []interface{}{[]byte(nil)}}, []byte("TEST")},
		{"AppendOne", args{[]byte("TEST"), []interface{}{[]byte("1")}}, []byte("TEST1")},
		{"AppendThree", args{[]byte("TEST"), []interface{}{[]byte("1"), []byte("2"), []byte("3456")}}, []byte("TEST123456")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, AppendRawKeys(tt.args.key, tt.args.keys...), "AppendRawKeys(%v, %v)", tt.args.key, tt.args.keys)
		})
	}
}

func TestAppendKeys(t *testing.T) {
	type args struct {
		key  []byte
		keys []interface{}
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"Empty", args{nil, nil}, []byte{}},
		{"PrefixOnly", args{[]byte("TEST"), nil}, []byte("TEST")},
		{"PrefixWithKeys", args{[]byte("TEST"), []interface{}{[]byte("1"), []byte("23")}}, []byte{0x54, 0x45, 0x53, 0x54, 0x31, 0x82, 0x32, 0x33}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppendKeys(tt.args.key, tt.args.keys...)
			assert.Equalf(t, tt.want, got, "AppendKeys(%v, %v)", tt.args.key, tt.args.keys)
			bytesOfKeys := got[len(tt.args.key):]
			keys, err := SplitKeys(bytesOfKeys)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.args.keys), len(keys))
			for idx, ko := range tt.args.keys {
				bs := ToBytes(ko)
				assert.Equal(t, bs, keys[idx])
			}
		})
	}
}
