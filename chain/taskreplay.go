/*
 * Copyright 2023 ICON Foundation
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
 *
 */

package chain

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
)

type verifyParams struct {
	Start  int64 `json:"start"`
	End    int64 `json:"end"`
	Detail bool  `json:"detail"`
}

type taskReplay struct {
	chain  *singleChain
	tmpDB  db.LayerDB
	result resultStore
	height int64
	start  int64
	end    int64
	detail bool
	stop   chan error
}

func (t *taskReplay) Stop() {
	t.stop <- errors.ErrInterrupted
}

func (t *taskReplay) Wait() error {
	return t.result.Wait()
}

func (t *taskReplay) String() string {
	return fmt.Sprintf("Replay(start=%d,end=%d,detail=%v)",
		t.start, t.end, t.detail)
}

func (t *taskReplay) DetailOf(s State) string {
	switch s {
	case Started:
		return fmt.Sprintf("replay started height=%d", t.height)
	default:
		return "replay " + s.String()
	}
}

func (t *taskReplay) initTransition() (module.Block, module.Transition, error) {
	sm := t.chain.ServiceManager()
	bm := t.chain.BlockManager()
	blk, err := bm.GetBlockByHeight(t.height)
	if err != nil {
		return nil, nil, err
	}
	tr, err := sm.CreateInitialTransition(blk.Result(), blk.NextValidators())
	return blk, tr, err
}

type transitionCallback chan error

func (t transitionCallback) OnValidate(transition module.Transition, err error) {
	t <- err
}

func (t transitionCallback) OnExecute(transition module.Transition, err error) {
	t <- err
}

func (t *taskReplay) doReplay() error {
	defer func() {
		t.chain.releaseManagers()
		t.chain.database = t.tmpDB.Unwrap()
	}()
	var err error

	t.height = t.start

	bm := t.chain.BlockManager()
	sm := t.chain.ServiceManager()
	logger := t.chain.Logger()

	end := t.end
	if last, err := bm.GetLastBlock(); err != nil {
		return err
	} else {
		lastHeight := last.Height()
		if end == 0 || end > lastHeight-1 {
			end = lastHeight - 1
		}
	}

	blk, ptr, err := t.initTransition()
	if err != nil {
		return err
	}
	var nblk module.Block
	var tr module.Transition
	for t.height <= end {
		// next block for votes and consensus information
		nblk, err = bm.GetBlockByHeight(t.height + 1)
		if err != nil {
			return err
		}
		csi, err := bm.NewConsensusInfo(blk)
		if err != nil {
			return err
		}
		tr, err = sm.CreateTransition(ptr, blk.NormalTransactions(), blk, csi, true)
		if err != nil {
			return err
		}
		ptxs := nblk.PatchTransactions()
		if len(ptxs.Hash()) > 0 {
			tr = sm.PatchTransition(tr, ptxs, nblk)
		}
		cb := make(chan error, 2)
		cancel, err := tr.Execute(transitionCallback(cb))
		if err != nil {
			return err
		}

		// wait for OnValidate
		select {
		case err := <-t.stop:
			cancel()
			return err
		case err := <-cb:
			if err != nil {
				return err
			}
		}

		// wait for OnExecute
		select {
		case err := <-t.stop:
			cancel()
			return err
		case err := <-cb:
			if err != nil {
				return err
			}
		}

		// check the result
		if !bytes.Equal(tr.Result(), nblk.Result()) {
			logger.Errorf("INVALID RESULT res=%#x exp=%#x",
				tr.Result(), nblk.Result())
			if t.detail {
				_ = sm.Finalize(tr, module.FinalizeResult)
				if err := service.ShowResultDiff(t.tmpDB, t.chain.plt, logger, nblk.Result(), tr.Result()); err != nil {
					logger.Errorf("FAIL to show diff err=%+v", err)
				}
			}
			return errors.InvalidStateError.New("InvalidResult")
		} else {
			if err := service.FinalizeTransition(tr,
				module.FinalizeNormalTransaction|module.FinalizePatchTransaction|module.FinalizeResult,
				false); err != nil {
				return err
			}
			_ = t.tmpDB.Flush(false)
		}
		t.height += 1
		ptr, tr = tr, nil
		blk, nblk = nblk, nil
	}
	return nil
}

func (t *taskReplay) Start() error {
	t.tmpDB = db.NewLayerDB(t.chain.database)
	t.chain.database = t.tmpDB

	if err := t.chain.prepareManagers(); err != nil {
		t.chain.database = t.tmpDB.Unwrap()
		t.result.SetValue(err)
		return err
	}
	t.stop = make(chan error, 1)
	go func() {
		err := t.doReplay()
		defer t.result.SetValue(err)
	}()
	return nil
}

func taskReplayFactory(chain *singleChain, param json.RawMessage) (chainTask, error) {
	var p verifyParams
	if err := json.Unmarshal(param, &p); err != nil {
		return nil, err
	}
	if (p.End != 0 && p.End < p.Start) || p.Start < 0 {
		return nil, errors.IllegalArgumentError.Errorf(
			"InvalidParameter(start=%d,end=%d)", p.Start, p.End)
	}
	task := &taskReplay{
		chain:  chain,
		start:  p.Start,
		end:    p.End,
		detail: p.Detail,
	}
	return task, nil
}

func init() {
	registerTaskFactory("replay", taskReplayFactory)
}
