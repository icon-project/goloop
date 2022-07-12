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
	"fmt"
	"os"
	"path"
	"sync/atomic"

	"github.com/icon-project/goloop/chain/gs"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

var pruningStates = map[State]string{
	Starting: "pruning starting",
	Stopping: "pruning stopping",
	Failed:   "pruning failed",
	Finished: "pruning done",
}

type taskPruning struct {
	chain   *singleChain
	result  resultStore
	gsfile  string
	dbtype  string
	height  int64
	blocks  int64
	current int64
}

func (t *taskPruning) String() string {
	return fmt.Sprintf("Pruning(height=%d)", t.height)
}

func (t *taskPruning) DetailOf(s State) string {
	switch s {
	case Started:
		i, a := t._progress()
		return fmt.Sprintf("pruning %d/%d", i, a)
	default:
		if st, ok := pruningStates[s]; ok {
			return st
		} else {
			return s.String()
		}
	}
}

func (t *taskPruning) Start() error {
	if err := t.chain.prepareManagers(); err != nil {
		return err
	}
	blk, err := t.chain.bm.GetLastBlock()
	if err != nil {
		t.chain.releaseManagers()
		return err
	}
	if t.height >= blk.Height() {
		t.chain.releaseManagers()
		return errors.IllegalArgumentError.Errorf(
			"InvalidHeight(height=%d,last=%d)", t.height, blk.Height())
	}
	t.blocks = blk.Height() - t.height + 1
	t.current = 0
	go t.doPruning()
	return nil
}

func (t *taskPruning) doPruning() {
	err := t._prune(t.gsfile, t.dbtype, t.height)
	t.result.SetValue(err)
}

func (t *taskPruning) OnExport(height int64) error {
	if atomic.LoadInt64(&t.blocks) == 0 {
		return errors.ErrInterrupted
	}
	atomic.StoreInt64(&t.current, height-t.height+1)
	return nil
}

func (t *taskPruning) _progress() (int64, int64) {
	blocks := atomic.LoadInt64(&t.blocks)
	if blocks == 0 {
		return 0, 0
	}
	current := atomic.LoadInt64(&t.current)
	return current, blocks
}

func (t *taskPruning) _exportGenesis(blk module.Block, votes module.CommitVoteSet, gsfile string) (rerr error) {
	os.RemoveAll(gsfile)
	fd, err := os.OpenFile(gsfile, os.O_CREATE|os.O_WRONLY|os.O_EXCL|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	gsw := gs.NewGenesisStorageWriter(fd)
	defer func() {
		gsw.Close()
		fd.Close()
		if rerr != nil {
			os.Remove(gsfile)
		}
	}()
	if err := t.chain.bm.ExportGenesis(blk, votes, gsw); err != nil {
		return errors.Wrap(err, "fail on exporting genesis storage")
	}
	return nil
}

func (t *taskPruning) _copyDatabase(dbpath, dbtype string, from, to int64) (rerr error) {
	os.RemoveAll(dbpath)
	dbase, err := t.chain.openDatabase(dbpath, dbtype)
	if err != nil {
		return err
	}
	defer func() {
		dbase.Close()
		if rerr != nil {
			os.RemoveAll(dbpath)
		}
	}()
	return t.chain.bm.ExportBlocks(from, to, dbase, t.OnExport)
}

func (t *taskPruning) _interrupted() bool {
	return atomic.LoadInt64(&t.blocks) == 0
}

func (t *taskPruning) _prune(gsfile, dbtype string, height int64) (rerr error) {
	c := t.chain
	defer c.releaseManagers()

	chainDir := c.cfg.ResolveAbsolute(c.cfg.BaseDir)
	dbpath := path.Join(chainDir, DefaultTmpDBDir)
	gsTmp := gsfile + ".tmp"

	blk, err := c.bm.GetBlockByHeight(height)
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

	nblk, err := c.bm.GetBlockByHeight(height + 1)
	if err != nil {
		return errors.InvalidStateError.Errorf("No next block height=%d", height)
	}

	c.logger.Infof("Export Genesis to=%s from=%d", gsTmp, height)
	if err := t._exportGenesis(blk, nblk.Votes(), gsTmp); err != nil {
		return err
	}
	defer func() {
		if rerr != nil {
			os.Remove(gsTmp)
		}
	}()

	if t._interrupted() {
		return errors.ErrInterrupted
	}

	lb, err := c.bm.GetLastBlock()
	if err != nil {
		return err
	}
	targetHeight := lb.Height()
	c.logger.Infof("Copy Database path=%s type=%s from=%d to=%d",
		dbpath, dbtype, height, targetHeight)
	err = t._copyDatabase(dbpath, dbtype, height, targetHeight)
	if err != nil {
		return err
	}
	defer func() {
		if rerr != nil {
			os.RemoveAll(dbpath)
		}
	}()

	if t._interrupted() {
		return errors.ErrInterrupted
	}

	c.releaseManagers()
	c.releaseDatabase()
	defer c.ensureDatabase()

	target := path.Join(chainDir, DefaultDBDir)
	dbbk := target + ".bk"
	gsbk := gsfile + ".bk"

	c.logger.Infof("Replace DB %s -> %s", dbpath, target)
	os.RemoveAll(dbbk)
	if err := os.Rename(target, dbbk); err != nil {
		return errors.UnknownError.Errorf("file on backup %s to %s",
			target, dbbk)
	}
	defer func() {
		if rerr != nil {
			os.RemoveAll(target)
			os.Rename(dbbk, target)
		} else {
			os.RemoveAll(dbbk)
		}
	}()
	if err := os.Rename(dbpath, target); err != nil {
		return errors.UnknownError.Errorf("fail on rename %s to %s",
			dbpath, target)
	}

	c.logger.Infof("Replace GS %s -> %s", gsTmp, gsfile)
	os.RemoveAll(gsbk)
	if err := os.Rename(gsfile, gsbk); err != nil {
		return errors.UnknownError.Errorf("fail on backup %s to %s",
			gsfile, gsbk)
	}
	defer func() {
		if rerr != nil {
			os.RemoveAll(gsfile)
			os.Rename(gsbk, gsfile)
		} else {
			os.RemoveAll(gsbk)
		}
	}()
	if err := os.Rename(gsTmp, gsfile); err != nil {
		return errors.UnknownError.Errorf("fail to rename %s to %s",
			gsTmp, gsfile)
	}

	// replace genesis
	fd, err := os.Open(gsfile)
	if err != nil {
		return errors.UnknownError.Wrapf(err, "fail to open file=%s", gsfile)
	}
	g, err := gs.NewFromFile(fd)
	if err != nil {
		return errors.UnknownError.Wrapf(err, "fail to parse gs=%s", gsfile)
	}

	c.logger.Infof("Reopen DB %s", chainDir)
	c.cfg.DBType = dbtype
	c.cfg.GenesisStorage = g
	c.cfg.Genesis = g.Genesis()

	if err := c.cfg.Save(); err != nil {
		return errors.UnknownError.Wrap(err, "fail to store configuration")
	}

	return nil
}

func (t *taskPruning) Stop() {
	atomic.StoreInt64(&t.blocks, 0)
}

func (t *taskPruning) Wait() error {
	return t.result.Wait()
}

func newTaskPruning(chain *singleChain, gsfile, dbtype string, height int64) chainTask {
	return &taskPruning{
		chain:  chain,
		gsfile: gsfile,
		dbtype: dbtype,
		height: height,
	}
}
