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

package consensus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func newVoteMsgWithHashPrefix(firstByteOfHash byte) *VoteMessage {
	vm := newVoteMessage()
	vm.BlockPartSetIDAndNTSVoteCount = &PartSetIDAndAppData{
		CountWord: 1,
		Hash:      make([]byte, 32),
	}
	vm.BlockPartSetIDAndNTSVoteCount.Hash[0] = firstByteOfHash
	return vm
}

func TestVoteSet_Add_ConflictingVoteAfterConsensus(t *testing.T) {
	assert := assert.New(t)
	vs := newVoteSet(4)
	vs.add(0, newVoteMsgWithHashPrefix(0))
	vs.add(1, newVoteMsgWithHashPrefix(0))
	vs.add(2, newVoteMsgWithHashPrefix(0))
	rdd, psid, ok := vs.getOverTwoThirdsRoundDecisionDigest()
	assert.True(ok)
	assert.False(vs.add(0, newVoteMsgWithHashPrefix(1)))
	assert.EqualValues(3, vs.count)
	rdd2, psid2, ok := vs.getOverTwoThirdsRoundDecisionDigest()
	assert.True(ok)
	assert.EqualValues(rdd, rdd2)
	assert.EqualValues(psid, psid2)
}

func TestVoteSet_Add_ConflictingVoteBeforeConsensus(t *testing.T) {
	assert := assert.New(t)
	vs := newVoteSet(4)
	vs.add(0, newVoteMsgWithHashPrefix(0))
	vs.add(1, newVoteMsgWithHashPrefix(1))
	vs.add(2, newVoteMsgWithHashPrefix(1))
	_, _, ok := vs.getOverTwoThirdsRoundDecisionDigest()
	assert.False(ok)
	assert.True(vs.add(0, newVoteMsgWithHashPrefix(1)))
	assert.EqualValues(3, vs.count)
	_, _, ok = vs.getOverTwoThirdsRoundDecisionDigest()
	assert.True(ok)
}

func TestVoteSet_voteListForOverTwoThirds(t *testing.T) {
	assert := assert.New(t)
	vs := newVoteSet(4)
	vs.add(0, newVoteMsgWithHashPrefix(1))
	vs.add(1, newVoteMsgWithHashPrefix(0))
	vs.add(2, newVoteMsgWithHashPrefix(0))
	vl := vs.voteListForOverTwoThirds()
	assert.Nil(vl)
	vm := newVoteMsgWithHashPrefix(0)
	vs.add(0, vm)
	vl = vs.voteListForOverTwoThirds()
	assert.EqualValues(3, vl.Len())
	for i := 0; i < vl.Len(); i++ {
		assert.True(vl.Get(i).Equal(&vm.voteBase))
	}
}

func newVoteMsgWithRound(round int32) *VoteMessage {
	vm := newVoteMessage()
	vm.BlockID = make([]byte, 32)
	vm.Round = round
	return vm
}

func TestVoteSet_getRoundEvidences(t *testing.T) {
	assert := assert.New(t)
	bid := make([]byte, 32)

	vs := newVoteSet(4)
	vs.add(0, newVoteMsgWithRound(3))
	vs.add(1, newVoteMsgWithRound(3))
	vs.add(2, newVoteMsgWithRound(3))
	vl := vs.getRoundEvidences(9, bid)
	assert.Nil(vl)

	vs = newVoteSet(4)
	vs.add(0, newVoteMsgWithRound(9))
	vl = vs.getRoundEvidences(9, bid)
	assert.Nil(vl)

	vs = newVoteSet(4)
	vs.add(0, newVoteMsgWithRound(9))
	vs.add(1, newVoteMsgWithRound(9))
	vl = vs.getRoundEvidences(9, bid)
	assert.NotNil(vl)
	assert.Equal(2, vl.Len())
	for i := 0; i < vl.Len(); i++ {
		assert.EqualValues(9, vl.Get(i).Round)
	}

	vs = newVoteSet(4)
	vs.add(0, newVoteMsgWithRound(10))
	vs.add(1, newVoteMsgWithRound(10))
	vs.add(2, newVoteMsgWithRound(10))
	vl = vs.getRoundEvidences(9, bid)
	assert.NotNil(vl)
	assert.Equal(3, vl.Len())
	for i := 0; i < vl.Len(); i++ {
		assert.EqualValues(10, vl.Get(i).Round)
	}
}

func TestHeightVoteSet_getRoundEvidences(t *testing.T) {
	assert := assert.New(t)
	var hvs heightVoteSet
	bid := make([]byte, 32)

	hvs.reset(4)
	hvs.add(0, newVoteMsgWithRound(3))
	hvs.add(1, newVoteMsgWithRound(3))
	hvs.add(2, newVoteMsgWithRound(3))
	vl := hvs.getRoundEvidences(9, bid)
	assert.Nil(vl)

	hvs.add(0, newVoteMsgWithRound(9))
	vl = hvs.getRoundEvidences(9, bid)
	assert.Nil(vl)

	hvs.add(1, newVoteMsgWithRound(9))
	vl = hvs.getRoundEvidences(9, bid)
	assert.NotNil(vl)
	assert.Equal(2, vl.Len())
	for i := 0; i < vl.Len(); i++ {
		assert.EqualValues(9, vl.Get(i).Round)
	}

	hvs.add(0, newVoteMsgWithRound(10))
	hvs.add(1, newVoteMsgWithRound(10))
	hvs.add(2, newVoteMsgWithRound(10))
	vl = hvs.getRoundEvidences(9, bid)
	assert.NotNil(vl)
	assert.GreaterOrEqual(vl.Len(), 2)
	for i := 0; i < vl.Len(); i++ {
		assert.GreaterOrEqual(vl.Get(i).Round, int32(9))
	}

	hvs.removeLowerRoundExcept(10, 3)
	vs := hvs.votesFor(3, VoteTypePrevote)
	assert.Equal(3, vs.count)
	assert.EqualValues(3, vs.voteList().Get(0).Round)
	vl = hvs.getRoundEvidences(9, bid)
	assert.NotNil(vl)
	assert.Equal(3, vl.Len())
	for i := 0; i < vl.Len(); i++ {
		assert.EqualValues(10, vl.Get(i).Round)
	}
}
