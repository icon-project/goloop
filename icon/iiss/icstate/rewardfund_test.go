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

package icstate

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/icmodule"
)

func TestRewardFund(t *testing.T) {
	iglobal := int64(100000)
	iprep := int64(50)
	ivoter := int64(50)
	rf := NewRewardFund()
	rf.Iglobal = big.NewInt(iglobal)
	rf.Iprep = icmodule.ToRate(iprep)
	rf.Ivoter = icmodule.ToRate(ivoter)

	bs := rf.Bytes()

	rf2, err := newRewardFundFromByte(bs)
	assert.NoError(t, err)

	assert.True(t, rf.Equal(rf2))
	assert.Equal(t, 0, rf.Iglobal.Cmp(rf2.Iglobal))

	rf3 := rf.Clone()
	assert.True(t, rf.Equal(rf3))

	assert.Equal(t, iglobal*iprep/100, rf.GetPRepFund().Int64())
	assert.Equal(t, iglobal*ivoter/100, rf.GetVoterFund().Int64())
}
