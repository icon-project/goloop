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

package consensus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/consensus/fastsync"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/test"
)

func TestConsensus_FastSyncServer(t *testing.T) {
	f := test.NewNode(t)
	defer f.Close()

	const maxHeight = 2
	blks := make([][]byte, maxHeight)
	f.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())

	err := f.CS.Start()
	assert.NoError(t, err)

	for h := 0; h < len(blks); h++ {
		blk, err := f.BM.GetBlockByHeight(int64(h))
		assert.NoError(t, err)
		blks[h], err = module.BlockDataToBytes(blk)
		assert.NoError(t, err)
	}

	_, h1 := f.NM.NewPeerFor(module.ProtoFastSync)
	for h := 0; h < len(blks); h++ {
		h1.Unicast(
			fastsync.ProtoBlockRequest,
			&fastsync.BlockRequest{
				RequestID:   uint32(h),
				Height:      int64(h),
				ProofOption: 0,
			},
			nil,
		)
	}
	for h := 0; h < len(blks); h++ {
		h1.AssertReceiveUnicast(
			fastsync.ProtoBlockMetadata,
			&fastsync.BlockMetadata{
				RequestID:   uint32(h),
				BlockLength: int32(len(blks[h])),
				Proof:       consensus.NewEmptyCommitVoteList().Bytes(),
			},
		)
		var bs []byte
		for len(bs) < len(blks[h]) {
			var bd fastsync.BlockData
			_ = h1.Receive(
				fastsync.ProtoBlockData,
				nil,
				&bd,
			)
			assert.EqualValues(t, h, bd.RequestID)
			bs = append(bs, bd.Data...)
		}
		assert.Equal(t, blks[h], bs)
	}
}

func TestConsensus_FastSyncServerFail(t *testing.T) {
	f := test.NewNode(t)
	defer f.Close()
	err := f.CS.Start()
	assert.NoError(t, err)

	_, h1 := f.NM.NewPeerFor(module.ProtoFastSync)
	h1.Unicast(
		fastsync.ProtoBlockRequest,
		&fastsync.BlockRequest{
			RequestID:   0,
			Height:      1,
			ProofOption: 0,
		},
		nil,
	)
	h1.AssertReceiveUnicast(
		fastsync.ProtoBlockMetadata,
		&fastsync.BlockMetadata{
			RequestID:   0,
			BlockLength: -1,
			Proof:       nil,
		},
	)
}

func TestConsensus_ClientBasics(t *testing.T) {
	f := test.NewNode(t)
	defer f.Close()

	err := f.CS.Start()
	assert.NoError(t, err)

	_, csh := f.NM.NewPeerFor(module.ProtoConsensusSync)
	fsh := csh.Peer().Join(module.ProtoFastSync)

	var rsm consensus.RoundStateMessage
	rsm.Height = 10
	rsm.Sync = true
	csh.Unicast(consensus.ProtoRoundState, &rsm, nil)

	var brm fastsync.BlockRequest
	fsh.Receive(fastsync.ProtoBlockRequest, nil, &brm)
	assert.EqualValues(t, 1, brm.Height)
}

func TestConsensus_BasicConsensus(t *testing.T) {
	f := test.NewNode(t)
	defer f.Close()

	h := make([]*test.SimplePeerHandler, 3)
	for i := 0; i < len(h); i++ {
		_, h[i] = f.NM.NewPeerFor(module.ProtoConsensus)
	}

	f.ProposeImportFinalizeBlockWithTX(
		consensus.NewEmptyCommitVoteList(),
		test.NewTx().SetValidatorsAddresser(
			h[0], h[1], h[2], f.Chain.Wallet(),
		).String(),
	)
	f.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())

	err := f.CS.Start()
	assert.NoError(t, err)

	var pm consensus.ProposalMessage
	h[0].Receive(
		consensus.ProtoProposal,
		nil,
		&pm,
	)
	assert.EqualValues(t, pm.Height, 3)
	assert.EqualValues(t, pm.Round, 0)

	ps := consensus.NewPartSetFromID(pm.BlockPartSetID)
	for !ps.IsComplete() {
		var bpm consensus.BlockPartMessage
		h[0].Receive(consensus.ProtoBlockPart, nil, &bpm)
		pt, err := consensus.NewPart(bpm.BlockPart)
		assert.NoError(t, err)
		err = ps.AddPart(pt)
		assert.NoError(t, err)
	}
	blk, err := f.BM.NewBlockDataFromReader(ps.NewReader())
	assert.NoError(t, err)

	for i := 0; i < len(h); i++ {
		h[i].Unicast(
			consensus.ProtoVote,
			consensus.NewVoteMessage(
				h[i].Wallet(),
				consensus.VoteTypePrevote, 3, 0, blk.ID(),
				ps.ID(), blk.Timestamp()+1,
			),
			func(rb bool, e error) {
				assert.True(t, rb)
				assert.NoError(t, e)
			},
		)
	}

	for i := 0; i < len(h); i++ {
		h[i].Unicast(
			consensus.ProtoVote,
			consensus.NewVoteMessage(
				h[i].Wallet(),
				consensus.VoteTypePrecommit, 3, 0, blk.ID(),
				ps.ID(), blk.Timestamp()+1,
			),
			func(rb bool, e error) {
				assert.True(t, rb)
				assert.NoError(t, e)
			},
		)
	}

	hcs0 := h[0].Peer().Join(module.ProtoConsensusSync)
	for {
		var rs consensus.RoundStateMessage
		hcs0.Receive(consensus.ProtoRoundState, nil, &rs)
		if rs.Height == 4 {
			break
		}
	}
}

func TestConsensus_BasicConsensus2(t *testing.T) {
	f := test.NewFixture(t, test.AddValidatorNodes(4))
	defer f.Close()

	test.NodeInterconnect(f.Nodes)
	for _, n := range f.Nodes {
		err := n.CS.Start()
		assert.NoError(t, err)
	}
	chn, err := f.BM.WaitForBlock(3)
	assert.NoError(t, err)
	blk := <-chn
	assert.EqualValues(t, 3, blk.Height())
	assert.EqualValues(t, 4, f.CS.GetStatus().Height)
}
