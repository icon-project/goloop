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
	"path"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/chain/imports"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
)

var importStates = map[State]string{
	Starting: "import starting",
	Stopping: "import stopping",
	Failed:   "import failed",
	Finished: "import done",
}

type taskImport struct {
	chain  *singleChain
	src    string
	height int64
	result resultStore
}

func (t *taskImport) String() string {
	return fmt.Sprintf("Import(src=%s,height=%d)", t.src, t.height)
}

func (t *taskImport) OnError(err error) {
	t.result.SetValue(err)
}

func (t *taskImport) OnEnd(errCh <-chan error) {
	if t.chain.cs != nil {
		t.chain.cs.Term()
		t.chain.cs = nil
	}
	err := <-errCh
	if err != nil {
		t.chain.logger.Warnf("Fail to import err=%+v", err)
	}
	t.result.SetValue(err)
}

func (t *taskImport) DetailOf(s State) string {
	switch s {
	case Started:
		i, a := t._progress()
		return fmt.Sprintf("import %d/%d", i, a)
	default:
		if st, ok := importStates[s]; ok {
			return st
		} else {
			return s.String()
		}
	}
}

func (t *taskImport) _progress() (int64, int64) {
	if bm := t.chain.BlockManager(); bm != nil {
		if blk, err := bm.GetLastBlock(); err == nil {
			return blk.Height(), t.height
		}
	}
	return 0, 0
}

func (t *taskImport) Start() error {
	if err := t._import(); err != nil {
		t.chain.releaseManagers()
		t.result.SetValue(err)
		return err
	}
	return nil
}

func (t *taskImport) _import() error {
	c := t.chain
	chainDir := c.cfg.AbsBaseDir()

	pr := network.PeerRoleFlag(c.cfg.Role)
	c.nm = network.NewManager(c, c.nt, c.cfg.SeedAddr, pr.ToRoles()...)

	ContractDir := path.Join(chainDir, DefaultContractDir)
	var err error
	var ts module.Timestamper
	c.sm, ts, err = imports.NewServiceManagerForImport(c, c.nm, c.pm, c.plt,
		ContractDir, t.src, t.height, t)
	if err != nil {
		return err
	}
	c.bm, err = block.NewManager(c, ts, nil)
	if err != nil {
		return err
	}
	blk, err := c.bm.GetLastBlock()
	if err != nil {
		return err
	}
	if blk.Height() > t.height {
		return errors.Errorf("chain already have height %d\n", blk.Height())
	}

	WALDir := path.Join(chainDir, DefaultWALDir)
	c.cs = consensus.NewConsensus(c, WALDir, ts, nil)

	if err := c.cs.Start(); err != nil {
		return err
	}
	if err := c.nm.Start(); err != nil {
		return err
	}
	return nil
}

func (t *taskImport) Stop() {
	t.result.SetValue(errors.ErrInterrupted)
}

func (t *taskImport) Wait() error {
	result := t.result.Wait()
	t.chain.releaseManagers()
	return result
}

func newTaskImport(chain *singleChain, src string, height int64) chainTask {
	return &taskImport{
		chain:  chain,
		src:    src,
		height: height,
	}
}
