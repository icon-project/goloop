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

package ictest

import (
	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/icdb"
	"github.com/icon-project/goloop/icon/merkle/hexary"
	"github.com/icon-project/goloop/test"
)

func NodeFinalizeMerkle(n* test.Node) *hexary.MerkleHeader {
	t := n.T
	temp, err := n.Chain.Database().GetBucket("temp")
	assert.NoError(t, err)
	mkl, err := n.Chain.Database().GetBucket(icdb.BlockMerkle)
	assert.NoError(t, err)
	ac, err := hexary.NewAccumulator(mkl, temp, "")
	for i:=int64(0); i<=n.LastBlock.Height(); i++ {
		blk , err := n.BM.GetBlockByHeight(i)
		assert.NoError(t, err)
		err = ac.Add(blk.Hash())
		assert.NoError(t, err)
	}
	header, err := ac.Finalize()
	assert.NoError(t, err)
	return header
}

func NodeNewVoteListV1ForLastBlock(t *test.Node) *blockv0.BlockVoteList {
	bv := blockv0.NewBlockVote(
		t.Chain.Wallet(),
		t.LastBlock.Height(),
		0,
		t.LastBlock.ID(),
		t.LastBlock.Timestamp()+1,
	)
	return blockv0.NewBlockVoteList(bv)
}
