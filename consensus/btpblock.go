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
	"github.com/icon-project/goloop/btp"
	"github.com/icon-project/goloop/module"
)

func (cs *consensus) GetBTPBlockHeaderAndProof(
	blk module.Block,
	nid int64,
	flag uint,
) (btpBlk module.BTPBlockHeader, proof []byte, err error) {
	bs, err := blk.BTPSection()
	if err != nil {
		return nil, nil, err
	}
	bd := bs.Digest()
	ntid, err := bd.NetworkTypeIDFromNID(nid)
	if err != nil {
		return nil, nil, err
	}
	nts, err := bs.NetworkTypeSectionFor(ntid)
	if err != nil {
		return nil, nil, err
	}
	var cvs module.CommitVoteSet
	if flag&module.FlagBTPBlockHeader != 0 || flag&module.FlagBTPBlockProof != 0 {
		cvs, err = cs.GetVotesByHeight(blk.Height())
		if err != nil {
			return nil, nil, err
		}
	}
	if flag&module.FlagBTPBlockHeader != 0 {
		btpBlk, err = btp.NewBTPBlockHeader(blk.Height(), cvs.VoteRound(), nts, nid, flag)
		if err != nil {
			return nil, nil, err
		}
	}
	if flag&module.FlagBTPBlockProof != 0 {
		prevBlk, err := cs.c.BlockManager().GetBlockByHeight(blk.Height() - 1)
		if err != nil {
			return btpBlk, nil, err
		}
		idx, err := cs.ntsdIndexFor(ntid, bd, prevBlk.Result())
		if err != nil {
			return btpBlk, nil, err
		}
		proof = cvs.NTSDProofAt(idx)
	}
	return btpBlk, proof, nil
}
