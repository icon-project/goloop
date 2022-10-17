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
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
)

type Node struct {
	*testing.T
	Chain     *Chain
	Base      string
	em        eeproxy.Manager
	NM        *NetworkManager
	SM        module.ServiceManager
	BM        module.BlockManager
	CS        module.Consensus
	PrevBlock module.Block
	LastBlock module.Block
	Platform  base.Platform
}

type NodeContext struct {
	C        *Chain
	Config   *FixtureConfig
	Base     string
	Platform base.Platform
	CM       contract.ContractManager
	EM       eeproxy.Manager
}

func NewNode(t *testing.T, o ...FixtureOption) *Node {
	cf := NewFixtureConfig(t, o...)
	base, err := ioutil.TempDir("", cf.Prefix)
	assert.NoError(t, err)
	dbase := cf.Dbase()
	logger := log.New()
	w := cf.Wallet
	if w == nil {
		w = wallet.New()
	}
	c, err := NewChain(t, w, dbase, logger, cf.CVSD, cf.Genesis)
	assert.NoError(t, err)
	c.Logger().SetLevel(log.TraceLevel)

	// set up sm
	RegisterTransactionFactory()
	const (
		ContractPath = "contract"
		EESocketPath = "ee.sock"
	)
	ctx := &NodeContext{
		C:      c,
		Config: cf,
		Base:   base,
	}
	plt := cf.NewPlatform(ctx)
	ctx.Platform = plt
	cm, err := plt.NewContractManager(c.Database(), path.Join(base, ContractPath), c.Logger())
	assert.NoError(t, err)
	ee, err := eeproxy.AllocEngines(c.Logger(), "python")
	assert.NoError(t, err)
	em, err := eeproxy.NewManager("unix", path.Join(base, EESocketPath), c.Logger(), ee...)
	assert.NoError(t, err)

	go func() {
		_ = em.Loop()
	}()
	err = em.SetInstances(0, 0, 0)
	assert.NoError(t, err)

	ctx.CM = cm
	ctx.EM = em
	c.sm = NewServiceManager(c, plt, cm, em)

	c.bm = cf.NewBM(ctx)
	lastBlk, err := c.bm.GetLastBlock()
	assert.NoError(t, err)

	c.cs = cf.NewCS(ctx)

	return &Node{
		T:         t,
		Chain:     c,
		Base:      base,
		em:        em,
		NM:        c.nm,
		SM:        c.sm,
		BM:        c.bm,
		CS:        c.cs,
		PrevBlock: nil,
		LastBlock: lastBlk,
		Platform:  plt,
	}
}

func (t *Node) Close() {
	err := t.em.Close()
	assert.NoError(t, err)
	err = os.RemoveAll(t.Base)
	assert.NoError(t, err)
	t.CS.Term()
	t.BM.Term()
	time.AfterFunc(time.Second*5, func() {
		t.Chain.Close()
	})
}

func (t *Node) GetLastBlock() module.Block {
	return GetLastBlock(t.T, t.BM)
}

func (t *Node) AssertLastBlock(
	prev module.Block, version int,
) {
	var height int64
	var prevID []byte
	if prev != nil {
		height = prev.Height() + 1
		prevID = prev.ID()
	}
	AssertLastBlock(t.T, t.BM, height, prevID, version)
}

func (t *Node) ProposeBlock(
	votes module.CommitVoteSet,
) module.BlockCandidate {
	blk, err, cbErr := ProposeBlock(t.BM, t.LastBlock.ID(), votes)
	assert.NoError(t, err)
	assert.NoError(t, cbErr)
	assert.Equal(t, t.LastBlock.ID(), blk.PrevID())
	assert.Equal(t, t.LastBlock.Height()+1, blk.Height())
	return blk
}

func (t *Node) ImportBlock(bc module.BlockCandidate, flag int) {
	assert.Equal(t, t.LastBlock.ID(), bc.PrevID())
	assert.Equal(t, t.LastBlock.Height()+1, bc.Height())
	blk, err, cbErr := ImportBlock(t.T, t.BM, bc, flag)
	assert.NoError(t, err)
	assert.NoError(t, cbErr)
	assert.Equal(t, bc.ID(), blk.ID())
}

func (t *Node) ImportBlockByReader(
	r io.Reader, flag int,
) module.BlockCandidate {
	bc, err, cbErr := ImportBlockByReader(t.T, t.BM, r, flag)
	assert.NoError(t, err)
	assert.NoError(t, cbErr)
	assert.Equal(t, t.LastBlock.ID(), bc.PrevID())
	assert.Equal(t, t.LastBlock.Height()+1, bc.Height())
	return bc
}

func (t *Node) FinalizeBlock(bc module.BlockCandidate) {
	prevBlock := t.LastBlock
	FinalizeBlock(t.T, t.BM, bc)
	blk, err := t.BM.GetLastBlock()
	assert.NoError(t, err)
	assert.Equal(t, bc.ID(), blk.ID())
	t.AssertLastBlock(prevBlock, bc.Version())
	t.PrevBlock = t.LastBlock
	t.LastBlock = blk
}

func (t *Node) ProposeFinalizeBlock(votes module.CommitVoteSet) {
	bc := t.ProposeBlock(votes)
	t.FinalizeBlock(bc)
	bc.Dispose()
}

func (t *Node) ProposeImportFinalizeBlock(
	votes module.CommitVoteSet,
) {
	bc := t.ProposeBlock(votes)
	t.ImportBlock(bc, 0)
	t.FinalizeBlock(bc)
	bc.Dispose()
}

func (t *Node) ImportFinalizeBlockByReader(r io.Reader) {
	bc := t.ImportBlockByReader(r, 0)
	t.FinalizeBlock(bc)
	bc.Dispose()
}

func (t *Node) ProposeFinalizeBlockWithTX(
	votes module.CommitVoteSet, txJson string,
) {
	tid, err := t.SM.SendTransaction(nil, 0, txJson)
	assert.NoError(t, err)
	bc := t.ProposeBlock(votes)
	t.FinalizeBlock(bc)
	tx, err := t.LastBlock.NormalTransactions().Get(0)
	assert.NoError(t, err)
	assert.Equal(t, tid, tx.ID())
	bc.Dispose()
}

func (t *Node) ProposeImportFinalizeBlockWithTX(
	votes module.CommitVoteSet, txJson string,
) {
	tid, err := t.SM.SendTransaction(nil, 0, txJson)
	assert.NoError(t, err)
	bc := t.ProposeBlock(votes)
	t.ImportBlock(bc, 0)
	t.FinalizeBlock(bc)
	tx, err := t.LastBlock.NormalTransactions().Get(0)
	assert.NoError(t, err)
	assert.Equal(t, tid, tx.ID())
	bc.Dispose()
}

func (t *Node) NewVoteListForLastBlock() module.CommitVoteSet {
	return consensus.NewCommitVoteList(consensus.NewPrecommitMessage(
		t.Chain.Wallet(),
		t.LastBlock.Height(),
		0,
		t.LastBlock.ID(),
		nil,
		t.LastBlock.Timestamp()+1,
	))
}

func (t *Node) Address() module.Address {
	return t.Chain.Wallet().Address()
}

func NodeInterconnect(nodes []*Node) {
	l := len(nodes)
	for i := 0; i < l; i++ {
		for j := i + 1; j < l; j++ {
			nodes[i].NM.Connect(nodes[j].NM)
		}
	}
}
