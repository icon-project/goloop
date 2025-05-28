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
	"bytes"
	"fmt"
	"os"
	"path"
	"sync"
	"sync/atomic"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/chain/gs"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/consensus/fastsync"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/state"
)

const (
	TempSuffix = ".tmp"
)

type taskReset struct {
	chain     *singleChain
	result    resultStore
	gsfile    string
	height    int64
	blockHash []byte
	cancelCh  chan struct{}

	reportHeight     int64
	reportResolved   uint64
	reportUnresolved uint64
}

var resetStates = map[State]string{
	Starting: "reset starting",
	Started:  "reset started",
	Stopping: "reset stopping",
	Failed:   "reset failed",
	Finished: "reset finished",
}

func (t *taskReset) String() string {
	if t.height != 0 {
		return fmt.Sprintf("Reset(height=%d,blockHash=%#x)", t.height, t.blockHash)
	}
	return "Reset"
}

func (t *taskReset) DetailOf(s State) string {
	if s == Started {
		height := atomic.LoadInt64(&t.reportHeight)
		resolved := atomic.LoadUint64(&t.reportResolved)
		unresolved := atomic.LoadUint64(&t.reportUnresolved)
		if height != 0 {
			return fmt.Sprintf("reset started height=%d resolved=%d unresolved=%d", height, resolved, unresolved)
		}
	}
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

	TmpDir := path.Join(chainDir, DefaultTmpDBDir)
	if err := os.RemoveAll(TmpDir); err != nil {
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

type progressSum struct {
	lock     sync.Mutex
	byHeight map[int64]*[2]int
	sum      [2]int
	callback module.ProgressCallback
}

func (p *progressSum) onProgress(h int64, r, u int) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.callback == nil {
		return nil
	}
	s := p.byHeight[h]
	if s == nil {
		s = new([2]int)
		p.byHeight[h] = s
	}
	p.sum[0] += r - s[0]
	p.sum[1] += u - s[1]
	s[0] = r
	s[1] = u
	return p.callback(h, p.sum[0], p.sum[1])
}

func newProgressSum(callback module.ProgressCallback) *progressSum {
	return &progressSum{
		byHeight: make(map[int64]*[2]int),
		callback: callback,
	}
}

func (t *taskReset) _prepareBlocks(height int64, blockHash []byte) (module.BlockData, module.CommitVoteSet, error) {
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
	bdf, err := block.NewBlockDataFactory(c, nil)
	if err != nil {
		return nil, nil, err
	}
	fsm, err := fastsync.NewManagerOnlyForClient(c.nm, bdf, c.logger, base.MaxBlockSize(c))
	if err != nil {
		return nil, nil, err
	}

	c.sm.Start()

	if err = c.nm.Start(); err != nil {
		return nil, nil, err
	}

	blk, votes, err := t._fetchBlock(fsm, height, blockHash)
	if err != nil {
		return nil, nil, err
	}
	pBlk, _, err := t._fetchBlock(fsm, height-1, blk.PrevID())
	if err != nil {
		return nil, nil, err
	}
	ppBlk, _, err := t._fetchBlock(fsm, height-2, pBlk.PrevID())
	if err != nil {
		return nil, nil, err
	}

	p := newProgressSum(t._reportProgress)
	if err = block.UnsafeFinalize(c.sm, c, ppBlk, t.cancelCh, p.onProgress); err != nil {
		return nil, nil, err
	}
	if err = block.UnsafeFinalize(c.sm, c, pBlk, t.cancelCh, p.onProgress); err != nil {
		return nil, nil, err
	}
	if err = block.UnsafeFinalize(c.sm, c, blk, t.cancelCh, p.onProgress); err != nil {
		return nil, nil, err
	}

	vh := pBlk.NextValidatorsHash()
	vl, err := state.ValidatorSnapshotFromHash(c.Database(), vh)
	if err != nil {
		return nil, nil, err
	}
	_, err = votes.VerifyBlock(blk, vl)
	if err != nil {
		return nil, nil, err
	}

	if err = block.SetLastHeight(c.Database(), nil, height); err != nil {
		return nil, nil, err
	}

	return blk, votes, nil
}

func (t *taskReset) _exportGenesis(blk module.BlockData, votes module.CommitVoteSet, gsfile string) (rerr error) {
	if err := t.chain.prepareManagers(); err != nil {
		return err
	}
	defer t.chain.releaseManagers()
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

func (t *taskReset) _reportProgress(h int64, resolved, unresolved int) error {
	if r, ok := t.result.GetValue(); ok && r != nil {
		return r
	}
	atomic.StoreInt64(&t.reportHeight, h)
	atomic.StoreUint64(&t.reportResolved, uint64(resolved))
	atomic.StoreUint64(&t.reportUnresolved, uint64(unresolved))
	return nil
}

func (t *taskReset) _exportBlocks(dbDirNew string, dbTypeNew string, height int64, blockHash []byte, votes module.CommitVoteSet) (rblk module.Block, rvotes module.CommitVoteSet, ret error) {
	// open database for export
	_ = os.RemoveAll(dbDirNew)
	newDB, err := t.chain.openDatabase(dbDirNew, dbTypeNew)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		log.Must(newDB.Close())
		if ret != nil {
			log.Must(os.RemoveAll(dbDirNew))
		}
	}()

	// prepare managers
	if err := t.chain.prepareManagers(); err != nil {
		return nil, nil, err
	}
	defer t.chain.releaseManagers()

	// check block hash with height
	blk, err := t.chain.bm.GetBlockByHeight(height)
	if err != nil {
		return nil, nil, err
	}
	if !bytes.Equal(blk.ID(), blockHash) {
		return nil, nil, errors.InvalidStateError.Errorf("BlockIDInvalid(exp=%#x,real=%#x)",
			blockHash, blk.ID())
	}

	// copy blocks for new genesis
	if err := t.chain.bm.ExportBlocks(height, height, newDB, t._reportProgress); err != nil {
		return nil, nil, err
	}

	// use given votes or get votes from the next block
	if votes == nil {
		if nblk, err := t.chain.bm.GetBlockByHeight(height + 1); err != nil {
			return nil, nil, err
		} else {
			votes = nblk.Votes()
		}
	}
	return blk, votes, nil
}

func (t *taskReset) _syncBlocks(height int64, blockHash []byte, votes module.CommitVoteSet) (rblk module.BlockData, rvotes module.CommitVoteSet, rrb Revertible, ret error) {
	logger := t.chain.Logger()
	logger.Debugf("syncBlocks: START height=%d blockHash=%#x", height, blockHash)
	defer logger.Debugf("syncBlocks: DONE err=%+v", ret)
	rblk, rvotes, rrb, ret = t._syncBlocksWithDB(height, blockHash, votes)
	if ret == nil {
		return
	}
	logger.Debugf("syncBlocks: syncBlocksWithDB fails err=%v continue with syncBlocksWithNetwork", ret)
	return t._syncBlocksWithNetwork(height, blockHash)
}

func (t *taskReset) _syncBlocksWithDB(height int64, blockHash []byte, votes module.CommitVoteSet) (rblk module.BlockData, rvotes module.CommitVoteSet, rrb Revertible, ret error) {
	var rb Revertible
	defer func() {
		if ret != nil {
			rb.RevertOrCommit(true)
		} else {
			rrb = rb
		}
	}()

	// prepare database for exporting blocks
	chainDir := t.chain.cfg.AbsBaseDir()
	dbDir := path.Join(chainDir, DefaultDBDir)

	dbDirNew := dbDir + TempSuffix
	dbTypeNew := t.chain.cfg.DBType
	rblk, rvotes, ret = t._exportBlocks(dbDirNew, dbTypeNew, height, blockHash, votes)
	if ret != nil {
		return
	}
	rb.Append(func(revert bool) {
		if revert {
			log.Must(os.RemoveAll(dbDirNew))
		}
	})

	// replace with new database
	t.chain.releaseDatabase()
	rb.Append(func(revert bool) {
		if revert {
			t.chain.ensureDatabase()
		}
	})
	if ret = rb.Delete(dbDir); ret != nil {
		return
	}
	if ret = rb.Rename(dbDirNew, dbDir); ret != nil {
		return
	}
	t.chain.ensureDatabase()
	rb.Append(func(revert bool) {
		if revert {
			t.chain.releaseDatabase()
		}
	})

	// remove other directories
	contractDir := path.Join(chainDir, DefaultContractDir)
	if ret = rb.Delete(contractDir); ret != nil {
		return
	}
	walDir := path.Join(chainDir, DefaultWALDir)
	if ret = rb.Delete(walDir); ret != nil {
		return
	}
	cacheDir := path.Join(chainDir, DefaultCacheDir)
	if ret = rb.Delete(cacheDir); ret != nil {
		return
	}
	return
}

func (t *taskReset) _syncBlocksWithNetwork(height int64, blockHash []byte) (rblk module.BlockData, rvotes module.CommitVoteSet, rrb Revertible, ret error) {
	c := t.chain
	chainDir := c.cfg.AbsBaseDir()

	// prepare revert
	var rb Revertible
	defer func() {
		if ret != nil {
			rb.RevertOrCommit(true)
		} else {
			rrb = rb
		}
	}()

	// reset database
	c.releaseDatabase()
	rb.Append(func(revert bool) {
		if revert {
			c.ensureDatabase()
		}
	})
	dbDir := path.Join(chainDir, DefaultDBDir)
	if ret = rb.Delete(dbDir); ret != nil {
		return
	}
	c.ensureDatabase()
	rb.Append(func(revert bool) {
		if revert {
			c.releaseDatabase()
			log.Must(os.RemoveAll(dbDir))
		}
	})

	// remove other directories
	contractDir := path.Join(chainDir, DefaultContractDir)
	if ret = rb.Delete(contractDir); ret != nil {
		return
	}
	WALDir := path.Join(chainDir, DefaultWALDir)
	if ret = rb.Delete(WALDir); ret != nil {
		return
	}
	CacheDir := path.Join(chainDir, DefaultCacheDir)
	if ret = rb.Delete(CacheDir); ret != nil {
		return
	}

	rblk, rvotes, ret = t._prepareBlocks(height, blockHash)
	rb.Append(func(revert bool) {
		if revert {
			log.Must(os.RemoveAll(contractDir))
			log.Must(os.RemoveAll(WALDir))
			log.Must(os.RemoveAll(CacheDir))
		}
	})
	return
}

func (t *taskReset) _resetToGenesis() (ret error) {
	gsType, err := t.chain.GenesisStorage().Type()
	if err != nil {
		return err
	}
	switch gsType {
	case module.GenesisNormal:
		return t._cleanUp()
	case module.GenesisPruned:
		c := t.chain
		genesis, err := gs.NewPrunedGenesis(c.GenesisStorage().Genesis())
		if err != nil {
			return err
		}
		votesBytes, err := c.GenesisStorage().Get(genesis.Votes.Bytes())
		if err != nil {
			return err
		}
		votes := c.CommitVoteSetDecoder()(votesBytes)
		_, _, rb, err := t._syncBlocks(genesis.Height.Value, genesis.Block.Bytes(), votes)
		if err != nil {
			return err
		}
		defer func() {
			rb.RevertOrCommit(ret != nil)
		}()
		return nil
	default:
		return errors.InvalidStateError.Errorf("UnknownGenesisType(type=%d)", gsType)
	}
}

func loadGenesisStorage(file string) (g module.GenesisStorage, ret error) {
	fd, err := os.OpenFile(file, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return gs.NewFromFile(fd)
}

func (t *taskReset) _resetToHeight(height int64, blockHash []byte) (ret error) {
	blk, votes, rb, err := t._syncBlocks(height, blockHash, nil)
	if err != nil {
		return err
	}
	defer func() {
		rb.RevertOrCommit(ret != nil)
	}()

	// create pruned genesis
	if err := rb.Delete(t.gsfile); err != nil {
		return err
	}
	if err := t._exportGenesis(blk, votes, t.gsfile); err != nil {
		return err
	}
	rb.Append(func(revert bool) {
		if revert {
			_ = os.Remove(t.gsfile)
		}
	})

	// reload new genesis
	g, err := loadGenesisStorage(t.gsfile)
	if err != nil {
		return err
	}
	t.chain.cfg.GenesisStorage = g
	t.chain.cfg.Genesis = g.Genesis()
	return nil
}

func (t *taskReset) _reset() (ret error) {
	if t.height == 0 {
		return t._resetToGenesis()
	} else {
		return t._resetToHeight(t.height, t.blockHash)
	}
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
		cancelCh:  make(chan struct{}, 1),
	}
}
