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

package containerdb

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/intconv"
)

func TestBigIntSafe(t *testing.T) {
	type args struct {
		v Value
	}
	tests := []struct {
		name string
		args args
		want *big.Int
	}{
		{ "Nil", args{ nil }, intconv.BigIntZero},
		{ "Value", args{ &valueImpl{bytesEntry{0x02,0x18}} }, big.NewInt(0x218)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, BigIntSafe(tt.args.v), "BigIntSafe(%v)", tt.args.v)
		})
	}
}

func TestInt64Safe(t *testing.T) {
	type args struct {
		v Value
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{ "Nil", args{ nil }, 0},
		{ "Value", args{ &valueImpl{bytesEntry{0x02,0x18}} }, 0x218},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, Int64Safe(tt.args.v), "Int64Safe(%v)", tt.args.v)
		})
	}
}