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

package trie

import (
	"testing"
)

func Test_bytesObject_Equal(t *testing.T) {
	type args struct {
		n Object
	}
	tests := []struct {
		name string
		o    BytesObject
		args args
		want bool
	}{
		{"NilWithNil", nil, args{nil}, true},
		{"NilWithNilPtr", nil, args{BytesObject(nil)}, true},
		{"NilWithEmpty", nil, args{BytesObject([]byte{})}, true},
		{"NilWithNonNil", nil, args{BytesObject([]byte{0x00})}, false},
		{"EmptyWithNil", BytesObject([]byte{}), args{nil}, true},
		{"EmptyWithNilPtr", BytesObject([]byte{}), args{BytesObject(nil)}, true},
		{"NonNilWithNil", BytesObject([]byte{0x00}), args{nil}, false},
		{"NonNilWithNilPtr", BytesObject([]byte{0x00}), args{BytesObject(nil)}, false},
		{"Case1", BytesObject([]byte{0x00}), args{BytesObject([]byte{0x00})}, true},
		{"Case2", BytesObject([]byte{0x00}), args{BytesObject([]byte{0x02})}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.Equal(tt.args.n); got != tt.want {
				t.Errorf("BytesObject.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}
