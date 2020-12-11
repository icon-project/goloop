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

package icstate

import (
	"github.com/icon-project/goloop/common"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnbonds(t *testing.T) {
	addr1 := "hx1"
	addr2 := "hx2"
	v1 := int64(1)
	v2 := int64(2)
	ub1 := Unbond{
		Address: common.NewAddressFromString(addr1),
		Value:   big.NewInt(v1),
	}
	ub2 := Unbond{
		Address: common.NewAddressFromString(addr2),
		Value:   big.NewInt(v2),
	}
	ubl1 := Unbonds{
		&ub1, &ub2,
	}

	ubl2 := ubl1.Clone()

	assert.True(t, ubl1.Has())
	assert.True(t, ubl1.Equal(ubl2))
	assert.Equal(t, v1+v2, ubl2.GetUnbondAmount().Int64())
}
