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

package icstage

import (
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type BlockProduce struct {
	icobject.NoDatabase
	proposerIndex int
	voteCount     int
	voteMask      *big.Int
}

func (bp *BlockProduce) Version() int {
	return 0
}

func (bp *BlockProduce) ProposerIndex() int {
	return bp.proposerIndex
}

func (bp *BlockProduce) SetProposerIndex(index int) {
	bp.proposerIndex = index
}

func (bp *BlockProduce) VoteCount() int {
	return bp.voteCount
}

func (bp *BlockProduce) SetVoteCount(count int) {
	bp.voteCount = count
}

func (bp *BlockProduce) VoteMask() *big.Int {
	return bp.voteMask
}

func (bp *BlockProduce) SetVoteMask(mask *big.Int) {
	bp.voteMask = mask
}

func (bp *BlockProduce) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&bp.proposerIndex,
		&bp.voteCount,
		&bp.voteMask,
	)
	return err
}

func (bp *BlockProduce) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		bp.proposerIndex,
		bp.voteCount,
		bp.voteMask,
	)
}

func (bp *BlockProduce) Equal(o icobject.Impl) bool {
	if bp2, ok := o.(*BlockProduce); ok {
		return bp.proposerIndex == bp2.proposerIndex &&
			bp.voteCount == bp2.voteCount &&
			bp.voteMask.Cmp(bp2.voteMask) == 0
	} else {
		return false
	}
}

func (bp *BlockProduce) Clear() {
	bp.proposerIndex = 0
	bp.voteCount = 0
	bp.voteMask = new(big.Int)
}

func (bp *BlockProduce) IsEmpty() bool {
	return bp.voteCount == 0
}

func (bp *BlockProduce) String() string {
	return fmt.Sprintf("proposerIndex=%d votecount=%d voteMask=%b",
		bp.proposerIndex, bp.voteCount, bp.voteMask)
}

func (bp *BlockProduce) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "BlockProduce{proposerIndex=%d votecount=%d voteMask=%b}",
				bp.proposerIndex, bp.voteCount, bp.voteMask)
		} else {
			fmt.Fprintf(f, "BlockProduce{%d %d %b}",
				bp.proposerIndex, bp.voteCount, bp.voteMask)
		}
	case 's':
		fmt.Fprint(f, bp.String())
	}
}

func newBlockProduce(_ icobject.Tag) *BlockProduce {
	return new(BlockProduce)
}

func NewBlockProduce(pIndex, vCount int, vMask *big.Int) *BlockProduce {
	return &BlockProduce{
		proposerIndex: pIndex,
		voteCount:     vCount,
		voteMask:      vMask,
	}
}
