/*
 * Copyright 2020 ICON Foundation
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

package chain

import (
	"os"
	"path"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/chain/gs"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/consensus/fastsync"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/service"
)

type taskReset struct {
	chain     *singleChain
	result    resultStore
	gsfile    string
	height    int64
	blockHash []byte
	cancelCh  chan struct{}
}

var resetStates = map[State]string{
	Starting: "reset starting",
	Started:  "reset started",
	Stopping: "reset stopping",
	Failed:   "reset failed",
	Finished: "reset finished",
}

func (t *taskReset) String() string {
	return "Reset"
}

func (t *taskReset) DetailOf(s State) string {
	if name, ok := resetStates[s]; ok {
		return name
	} else {
		return s.String()
	}
}

func (t *taskReset) Start() error {
	if t.height < 0 || t.height == 1 {
		return errors.IllegalArgumentError.Errorf("InvalidHeight(height=%d)", t.height)
	}
	if len(t.blockHash) != 0 && len(t.blockHash) != crypto.HashLen {
		return errors.IllegalArgumentError.Errorf("InvalidBlockHash(hash=%#x len=%d)", t.blockHash, len(t.blockHash))
	}
	if t.height == 0 && len(t.blockHash) == crypto.HashLen {
		return errors.IllegalArgumentError.Errorf("BlockHashForZeroHeight")
	}
	if t.height != 0 && len(t.blockHash) == 0 {
		return errors.IllegalArgumentError.Errorf("NoBlockHash(height=%d)", t.height)
	}
	go t.doReset()
	return nil
}

func (t *taskReset) doReset() {
	err := t._reset()
	t.result.SetValue(err)
}

func (t *taskReset) _cleanUp() error {
	c := t.chain
	chainDir := c.cfg.AbsBaseDir()

	c.releaseDatabase()
	defer c.ensureDatabase()

	DBDir := path.Join(chainDir, DefaultDBDir)
	if err := os.RemoveAll(DBDir); err != nil {
		return err
	}
	CacheDir := path.Join(chainDir, DefaultCacheDir)
	if err := os.RemoveAll(CacheDir); err != nil {
		return err
	}

	WALDir := path.Join(chainDir, DefaultWALDir)
	if err := os.RemoveAll(WALDir); err != nil {
		return err
	}

	ContractDir := path.Join(chainDir, DefaultContractDir)
	if err := os.RemoveAll(ContractDir); err != nil {
		return err
	}
	return nil
}

func (t *taskReset) _fetchBlock(fsm fastsync.Manager, h int64, hash []byte) (module.BlockData, module.CommitVoteSet, error) {
	blk, voteBytes, err := fastsync.FetchBlockByHeightAndHash(fsm, h, hash, t.cancelCh)
	if err != nil {
		return nil, nil, err
	}
	votes := t.chain.CommitVoteSetDecoder()(voteBytes)
	return blk, votes, nil
}

func (t *taskReset) _prepareBlocks() (module.BlockData, module.CommitVoteSet, error) {
	c := t.chain
	defer c.releaseManagers()

	chainDir := c.cfg.AbsBaseDir()

	pr := network.PeerRoleFlag(c.cfg.Role)
	c.nm = network.NewManager(c, c.nt, c.cfg.SeedAddr, pr.ToRoles()...)

	ContractDir := path.Join(chainDir, DefaultContractDir)
	var err error
	c.sm, err = service.NewManager(c, c.nm, c.pm, c.plt, ContractDir)
	if err != nil {
		return nil, nil, err
	}
	bdf, err := block.NewBlockDataFactory(c, c.sm, nil)
	if err != nil {
		return nil, nil, err
	}
	fsm, err := fastsync.NewManagerOnlyForClient(c.nm, bdf, c.logger)
	if err != nil {
		return nil, nil, err
	}

	c.sm.Start()

	if err = c.nm.Start(); err != nil {
		return nil, nil, err
	}

	blk, votes, err := t._fetchBlock(fsm, t.height, t.blockHash)
	if err != nil {
		return nil, nil, err
	}
	pBlk, _, err := t._fetchBlock(fsm, t.height-1, blk.PrevID())
	if err != nil {
		return nil, nil, err
	}
	ppBlk, _, err := t._fetchBlock(fsm, t.height-2, pBlk.PrevID())
	if err != nil {
		return nil, nil, err
	}

	if err = block.UnsafeFinalize(c.sm, c, ppBlk, t.cancelCh); err != nil {
		return nil, nil, err
	}
	if err = block.UnsafeFinalize(c.sm, c, pBlk, t.cancelCh); err != nil {
		return nil, nil, err
	}
	if err = block.UnsafeFinalize(c.sm, c, blk, t.cancelCh); err != nil {
		return nil, nil, err
	}

	if err = block.SetLastHeight(c.Database(), nil, t.height); err != nil {
		return nil, nil, err
	}

	return blk, votes, nil
}

func (t *taskReset) _exportGenesis(blk module.Block, votes module.CommitVoteSet, gsfile string) (rerr error) {
	_ = os.RemoveAll(gsfile)
	fd, err := os.OpenFile(gsfile, os.O_CREATE|os.O_WRONLY|os.O_EXCL|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	gsw := gs.NewGenesisStorageWriter(fd)
	defer func() {
		_ = gsw.Close()
		_ = fd.Close()
		if rerr != nil {
			_ = os.Remove(gsfile)
		}
	}()
	if err := t.chain.bm.ExportGenesis(blk, votes, gsw); err != nil {
		return errors.Wrap(err, "fail on exporting genesis storage")
	}
	return nil
}

func (t *taskReset) _makePrunedGenesis(blkData module.BlockData, votes module.CommitVoteSet) (err error) {
	c := t.chain
	if err := c.prepareManagers(); err != nil {
		return err
	}
	defer c.releaseManagers()

	blk, err := t.chain.bm.GetBlockByHeight(blkData.Height())
	if err != nil {
		return err
	}
	if cid, err := c.sm.GetChainID(blk.Result()); err != nil {
		return errors.InvalidStateError.New("No ChainID is recorded (require Revision 8)")
	} else {
		if cid != int64(c.CID()) {
			return errors.InvalidStateError.Errorf("Invalid chain ID real=%d exp=%d", cid, c.CID())
		}
	}

	gsTmp := t.gsfile + ".tmp"
	if err := t._exportGenesis(blk, votes, gsTmp); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(gsTmp)
		}
	}()

	_, err = os.Stat(t.gsfile)
	if err == nil {
		gsbk := t.gsfile + ".bk"
		_ = os.RemoveAll(gsbk)
		if err := os.Rename(t.gsfile, gsbk); err != nil {
			return errors.UnknownError.Wrapf(err, "fail on backup %s to %s",
				t.gsfile, gsbk)
		}
		defer func() {
			if err != nil {
				_ = os.RemoveAll(t.gsfile)
				_ = os.Rename(gsbk, t.gsfile)
			} else {
				_ = os.RemoveAll(gsbk)
			}
		}()
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.UnknownError.Wrapf(err, "cannot stat %s", t.gsfile)
	}
	if err := os.Rename(gsTmp, t.gsfile); err != nil {
		return errors.UnknownError.Errorf("fail to rename %s to %s",
			gsTmp, t.gsfile)
	}

	// replace genesis
	fd, err := os.Open(t.gsfile)
	if err != nil {
		return errors.UnknownError.Wrapf(err, "fail to open file=%s", t.gsfile)
	}
	g, err := gs.NewFromFile(fd)
	if err != nil {
		return errors.UnknownError.Wrapf(err, "fail to parse gs=%s", t.gsfile)
	}

	c.cfg.GenesisStorage = g
	c.cfg.Genesis = g.Genesis()

	return nil
}

func (t *taskReset) _reset() (ret error) {
	err := t._cleanUp()
	if err != nil {
		return err
	}
	if t.height == 0 {
		return nil
	}

	// do clean up again on failure
	defer func() {
		if ret != nil {
			_ = t._cleanUp()
		}
	}()
	blk, votes, err := t._prepareBlocks()
	if err != nil {
		return err
	}
	return t._makePrunedGenesis(blk, votes)
}

func (t *taskReset) Stop() {
	select {
	case t.cancelCh <- struct{}{}:
	default:
	}
}

func (t *taskReset) Wait() error {
	return t.result.Wait()
}

func newTaskReset(chain *singleChain, gsfile string, height int64, blockHash []byte) chainTask {
	return &taskReset{
		chain:     chain,
		gsfile:    gsfile,
		height:    height,
		blockHash: blockHash,
		cancelCh:  make(chan struct{}),
	}
}
