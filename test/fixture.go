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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/platform/basic"
)

type Fixture struct {
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
}

type FixtureConfig struct {
	T            *testing.T
	MerkleRoot   []byte
	MerkleLeaves int64
	Prefix       string
	Dbase        db.Database
	CVSD         module.CommitVoteSetDecoder
	NewPlatform  func(ctx *FixtureContext) service.Platform
	NewSM        func(ctx *FixtureContext) module.ServiceManager
	NewBM        func(ctx *FixtureContext) module.BlockManager
	NewCS        func(ctx *FixtureContext) module.Consensus
}

type FixtureContext struct {
	C        *Chain
	Config   *FixtureConfig
	Base     string
	Platform service.Platform
	CM       contract.ContractManager
	EM       eeproxy.Manager
}

func NewFixtureConfig(t *testing.T) *FixtureConfig {
	return &FixtureConfig{
		T:      t,
		Prefix: "goloop-block-fixture",
		Dbase:  db.NewMapDB(),
		CVSD:   consensus.NewCommitVoteSetFromBytes,
		NewPlatform: func(ctx *FixtureContext) service.Platform {
			return basic.Platform
		},
		NewSM: func(ctx *FixtureContext) module.ServiceManager {
			return NewServiceManager(ctx.C, ctx.Platform, ctx.CM, ctx.EM)
		},
		NewBM: func(ctx *FixtureContext) module.BlockManager {
			bm, err := block.NewManager(ctx.C, nil, nil)
			assert.NoError(ctx.Config.T, err)
			return bm
		},
		NewCS: func(ctx *FixtureContext) module.Consensus {
			wm := NewWAL()
			wal := path.Join(ctx.Base, "wal")
			cs := consensus.New(ctx.C, wal, wm, nil, nil)
			assert.NotNil(ctx.Config.T, cs)
			return cs
		},
	}
}

type FixtureOption func(cf *FixtureConfig) *FixtureConfig

func (cf *FixtureConfig) Override(cf2 *FixtureConfig) *FixtureConfig {
	res := *cf
	if cf2.T != nil {
		res.T = cf2.T
	}
	if cf2.MerkleRoot != nil {
		res.MerkleRoot = cf2.MerkleRoot
		res.MerkleLeaves = cf2.MerkleLeaves
	}
	if len(cf2.Prefix) != 0 {
		res.Prefix = cf2.Prefix
	}
	if cf2.Dbase != nil {
		res.Dbase = cf2.Dbase
	}
	if cf2.CVSD != nil {
		res.CVSD = cf2.CVSD
	}
	if cf2.NewPlatform != nil {
		res.NewPlatform = cf2.NewPlatform
	}
	if cf2.NewSM != nil {
		res.NewSM = cf2.NewSM
	}
	if cf2.NewBM != nil {
		res.NewBM = cf2.NewBM
	}
	if cf2.NewCS != nil {
		res.NewCS = cf2.NewCS
	}
	return &res
}

func NewFixture(t *testing.T, opt ...FixtureOption) *Fixture {
	cf := NewFixtureConfig(t)
	for _, o := range opt {
		cf = o(cf)
	}
	base, err := ioutil.TempDir("", cf.Prefix)
	assert.NoError(t, err)
	dbase := cf.Dbase
	logger := log.New()
	c, err := NewChain(t, dbase, logger, cf.CVSD)
	assert.NoError(t, err)
	c.Logger().SetLevel(log.TraceLevel)

	// set up sm
	RegisterTransactionFactory()
	const (
		ContractPath = "contract"
		EESocketPath = "ee.sock"
	)
	ctx := &FixtureContext{
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

	return &Fixture{
		T:         t,
		Chain:     c,
		Base:      base,
		em:        em,
		NM:        c.nm.(*NetworkManager),
		SM:        c.sm,
		BM:        c.bm,
		CS:        c.cs,
		PrevBlock: nil,
		LastBlock: lastBlk,
	}
}

func (t *Fixture) Close() {
	err := t.em.Close()
	assert.NoError(t, err)
	err = os.RemoveAll(t.Base)
	assert.NoError(t, err)
	t.CS.Term()
	ResetJobChan()
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

var jobChan chan func()
var lock sync.Mutex

const jobChLen = 1024

func Go(f func()) {
	lock.Lock()
	defer lock.Unlock()

	if jobChan == nil {
		jobChan = make(chan func(), jobChLen)
		jc := jobChan
		go func() {
			for job := range jc {
				job()
			}
		}()
	}
	jobChan <- f
}

func ResetJobChan() {
	lock.Lock()
	defer lock.Unlock()

	if jobChan != nil {
		close(jobChan)
		jobChan = nil
	}
}
