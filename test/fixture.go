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

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/platform/basic"
)

type Fixture struct {
	*testing.T
	Chain     *Chain
	Base      string
	em        eeproxy.Manager
	SM        module.ServiceManager
	BM        module.BlockManager
	PrevBlock module.Block
	LastBlock module.Block
}

type FixtureOption struct {
	Prefix      string
	Dbase       db.Database
	CVSD        module.CommitVoteSetDecoder
	NewPlatform func(plt service.Platform) service.Platform
	NewSM       func(sm *ServiceManager) module.ServiceManager
	NewBM       func(bm module.BlockManager, c *Chain) module.BlockManager
}

func (o *FixtureOption) fillDefault() *FixtureOption {
	var res FixtureOption
	if o != nil {
		res = *o
	}
	if len(res.Prefix) == 0 {
		res.Prefix = "goloop-block-fixture"
	}
	if res.Dbase == nil {
		res.Dbase = db.NewMapDB()
	}
	if res.CVSD == nil {
		res.CVSD = consensus.NewCommitVoteSetFromBytes
	}
	if res.NewPlatform == nil {
		res.NewPlatform = func(plt service.Platform) service.Platform {
			return plt
		}
	}
	if res.NewSM == nil {
		res.NewSM = func(sm *ServiceManager) module.ServiceManager {
			return sm
		}
	}
	if res.NewBM == nil {
		res.NewBM = func(bm module.BlockManager, c *Chain) module.BlockManager {
			return bm
		}
	}
	return &res
}

func NewFixture(t *testing.T, opt *FixtureOption) *Fixture {
	opt = opt.fillDefault()
	base, err := ioutil.TempDir("", opt.Prefix)
	assert.NoError(t, err)
	dbase := opt.Dbase
	logger := log.New()
	c, err := NewChain(dbase, logger, opt.CVSD)
	assert.NoError(t, err)

	// set up sm
	RegisterTransactionFactory()
	const (
		ContractPath = "contract"
		EESocketPath = "ee.sock"
	)
	plt := opt.NewPlatform(basic.Platform)
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

	sm := NewServiceManager(c, plt, cm, em)
	c.sm = opt.NewSM(sm)

	bm, err := block.NewManager(c, nil, nil)
	assert.NoError(t, err)
	c.bm = opt.NewBM(bm, c)
	lastBlk, err := c.bm.GetLastBlock()
	assert.NoError(t, err)

	return &Fixture{
		T:         t,
		Chain:     c,
		Base:      base,
		em:        em,
		SM:        c.sm,
		BM:        c.bm,
		PrevBlock: nil,
		LastBlock: lastBlk,
	}
}

func (t *Fixture) Close() {
	err := t.em.Close()
	assert.NoError(t, err)
	err = os.RemoveAll(t.Base)
	assert.NoError(t, err)
}

func (t *Fixture) GetLastBlock() module.Block {
	return GetLastBlock(t.T, t.BM)
}

func (t *Fixture) AssertLastBlock(
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

func (t *Fixture) ProposeBlock(
	votes module.TimestampedCommitVoteSet,
) module.BlockCandidate {
	blk, err, cbErr := ProposeBlock(t.BM, t.LastBlock.ID(), votes)
	assert.NoError(t, err)
	assert.NoError(t, cbErr)
	assert.Equal(t, t.LastBlock.ID(), blk.PrevID())
	assert.Equal(t, t.LastBlock.Height()+1, blk.Height())
	return blk
}

func (t *Fixture) ImportBlock(bc module.BlockCandidate, flag int) {
	assert.Equal(t, t.LastBlock.ID(), bc.PrevID())
	assert.Equal(t, t.LastBlock.Height()+1, bc.Height())
	blk, err, cbErr := ImportBlock(t.T, t.BM, bc, flag)
	assert.NoError(t, err)
	assert.NoError(t, cbErr)
	assert.Equal(t, bc.ID(), blk.ID())
}

func (t *Fixture) ImportBlockByReader(
	r io.Reader, flag int,
) module.BlockCandidate {
	bc, err, cbErr := ImportBlockByReader(t.T, t.BM, r, flag)
	assert.NoError(t, err)
	assert.NoError(t, cbErr)
	assert.Equal(t, t.LastBlock.ID(), bc.PrevID())
	assert.Equal(t, t.LastBlock.Height()+1, bc.Height())
	return bc
}

func (t *Fixture) FinalizeBlock(bc module.BlockCandidate) {
	prevBlock := t.LastBlock
	FinalizeBlock(t.T, t.BM, bc)
	blk, err := t.BM.GetLastBlock()
	assert.NoError(t, err)
	assert.Equal(t, bc.ID(), blk.ID())
	t.AssertLastBlock(prevBlock, bc.Version())
	t.PrevBlock = t.LastBlock
	t.LastBlock = blk
}

func (t *Fixture) ProposeFinalizeBlock(votes module.TimestampedCommitVoteSet) {
	bc := t.ProposeBlock(votes)
	t.FinalizeBlock(bc)
	bc.Dispose()
}

func (t *Fixture) ProposeImportFinalizeBlock(
	votes module.TimestampedCommitVoteSet,
) {
	bc := t.ProposeBlock(votes)
	t.ImportBlock(bc, 0)
	t.FinalizeBlock(bc)
	bc.Dispose()
}

func (t *Fixture) ImportFinalizeBlockByReader(r io.Reader) {
	bc := t.ImportBlockByReader(r, 0)
	t.FinalizeBlock(bc)
	bc.Dispose()
}

func (t *Fixture) ProposeImportFinalizeBlockWithTX(
	votes module.TimestampedCommitVoteSet, txJson string,
) {
	tid, err := t.SM.SendTransaction(txJson)
	assert.NoError(t, err)
	bc := t.ProposeBlock(votes)
	t.ImportBlock(bc, 0)
	t.FinalizeBlock(bc)
	tx, err := t.LastBlock.NormalTransactions().Get(0)
	assert.NoError(t, err)
	assert.Equal(t, tid, tx.ID())
	bc.Dispose()
}

func (t *Fixture) NewVoteListForLastBlock() module.TimestampedCommitVoteSet {
	return consensus.NewCommitVoteList(consensus.NewPrecommitMessage(
		t.Chain.Wallet(),
		t.LastBlock.Height(),
		0,
		t.LastBlock.ID(),
		nil,
		t.LastBlock.Timestamp()+1,
	))
}
