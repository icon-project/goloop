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
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/btp/ntm"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/platform/basic"
	"github.com/icon-project/goloop/test"
)

func TestBlockManager_Basics(t_ *testing.T) {
	t := test.NewNode(t_)
	defer t.Close()

	t.AssertLastBlock(nil, module.BlockVersion2)

	t.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	t.AssertLastBlock(t.PrevBlock, module.BlockVersion2)
}

func TestBlockManager_BTPDigest(t_ *testing.T) {
	assert := assert.New(t_)
	t := test.NewNode(t_)
	defer t.Close()

	t.ProposeFinalizeBlockWithTX(
		consensus.NewEmptyCommitVoteList(),
		test.NewTx().SetValidatorsNode(
			t,
		).Call("setRevision", map[string]string{
			"code": fmt.Sprintf("0x%x", basic.MaxRevision),
		}).CallFrom(t.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   "eth",
			"pubKey": fmt.Sprintf("0x%x", t.Chain.WalletFor("eth").PublicKey()),
		}).Call("openBTPNetwork", map[string]string{
			"networkTypeName": "eth",
			"name":            "eth-test",
			"owner":           t.CommonAddress().String(),
		}).String(),
	)
	bd, err := t.LastBlock.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(0, bd.NTSHashEntryCount())

	// advance one block for tx result
	t.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	bd, err = t.LastBlock.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, bd.NTSHashEntryCount())
	assert.EqualValues(1, bd.NetworkTypeDigests()[0].NetworkTypeID())
	assert.EqualValues(1, bd.NetworkTypeDigests()[0].NetworkDigests()[0].NetworkID())
	assert.EqualValues([]byte{0x2}, t.LastBlock.NetworkSectionFilter().Bytes())
	ml, err := bd.NetworkTypeDigests()[0].NetworkDigests()[0].MessageList(t.Chain.Database(), ntm.ForUID("eth"))
	assert.NoError(err)
	assert.EqualValues(0, ml.Len())

	testMsg := ([]byte)("test message")
	t.ProposeFinalizeBlockWithTX(
		consensus.NewEmptyCommitVoteList(),
		test.NewTx().CallFrom(t.CommonAddress(), "sendBTPMessage", map[string]string{
			"networkId": "0x1",
			"message":   fmt.Sprintf("0x%x", testMsg),
		}).String(),
	)
	bd, err = t.LastBlock.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(0, bd.NTSHashEntryCount())

	// advance one block for tx result
	t.ProposeFinalizeBlock(t.NewVoteListForLastBlock())
	bd, err = t.LastBlock.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(1, bd.NTSHashEntryCount())
	assert.EqualValues(1, bd.NetworkTypeDigests()[0].NetworkTypeID())
	assert.EqualValues(1, bd.NetworkTypeDigests()[0].NetworkDigests()[0].NetworkID())
	assert.EqualValues([]byte{0x2}, t.LastBlock.NetworkSectionFilter().Bytes())
	ml, err = bd.NetworkTypeDigests()[0].NetworkDigests()[0].MessageList(t.Chain.Database(), ntm.ForUID("eth"))
	assert.NoError(err)
	assert.EqualValues(1, ml.Len())
	msg, err := ml.Get(0)
	assert.NoError(err)
	assert.EqualValues(testMsg, msg.Bytes())

	v := t.NewVoteListForLastBlock()
	assert.EqualValues(1, v.NTSDProofCount())
	t.ProposeFinalizeBlock(v)
}

func getReaderForBlock(t *testing.T, blk module.Block) io.Reader {
	var buf bytes.Buffer
	err := blk.Marshal(&buf)
	assert.NoError(t, err)
	return &buf
}

func TestBlockManager_BTPImport(t_ *testing.T) {
	assert := assert.New(t_)
	f := test.NewFixture(t_, test.AddValidatorNodes(1))
	defer f.Close()

	vNode := f.Nodes[0]
	vNode.ProposeFinalizeBlockWithTX(
		consensus.NewEmptyCommitVoteList(),
		test.NewTx().Call("setRevision", map[string]string{
			"code": fmt.Sprintf("0x%x", basic.MaxRevision),
		}).CallFrom(vNode.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   "eth",
			"pubKey": fmt.Sprintf("0x%x", vNode.Chain.WalletFor("eth").PublicKey()),
		}).Call("openBTPNetwork", map[string]string{
			"networkTypeName": "eth",
			"name":            "eth-test",
			"owner":           vNode.CommonAddress().String(),
		}).String(),
	)
	f.ImportFinalizeBlockByReader(getReaderForBlock(vNode.T, vNode.LastBlock))

	vNode.ProposeFinalizeBlock(vNode.NewVoteListForLastBlock())
	f.ImportFinalizeBlockByReader(getReaderForBlock(vNode.T, vNode.LastBlock))

	// send message
	testMsg := ([]byte)("test message")
	vNode.ProposeFinalizeBlockWithTX(
		vNode.NewVoteListForLastBlock(),
		test.NewTx().CallFrom(vNode.CommonAddress(), "sendBTPMessage", map[string]string{
			"networkId": "0x1",
			"message":   fmt.Sprintf("0x%x", testMsg),
		}).String(),
	)
	f.ImportFinalizeBlockByReader(getReaderForBlock(vNode.T, vNode.LastBlock))

	// generate result block (BTPDigest has 1 NTS)
	vNode.ProposeFinalizeBlock(vNode.NewVoteListForLastBlock())
	f.ImportFinalizeBlockByReader(getReaderForBlock(vNode.T, vNode.LastBlock))
	bd, _ := f.LastBlock.BTPDigest()
	assert.Equal(1, len(bd.NetworkTypeDigests()))

	// generate block that has the vote of the result block.
	// the vote must have 1 NTSDProof
	oriCvl := vNode.NewVoteListForLastBlock()
	cvl := vNode.NewVoteListForLastBlock()
	t_.Logf("original vote = %s", codec.DumpRLP("  ", oriCvl.Bytes()))
	vNode.ProposeFinalizeBlock(vNode.NewVoteListForLastBlock())

	// fail import with modified vote
	h, b, err := block.FormatFromBlock(vNode.LastBlock)
	assert.NoError(err)
	cvl.(*consensus.CommitVoteList).NTSDProves = nil
	t_.Logf("modified vote = %s", codec.DumpRLP("  ", cvl.Bytes()))
	h.VotesHash = cvl.Hash()
	b.Votes = cvl.Bytes()
	_, err, cbErr := test.ImportBlockByReader(f.T, f.BM, block.NewBlockReaderFromFormat(h, b), 0)
	assert.Error(err)
	assert.NoError(cbErr)

	// success import with original vote
	f.ImportFinalizeBlockByReader(getReaderForBlock(vNode.T, vNode.LastBlock))
}

func TestManager_ChangePubKey(t_ *testing.T) {
	assert := assert.New(t_)
	f := test.NewFixture(t_, test.AddDefaultNode(false), test.AddValidatorNodes(4))
	defer f.Close()

	// 1
	tx := test.NewTx().Call("setRevision", map[string]string{
		"code": fmt.Sprintf("0x%x", basic.MaxRevision),
	})
	for i, v := range f.Validators {
		tx.CallFrom(v.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   "eth",
			"pubKey": fmt.Sprintf("0x%x", v.Chain.WalletFor("eth").PublicKey()),
		})
		t_.Logf("register eth key index=%d key=%x", i, v.Chain.WalletFor("eth").PublicKey())
	}
	f.ProposeFinalizeBlockWithTX(
		consensus.NewEmptyCommitVoteList(),
		tx.Call("openBTPNetwork", map[string]string{
			"networkTypeName": "eth",
			"name":            "eth-test",
			"owner":           f.CommonAddress().String(),
		}).String(),
	)

	// 2
	wp := test.NewWalletProvider()
	wp2 := test.NewWalletProvider()
	f.ProposeFinalizeBlockWithTX(
		f.NewCommitVoteListForLastBlock(0, 0),
		test.NewTx().CallFrom(f.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   "eth",
			"pubKey": fmt.Sprintf("0x%x", wp.WalletFor("eth").PublicKey()),
		}).CallFrom(f.Nodes[1].CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   "eth",
			"pubKey": fmt.Sprintf("0x%x", wp2.WalletFor("eth").PublicKey()),
		}).String(),
	)
	t_.Logf("register eth key index=%d key=%x", 0, wp.WalletFor("eth").PublicKey())
	t_.Logf("register eth key index=%d key=%x", 1, wp2.WalletFor("eth").PublicKey())

	// 3
	testMsg := ([]byte)("test message")
	f.ProposeFinalizeBlockWithTX(
		f.NewCommitVoteListForLastBlock(0, 1),
		test.NewTx().CallFrom(f.CommonAddress(), "sendBTPMessage", map[string]string{
			"networkId": "0x1",
			"message":   fmt.Sprintf("0x%x", testMsg),
		}).String(),
	)
	bs, err := f.LastBlock.BTPSection()
	assert.NoError(err)
	nts, err := bs.NetworkTypeSectionFor(1)
	assert.NoError(err)
	ns, err := nts.NetworkSectionFor(1)
	assert.NoError(err)
	assert.True(ns.NextProofContextChanged())

	_, err = nts.NextProofContext().NewProofPart(make([]byte, 32), wp)
	assert.NoError(err)
}
