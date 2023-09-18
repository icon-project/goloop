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
 */

package icmodule

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/module"
)

func TestValueToRevision(t *testing.T) {
	for i := 0; i < RevisionReserved; i++ {
		// Revision value check
		rev := ValueToRevision(i)
		assert.True(t, rev.Value() == i)

		// Flag Check
		flags := revisionFlags[i]
		mask := module.Revision(1 << 8)
		for ; flags != 0; mask <<= 1 {
			flag := flags & mask
			if flag != 0 {
				assert.True(t, rev.Has(flag))
				flags &= ^mask
			}
		}
	}
}
