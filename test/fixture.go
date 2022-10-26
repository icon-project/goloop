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

package test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
)

type Fixture struct {
	// default node
	*Node
	BaseConfig *FixtureConfig

	// all nodes
	Nodes      []*Node
	Validators []*Node
	Height     int64
}

func NewFixture(t *testing.T, o ...FixtureOption) *Fixture {
	cf := NewFixtureConfig(t, o...)
	f := &Fixture{
		BaseConfig: cf,
	}
	var gs string
	if cf.AddValidatorNodes > 0 {
		wallets := make([]module.Wallet, cf.AddValidatorNodes)
		for i := range wallets {
			wallets[i] = wallet.New()
		}
		var validators string
		for i, w := range wallets {
			if i > 0 {
				validators += ", "
			}
			validators += fmt.Sprintf(`"%s"`, w.Address())
		}
		gs = fmt.Sprintf(`{
			"accounts": [
				{
					"name" : "treasury",
					"address" : "hx1000000000000000000000000000000000000000",
					"balance" : "0x0"
				},
				{
					"name" : "god",
					"address" : "hx0000000000000000000000000000000000000000",
					"balance" : "0x0"
				}
			],
			"message": "",
			"nid" : "0x1",
			"chain" : {
				"validatorList" : [ %s ]
			}
		}`, validators)
		for i := range wallets {
			node := f.AddNode(UseGenesis(gs), UseWallet(wallets[i]))
			f.Validators = append(f.Validators, node)
		}
	}
	if *cf.AddDefaultNode {
		node := f.AddNode(UseGenesis(gs))
		f.Node = node
	}
	return f
}

func (f *Fixture) AddNode(o ...FixtureOption) *Node {
	eo := make([]FixtureOption, 0, len(o)+1)
	eo = append(eo, UseConfig(f.BaseConfig))
	eo = append(eo, o...)
	node := NewNode(f.BaseConfig.T, eo...)
	f.Nodes = append(f.Nodes, node)
	if f.Node == nil {
		f.Node = node
	}
	return node
}

func (f *Fixture) AddNodes(n int, o ...FixtureOption) []*Node {
	nodes := make([]*Node, n)
	for i := 0; i < n; i++ {
		nodes[i] = f.AddNode(o...)
	}
	return nodes
}

func (f *Fixture) Close() {
	for _, n := range f.Nodes {
		n.Close()
	}
}

func (f *Fixture) newPrecommitsAndPCM(blk module.BlockData, round int32, ntsVoteCount int) ([]*consensus.VoteMessage, module.BTPProofContextMap) {
	var pcm module.BTPProofContextMap
	if blk.Height() == 0 {
		pcm = nil
	} else {
		prevBlk, err := f.BM.GetBlockByHeight(blk.Height() - 1)
		assert.NoError(f.T, err)
		pcm, err = prevBlk.NextProofContextMap()
		assert.NoError(f.T, err)
	}
	var buf bytes.Buffer
	err := blk.Marshal(&buf)
	assert.NoError(f.T, err)
	pb := consensus.NewPartSetBuffer(consensus.ConfigBlockPartSize)
	_, err = pb.Write(buf.Bytes())
	assert.NoError(f.T, err)
	ps := pb.PartSet()
	bpsID := ps.ID()
	var votes []*consensus.VoteMessage
	for _, v := range f.Validators {
		vote, err := consensus.NewVoteMessageFromBlock(
			v.Chain.Wallet(),
			v.Chain,
			blk,
			round,
			consensus.VoteTypePrecommit,
			bpsID.WithAppData(uint16(ntsVoteCount)),
			blk.Timestamp()+1,
			f.Chain.NID(),
			pcm,
		)
		assert.NoError(f.T, err)
		votes = append(votes, vote)
	}
	return votes, pcm
}

func (f *Fixture) NewPrecommitList(blk module.BlockData, round int32, ntsVoteCount int) []*consensus.VoteMessage {
	votes, _ := f.newPrecommitsAndPCM(blk, round, ntsVoteCount)
	return votes
}

func (f *Fixture) NewCommitVoteListForLastBlock(round int32, ntsVoteCount int) module.CommitVoteSet {
	votes, pcm := f.newPrecommitsAndPCM(f.LastBlock, round, ntsVoteCount)
	return consensus.NewCommitVoteList(pcm, votes...)
}

func (f *Fixture) SendTransactionToAll(tx interface{ String() string }) {
	for _, node := range f.Nodes {
		_, err := node.SM.SendTransaction(nil, 0, tx.String())
		assert.NoError(f.T, err)
	}
}

func (f *Fixture) SendTransactionToProposer(tx interface{ String() string }) {
	blk, err := f.BM.GetLastBlock()
	assert.NoError(f.T, err)
	h := blk.Height() + 1
	r := 0
	idx := int((h + int64(r)) % int64(blk.NextValidators().Len()))
	val, ok := blk.NextValidators().Get(idx)
	assert.True(f.T, ok)
	found := false
	for i, v := range f.Validators {
		if bytes.Equal(val.Address().Bytes(), v.Chain.Wallet().Address().Bytes()) {
			log.Infof("SendTransaction tx=%s val=%s", tx.String(), f.Validators[i].CommonAddress())
			_, err = f.Validators[i].SM.SendTransaction(nil, 0, tx.String())
			assert.NoError(f.T, err)
			found = true
		}
	}
	assert.True(f.T, found)
}

func (f *Fixture) WaitForBlock(h int64) module.Block {
	res := NodeWaitForBlock(f.Nodes, h)
	if f.Height < h {
		f.Height = h
	}
	return res
}

func (f *Fixture) WaitForNextBlock() module.Block {
	return f.WaitForNextNthBlock(1)
}

func (f *Fixture) WaitForNextNthBlock(n int) module.Block {
	return f.WaitForBlock(f.Height + int64(n))
}
