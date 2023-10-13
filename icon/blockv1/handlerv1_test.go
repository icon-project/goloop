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

package blockv1_test

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/icon"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/blockv1"
	"github.com/icon-project/goloop/icon/ictest"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/test"
)

func newFixture(t test.T) *test.Node {
	return test.NewNode(t, ictest.UseBMForBlockV1)
}

func TestHandler_Basics(t_ *testing.T) {
	t := newFixture(t_)
	defer t.Close()

	t.AssertLastBlock(nil, module.BlockVersion1)
	t.ProposeImportFinalizeBlock((*blockv0.BlockVoteList)(nil))
	t.AssertLastBlock(t.PrevBlock, module.BlockVersion1)
	t.ProposeImportFinalizeBlock((*blockv0.BlockVoteList)(nil))
	t.AssertLastBlock(t.PrevBlock, module.BlockVersion1)
}

func TestHandler_BlockV13(t_ *testing.T) {
	t := newFixture(t_)
	defer t.Close()

	t.AssertLastBlock(nil, module.BlockVersion1)
	bc := t.ProposeBlock((*blockv0.BlockVoteList)(nil))

	var b blockv1.Format
	b.Version = module.BlockVersion1
	b.Height = t.LastBlock.Height() + 1
	b.PrevHash = t.LastBlock.Hash()
	b.PrevID = t.LastBlock.ID()
	b.Result = bc.Result()
	b.VersionV0 = blockv0.Version03
	bs := codec.MustMarshalToBytes(&b)
	t.ImportFinalizeBlockByReader(bytes.NewReader(bs))
	t.AssertLastBlock(t.PrevBlock, module.BlockVersion1)
}

func TestHandler_ValidatorChange(t_ *testing.T) {
	t := newFixture(t_)
	defer t.Close()

	t.AssertLastBlock(nil, module.BlockVersion1)

	nilVotes := (*blockv0.BlockVoteList)(nil)
	t.ProposeImportFinalizeBlockWithTX(nilVotes, fmt.Sprintf(`{
		"type": "test",
		"timestamp": "0x0",
		"validators": [ "%s" ]
	}`, t.Chain.Wallet().Address()))
	assert.Nil(t, t.LastBlock.NextValidatorsHash())

	t.ProposeImportFinalizeBlock(nilVotes)
	// now the tx is applied in the tx
	assert.NotNil(t, t.LastBlock.NextValidatorsHash())
	// still no votes
	assert.Nil(t, t.LastBlock.Votes())

	t.ProposeImportFinalizeBlock(ictest.NodeNewVoteListV1ForLastBlock(t))
	// now there is some votes
	assert.NotNil(t, t.LastBlock.Votes())
}

func TestHandler_BlockVersionChange(t_ *testing.T) {
	t := newFixture(t_)
	defer t.Close()

	t.AssertLastBlock(nil, module.BlockVersion1)

	nilVotes := (*blockv0.BlockVoteList)(nil)
	t.ProposeImportFinalizeBlockWithTX(nilVotes, fmt.Sprintf(`{
		"type": "test",
		"timestamp": "0x0",
		"validators": [ "%s" ]
	}`, t.Chain.Wallet().Address()))
	assert.Nil(t, t.LastBlock.NextValidatorsHash())

	t.ProposeImportFinalizeBlock(nilVotes)
	// now the tx is applied in the tx
	assert.NotNil(t, t.LastBlock.NextValidatorsHash())
	// still no votes
	assert.Nil(t, t.LastBlock.Votes())

	t.ProposeImportFinalizeBlock(ictest.NodeNewVoteListV1ForLastBlock(t))
	// now there is some votes
	assert.NotNil(t, t.LastBlock.Votes())

	t.ProposeImportFinalizeBlockWithTX(
		ictest.NodeNewVoteListV1ForLastBlock(t),
		`{
			"type": "test",
			"timestamp": "0x0",
			"nextBlockVersion": "0x2"
		}`,
	)
	assert.EqualValues(
		t, module.BlockVersion1, t.SM.GetNextBlockVersion(t.LastBlock.Result()),
	)

	t.ProposeImportFinalizeBlock(ictest.NodeNewVoteListV1ForLastBlock(t))
	t.AssertLastBlock(t.PrevBlock, module.BlockVersion1)
	// now the tx is applied in the tx
	assert.EqualValues(
		t, module.BlockVersion2, t.SM.GetNextBlockVersion(t.LastBlock.Result()),
	)

	t.ProposeImportFinalizeBlock(t.NewVoteListForLastBlock())
	t.AssertLastBlock(t.PrevBlock, module.BlockVersion2)

	t.ProposeImportFinalizeBlock(t.NewVoteListForLastBlock())
	t.AssertLastBlock(t.PrevBlock, module.BlockVersion2)
}

func TestNewBlockManager_DefaultIsV1AndPrevBlockOfLastBlockIsNonGenesisV2(t *testing.T) {
	nd := test.NewNode(t, ictest.UseBMForBlockV1)
	defer nd.Close()
	assert := assert.New(t)

	nilVotes := (*blockv0.BlockVoteList)(nil)
	nd.ProposeFinalizeBlockWithTX(
		nilVotes,
		`{
			"type": "test",
			"timestamp": "0x0",
			"nextBlockVersion": "0x2"
		}`,
	)
	assert.EqualValues(module.BlockVersion1, nd.LastBlock.Version())
	nd.ProposeFinalizeBlock(nilVotes)
	assert.EqualValues(module.BlockVersion1, nd.LastBlock.Version())
	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	assert.EqualValues(module.BlockVersion2, nd.LastBlock.Version())
	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	assert.EqualValues(module.BlockVersion2, nd.LastBlock.Version())
	blk := nd.LastBlock

	nd2 := test.NewNode(t, ictest.UseBMForBlockV1, test.UseDB(nd.Chain.Database()))
	blk2, err := nd2.BM.GetLastBlock()
	assert.NoError(err)
	assert.Equal(blk.ID(), blk2.ID())
}

func FuzzHandler_NewBlockDataFromReader(fz *testing.F) {
	nd := test.NewNode(fz, ictest.UseBMForBlockV1)
	defer nd.Close()

	nilVotes := (*blockv0.BlockVoteList)(nil)
	nd.ProposeFinalizeBlockWithTX(
		nilVotes,
		`{
			"type": "test",
			"timestamp": "0x0",
			"nextBlockVersion": "0x2"
		}`,
	)

	plf, err := icon.NewPlatform("", nd.Chain.CID())
	assert.NoError(fz, err)
	bh := plf.NewBlockHandlers(nd.Chain)
	bdf, err := block.NewBlockDataFactory(nd.Chain, bh)
	assert.NoError(fz, err)
	testcases := [][]byte{
		[]byte("\xcb\x02000000\x80\x8000\xc8\xc0\xc0\x85\xc4\xc0\xc1\xc000"),
	}
	for _, tc := range testcases {
		fz.Add(tc)
	}
	fz.Fuzz(func(t *testing.T, bs []byte) {
		bd, err := bdf.NewBlockDataFromReader(bytes.NewReader(bs))
		if err != nil {
			return
		}
		bd.Marshal(io.Discard)
	})
}
