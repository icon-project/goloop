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
	"reflect"
	"testing"
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
