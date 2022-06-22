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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/btp/ntm"
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
		test.NewTx().SetValidatorsNode(t).String(),
	)
	t.ProposeFinalizeBlockWithTX(
		consensus.NewEmptyCommitVoteList(),
		test.NewTx().Call("setRevision", map[string]string{
			"code": fmt.Sprintf("0x%x", basic.MaxRevision),
		}).String(),
	)
	t.ProposeFinalizeBlockWithTX(
		consensus.NewEmptyCommitVoteList(),
		test.NewTx().CallFrom(t.CommonAddress(), "setPublicKey", map[string]string{
			"name":   "eth",
			"pubKey": fmt.Sprintf("0x%x", t.Chain.WalletFor("eth").PublicKey()),
		}).String())
	t.ProposeFinalizeBlockWithTX(
		t.NewVoteListForLastBlock(),
		test.NewTx().Call("openBTPNetwork", map[string]string{
			"networkTypeName": "eth",
			"name":            "eth-test",
			"owner":           t.CommonAddress().String(),
		}).String(),
	)
	bd, err := t.LastBlock.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(0, bd.NTSHashEntryCount())

	t.ProposeFinalizeBlock(t.NewVoteListForLastBlock())
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
		t.NewVoteListForLastBlock(),
		test.NewTx().CallFrom(t.CommonAddress(), "sendBTPMessage", map[string]string{
			"networkId": "0x1",
			"message":   fmt.Sprintf("0x%x", testMsg),
		}).String(),
	)
	bd, err = t.LastBlock.BTPDigest()
	assert.NoError(err)
	assert.EqualValues(0, bd.NTSHashEntryCount())

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

	t.ProposeFinalizeBlock(t.NewVoteListForLastBlock())
}
