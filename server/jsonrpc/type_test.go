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
 *
 */

package jsonrpc

import (
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestHexIntFromInt64(t *testing.T) {
	type args struct {
		v int64
	}
	tests := []struct {
		name string
		args args
		want HexInt
	}{
		{ "zero", args{ 0 }, "0x0" },
		{ "one-digit", args{ 10 }, "0xa" },
		{ "multiple-digits", args{ 99 }, "0x63" },
		{ "negative-one-digit", args{ -8 }, "-0x8" },
		{ "negative-multi-digit", args{ -99 }, "-0x63" },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, HexIntFromInt64(tt.args.v), "HexIntFromInt64(%v)", tt.args.v)
		})
	}
}

func TestHexIntFromBigInt(t *testing.T) {
	type args struct {
		v *big.Int
	}
	tests := []struct {
		name string
		args args
		want HexInt
	}{
		{ "zero", args{ big.NewInt(0) }, "0x0" },
		{ "one-digit", args{ big.NewInt(10) }, "0xa" },
		{ "multiple-digits", args{ big.NewInt(99) }, "0x63" },
		{ "negative-one-digit", args{ big.NewInt(-8) }, "-0x8" },
		{ "negative-multi-digit", args{ big.NewInt(-99) }, "-0x63" },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, HexIntFromBigInt(tt.args.v), "HexIntFromBigInt(%v)", tt.args.v)
		})
	}
}