/*
 * Copyright 2021 ICON Foundation
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

package contract

import (
	"math"
	"reflect"
	"testing"

	"github.com/icon-project/goloop/common"
)

func newHexInt(s string, base int) *common.HexInt {
	v := new(common.HexInt)
	if _, yn := v.SetString(s, base); !yn {
		return nil
	}
	return v
}

func TestAssignHexInt(t *testing.T) {
	type args struct {
		dstValue reflect.Value
		srcValue *common.HexInt
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"HexInt", args{reflect.ValueOf(new(*common.HexInt)).Elem(), common.NewHexInt(10)}, false},
		{"Int", args{reflect.ValueOf(new(int)).Elem(), common.NewHexInt(10)}, false},
		{"Uint", args{reflect.ValueOf(new(uint)).Elem(), common.NewHexInt(-10)}, true},
		{"Int32Edge", args{reflect.ValueOf(new(int32)).Elem(), common.NewHexInt(math.MinInt32)}, false},
		{"Int64Edge", args{reflect.ValueOf(new(int64)).Elem(), common.NewHexInt(math.MinInt64)}, false},
		{"Int64Edge", args{reflect.ValueOf(new(int64)).Elem(), common.NewHexInt(math.MaxInt64)}, false},
		{"Uint32Edge", args{reflect.ValueOf(new(uint32)).Elem(), common.NewHexInt(math.MaxUint32)}, false},
		{"Uint64Edge", args{reflect.ValueOf(new(uint64)).Elem(), newHexInt("0xffffffffffffffff", 0)}, false},
		{"Int32Over", args{reflect.ValueOf(new(int32)).Elem(), common.NewHexInt(math.MaxInt32 + 1)}, true},
		{"Int32Under", args{reflect.ValueOf(new(int32)).Elem(), common.NewHexInt(math.MinInt32 - 1)}, true},
		{"Uint32Over", args{reflect.ValueOf(new(uint32)).Elem(), common.NewHexInt(math.MaxUint32 + 1)}, true},
		{"Int64Over", args{reflect.ValueOf(new(int64)).Elem(), newHexInt("0x8000000000000000", 0)}, true},
		{"Int64Under", args{reflect.ValueOf(new(int64)).Elem(), newHexInt("-0x8000000000000001", 0)}, true},
		{"Uint64Over", args{reflect.ValueOf(new(uint32)).Elem(), newHexInt("0x10000000000000000", 0)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AssignHexInt(tt.args.dstValue, tt.args.srcValue); (err != nil) != tt.wantErr {
				t.Errorf("AssignHexInt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
