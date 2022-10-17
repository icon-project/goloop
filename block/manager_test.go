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

package block_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/test"
)

func TestBlockManager_Basics(t *testing.T) {
	nd := test.NewNode(t)
	defer nd.Close()

	nd.AssertLastBlock(nil, module.BlockVersion2)

	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	nd.AssertLastBlock(nd.PrevBlock, module.BlockVersion2)
}

func TestBlockManager_GetBlock(t *testing.T) {
	nd := test.NewNode(t)
	defer nd.Close()
	assert := assert.New(t)

	blk, err := nd.BM.GetBlock(make([]byte, crypto.HashLen))
	assert.Error(err)

	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	blk = nd.GetLastBlock()
	blk2, err := nd.BM.GetBlock(blk.ID())
	assert.NoError(err)
	assert.EqualValues(blk.ID(), blk2.ID())

	for i := 0; i < block.ConfigCacheCap; i++ {
		nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	}
	blk2, err = nd.BM.GetBlock(blk.ID())
	assert.NoError(err)
	assert.EqualValues(blk.ID(), blk2.ID())
}

func TestBlockManager_NewManager(t *testing.T) {
	nd := test.NewNode(t)
	defer nd.Close()
	assert := assert.New(t)

	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	blk := nd.GetLastBlock()
	db := nd.Chain.Database()

	nd2 := test.NewNode(t, test.UseDB(db))
	defer nd2.Close()
	blk2 := nd2.GetLastBlock()
	assert.EqualValues(blk.ID(), blk2.ID())
}

func TestBlockManager_ImportBlock_OK(t *testing.T) {
	nd := test.NewNode(t)
	defer nd.Close()
	assert := assert.New(t)

	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	blk := nd.GetLastBlock()

	ch := make(chan module.BlockCandidate)
	nd2 := test.NewNode(t)
	_, err := nd2.BM.ImportBlock(blk, 0, func(bc module.BlockCandidate, err error) {
		ch <- bc
	})
	assert.NoError(err)
	blk2 := <-ch
	assert.EqualValues(blk.ID(), blk2.ID())
}

func TestFreeFunctions(t *testing.T) {
	nd := test.NewNode(t)
	defer nd.Close()
	assert := assert.New(t)

	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	db := nd.Chain.Database()
	blk := nd.GetLastBlock()

	hash, err := block.GetBlockHeaderHashByHeight(db, nil, 1)
	assert.NoError(err)
	assert.EqualValues(blk.ID(), hash)

	ver, err := block.GetBlockVersion(db, nil, 1)
	assert.NoError(err)
	assert.EqualValues(module.BlockVersion2, ver)

	// add one more block
	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())

	cvlBytes, err := block.GetCommitVoteListBytesByHeight(db, nil, 1)
	assert.NoError(err)
	assert.EqualValues(consensus.NewEmptyCommitVoteList().Bytes(), cvlBytes)

	h, err := block.GetLastHeight(db)
	assert.NoError(err)
	assert.EqualValues(2, h)
	assert.EqualValues(2, block.GetLastHeightOf(db))

	err = block.ResetDB(db, nil, 1)
	assert.NoError(err)
	assert.EqualValues(1, block.GetLastHeightOf(db))
}
