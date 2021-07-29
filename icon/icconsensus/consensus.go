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

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/icon/icdb"
	"github.com/icon-project/goloop/icon/merkle/hexary"
	"github.com/icon-project/goloop/module"
)

type wrapper struct {
	mu sync.Mutex
	module.Consensus
	c           module.Chain
	walDir      string
	wm          consensus.WALManager
	timestamper module.Timestamper
	mtCap       int64
	bpp         *bpp
}

func NewConsensus(
	c module.Chain,
	walDir string,
	timestamper module.Timestamper,
	mtRoot []byte,
	mtCap int64,
) (module.Consensus, error) {
	return New(c, walDir, nil, timestamper, mtRoot, mtCap)
}

func New(
	c module.Chain,
	walDir string,
	wm consensus.WALManager,
	timestamper module.Timestamper,
	mtRoot []byte,
	mtCap int64,
) (module.Consensus, error) {
	h, err := block.GetLastHeight(c.Database())
	if err != nil {
		return nil, err
	}
	bk, err := c.Database().GetBucket(icdb.BlockMerkle)
	if err != nil {
		return nil, err
	}
	mt, err := hexary.NewMerkleTree(bk, mtRoot, mtCap, -1)
	if err != nil {
		return nil, err
	}
	cse := &wrapper{
		c:           c,
		walDir:      walDir,
		wm:          wm,
		timestamper: timestamper,
		mtCap:       mtCap,
	}
	cse.bpp = newBPP(mt)
	if h < mtCap {
		cse.Consensus = newFastSyncer(h+1, mtCap-1, c, cse)
	} else {
		cse.Consensus = consensus.New(c, walDir, wm, timestamper, cse.bpp)
	}
	return cse, nil
}

func (c *wrapper) GetVotesByHeight(height int64) (module.CommitVoteSet, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if height < c.mtCap {
		blk, err := c.c.BlockManager().GetBlockByHeight(height + 1)
		if err != nil {
			return nil, err
		}
		return blk.Votes(), nil
	}
	return c.Consensus.GetVotesByHeight(height)
}

func (c *wrapper) upgrade() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Consensus = consensus.New(c.c, c.walDir, c.wm, c.timestamper, c.bpp)
}
