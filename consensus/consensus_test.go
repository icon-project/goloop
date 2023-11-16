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
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/btp/ntm"
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
	rsm.PrevotesMask = consensus.NewBitArray(0)
	rsm.PrecommitsMask = consensus.NewBitArray(0)
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
				0,
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
				0,
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
	for f.CS.GetStatus().Height < 4 {
		time.Sleep(200 * time.Millisecond)
	}
}

type btpTest struct {
	*testing.T
	*assert.Assertions
	*test.Fixture
}

func newBTPTest(t *testing.T, opt ...test.FixtureOption) *btpTest {
	const dsa = "ecdsa/secp256k1"
	const uid = "eth"
	assert := assert.New(t)
	opt = append(opt, test.AddDefaultNode(false), test.AddValidatorNodes(4), test.SetTimeoutPropose(4*time.Second))
	f := test.NewFixture(t, opt...)

	tx := test.NewTx().Call("setRevision", map[string]string{
		"code": fmt.Sprintf("0x%x", basic.MaxRevision),
	}).Call("setMinimizeBlockGen", map[string]string{
		"yn": "0x1",
	})
	for i, v := range f.Validators {
		tx.CallFrom(v.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   dsa,
			"pubKey": fmt.Sprintf("0x%x", v.Chain.WalletFor(dsa).PublicKey()),
		})
		pk := v.Chain.WalletFor(dsa).PublicKey()
		iconAddr, err := ntm.NewIconAddressFromPubKey(pk)
		assert.NoError(err)
		ethAddr, err := ntm.ForUID("eth").AddressFromPubKey(pk)
		assert.NoError(err)
		t.Logf("register key index=%d %s=%x icon=%x eth=%x", i, dsa, v.Chain.WalletFor(dsa).PublicKey(), iconAddr, ethAddr)
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
	return &btpTest{
		T:          t,
		Assertions: assert,
		Fixture:    f,
	}
}

func (tst *btpTest) Close() {
	if tst.Fixture != nil {
		tst.Fixture.Close()
	}
}

func TestConsensus_NoNTSVoteCountForFirstNTS(t *testing.T) {
	tst := newBTPTest(t)
	defer tst.Close()
	f := tst.Fixture
	assert := tst.Assertions

	blk := f.WaitForBlock(2)
	bd, err := blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	blk = f.SendTXToAllAndWaitForBlock(f.NewTx())
	assert.EqualValues(0, blk.Votes().NTSDProofCount())
}

func TestConsensus_BTPBasic(t *testing.T) {
	tst := newBTPTest(t)
	defer tst.Close()
	f := tst.Fixture
	assert := tst.Assertions

	blk := f.WaitForBlock(2)
	bd, err := blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))
	assert.EqualValues(1, bd.NetworkTypeDigestFor(1).NetworkTypeID())
	assert.EqualValues(1, bd.NetworkTypeDigestFor(1).NetworkDigestFor(1).NetworkID())

	testMsg := ([]byte)("test message")
	blk = f.SendTXToAllAndWaitForResultBlock(
		f.NewTx().CallFrom(f.CommonAddress(), "sendBTPMessage", map[string]string{
			"networkId": "0x1",
			"message":   fmt.Sprintf("0x%x", testMsg),
		}),
	)
	assert.EqualValues(4, blk.Height())
	bd, err = blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	bbh, pfBytes, err := f.CS.GetBTPBlockHeaderAndProof(
		blk, 1,
		module.FlagBTPBlockHeader|module.FlagBTPBlockProof,
	)
	assert.NoError(err)
	assert.EqualValues(1, bbh.NetworkID())
	assert.EqualValues(0, bbh.FirstMessageSN())
	prevBlk, err := f.BM.GetBlockByHeight(blk.Height() - 1)
	assert.NoError(err)
	pcm, err := prevBlk.NextProofContextMap()
	assert.NoError(err)
	pc, err := pcm.ProofContextFor(1)
	assert.NoError(err)
	pf, err := pc.NewProofFromBytes(pfBytes)
	assert.NoError(err)
	ntsd := pc.NewDecision(module.SourceNetworkUID(1), 1, 4, bbh.Round(), bd.NetworkTypeDigestFor(1).NetworkTypeSectionHash())
	err = pc.Verify(ntsd.Hash(), pf)
	assert.NoError(err)

	// increase height so that early blocks are not in block cache
	for i := 0; i < 10; i++ {
		f.SendTXToAllAndWaitForResultBlock(f.NewTx())
	}

	blk, err = f.BM.GetBlockByHeight(2)
	assert.NoError(err)
	bb, _, err := f.CS.GetBTPBlockHeaderAndProof(blk, 1, module.FlagBTPBlockHeader)
	assert.NoError(err)
	assert.EqualValues(2, bb.MainHeight())

	blk, err = f.BM.GetBlockByHeight(4)
	assert.NoError(err)
	bb, _, err = f.CS.GetBTPBlockHeaderAndProof(blk, 1, module.FlagBTPBlockHeader)
	assert.NoError(err)
	assert.EqualValues(4, bb.MainHeight())
	assert.NotNil(bb.MessagesRoot())
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
	const dsa = "ecdsa/secp256k1"
	for _, v := range f.Validators {
		tx.CallFrom(v.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   dsa,
			"pubKey": fmt.Sprintf("0x%x", v.Chain.WalletFor(dsa).PublicKey()),
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
	assert.EqualValues(2, bbh.MainHeight())
	assert.EqualValues(true, bbh.NextProofContextChanged())
	assert.EqualValues(1, bbh.UpdateNumber())
	assert.EqualValues(0, bbh.MessageCount())
	assert.EqualValues([]byte(nil), bbh.MessagesRoot())
	assert.EqualValues([]byte(nil), bbh.PrevNetworkSectionHash())
	blk := f.LastBlock
	pcm, err := blk.NextProofContextMap()
	assert.NoError(err)
	pc, err := pcm.ProofContextFor(1)
	assert.NoError(err)
	assert.EqualValues(pc.Bytes(), bbh.NextProofContext())
	assert.EqualValues(pc.Hash(), bbh.NextProofContextHash())
	assert.EqualValues(0, len(bbh.NetworkSectionToRoot()))
}

func TestConsensus_ChangeBTPKey(t_ *testing.T) {
	const dsa = "ecdsa/secp256k1"
	tst := newBTPTest(t_)
	defer tst.Close()
	f := tst.Fixture
	assert := tst.Assertions

	blk := f.WaitForBlock(2)
	bd, err := blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	wp := test.NewWalletProvider()
	wp2 := test.NewWalletProvider()
	tx := test.NewTx().CallFrom(f.CommonAddress(), "setBTPPublicKey", map[string]string{
		"name":   dsa,
		"pubKey": fmt.Sprintf("0x%x", wp.WalletFor(dsa).PublicKey()),
	}).CallFrom(f.Nodes[1].CommonAddress(), "setBTPPublicKey", map[string]string{
		"name":   dsa,
		"pubKey": fmt.Sprintf("0x%x", wp2.WalletFor(dsa).PublicKey()),
	}).SetTimestamp(blk.Timestamp())
	blk = f.SendTXToAllAndWaitForBlock(tx)
	_, err = blk.NormalTransactions().Get(0)
	assert.NoError(err)

	blk = f.WaitForNextBlock()
	bd, err = blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	f.Nodes[0].Chain.SetWalletFor(dsa, wp.WalletFor(dsa))
	f.Nodes[1].Chain.SetWalletFor(dsa, wp2.WalletFor(dsa))

	testMsg := ([]byte)("test message")
	tx = test.NewTx().CallFrom(f.CommonAddress(), "sendBTPMessage", map[string]string{
		"networkId": "0x1",
		"message":   fmt.Sprintf("0x%x", testMsg),
	}).SetTimestamp(blk.Timestamp())
	blk = f.SendTXToAllAndWaitForBlock(tx)
	_, err = blk.NormalTransactions().Get(0)
	assert.NoError(err)

	_ = f.WaitForNextBlock()
}

func TestConsensus_SetWrongBTPKey(t_ *testing.T) {
	const dsa = "ecdsa/secp256k1"
	tst := newBTPTest(t_)
	defer tst.Close()
	f := tst.Fixture
	assert := tst.Assertions

	blk := f.WaitForBlock(2)
	bd, err := blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	wp := test.NewWalletProvider()
	wp2 := test.NewWalletProvider()
	blk = f.SendTXToAllAndWaitForBlock(
		f.NewTx().CallFrom(f.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   dsa,
			"pubKey": fmt.Sprintf("0x%x", wp.WalletFor(dsa).PublicKey()),
		}).CallFrom(f.Nodes[1].CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   dsa,
			"pubKey": fmt.Sprintf("0x%x", wp2.WalletFor(dsa).PublicKey()),
		}),
	)
	_, err = blk.NormalTransactions().Get(0)
	assert.NoError(err)

	blk = f.WaitForNextBlock()
	bd, err = blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	f.Nodes[0].Chain.SetWalletFor(dsa, wp.WalletFor(dsa))
	f.Nodes[1].Chain.SetWalletFor(dsa, wp2.WalletFor(dsa))

	testMsg := ([]byte)("test message")
	blk = f.SendTXToAllAndWaitForBlock(
		f.NewTx().CallFrom(f.CommonAddress(), "sendBTPMessage", map[string]string{
			"networkId": "0x1",
			"message":   fmt.Sprintf("0x%x", testMsg),
		}),
	)
	_, err = blk.NormalTransactions().Get(0)
	assert.NoError(err)

	f.WaitForNextBlock()

	// set wrong pub key
	wrongWP := test.NewWalletProvider()
	f.SendTXToAllAndWaitForResultBlock(
		f.NewTx().CallFrom(f.Nodes[0].CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   dsa,
			"pubKey": fmt.Sprintf("0x%x", wrongWP.WalletFor(dsa).PublicKey()),
		}),
	)

	// send message
	blk = f.SendTXToAllAndWaitForResultBlock(
		f.NewTx().CallFrom(f.CommonAddress(), "sendBTPMessage", map[string]string{
			"networkId": "0x1",
			"message":   fmt.Sprintf("0x%x", testMsg),
		}),
	)
	assert.NoError(err)
	votes, err := f.CS.GetVotesByHeight(blk.Height())
	assert.NoError(err)
	assert.EqualValues(3, len(votes.(*consensus.CommitVoteList).Items))
}

func TestConsensus_RevokeValidator(t_ *testing.T) {
	ntm.InitIconModule()
	const uid2 = "icon"
	tst := newBTPTest(t_)
	defer tst.Close()
	f := tst.Fixture
	assert := tst.Assertions

	blk := f.WaitForBlock(2)
	bd, err := blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	blk = f.SendTXToAllAndWaitForBlock(
		f.NewTx().Call("revokeValidator", map[string]string{
			"address": f.Nodes[0].CommonAddress().String(),
		}),
	)
	assert.EqualValues(4, blk.NextValidators().Len())

	tx := f.NewTx().Call("openBTPNetwork", map[string]string{
		"networkTypeName": uid2,
		"name":            fmt.Sprintf("%s-test", uid2),
		"owner":           f.CommonAddress().String(),
	})
	_ = f.SendTXToAllAndWaitForBlock(tx)
	// prepare the case when the above tx is included in blk.Height() + 2
	blk, err = f.BM.GetBlockByHeight(blk.Height() + 1)
	assert.NoError(err)
	assert.EqualValues(3, blk.NextValidators().Len())
	bs, err := blk.BTPSection()
	assert.NoError(err)
	nts, err := bs.NetworkTypeSectionFor(1)
	assert.NoError(err)
	_ = nts.NextProofContext()
	bysl := nts.NextProofContext().Bytes()
	log.Infof("%s", test.DumpRLP("  ", bysl))

	blk = f.WaitForNextBlock()
	assert.EqualValues(3, blk.NextValidators().Len())
	bs, err = blk.BTPSection()
	assert.NoError(err)
	nts, err = bs.NetworkTypeSectionFor(2)
	assert.NoError(err)
	_ = nts.NextProofContext()
	bysl = nts.NextProofContext().Bytes()
	log.Infof("%s", test.DumpRLP("  ", bysl))

	f.SendTXToAllAndWaitForResultBlock(f.NewTx())

	testMsg := ([]byte)("test message")
	f.SendTXToAllAndWaitForResultBlock(
		f.NewTx().CallFrom(f.CommonAddress(), "sendBTPMessage", map[string]string{
			"networkId": "0x1",
			"message":   fmt.Sprintf("0x%x", testMsg),
		}),
	)
}

func TestConsensus_OpenCloseRevokeValidatorOpen(t_ *testing.T) {
	const uid = "eth"
	tst := newBTPTest(t_)
	defer tst.Close()
	f := tst.Fixture
	assert := tst.Assertions

	blk := f.WaitForBlock(2)
	bd, err := blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	f.SendTXToAllAndWaitForResultBlock(
		f.NewTx().Call("closeBTPNetwork", map[string]string{
			"id": "0x1",
		}),
	)

	blk = f.SendTXToAllAndWaitForBlock(
		f.NewTx().Call("revokeValidator", map[string]string{
			"address": f.Nodes[0].CommonAddress().String(),
		}),
	)
	assert.EqualValues(4, blk.NextValidators().Len())
	revokeHeight := blk.Height()

	_ = f.SendTXToAllAndWaitForBlock(
		f.NewTx().Call("openBTPNetwork", map[string]string{
			"networkTypeName": uid,
			"name":            fmt.Sprintf("%s-test", uid),
			"owner":           f.CommonAddress().String(),
		}),
	)
	blk, err = f.BM.GetBlockByHeight(revokeHeight + 1)
	assert.NoError(err)
	assert.EqualValues(3, blk.NextValidators().Len())

	blk = f.WaitForNextBlock()
	bs, err := blk.BTPSection()
	assert.NoError(err)
	nts, err := bs.NetworkTypeSectionFor(1)
	assert.NoError(err)
	bysl := nts.NextProofContext().Bytes()
	log.Infof("NextProofContext=%s", test.DumpRLP("  ", bysl))

	f.SendTXToAllAndWaitForResultBlock(f.NewTx())
}

func TestConsensus_GetBTPBlockHeaderAndProof_NotCached(t_ *testing.T) {
	tst := newBTPTest(t_)
	defer tst.Close()
	f := tst.Fixture
	assert := tst.Assertions
	_ = f.WaitForBlock(2)

	// increase height so that B(2) is not in block cache
	for i := 0; i < 10; i++ {
		f.SendTXToAllAndWaitForResultBlock(f.NewTx())
	}

	blk, err := f.BM.GetBlockByHeight(2)
	assert.NoError(err)
	bb, _, err := f.CS.GetBTPBlockHeaderAndProof(blk, 1, module.FlagBTPBlockHeader)
	assert.NoError(err)
	assert.EqualValues(2, bb.MainHeight())
}

func TestConsensus_OpenSetNilKey(t_ *testing.T) {
	const dsa = "ecdsa/secp256k1"
	tst := newBTPTest(t_)
	defer tst.Close()
	f := tst.Fixture
	assert := tst.Assertions

	blk := f.WaitForBlock(2)
	bd, err := blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	blk = f.SendTXToAllAndWaitForBlock(
		f.NewTx().CallFrom(f.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   dsa,
			"pubKey": "0x",
		}),
	)
	assert.EqualValues(4, blk.NextValidators().Len())

	blk = f.WaitForNextBlock()
	assert.EqualValues(4, blk.NextValidators().Len())
	bs, err := blk.BTPSection()
	assert.NoError(err)
	nts, err := bs.NetworkTypeSectionFor(1)
	assert.NoError(err)
	_ = nts.NextProofContext()
	bysl := nts.NextProofContext().Bytes()
	log.Infof("%s", test.DumpRLP("  ", bysl))
}

func TestConsensus_Restart(t *testing.T) {
	tst := newBTPTest(t)
	defer tst.Close()
	f := tst.Fixture
	assert := tst.Assertions

	blk := f.WaitForBlock(2)
	bd, err := blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))

	testMsg := ([]byte)("test message")
	blk = f.SendTXToAllAndWaitForResultBlock(
		f.NewTx().CallFrom(f.CommonAddress(), "sendBTPMessage", map[string]string{
			"networkId": "0x1",
			"message":   fmt.Sprintf("0x%x", testMsg),
		}),
	)
	bs, err := blk.BTPSection()
	assert.EqualValues(4, blk.Height())
	assert.NoError(err)
	assert.EqualValues(1, len(bs.NetworkTypeSections()))

	bd, err = blk.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, len(bd.NetworkTypeDigests()))
	f.Close()
	oldF := f
	tst.Fixture = nil

	f2 := test.NewFixture(t, test.UseWallet(oldF.Chain.Wallet()), test.UseDB(oldF.Chain.Database()), test.UseGenesis(string(oldF.Chain.Genesis())))
	defer f2.Close()
	err = f2.CS.Start()
	assert.NoError(err)
}

func TestConsensus_Sync(t *testing.T) {
	assert := assert.New(t)
	f := test.NewFixture(t, test.AddValidatorNodes(4))
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	tx := test.NewTx().Call("setRevision", map[string]string{
		"code": fmt.Sprintf("0x%x", basic.MaxRevision),
	}).Call("setMinimizeBlockGen", map[string]string{
		"yn": "0x1",
	})
	f.SendTransactionToProposer(tx)

	validators := f.Nodes[:4]
	test.NodeInterconnect(validators)
	for _, v := range validators {
		err := v.CS.Start()
		assert.NoError(err)
	}

	blk := test.NodeWaitForBlock(validators, 2)
	assert.EqualValues(2, blk.Height())

	// just increase block heights
	for h := int64(4); h <= 10; h += 2 {
		f.SendTransactionToAll(validators[0].NewTx())
		_ = test.NodeWaitForBlock(validators, h)
	}

	nd := f.Nodes[4]
	blk = nd.GetLastBlock()
	assert.EqualValues(0, blk.Height())

	for _, n := range validators {
		nd.NM.Connect(n.NM)
	}
	err := nd.CS.Start()
	assert.NoError(err)
	blk = nd.WaitForBlock(10)
	assert.EqualValues(10, blk.Height())
}

func newSignedNilVote(w module.Wallet, vt consensus.VoteType, h int64, r int32, nid []byte, ts int64) *consensus.VoteMessage {
	return consensus.NewVoteMessage(
		w, vt, h, r, nid, nil, ts,
		nil, nil, 0,
	)
}

func testWALMessageList(t *testing.T, walID string) {
	assert := assert.New(t)
	f := test.NewFixture(t, test.AddDefaultNode(false), test.AddValidatorNodes(4))
	defer f.Close()

	v1 := newSignedNilVote(f.Nodes[0].Chain.Wallet(), consensus.VoteTypePrevote, 1, 3, codec.MustMarshalToBytes(f.Chain.NID()), 10)
	v2 := newSignedNilVote(f.Nodes[1].Chain.Wallet(), consensus.VoteTypePrevote, 1, 3, codec.MustMarshalToBytes(f.Chain.NID()), 10)
	v3 := newSignedNilVote(f.Nodes[2].Chain.Wallet(), consensus.VoteTypePrevote, 1, 3, codec.MustMarshalToBytes(f.Chain.NID()), 10)

	vl := consensus.NewVoteList()
	vl.AddVote(v1)
	vl.AddVote(v2)
	vl.AddVote(v3)
	vlm := &consensus.VoteListMessage{
		VoteList: vl,
	}

	wal := consensus.NewTestWAL()
	ww, err := wal.OpenForWrite(walID, &consensus.WALConfig{})
	assert.NoError(err)
	mww := consensus.WalMessageWriter{WALWriter: ww}
	err = mww.WriteMessage(vlm)
	assert.NoError(err)
	err = mww.Close()
	assert.NoError(err)

	nd := f.AddNode(test.UseGenesis(string(f.Chain.Genesis())), test.UseWAL(wal))

	err = nd.CS.Start()
	assert.NoError(err)
	status := nd.CS.GetStatus()
	assert.EqualValues(3, status.Round)
}

func TestConsensus_WALMessageList(t *testing.T) {
	testWALMessageList(t, "round")
	testWALMessageList(t, "lock")
	testWALMessageList(t, "commit")
}

func TestConsensus_RoundWALMyVote(t *testing.T) {
	assert := assert.New(t)
	f := test.NewFixture(t, test.AddDefaultNode(false), test.AddValidatorNodes(4))
	defer f.Close()

	v1 := newSignedNilVote(f.Nodes[0].Chain.Wallet(), consensus.VoteTypePrevote, 1, 3, codec.MustMarshalToBytes(f.Chain.NID()), 10)
	wal := consensus.NewTestWAL()
	ww, err := wal.OpenForWrite("round", &consensus.WALConfig{})
	assert.NoError(err)
	mww := consensus.WalMessageWriter{WALWriter: ww}
	err = mww.WriteMessage(v1)
	assert.NoError(err)
	err = mww.Close()
	assert.NoError(err)

	nd := f.AddNode(test.UseGenesis(string(f.Chain.Genesis())), test.UseWAL(wal), test.UseWallet(f.Nodes[0].Chain.Wallet()))

	err = nd.CS.Start()
	assert.NoError(err)
	status := nd.CS.GetStatus()
	assert.EqualValues(3, status.Round)
}

type ConsensusInternal interface {
	module.Consensus
	OnReceive(sp module.ProtocolInfo, bs []byte, id module.PeerID) (bool, error)
	ReceiveBlockResult(br fastsync.BlockResult)
}

type peerID []byte

func (p peerID) Bytes() []byte {
	return p
}

func (p peerID) Equal(id module.PeerID) bool {
	return bytes.Equal(p.Bytes(), id.Bytes())
}

func (p peerID) String() string {
	return hex.EncodeToString(p)
}

type blockResult struct {
	blk      module.BlockData
	votes    []byte
	consume  func()
	reject   func()
	consumed bool
}

func (br *blockResult) Block() module.BlockData {
	return br.blk
}

func (br *blockResult) Votes() []byte {
	return br.votes
}

func (br *blockResult) Consume() {
	if br.consume != nil {
		br.consume()
	}
	br.consumed = true
}

func (br *blockResult) Reject() {
	if br.reject != nil {
		br.reject()
	}
}

type importResult struct {
	blk module.BlockCandidate
	err error
}

type blockManager struct {
	module.BlockManager
	BeforeImport func(module.BlockCandidate, error)
	AfterImport  func(module.BlockCandidate, error)
}

func wrapBlockManager(bm module.BlockManager) *blockManager {
	return &blockManager{
		BlockManager: bm,
	}
}

func useWrappedBM(t *testing.T) test.FixtureOption {
	return test.UseBMFactory(func(ctx *test.NodeContext) module.BlockManager {
		defCf := test.NewFixtureConfig(t)
		bm := defCf.NewBM(ctx)
		return wrapBlockManager(bm)
	})
}

func (bm *blockManager) ImportBlock(blk module.BlockData, flags int, cb func(module.BlockCandidate, error)) (canceler module.Canceler, err error) {
	return bm.BlockManager.ImportBlock(blk, flags, func(bc module.BlockCandidate, err error) {
		if bm.BeforeImport != nil {
			bm.BeforeImport(bc, err)
		}
		cb(bc, err)
		if bm.AfterImport != nil {
			bm.AfterImport(bc, err)
		}
	})
}

func (bm *blockManager) InstallImportWaiter(c int) <-chan importResult {
	ch := make(chan importResult, c)
	bm.AfterImport = func(bc module.BlockCandidate, err error) {
		ch <- importResult{bc, err}
	}
	return ch
}

func (bm *blockManager) InstallImportBlocker(c int) chan<- struct{} {
	ch := make(chan struct{}, c)
	bm.BeforeImport = func(bc module.BlockCandidate, err error) {
		<-ch
	}
	return ch
}

func TestConsensus_BlockCandidateDisposal(t *testing.T) {
	// ImportBlock -> enterPrecommit -> ReceiveBlockResult -> ReceiveBlockResult

	assert := assert.New(t)
	f := test.NewFixture(
		t, test.AddDefaultNode(false), test.AddValidatorNodes(4), useWrappedBM(t),
	)
	defer f.Close()
	bm, ok := f.BM.(*blockManager)
	assert.True(ok)
	ch := bm.InstallImportWaiter(1)

	f.Nodes[1].ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	blk, err := f.Nodes[1].BM.GetBlockByHeight(1)
	assert.NoError(err)

	msgBS, bpmBS, bps := f.Nodes[1].ProposalBytesFor(blk, 0)

	peer := peerID(make([]byte, 4))
	cs, ok := f.CS.(ConsensusInternal)
	assert.True(ok)
	assert.NoError(cs.Start())

	// runs ImportBlock
	_, _ = cs.OnReceive(consensus.ProtoProposal, msgBS, peer)
	_, _ = cs.OnReceive(consensus.ProtoBlockPart, bpmBS, peer)
	<-ch

	// enterPrecommit
	pv1 := f.Nodes[1].VoteFor(consensus.VoteTypePrevote, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pv1), peer)
	pv2 := f.Nodes[2].VoteFor(consensus.VoteTypePrevote, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pv2), peer)
	pv3 := f.Nodes[3].VoteFor(consensus.VoteTypePrevote, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pv3), peer)

	vl := consensus.NewCommitVoteList(
		nil,
		f.Nodes[1].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0),
		f.Nodes[2].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0),
		f.Nodes[3].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0),
	)
	br := blockResult{
		blk:   blk,
		votes: vl.Bytes(),
		reject: func() {
			assert.Fail("shall not reject")
		},
	}

	// finalize, move to next height
	cs.ReceiveBlockResult(&br)
	assert.True(br.consumed)

	// wait commit wait time
	for cs.GetStatus().Height < 2 {
		time.Sleep(200 * time.Millisecond)
	}

	buf := bytes.NewBuffer(nil)
	assert.NoError(blk.Marshal(buf))
	f.Nodes[2].ImportFinalizeBlockByReader(buf)
	f.Nodes[2].ProposeFinalizeBlock(vl)
	blk, err = f.Nodes[2].BM.GetBlockByHeight(2)
	assert.NoError(err)
	_, _, bps = f.Nodes[2].ProposalBytesFor(blk, 0)

	vl = consensus.NewCommitVoteList(
		nil,
		f.Nodes[1].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0),
		f.Nodes[2].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0),
		f.Nodes[3].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0),
	)
	br = blockResult{
		blk:   blk,
		votes: vl.Bytes(),
		reject: func() {
			assert.Fail("shall not reject")
		},
	}

	// finalize another block
	cs.ReceiveBlockResult(&br)
	assert.True(br.consumed)
	<-ch
	blk, err = bm.GetLastBlock()
	assert.NoError(err)
	assert.EqualValues(2, blk.Height())
}

func TestConsensus_BlockCandidateDisposal2(t *testing.T) {
	// ImportBlock -> ReceiveBlockResult

	assert := assert.New(t)
	f := test.NewFixture(
		t, test.AddDefaultNode(false), test.AddValidatorNodes(4), useWrappedBM(t),
	)
	defer f.Close()
	bm, ok := f.BM.(*blockManager)
	assert.True(ok)
	ch := bm.InstallImportWaiter(1)

	f.Nodes[1].ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	blk, err := f.Nodes[1].BM.GetBlockByHeight(1)
	assert.NoError(err)

	msgBS, bpmBS, bps := f.Nodes[1].ProposalBytesFor(blk, 0)

	peer := peerID(make([]byte, 4))
	cs, ok := f.CS.(ConsensusInternal)
	assert.True(ok)
	assert.NoError(cs.Start())

	// runs ImportBlock
	_, _ = cs.OnReceive(consensus.ProtoProposal, msgBS, peer)
	_, _ = cs.OnReceive(consensus.ProtoBlockPart, bpmBS, peer)
	<-ch

	vl := consensus.NewCommitVoteList(
		nil,
		f.Nodes[1].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0),
		f.Nodes[2].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0),
		f.Nodes[3].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0),
	)
	br := blockResult{
		blk:   blk,
		votes: vl.Bytes(),
		reject: func() {
			assert.Fail("shall not reject")
		},
	}

	// finalize, move to next height
	cs.ReceiveBlockResult(&br)
	assert.True(br.consumed)
	blk, err = bm.GetLastBlock()
	assert.NoError(err)
	assert.EqualValues(1, blk.Height())
}

func TestConsensus_BlockCreationFail(t *testing.T) {
	// proposal(byz) -> block part (byz) -> 3 nil precommits
	// -> new proposal -> block part -> 3 precommits -> commit

	assert := assert.New(t)
	f := test.NewFixture(
		t, test.AddDefaultNode(false), test.AddValidatorNodes(4), useWrappedBM(t),
	)
	defer f.Close()
	bm, ok := f.BM.(*blockManager)
	assert.True(ok)
	ch := bm.InstallImportWaiter(1)

	blk := f.Nodes[1].ProposeBlock(consensus.NewEmptyCommitVoteList())
	byzPMBytes, byzBPMBytes, _ := f.Nodes[1].InvalidProposalBytesFor(blk)

	peer := peerID(make([]byte, 4))
	cs, ok := f.CS.(ConsensusInternal)
	assert.True(ok)
	assert.NoError(cs.Start())

	// give invalid block part set
	_, _ = cs.OnReceive(consensus.ProtoProposal, byzPMBytes, peer)
	_, _ = cs.OnReceive(consensus.ProtoBlockPart, byzBPMBytes, peer)

	// nil precommit, move next round
	pc1 := f.Nodes[1].NilVoteFor(consensus.VoteTypePrecommit, blk, 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc1), peer)
	pc2 := f.Nodes[2].NilVoteFor(consensus.VoteTypePrecommit, blk, 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc2), peer)
	pc3 := f.Nodes[3].NilVoteFor(consensus.VoteTypePrecommit, blk, 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc3), peer)

	blk = f.Nodes[2].ProposeBlock(consensus.NewEmptyCommitVoteList())
	msgBS, bpmBS, bps := f.Nodes[2].ProposalBytesFor(blk, 1)

	_, _ = cs.OnReceive(consensus.ProtoProposal, msgBS, peer)
	_, _ = cs.OnReceive(consensus.ProtoBlockPart, bpmBS, peer)

	pc1 = f.Nodes[1].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 1)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc1), peer)
	pc2 = f.Nodes[2].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 1)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc2), peer)
	pc3 = f.Nodes[3].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 1)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc3), peer)
	<-ch

	for cs.GetStatus().Height < 2 {
		time.Sleep(200 * time.Millisecond)
	}
}

func TestConsensus_BlockCreationFail2(t *testing.T) {
	// proposal(byz) -> block part (byz) -> 3 non-nil precommits

	assert := assert.New(t)
	f := test.NewFixture(
		t, test.AddDefaultNode(false), test.AddValidatorNodes(4),
	)
	defer f.Close()

	blk := f.Nodes[1].ProposeBlock(consensus.NewEmptyCommitVoteList())
	byzPMBytes, byzBPMBytes, bps := f.Nodes[1].InvalidProposalBytesFor(blk)

	peer := peerID(make([]byte, 4))
	cs, ok := f.CS.(ConsensusInternal)
	assert.True(ok)
	assert.NoError(cs.Start())

	// give invalid block part set
	_, _ = cs.OnReceive(consensus.ProtoProposal, byzPMBytes, peer)
	_, _ = cs.OnReceive(consensus.ProtoBlockPart, byzBPMBytes, peer)

	// non-nil precommit
	pc1 := f.Nodes[1].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc1), peer)
	pc2 := f.Nodes[2].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc2), peer)
	pc3 := f.Nodes[3].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	assert.Panics(func() {
		_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc3), peer)
	})
}

func TestConsensus_RejectInvalidBlockResult(t *testing.T) {
	// precommit -> ReceiveBlockResult(byz)

	assert := assert.New(t)
	f := test.NewFixture(
		t, test.AddDefaultNode(false), test.AddValidatorNodes(4),
	)
	defer f.Close()

	byzBlk := f.Nodes[1].ProposeBlock(consensus.NewEmptyCommitVoteList())
	f.Nodes[1].ProposeFinalizeBlockWithTX(consensus.NewEmptyCommitVoteList(), f.NewTx().String())
	blk, err := f.Nodes[1].BM.GetBlockByHeight(1)
	assert.NoError(err)
	assert.NotEqual(byzBlk.ID(), blk.ID())

	_, _, bps := f.Nodes[1].ProposalBytesFor(blk, 0)
	cs, ok := f.CS.(ConsensusInternal)
	assert.True(ok)
	assert.NoError(cs.Start())

	// make precommit
	peer := peerID(make([]byte, 4))
	pc1 := f.Nodes[1].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc1), peer)
	pc2 := f.Nodes[2].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc2), peer)
	pc3 := f.Nodes[3].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc3), peer)

	var rejected bool
	br := blockResult{
		blk:   byzBlk,
		votes: nil,
		consume: func() {
			assert.Fail("shall not consume")
		},
		reject: func() {
			rejected = true
		},
	}
	cs.ReceiveBlockResult(&br)
	assert.False(br.consumed)
	assert.True(rejected)
}

func TestConsensus_InvalidProposal(t *testing.T) {
	// proposal(b') -> start import(b') -> 3 prevotes(b) -> import done
	// -> block part(b)

	assert := assert.New(t)
	f := test.NewFixture(
		t, test.AddDefaultNode(false), test.AddValidatorNodes(4), useWrappedBM(t),
	)
	defer f.Close()
	bm, ok := f.BM.(*blockManager)
	assert.True(ok)
	bCh := bm.InstallImportBlocker(1)
	wCh := bm.InstallImportWaiter(1)

	byzBlk := f.Nodes[1].ProposeBlock(consensus.NewEmptyCommitVoteList())
	f.Nodes[1].ProposeFinalizeBlockWithTX(consensus.NewEmptyCommitVoteList(), f.NewTx().String())
	byzPMBytes, byzBPMBytes, _ := f.Nodes[1].ProposalBytesFor(byzBlk, 0)

	peer := peerID(make([]byte, 4))
	cs, ok := f.CS.(ConsensusInternal)
	assert.True(ok)
	assert.NoError(cs.Start())

	// proposal(B'), start import
	_, _ = cs.OnReceive(consensus.ProtoProposal, byzPMBytes, peer)
	_, _ = cs.OnReceive(consensus.ProtoBlockPart, byzBPMBytes, peer)

	// 3 prevotes(B)
	blk, err := f.Nodes[1].BM.GetLastBlock()
	assert.NoError(err)
	assert.NotEqual(byzBlk.ID(), blk.ID())
	_, bpmBS, bps := f.Nodes[1].ProposalBytesFor(blk, 0)
	pv1 := f.Nodes[1].VoteFor(consensus.VoteTypePrevote, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pv1), peer)
	pv2 := f.Nodes[2].VoteFor(consensus.VoteTypePrevote, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pv2), peer)
	pv3 := f.Nodes[3].VoteFor(consensus.VoteTypePrevote, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pv3), peer)

	// import cb
	bCh <- struct{}{}
	<-wCh

	// 3 precommit(B)
	pc1 := f.Nodes[1].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc1), peer)
	pc2 := f.Nodes[2].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc2), peer)
	pc3 := f.Nodes[3].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc3), peer)

	// block part(B)
	_, _ = cs.OnReceive(consensus.ProtoBlockPart, bpmBS, peer)
	bCh <- struct{}{}
	for cs.GetStatus().Height < 2 {
		time.Sleep(200 * time.Millisecond)
	}
	cBlk, err := bm.GetLastBlock()
	assert.NoError(err)
	assert.Equal(blk.ID(), cBlk.ID())
}

func TestConsensus_BPMBuffer(t *testing.T) {
	assert := assert.New(t)
	f := test.NewFixture(
		t, test.AddDefaultNode(false), test.AddValidatorNodes(4),
	)
	defer f.Close()

	blk := f.Nodes[1].ProposeBlock(consensus.NewEmptyCommitVoteList())
	pmBytes, bpmBytes, bps := f.Nodes[1].ProposalBytesFor(blk, 0)

	peer := peerID(make([]byte, 4))
	cs, ok := f.CS.(ConsensusInternal)
	assert.True(ok)
	assert.NoError(cs.Start())

	_, _ = cs.OnReceive(consensus.ProtoBlockPart, bpmBytes, peer)
	_, _ = cs.OnReceive(consensus.ProtoProposal, pmBytes, peer)

	// non-nil precommit
	pc1 := f.Nodes[1].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc1), peer)
	pc2 := f.Nodes[2].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc2), peer)
	pc3 := f.Nodes[3].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc3), peer)
	for cs.GetStatus().Height < 2 {
		time.Sleep(200 * time.Millisecond)
	}
}

func TestConsensus_BPMBuffer2(t *testing.T) {
	assert := assert.New(t)
	f := test.NewFixture(
		t, test.AddDefaultNode(false), test.AddValidatorNodes(4),
	)
	defer f.Close()

	blk := f.Nodes[1].ProposeBlock(consensus.NewEmptyCommitVoteList())
	_, bpmBytes, bps := f.Nodes[1].ProposalBytesFor(blk, 0)

	peer := peerID(make([]byte, 4))
	cs, ok := f.CS.(ConsensusInternal)
	assert.True(ok)
	assert.NoError(cs.Start())

	_, _ = cs.OnReceive(consensus.ProtoBlockPart, bpmBytes, peer)

	// non-nil precommit
	pc1 := f.Nodes[1].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc1), peer)
	pc2 := f.Nodes[2].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc2), peer)
	pc3 := f.Nodes[3].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc3), peer)
	for cs.GetStatus().Height < 2 {
		time.Sleep(200 * time.Millisecond)
	}
}

type serviceManager struct {
	module.ServiceManager
	OnSendDoubleSignReport func(result []byte, vh []byte, data []module.DoubleSignData) error
}

func wrapServiceManager(sm module.ServiceManager) *serviceManager {
	return &serviceManager{
		ServiceManager: sm,
	}
}

func (sm *serviceManager) SendDoubleSignReport(result []byte, vh []byte, data []module.DoubleSignData) error {
	return sm.OnSendDoubleSignReport(result, vh, data)
}

func useWrappedSM(t *testing.T) test.FixtureOption {
	return test.UseSMFactory(func(ctx *test.NodeContext) module.ServiceManager {
		defCf := test.NewFixtureConfig(t)
		sm := defCf.NewSM(ctx)
		return wrapServiceManager(sm)
	})
}

func TestConsensus_DSR(t *testing.T) {
	assert := assert.New(t)
	f := test.NewFixture(
		t, test.AddDefaultNode(false), test.AddValidatorNodes(4),
		useWrappedSM(t),
	)
	defer f.Close()

	var reported bool
	f.SM.(*serviceManager).OnSendDoubleSignReport = func(result []byte, vh []byte, data []module.DoubleSignData) error {
		reported = true
		return nil
	}

	blk := f.Nodes[1].ProposeBlock(consensus.NewEmptyCommitVoteList())
	pmBytes, bpmBytes, bps := f.Nodes[1].ProposalBytesFor(blk, 0)

	peer := peerID(make([]byte, 4))
	cs, ok := f.CS.(ConsensusInternal)
	assert.True(ok)
	assert.NoError(cs.Start())

	_, _ = cs.OnReceive(consensus.ProtoProposal, pmBytes, peer)
	_, _ = cs.OnReceive(consensus.ProtoBlockPart, bpmBytes, peer)

	pc1 := f.Nodes[1].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc1), peer)
	pc2 := f.Nodes[2].VoteFor(consensus.VoteTypePrecommit, blk, bps.ID(), 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc2), peer)
	assert.False(reported)

	pc1 = f.Nodes[1].NilVoteFor(consensus.VoteTypePrecommit, blk, 0)
	_, _ = cs.OnReceive(consensus.ProtoVote, codec.MustMarshalToBytes(pc1), peer)
	assert.True(reported)
}
