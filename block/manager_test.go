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
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/platform/basic"
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

	_, err := nd.BM.GetBlock(make([]byte, crypto.HashLen))
	assert.Error(err)

	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	blk := nd.GetLastBlock()
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
	defer nd2.Close()
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

	cvlBytes, err := block.GetCommitVoteListBytesForHeight(db, nil, 1)
	assert.NoError(err)
	assert.EqualValues(consensus.NewEmptyCommitVoteList().Bytes(), cvlBytes)

	h, err := block.GetLastHeight(db)
	assert.NoError(err)
	assert.EqualValues(2, h)
	assert.EqualValues(2, block.GetLastHeightOf(db))

	err = block.ResetDB(db, nil, 1)
	assert.NoError(err)
	assert.EqualValues(1, block.GetLastHeightOf(db))

	res, err := block.GetBlockResultByHeight(db, nil, 1)
	assert.NoError(err)
	assert.EqualValues(1, blk.Height())
	assert.EqualValues(blk.Result(), res)

	bd, err := block.GetBTPDigestFromResult(db, nil, res)
	assert.NoError(err)
	assert.EqualValues([]byte(nil), bd.Bytes())

	vl, err := block.GetNextValidatorsByHeight(db, nil, 1)
	assert.NoError(err)
	assert.EqualValues(0, vl.Len())
	assert.EqualValues([]byte(nil), vl.Bytes())
}

func TestBlockManager_BTPDigest(t_ *testing.T) {
	const dsa = "ecdsa/secp256k1"
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
			"name":   dsa,
			"pubKey": fmt.Sprintf("0x%x", t.Chain.WalletFor(dsa).PublicKey()),
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
	const dsa = "ecdsa/secp256k1"
	assert := assert.New(t_)
	f := test.NewFixture(t_, test.AddValidatorNodes(1))
	defer f.Close()

	vNode := f.Nodes[0]
	vNode.ProposeFinalizeBlockWithTX(
		consensus.NewEmptyCommitVoteList(),
		test.NewTx().Call("setRevision", map[string]string{
			"code": fmt.Sprintf("0x%x", basic.MaxRevision),
		}).CallFrom(vNode.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   dsa,
			"pubKey": fmt.Sprintf("0x%x", vNode.Chain.WalletFor(dsa).PublicKey()),
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
	const dsa = "ecdsa/secp256k1"
	assert := assert.New(t_)
	f := test.NewFixture(t_, test.AddDefaultNode(false), test.AddValidatorNodes(4))
	defer f.Close()

	// 1
	tx := test.NewTx().Call("setRevision", map[string]string{
		"code": fmt.Sprintf("0x%x", basic.MaxRevision),
	})
	for i, v := range f.Validators {
		tx.CallFrom(v.CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   dsa,
			"pubKey": fmt.Sprintf("0x%x", v.Chain.WalletFor(dsa).PublicKey()),
		})
		t_.Logf("register eth key index=%d key=%x", i, v.Chain.WalletFor(dsa).PublicKey())
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
			"name":   dsa,
			"pubKey": fmt.Sprintf("0x%x", wp.WalletFor(dsa).PublicKey()),
		}).CallFrom(f.Nodes[1].CommonAddress(), "setBTPPublicKey", map[string]string{
			"name":   dsa,
			"pubKey": fmt.Sprintf("0x%x", wp2.WalletFor(dsa).PublicKey()),
		}).String(),
	)
	t_.Logf("register eth key index=%d key=%x", 0, wp.WalletFor(dsa).PublicKey())
	t_.Logf("register eth key index=%d key=%x", 1, wp2.WalletFor(dsa).PublicKey())

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

func TestManager_WaitForBlock(t *testing.T) {
	assert := assert.New(t)
	nd := test.NewNode(t)
	defer nd.Close()

	ch0, err := nd.BM.WaitForBlock(0)
	assert.NoError(err)
	blk := <-ch0
	assert.EqualValues(0, blk.Height())

	ch1, err := nd.BM.WaitForBlock(1)
	assert.NoError(err)
	select {
	case <-ch1:
		assert.Fail("Shall not happen")
	default:
	}

	ch2, err := nd.BM.WaitForBlock(2)
	assert.NoError(err)

	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	blk = <-ch1
	assert.EqualValues(1, blk.Height())

	select {
	case <-ch2:
		assert.Fail("Shall not happen")
	default:
	}

	nd.BM.Term()
	blk = <-ch2
	assert.Nil(blk)
}

func TestManager_WaitTransactionResult(t *testing.T) {
	assert := assert.New(t)
	nd := test.NewNode(t)
	defer nd.Close()

	tx := nd.NewTx()
	nd.ProposeFinalizeBlockWithTX(
		consensus.NewEmptyCommitVoteList(), tx.String(),
	)
	ch, err := nd.BM.WaitTransactionResult(tx.ID())
	assert.NoError(err)

	select {
	case <-ch:
		assert.Fail("shall not receive")
	default:
	}

	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	ti := (<-ch).(module.TransactionInfo)
	tx2, err := ti.Transaction()
	assert.NoError(err)
	assert.Equal(tx.ID(), tx2.ID())
}

type genesisBuffer struct {
	gtx  []byte
	data map[string][]byte
}

func newGenesisBuffer() *genesisBuffer {
	return &genesisBuffer{
		data: make(map[string][]byte),
	}
}

func (gb *genesisBuffer) WriteGenesis(gtx []byte) error {
	gb.gtx = gtx
	return nil
}

func (gb *genesisBuffer) WriteData(value []byte) ([]byte, error) {
	hv := crypto.SHA3Sum256(value)
	gb.data[string(hv)] = value
	return hv, nil
}

func (gb *genesisBuffer) Close() error {
	return nil
}

type genesisStorage struct {
	gType  module.GenesisType
	cid    int
	nid    int
	height int64
	gtx    []byte
	data   map[string][]byte
}

func newGenesisStorage(gType module.GenesisType, cid, nid int, height int64, gb *genesisBuffer) *genesisStorage {
	return &genesisStorage{
		gType:  gType,
		cid:    cid,
		nid:    nid,
		height: height,
		gtx:    gb.gtx,
		data:   gb.data,
	}
}

func (gs *genesisStorage) CID() (int, error) {
	return gs.cid, nil
}

func (gs *genesisStorage) NID() (int, error) {
	return gs.nid, nil
}

func (gs *genesisStorage) Height() int64 {
	return gs.height
}

func (gs *genesisStorage) Type() (module.GenesisType, error) {
	return gs.gType, nil
}

func (gs *genesisStorage) Genesis() []byte {
	return gs.gtx
}

func (gs *genesisStorage) Get(key []byte) ([]byte, error) {
	return gs.data[string(key)], nil
}

func TestManager_ExportGenesis(t *testing.T) {
	assert := assert.New(t)

	nd := test.NewNode(t)
	defer nd.Close()

	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	blk := nd.GetLastBlock()
	gb := newGenesisBuffer()
	err := nd.BM.ExportGenesis(blk, consensus.NewEmptyCommitVoteList(), gb)
	assert.NoError(err)
	gs := newGenesisStorage(module.GenesisPruned, nd.Chain.CID(), nd.Chain.NID(), 2, gb)
	dbase := db.NewMapDB()
	err = nd.BM.ExportBlocks(2, 2, dbase, func(h int64) error {
		return nil
	})
	assert.NoError(err)

	nd2 := test.NewNode(t, test.UseGenesisStorage(gs), test.UseDB(dbase))
	defer nd2.Close()
	blk2, _, err := nd2.BM.GetGenesisData()
	assert.NoError(err)
	assert.EqualValues(blk.ID(), blk2.ID())
}

func TestManager_ExportBlocks(t *testing.T) {
	assert := assert.New(t)
	nd := test.NewNode(t)
	defer nd.Close()
	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	dbase := db.NewMapDB()
	ch := make(chan int64, 3)
	err := nd.BM.ExportBlocks(0, 2, dbase, func(h int64) error {
		ch <- h
		return nil
	})
	assert.NoError(err)
	assert.EqualValues(0, <-ch)
	assert.EqualValues(1, <-ch)
	assert.EqualValues(2, <-ch)
	block.ResetDB(dbase, nil, 1)

	nd2 := test.NewNode(t, test.UseDB(dbase))
	defer nd2.Close()
	blk, err := nd.BM.GetBlockByHeight(1)
	assert.NoError(err)
	blk2, err := nd2.BM.GetBlockByHeight(1)
	assert.NoError(err)
	assert.EqualValues(blk.ID(), blk2.ID())
}
