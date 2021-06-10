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
	timestamper module.Timestamper
	mtCap       int64
}

func NewConsensus(
	c module.Chain,
	walDir string,
	timestamper module.Timestamper,
) (module.Consensus, error) {
	h, err := block.GetLastHeight(c.Database())
	if err != nil {
		return nil, err
	}
	bk, err := c.Database().GetBucket(icdb.BlockMerkle)
	if err != nil {
		return nil, err
	}
	mtCap, err := hexary.CapOfMerkleTree(bk, "")
	if err != nil {
		return nil, err
	}
	cse := &wrapper{
		c:           c,
		walDir:      walDir,
		timestamper: timestamper,
		mtCap:       mtCap,
	}
	if h < mtCap {
		cse.Consensus, err = newFastSyncer(h+1, mtCap-1, c, cse)
	} else {
		cse.Consensus = consensus.NewConsensus(c, walDir, timestamper)
	}
	if err != nil {
		return nil, err
	}
	return cse, nil
}

func (c *wrapper) GetVotesByHeight(height int64) (module.CommitVoteSet, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if height < c.mtCap {
		blk, err := c.c.BlockManager().GetBlockByHeight(height+1)
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

	c.Consensus = consensus.NewConsensus(c.c, c.walDir, c.timestamper)
}
