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
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icutils"
)

func getRandomVoteState() VoteState {
	return []VoteState{None, Success, Failure}[rand.Intn(3)]
}

func checkDuplicatePReps(t *testing.T, prepSet PRepSet, originPReps []*PRep) {
	prepMap := make(map[string]int)
	for i := 0; i < prepSet.Size(); i++ {
		prep := prepSet.GetPRepByIndex(i)
		_, ok := prepMap[icutils.ToKey(prep.Owner())]
		assert.False(t, ok)
		prepMap[icutils.ToKey(prep.Owner())] = 1
	}
	assert.Equal(t, len(prepMap), prepSet.Size())

	for _, prep := range originPReps {
		_, ok := prepMap[icutils.ToKey(prep.Owner())]
		assert.True(t, ok)
	}
}

func TestPRepSet_GetPRepByIndex(t *testing.T) {
	br := int64(5)
	preps := newDummyPReps(10, br)

	prevBd := big.NewInt(-1)
	for i := 0; i < preps.Size(); i++ {
		prep := preps.GetPRepByIndex(i)
		info := prep.info()
		assert.Equal(t, "KOR", *info.Country)

		bd := prep.GetBondedDelegation(br)
		assert.True(t, prevBd.Cmp(bd) <= 0)
	}
}

func TestPRepSet_OnTermEnd(t *testing.T) {
	var err error
	size := 150
	br := int64(5)
	mainPRepCount := 22
	subPRepCount := 78
	extraMainPRepCount := 0
	electedPRepCount := mainPRepCount + subPRepCount
	limit := 30
	revision := icmodule.RevisionResetPenaltyMask

	preps := newDummyPReps(size, br)
	assert.Equal(t, size, preps.Size())

	prep := preps.GetPRepByIndex(0)
	prep.vPenaltyMask = (rand.Uint32() & uint32(0x3FFFFFFF)) | uint32(1)
	assert.True(t, prep.GetVPenaltyCount() > 0)

	err = preps.OnTermEnd(revision, mainPRepCount, subPRepCount, extraMainPRepCount, limit, br)
	assert.NoError(t, err)
	assert.Equal(t, mainPRepCount, preps.GetPRepSize(GradeMain))
	assert.Equal(t, subPRepCount, preps.GetPRepSize(GradeSub))
	assert.Equal(t, size-mainPRepCount-subPRepCount, preps.GetPRepSize(GradeCandidate))

	for i := 0; i < size; i++ {
		prep = preps.GetPRepByIndex(i)
		if revision == icmodule.RevisionResetPenaltyMask {
			assert.Zero(t, prep.GetVPenaltyCount())
		}
		if i < mainPRepCount {
			assert.Equal(t, GradeMain, prep.Grade())
		} else if i < electedPRepCount {
			assert.Equal(t, GradeSub, prep.Grade())
		} else {
			assert.Equal(t, GradeCandidate, prep.Grade())
		}
	}
}

func TestPRepSet_NewPRepsIncludingExtraMainPRep(t *testing.T) {
	size := 200
	br := int64(5)
	mainPRepCount := 22
	subPRepCount := 78
	extraMainPRepCount := 3

	preps := make([]*PRep, size)
	for i := 0; i < size; i++ {
		prep := newDummyPRep(i)
		prep.lastHeight = rand.Int63n(10000)
		prep.lastState = getRandomVoteState()
		preps[i] = prep
	}

	prepSet := NewPRepsIncludingExtraMainPRep(
		preps, mainPRepCount, subPRepCount, extraMainPRepCount, br,
	)
	assert.Equal(t, size, prepSet.Size())
	checkDuplicatePReps(t, prepSet, preps)

	sort.Slice(preps, func(i, j int) bool {
		return lessByPower(preps[i], preps[j], br)
	})

	extraMainPRepIdxRange := []int{mainPRepCount, mainPRepCount + extraMainPRepCount}
	prevPRep := prepSet.GetPRepByIndex(0)
	for i := 1; i < size; i++ {
		if i >= extraMainPRepIdxRange[0] && i < extraMainPRepIdxRange[1] {
			// Skip extra main preps
			continue
		}
		prep := prepSet.GetPRepByIndex(i)
		assert.True(t, lessByPower(prevPRep, prep, br))
		prevPRep = prep
	}

	restPReps := make([]*PRep, subPRepCount)
	for i := 0; i < len(restPReps); i++ {
		restPReps[i] = prepSet.GetPRepByIndex(mainPRepCount + i)
	}
	sort.Slice(restPReps, func(i, j int) bool {
		return lessByLRU(restPReps[i], restPReps[j], br)
	})

	for i := 0; i < extraMainPRepCount; i++ {
		assert.True(t, restPReps[i] == prepSet.GetPRepByIndex(i+mainPRepCount))
	}
}

// In case of 0 ExtraMainPRepCount,
// Check if both of two NewPReps functions return the same results
func TestPRepSet_NewPReps_WithZeroExtraPRepCount(t *testing.T) {
	size := 200
	br := int64(5)
	mainPRepCount := 25
	subPRepCount := 75
	extraMainPRepCount := 0

	preps := make([]*PRep, size)
	for i := 0; i < size; i++ {
		prep := newDummyPRep(i)
		prep.lastHeight = rand.Int63n(10000)
		prep.lastState = getRandomVoteState()
		preps[i] = prep
	}

	prepSet0 := NewPRepsOrderedByPower(preps, br)
	prepSet1 := NewPRepsIncludingExtraMainPRep(
		preps, mainPRepCount, subPRepCount, extraMainPRepCount, br,
	)
	checkDuplicatePReps(t, prepSet0, preps)
	checkDuplicatePReps(t, prepSet1, preps)

	sort.Slice(preps, func(i, j int) bool {
		return lessByPower(preps[i], preps[j], br)
	})

	assert.Equal(t, len(preps), prepSet0.Size())
	assert.Equal(t, len(preps), prepSet1.Size())

	for i, prep := range preps {
		assert.Equal(t, prep, prepSet0.GetPRepByIndex(i))
		assert.Equal(t, prep, prepSet1.GetPRepByIndex(i))
	}
}

func TestPRepSet_NewPReps2_withNoExtraPReps(t *testing.T) {
	size := 23
	br := int64(5)
	mainPRepCount := 22
	subPRepCount := 78
	extraMainPRepCount := 3

	preps := make([]*PRep, size)
	for i := 0; i < size; i++ {
		prep := newDummyPRep(i)
		prep.lastHeight = rand.Int63n(10000)
		prep.lastState = getRandomVoteState()
		preps[i] = prep
	}

	prepSet0 := NewPRepsOrderedByPower(preps, br)
	prepSet1 := NewPRepsIncludingExtraMainPRep(
		preps, mainPRepCount, subPRepCount, extraMainPRepCount, br,
	)
	checkDuplicatePReps(t, prepSet0, preps)
	checkDuplicatePReps(t, prepSet1, preps)

	// preps is sorted in descending order by power
	sort.Slice(preps, func(i, j int) bool {
		return lessByPower(preps[i], preps[j], br)
	})

	assert.Equal(t, len(preps), prepSet0.Size())
	assert.Equal(t, len(preps), prepSet1.Size())

	for i := 0; i < size; i++ {
		prep := preps[i]
		assert.Equal(t, prep, prepSet0.GetPRepByIndex(i))
		assert.Equal(t, prep, prepSet1.GetPRepByIndex(i))
	}
}

func TestPRepSet_ExtraMainPRepsWithZeroPower(t *testing.T) {
	size := 30
	br := int64(5)
	mainPRepCount := 22
	subPRepCount := 78
	extraMainPRepCount := 3
	zeroPowerPRepCount := 8

	preps := make([]*PRep, size)
	for i := 0; i < size; i++ {
		prep := newDummyPRep(i)
		prep.lastHeight = rand.Int63n(10000)
		prep.lastState = getRandomVoteState()
		preps[i] = prep

		// there are at least 8 preps with 0 power
		if i < zeroPowerPRepCount {
			prep.SetBonded(new(big.Int))
			assert.Zero(t, prep.GetPower(br).Sign())
		}
	}

	prepSet0 := NewPRepsOrderedByPower(preps, br)
	prepSet1 := NewPRepsIncludingExtraMainPRep(
		preps, mainPRepCount, subPRepCount, extraMainPRepCount, br,
	)
	checkDuplicatePReps(t, prepSet0, preps)
	checkDuplicatePReps(t, prepSet1, preps)

	// preps is sorted in descending order by power
	sort.Slice(preps, func(i, j int) bool {
		return lessByPower(preps[i], preps[j], br)
	})

	assert.Equal(t, len(preps), prepSet0.Size())
	assert.Equal(t, len(preps), prepSet1.Size())

	for i, prep := range preps {
		assert.Equal(t, prep, prepSet0.GetPRepByIndex(i))
		assert.Equal(t, prep, prepSet1.GetPRepByIndex(i))
	}
}

func TestPRepSet_WithLRUAndOneZeroPowerPRep(t *testing.T) {
	size := 30
	br := int64(5)
	mainPRepCount := 22
	subPRepCount := 78
	extraMainPRepCount := 3
	zeroPowerPRepIndex := mainPRepCount

	preps := make([]*PRep, size)
	for i := 0; i < size; i++ {
		prep := newDummyPRep(i)
		prep.lastHeight = rand.Int63n(10000) + 1
		prep.lastState = getRandomVoteState()
		preps[i] = prep
	}
	preps[zeroPowerPRepIndex].lastHeight = 0
	preps[zeroPowerPRepIndex].lastState = None
	preps[zeroPowerPRepIndex].SetBonded(big.NewInt(0))

	prepSet := NewPRepsIncludingExtraMainPRep(
		preps, mainPRepCount, subPRepCount, extraMainPRepCount, br,
	)
	checkDuplicatePReps(t, prepSet, preps)
}
