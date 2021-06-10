/*
 * Copyright 2021 ICON Foundation
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
)

func TestPReps_GetPRepByIndex(t *testing.T) {
	br := int64(5)
	preps := newDummyPReps(10, br)

	prevBd := big.NewInt(-1)
	for i := 0; i < preps.Size(); i++ {
		prep := preps.GetPRepByIndex(i)
		assert.Equal(t, "KOR", prep.country)

		bd := prep.GetBondedDelegation(br)
		assert.True(t, prevBd.Cmp(bd) <= 0)
	}
}

func TestPReps_ResetAllStatus(t *testing.T) {
	var err error
	size := 150
	br := int64(5)
	mainPRepCount := 22
	subPRepCount := 78
	electedPRepCount := mainPRepCount + subPRepCount
	bh := int64(100)
	penaltyMask := 0x3FFFFFFF

	preps := newDummyPReps(size, br)
	assert.Equal(t, size, preps.Size())

	err = preps.ResetAllStatus(bh, mainPRepCount, subPRepCount, penaltyMask)
	assert.NoError(t, err)
	assert.Equal(t, mainPRepCount, preps.GetPRepSize(Main))
	assert.Equal(t, subPRepCount, preps.GetPRepSize(Sub))
	assert.Equal(t, size - mainPRepCount - subPRepCount, preps.GetPRepSize(Candidate))

	for i := 0; i < size; i++ {
		prep := preps.GetPRepByIndex(i)
		if i < mainPRepCount {
			assert.Equal(t, Main, prep.Grade())
		} else if i < electedPRepCount {
			assert.Equal(t, Sub, prep.Grade())
		} else {
			assert.Equal(t, Candidate, prep.Grade())
		}
	}
}
