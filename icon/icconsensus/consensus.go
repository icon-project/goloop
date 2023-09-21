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

package icconsensus

import (
	"sync"
	"time"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/icdb"
	"github.com/icon-project/goloop/icon/merkle/hexary"
	"github.com/icon-project/goloop/module"
)

type wrapper struct {
	mu sync.Mutex
	module.Consensus
	c              base.Chain
	walDir         string
	wm             consensus.WALManager
	timestamper    module.Timestamper
	merkleHeader   *hexary.MerkleHeader
	lastVoteData   *consensus.LastVoteData
	timeoutPropose time.Duration
}

func New(
	c base.Chain,
	walDir string,
	wm consensus.WALManager,
	timestamper module.Timestamper,
	merkleHeader *hexary.MerkleHeader,
	lastVotes *blockv0.BlockVoteList,
	timeoutPropose time.Duration,
) (module.Consensus, error) {
	return &wrapper{
		c:            c,
		walDir:       walDir,
		wm:           wm,
		timestamper:  timestamper,
		merkleHeader: merkleHeader,
		lastVoteData: &consensus.LastVoteData{
			Height: merkleHeader.Leaves - 1,
			VotesBytes: lastVotes.Bytes(),
		},
		timeoutPropose: timeoutPropose,
	}, nil
}

func (c *wrapper) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	h, err := block.GetLastHeight(c.c.Database())
	if err != nil {
		return err
	}
	bk, err := c.c.Database().GetBucket(icdb.BlockMerkle)
	if err != nil {
		return err
	}
	mt, err := hexary.NewMerkleTree(bk, c.merkleHeader, -1)
	if err != nil {
		return err
	}
	bpp := newBPP(mt)
	if h+1 < c.merkleHeader.Leaves {
		c.Consensus = newFastSyncer(h+1, c.merkleHeader.Leaves-1, c.c, c, bpp)
	} else {
		c.Consensus = consensus.New(
			c.c, c.walDir, c.wm, c.timestamper, bpp, c.lastVoteData, c.timeoutPropose,
		)
	}
	return c.Consensus.Start()
}

func (c *wrapper) GetStatus() *module.ConsensusStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Consensus == nil {
		return nil
	}

	return c.Consensus.GetStatus()
}

func (c *wrapper) GetVotesByHeight(height int64) (module.CommitVoteSet, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Consensus == nil {
		return nil, errors.WithStack(errors.ErrNotFound)
	}

	if height < c.merkleHeader.Leaves {
		blk, err := c.c.BlockManager().GetBlockByHeight(height + 1)
		if err != nil {
			return nil, err
		}
		return blk.Votes(), nil
	}
	return c.Consensus.GetVotesByHeight(height)
}

func (c *wrapper) Upgrade(bpp *bpp) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Consensus.Term()
	c.Consensus = consensus.New(c.c, c.walDir, c.wm, c.timestamper, bpp, c.lastVoteData, c.timeoutPropose)
	err := c.Consensus.Start()
	if err != nil {
		c.c.Logger().Panicf("fail to start consensus %+v", err)
	}
}

func (c *wrapper) Term() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Consensus != nil {
		c.Consensus.Term()
	}
}
