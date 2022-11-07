/*
 * Copyright 2022 ICON Foundation
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

package test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
)

func FuzzDumpRLP(f *testing.F) {
	f.Add([]byte{0x80})
	f.Add([]byte{0xf8, 0x00})
	f.Add([]byte("\xff\u007f\xff\xff\xff\xff\xff\xff\xff"))
	f.Fuzz(func(t *testing.T, data []byte) {
		DumpRLP("", data)
	})
}

func bytesTrimN(data []byte, n int) []byte {
	return data[:len(data)-n]
}

func replace(data []byte, o int, n []byte) []byte {
	copy(data[o:], n)
	return data
}

func TestDumpRLP(t *testing.T) {
	type args struct {
		indent string
		data   []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Nil", args{"", []byte{0xf8, 00}}, "null(0x2:2)\n"},
		{"Bytes", args{"", codec.RLP.MustMarshalToBytes([]byte{0x12, 0x34})}, "bytes(0x2:2) : 1234\n"},
		{"List", args{"", codec.RLP.MustMarshalToBytes([][]byte{{0x12, 0x34}, {0xf1, 0xc3}})}, "list(0x6:6) [\n  bytes(0x2:2) : 1234\n  bytes(0x2:2) : f1c3\n]\n"},
		{"Shorten", args{"", bytesTrimN(codec.RLP.MustMarshalToBytes([][]byte{{0x12, 0x34}, {0xf1, 0xc3}}), 2)}, "no data(offset=1,size=6,limit=5)\n"},
		{"Corrupt", args{"", replace(codec.RLP.MustMarshalToBytes([][]byte{{0x12, 0x34}, {0xf1, 0xc3}}), 4, []byte{0x22})}, "list(0x6:6) [\n  bytes(0x2:2) : 1234\n  bytes(0x1:1) : 22\n  no data(offset=6,size=49,limit=7)\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, DumpRLP(tt.args.indent, tt.args.data), "DumpRLP(%v, %v)", tt.args.indent, tt.args.data)
		})
	}
}
