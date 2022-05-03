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

package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBitSetFilter_Basic(t *testing.T) {
	assert := assert.New(t)
	f := MakeBitSetFilter(256 / 8)
	f.Set(3)
	f.Set(5)
	f.Set(7)
	f.Set(9)
	assert.Equal(false, f.Test(0))
	assert.Equal(false, f.Test(1))
	assert.Equal(false, f.Test(2))
	assert.Equal(true, f.Test(3))
	assert.Equal(false, f.Test(4))
	assert.Equal(true, f.Test(5))
	assert.Equal(false, f.Test(6))
	assert.Equal(true, f.Test(7))
	assert.Equal(false, f.Test(8))
	assert.Equal(true, f.Test(9))
	assert.Equal(false, f.Test(10))
}
