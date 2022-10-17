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

package fastsync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeightSet_Basics(t *testing.T) {
	assert := assert.New(t)

	hs := newHeightSet(20, 26)
	v, ok := hs.getLowest()
	assert.True(ok)
	assert.EqualValues(20, v)

	for i := 20; i < 25; i++ {
		v, ok = hs.popLowest()
		assert.True(ok)
		assert.EqualValues(i, v)
	}

	assert.Panics(func() {
		hs.add(29)
	})

	hs.add(21)
	hs.add(23)
	v, ok = hs.getLowest()
	assert.True(ok)
	assert.EqualValues(21, v)
	v, ok = hs.popLowest()
	assert.True(ok)
	assert.EqualValues(21, v)
	v, ok = hs.popLowest()
	assert.True(ok)
	assert.EqualValues(23, v)
	v, ok = hs.popLowest()
	assert.True(ok)
	assert.EqualValues(25, v)
	v, ok = hs.popLowest()
	assert.True(ok)
	assert.EqualValues(26, v)
	v, ok = hs.popLowest()
	assert.False(ok)
	assert.EqualValues(-1, v)
}
