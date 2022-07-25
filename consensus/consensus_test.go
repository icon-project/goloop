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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/consensus/fastsync"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/platform/basic"
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
	fsh := csh.Peer().RegisterProto(module.ProtoFastSync)

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
				ps.ID(), blk.Timestamp()+1, nil, nil,
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
				ps.ID(), blk.Timestamp()+1, nil, nil,
			),
			func(rb bool, e error) {
				assert.True(t, rb)
				assert.NoError(t, e)
			},
		)
	}

	hcs0 := h[0].Peer().RegisterProto(module.ProtoConsensusSync)
	for {
		var rs consensus.RoundStateMessage
		hcs0.Receive(consensus.ProtoRoundState, nil, &rs)
		if rs.Height == 4 {
			break
		}
	}
}

func TestConsensus_BasicConsensus2(t *testing.T) {
	f := test.NewFixture(t,
		test.AddDefaultNode(false),
		test.AddValidatorNodes(4),
	)
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

func TestConsensus_NoNTSVoteCountForFirstNTS(t *testing.T) {
	f := test.NewNode(t)
	defer f.Close()

	h := make([]*test.SimplePeerHandler, 3)
	for i := 0; i < len(h); i++ {
		_, h[i] = f.NM.NewPeerFor(module.ProtoConsensus)
	}

	f.ProposeFinalizeBlockWithTX(
		consensus.NewEmptyCommitVoteList(),
		test.NewTx().SetValidatorsAddresser(
			h[0], h[1], h[2], f.Chain.Wallet(),
		).Call("setRevision", map[string]string{
			"code": fmt.Sprintf("0x%x", basic.MaxRevision),
		}).CallFrom(f.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   "eth",
			"pubKey": fmt.Sprintf("0x%x", f.Chain.WalletFor("eth").PublicKey()),
		}).Call("openBTPNetwork", map[string]string{
			"networkTypeName": "eth",
			"name":            "eth-test",
			"owner":           f.CommonAddress().String(),
		}).String(),
	)
	f.ProposeFinalizeBlockWithTX(
		consensus.NewEmptyCommitVoteList(),
		test.NewTx().Call("openBTPNetwork", map[string]string{
			"networkTypeName": "eth",
			"name":            "eth-test",
			"owner":           f.CommonAddress().String(),
		}).String(),
	)
	err := f.CS.Start()
	assert.NoError(t, err)

	var pm consensus.ProposalMessage
	h[0].Receive(
		consensus.ProtoProposal,
		nil,
		&pm,
	)
	assert.EqualValues(t, 3, pm.Height)
	assert.EqualValues(t, 0, pm.Round)
	assert.EqualValues(t, 0, pm.BlockPartSetID.AppData())
}

func TestConsensus_BTPBasic(t_ *testing.T) {
	assert := assert.New(t_)
	f := test.NewFixture(t_, test.AddDefaultNode(false), test.AddValidatorNodes(4))
	defer f.Close()

	h := make([]*test.SimplePeerHandler, 3)
	for i := 0; i < len(h); i++ {
		_, h[i] = f.NM.NewPeerForWithAddress(module.ProtoConsensus, f.Validators[i+1].Chain.Wallet())
	}

	tx := test.NewTx().Call("setRevision", map[string]string{
		"code": fmt.Sprintf("0x%x", basic.MaxRevision),
	})
	for _, v := range f.Validators {
		tx.CallFrom(v.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   "eth",
			"pubKey": fmt.Sprintf("0x%x", v.Chain.WalletFor("eth").PublicKey()),
		})
	}
	f.ProposeFinalizeBlockWithTX(
		consensus.NewEmptyCommitVoteList(),
		tx.Call("openBTPNetwork", map[string]string{
			"networkTypeName": "eth",
			"name":            "eth-test",
			"owner":           f.CommonAddress().String(),
		}).String(),
	)
	f.ProposeFinalizeBlock(f.NewCommitVoteListForLastBlock(0, 0))

	testMsg := ([]byte)("test message")
	f.ProposeFinalizeBlockWithTX(
		f.NewCommitVoteListForLastBlock(0, 0),
		test.NewTx().CallFrom(f.CommonAddress(), "sendBTPMessage", map[string]string{
			"networkId": "0x1",
			"message":   fmt.Sprintf("0x%x", testMsg),
		}).String(),
	)

	cvl := f.NewCommitVoteListForLastBlock(0, 0)
	// start consensus from block(height=4) proposerIndex=0
	err := consensus.StartConsensusWithLastVotes(f.CS, &consensus.LastVoteData{
		Height:     f.LastBlock.Height(),
		VotesBytes: cvl.Bytes(),
	})
	assert.NoError(err)

	var pm consensus.ProposalMessage
	h[0].Receive(
		consensus.ProtoProposal,
		nil,
		&pm,
	)
	assert.EqualValues(4, pm.Height)
	assert.EqualValues(0, pm.Round)
	assert.EqualValues(1, pm.BlockPartSetID.AppData())

	ps := consensus.NewPartSetFromID(pm.BlockPartSetID)
	for !ps.IsComplete() {
		var bpm consensus.BlockPartMessage
		h[0].Receive(consensus.ProtoBlockPart, nil, &bpm)
		pt, err := consensus.NewPart(bpm.BlockPart)
		assert.NoError(err)
		err = ps.AddPart(pt)
		assert.NoError(err)
	}
	blk, err := f.BM.NewBlockDataFromReader(ps.NewReader())
	assert.NoError(err)
	bd, err := blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	for i := 0; i < len(h); i++ {
		h[i].Unicast(
			consensus.ProtoVote,
			consensus.NewVoteMessage(
				h[i].Wallet(),
				consensus.VoteTypePrevote, 4, 0, blk.ID(),
				ps.ID(), blk.Timestamp()+1, nil, nil,
			),
			func(rb bool, e error) {
				assert.True(rb)
				assert.NoError(e)
			},
		)
	}

	precommits := f.NewPrecommitList(blk, 0, 1)
	for i := 0; i < len(h); i++ {
		precommit := precommits[i+1]
		h[i].Unicast(
			consensus.ProtoVote,
			precommit,
			func(rb bool, e error) {
				assert.NoError(e)
			},
		)
	}

	hcs0 := h[0].Peer().RegisterProto(module.ProtoConsensusSync)
	for {
		var rs consensus.RoundStateMessage
		hcs0.Receive(consensus.ProtoRoundState, nil, &rs)
		if rs.Height == 5 {
			break
		}
	}

	f.UpdateLastBlock()
	assert.EqualValues(4, f.LastBlock.Height())
	bbh, pfBytes, err := f.CS.GetBTPBlockHeaderAndProof(
		f.LastBlock, 1,
		module.FlagBTPBlockHeader|module.FlagBTPBlockProof,
	)
	assert.NoError(err)
	assert.EqualValues(1, bbh.NetworkID())
	assert.EqualValues(0, bbh.FirstMessageSN())
	prevBlk, err := f.BM.GetBlockByHeight(f.LastBlock.Height() - 1)
	assert.NoError(err)
	pcm, err := prevBlk.NextProofContextMap()
	assert.NoError(err)
	pc, err := pcm.ProofContextFor(1)
	assert.NoError(err)
	pf, err := pc.NewProofFromBytes(pfBytes)
	assert.NoError(err)
	bd, err = blk.BTPDigest()
	ntsd := pc.NewDecision(module.SourceNetworkUID(1), 1, 4, 0, bd.NetworkTypeDigestFor(1).NetworkTypeSectionHash())
	err = pc.Verify(ntsd.Hash(), pf)
	assert.NoError(err)
}

func TestConsensus_BTPBlockBasic(t_ *testing.T) {
	assert := assert.New(t_)
	f := test.NewFixture(t_, test.AddDefaultNode(false), test.AddValidatorNodes(4))
	defer f.Close()

	h := make([]*test.SimplePeerHandler, 3)
	for i := 0; i < len(h); i++ {
		_, h[i] = f.NM.NewPeerForWithAddress(module.ProtoConsensus, f.Validators[i+1].Chain.Wallet())
	}

	tx := test.NewTx().Call("setRevision", map[string]string{
		"code": fmt.Sprintf("0x%x", basic.MaxRevision),
	})
	for _, v := range f.Validators {
		tx.CallFrom(v.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   "eth",
			"pubKey": fmt.Sprintf("0x%x", v.Chain.WalletFor("eth").PublicKey()),
		})
	}
	f.ProposeFinalizeBlockWithTX(
		consensus.NewEmptyCommitVoteList(),
		tx.Call("openBTPNetwork", map[string]string{
			"networkTypeName": "eth",
			"name":            "eth-test",
			"owner":           f.CommonAddress().String(),
		}).String(),
	)
	f.ProposeFinalizeBlock(f.NewCommitVoteListForLastBlock(0, 0))

	cvl := f.NewCommitVoteListForLastBlock(0, 0)

	err := consensus.StartConsensusWithLastVotes(f.CS, &consensus.LastVoteData{
		Height:     f.LastBlock.Height(),
		VotesBytes: cvl.Bytes(),
	})
	assert.NoError(err)
	assert.EqualValues(2, f.LastBlock.Height())
	bbh, _, err := f.CS.GetBTPBlockHeaderAndProof(
		f.LastBlock, 1,
		module.FlagBTPBlockHeader,
	)
	assert.NoError(err)
	assert.EqualValues(1, bbh.NetworkID())
	assert.EqualValues(0, bbh.FirstMessageSN())
}

func TestConsensus_ChangeBTPKey(t_ *testing.T) {
	testChangeBTPKey("eth", t_)
	testChangeBTPKey("icon", t_)
}

func testChangeBTPKey(uid string, t_ *testing.T) {
	assert := assert.New(t_)
	f := test.NewFixture(t_, test.AddDefaultNode(false), test.AddValidatorNodes(4))
	defer f.Close()

	tx := test.NewTx().Call("setRevision", map[string]string{
		"code": fmt.Sprintf("0x%x", basic.MaxRevision),
	}).Call("setMinimizeBlockGen", map[string]string{
		"yn": fmt.Sprintf("0x1"),
	})
	for i, v := range f.Validators {
		tx.CallFrom(v.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   uid,
			"pubKey": fmt.Sprintf("0x%x", v.Chain.WalletFor(uid).PublicKey()),
		})
		t_.Logf("register %s key index=%d key=%x", uid, i, v.Chain.WalletFor(uid).PublicKey())
	}
	tx.Call("openBTPNetwork", map[string]string{
		"networkTypeName": uid,
		"name":            fmt.Sprintf("%s-test", uid),
		"owner":           f.CommonAddress().String(),
	})

	f.SendTransactionToProposer(tx)

	test.NodeInterconnect(f.Nodes)
	for _, n := range f.Nodes {
		err := n.CS.Start()
		assert.NoError(err)
	}

	blk := f.WaitForBlock(2)
	bd, err := blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	wp := test.NewWalletProvider()
	wp2 := test.NewWalletProvider()
	tx = test.NewTx().CallFrom(f.CommonAddress(), "setBTPPublicKey", map[string]string{
		"name":   uid,
		"pubKey": fmt.Sprintf("0x%x", wp.WalletFor(uid).PublicKey()),
	}).CallFrom(f.Nodes[1].CommonAddress(), "setBTPPublicKey", map[string]string{
		"name":   uid,
		"pubKey": fmt.Sprintf("0x%x", wp2.WalletFor(uid).PublicKey()),
	}).SetTimestamp(blk.Timestamp())
	f.SendTransactionToProposer(tx)

	blk = f.WaitForBlock(3)
	_, err = blk.NormalTransactions().Get(0)
	assert.NoError(err)

	blk = f.WaitForBlock(4)
	bd, err = blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	testMsg := ([]byte)("test message")
	tx = test.NewTx().CallFrom(f.CommonAddress(), "sendBTPMessage", map[string]string{
		"networkId": "0x1",
		"message":   fmt.Sprintf("0x%x", testMsg),
	}).SetTimestamp(blk.Timestamp())
	f.SendTransactionToProposer(tx)

	blk = f.WaitForBlock(5)
	_, err = blk.NormalTransactions().Get(0)
	assert.NoError(err)

	f.Nodes[0].Chain.SetWalletFor(uid, wp.WalletFor(uid))
	f.Nodes[1].Chain.SetWalletFor(uid, wp2.WalletFor(uid))
	blk = f.WaitForBlock(6)
}

func TestConsensus_SetWrongBTPKey(t_ *testing.T) {
	testSetWrongBTPKey("eth", t_)
	testSetWrongBTPKey("icon", t_)
}

func testSetWrongBTPKey(uid string, t_ *testing.T) {
	assert := assert.New(t_)
	f := test.NewFixture(t_, test.AddDefaultNode(false), test.AddValidatorNodes(4))
	defer f.Close()

	tx := test.NewTx().Call("setRevision", map[string]string{
		"code": fmt.Sprintf("0x%x", basic.MaxRevision),
	}).Call("setMinimizeBlockGen", map[string]string{
		"yn": fmt.Sprintf("0x1"),
	})
	for i, v := range f.Validators {
		tx.CallFrom(v.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   uid,
			"pubKey": fmt.Sprintf("0x%x", v.Chain.WalletFor(uid).PublicKey()),
		})
		t_.Logf("register %s key index=%d key=%x", uid, i, v.Chain.WalletFor(uid).PublicKey())
	}
	tx.Call("openBTPNetwork", map[string]string{
		"networkTypeName": uid,
		"name":            fmt.Sprintf("%s-test", uid),
		"owner":           f.CommonAddress().String(),
	})
	f.SendTransactionToProposer(tx)

	test.NodeInterconnect(f.Nodes)
	for _, n := range f.Nodes {
		err := n.CS.Start()
		assert.NoError(err)
	}

	blk := f.WaitForBlock(2)
	bd, err := blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	wp := test.NewWalletProvider()
	wp2 := test.NewWalletProvider()
	f.SendTransactionToProposer(
		f.NewTx().CallFrom(f.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   uid,
			"pubKey": fmt.Sprintf("0x%x", wp.WalletFor(uid).PublicKey()),
		}).CallFrom(f.Nodes[1].CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   uid,
			"pubKey": fmt.Sprintf("0x%x", wp2.WalletFor(uid).PublicKey()),
		}),
	)

	blk = f.WaitForNextBlock()
	_, err = blk.NormalTransactions().Get(0)
	assert.NoError(err)

	blk = f.WaitForNextBlock()
	bd, err = blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	testMsg := ([]byte)("test message")
	f.SendTransactionToProposer(
		f.NewTx().CallFrom(f.CommonAddress(), "sendBTPMessage", map[string]string{
			"networkId": "0x1",
			"message":   fmt.Sprintf("0x%x", testMsg),
		}),
	)

	blk = f.WaitForNextBlock()
	_, err = blk.NormalTransactions().Get(0)
	assert.NoError(err)

	f.Nodes[0].Chain.SetWalletFor(uid, wp.WalletFor(uid))
	f.Nodes[1].Chain.SetWalletFor(uid, wp2.WalletFor(uid))
	f.WaitForNextBlock()

	// set wrong pub key
	f.SendTransactionToProposer(
		f.NewTx().CallFrom(f.Nodes[0].CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   uid,
			"pubKey": fmt.Sprintf("0x%x", wp2.WalletFor(uid).PublicKey()),
		}),
	)
	f.WaitForNextNthBlock(2)

	// send message
	f.SendTransactionToProposer(
		f.NewTx().CallFrom(f.CommonAddress(), "sendBTPMessage", map[string]string{
			"networkId": "0x1",
			"message":   fmt.Sprintf("0x%x", testMsg),
		}),
	)
	f.WaitForNextNthBlock(2)

	blk, err = f.BM.GetLastBlock()
	assert.NoError(err)
	votes, err := f.CS.GetVotesByHeight(blk.Height())
	assert.NoError(err)
	assert.EqualValues(3, len(votes.(*consensus.CommitVoteList).Items))
}

func TestConsensus_RevokeValidator(t_ *testing.T) {
	const dsa = "ecdsa/secp256k1"
	const uid = "eth"
	const uid2 = "icon"
	assert := assert.New(t_)
	f := test.NewFixture(t_, test.AddDefaultNode(false), test.AddValidatorNodes(4))
	defer f.Close()

	tx := test.NewTx().Call("setRevision", map[string]string{
		"code": fmt.Sprintf("0x%x", basic.MaxRevision),
	}).Call("setMinimizeBlockGen", map[string]string{
		"yn": fmt.Sprintf("0x1"),
	})
	for i, v := range f.Validators {
		tx.CallFrom(v.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   dsa,
			"pubKey": fmt.Sprintf("0x%x", v.Chain.WalletFor(dsa).PublicKey()),
		})
		t_.Logf("register key index=%d %s=%x %s=%x %s=%x", i, dsa, v.Chain.WalletFor(dsa).PublicKey(), uid, v.Chain.WalletFor(uid).PublicKey(), uid2, v.Chain.WalletFor(uid2).PublicKey())
	}
	tx.Call("openBTPNetwork", map[string]string{
		"networkTypeName": uid,
		"name":            fmt.Sprintf("%s-test", uid),
		"owner":           f.CommonAddress().String(),
	})
	f.SendTransactionToProposer(tx)

	test.NodeInterconnect(f.Nodes)
	for _, n := range f.Nodes {
		err := n.CS.Start()
		assert.NoError(err)
	}

	blk := f.WaitForBlock(2)
	bd, err := blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	f.SendTransactionToProposer(
		f.NewTx().Call("revokeValidator", map[string]string{
			"address": f.Nodes[0].CommonAddress().String(),
		}),
	)
	blk = f.WaitForNextBlock()
	assert.EqualValues(4, blk.NextValidators().Len())

	tx = f.NewTx().Call("openBTPNetwork", map[string]string{
		"networkTypeName": uid2,
		"name":            fmt.Sprintf("%s-test", uid2),
		"owner":           f.CommonAddress().String(),
	})
	f.SendTransactionToProposer(tx)
	blk = f.WaitForNextBlock()
	assert.EqualValues(3, blk.NextValidators().Len())
	bs, err := blk.BTPSection()
	nts, err := bs.NetworkTypeSectionFor(1)
	assert.NoError(err)
	_ = nts.NextProofContext()
	bysl := nts.NextProofContext().Bytes()
	log.Infof("%s", codec.DumpRLP("  ", bysl))

	blk = f.WaitForNextBlock()
	assert.EqualValues(3, blk.NextValidators().Len())
	bs, err = blk.BTPSection()
	nts, err = bs.NetworkTypeSectionFor(2)
	assert.NoError(err)
	_ = nts.NextProofContext()
	bysl = nts.NextProofContext().Bytes()
	log.Infof("%s", codec.DumpRLP("  ", bysl))

	f.SendTransactionToAll(f.NewTx())
	f.WaitForNextBlock()
	f.WaitForNextBlock()

	testMsg := ([]byte)("test message")
	f.SendTransactionToAll(
		f.NewTx().CallFrom(f.CommonAddress(), "sendBTPMessage", map[string]string{
			"networkId": "0x1",
			"message":   fmt.Sprintf("0x%x", testMsg),
		}),
	)
	f.WaitForNextBlock()
	f.WaitForNextBlock()
}
