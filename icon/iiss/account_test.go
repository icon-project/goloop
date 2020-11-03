/*
 * Copyright 2020 ICON Foundation
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package iiss

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
)

func TestAccountStateImpl(t *testing.T) {
	t1 := AccountStateImpl{
		version: 100,
		staked: common.NewHexInt(12),
	}

	fmt.Printf("version: %d, staked: %d\n", t1.version, t1.staked.Int.Int64())
	bs := t1.Bytes()
	fmt.Printf("bs: %b\n", bs)
	t2 := new(AccountStateImpl)
	t2.SetBytes(bs)

	assert.Equal(t, t1.version, t2.version)
	assert.Equal(t, 0, t1.staked.Cmp(&t2.staked.Int))
}
