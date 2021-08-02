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
	"github.com/icon-project/goloop/common/errors"
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
	mtRoot      []byte
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
	return &wrapper{
		c:           c,
		walDir:      walDir,
		wm:          wm,
		timestamper: timestamper,
		mtRoot:      mtRoot,
		mtCap:       mtCap,
	}, nil
}

func (c *wrapper) Start() error {
	h, err := block.GetLastHeight(c.c.Database())
	if err != nil {
		return err
	}
	bk, err := c.c.Database().GetBucket(icdb.BlockMerkle)
	if err != nil {
		return err
	}
	mt, err := hexary.NewMerkleTree(bk, c.mtRoot, c.mtCap, -1)
	if err != nil {
		return err
	}
	c.bpp = newBPP(mt)
	if h < c.mtCap {
		c.Consensus = newFastSyncer(h+1, c.mtCap-1, c.c, c)
	} else {
		c.Consensus = consensus.New(c.c, c.walDir, c.wm, c.timestamper, c.bpp)
	}
	return c.Consensus.Start()
}

func (c *wrapper) GetVotesByHeight(height int64) (module.CommitVoteSet, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Consensus == nil {
		return nil, errors.WithStack(errors.ErrNotFound)
	}

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

func (c *wrapper) Term() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Consensus != nil {
		c.Consensus.Term()
	}
}
