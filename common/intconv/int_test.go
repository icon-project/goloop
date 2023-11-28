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

package intconv

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBigIntSafe(t *testing.T) {
	assert.True(t, BigIntZero.Cmp(BigIntSafe(nil)) == 0)
	value := big.NewInt(1238)
	assert.True(t, BigIntSafe(value) == value)
}