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
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"math/big"
)

type BlockProduce struct {
	icobject.NoDatabase
	ProposerIndex int
	VoteCount     int
	VoteMask      *big.Int
}

func (bp *BlockProduce) Version() int {
	return 0
}

func (bp *BlockProduce) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&bp.ProposerIndex,
		&bp.VoteCount,
		&bp.VoteMask,
	)
	return err
}

func (bp *BlockProduce) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		bp.ProposerIndex,
		bp.VoteCount,
		bp.VoteMask,
	)
}

func (bp *BlockProduce) Equal(o icobject.Impl) bool {
	if bp2, ok := o.(*BlockProduce); ok {
		return bp.ProposerIndex == bp2.ProposerIndex &&
			bp.VoteCount == bp2.VoteCount &&
			bp.VoteMask.Cmp(bp2.VoteMask) == 0
	} else {
		return false
	}
}

func (bp *BlockProduce) Clear() {
	bp.ProposerIndex = 0
	bp.VoteCount = 0
	bp.VoteMask = nil
}

func (bp *BlockProduce) IsEmpty() bool {
	return bp.VoteCount == 0
}

func newBlockProduce(tag icobject.Tag) *BlockProduce {
	return new(BlockProduce)
}
