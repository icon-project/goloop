/*
 * Copyright 2022 ICON Foundation
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

package block

import (
	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
)

type finalizeRequest struct {
	sm     ServiceManager
	syncTr module.Transition
	dbase  db.Database
	resCh  chan error
}

func (r *finalizeRequest) finalize(blk module.BlockData) error {
	ntr, err := r.sm.CreateTransition(r.syncTr, blk.NormalTransactions(), blk, nil, true)
	if err != nil {
		return nil
	}
	if err = service.FinalizeTransition(ntr, module.FinalizeNormalTransaction, false); err != nil {
		return err
	}
	if err = service.FinalizeTransition(r.syncTr, module.FinalizePatchTransaction|module.FinalizeResult, false); err != nil {
		return err
	}

	if err = blk.(base.BlockVersionSpec).FinalizeHeader(r.dbase); err != nil {
		return err
	}
	if err = WriteTransactionLocators(r.dbase, blk.Height(), blk.PatchTransactions(), blk.NormalTransactions()); err != nil {
		return err
	}
	return nil
}

func (r *finalizeRequest) OnValidate(t module.Transition, err error) {
	if err != nil {
		log.Warnf("unexpected error during forced finalization: %+v", err)
		r.resCh <- err
	}
}

func (r *finalizeRequest) OnExecute(t module.Transition, err error) {
	r.resCh <- err
}

func UnsafeFinalize(
	sm ServiceManager,
	c module.Chain,
	blk module.BlockData,
	cancelCh <-chan struct{},
	progressCB module.ProgressCallback,
) error {
	initTr, err := sm.CreateInitialTransition(nil, nil)
	if err != nil {
		return err
	}
	bi := common.NewBlockInfo(blk.Height()-1, blk.Timestamp()-1)
	tr, err := sm.CreateTransition(initTr, nil, bi, nil, true)
	if err != nil {
		return err
	}
	tr = sm.PatchTransition(tr, blk.PatchTransactions(), blk)
	syncTr := sm.CreateSyncTransition(tr, blk.Result(), blk.NextValidatorsHash(), true)

	// Assume that the transition supports SetProgressCallback method
	// to monitoring progress.
	// This monitoring feature is not essential
	type setProgressCallbacker interface {
		SetProgressCallback(cb module.ProgressCallback)
	}
	if setter, ok := syncTr.(setProgressCallbacker); ok {
		setter.SetProgressCallback(progressCB)
	} else {
		log.Warnln("transition doesn't support SetProgressCallback()")
	}

	r := &finalizeRequest{
		sm:     sm,
		syncTr: syncTr,
		dbase:  c.Database(),
		resCh:  make(chan error, 2),
	}
	canceler, err := syncTr.Execute(r)
	if err != nil {
		return err
	}
	select {
	case err := <-r.resCh:
		if err != nil {
			return err
		}
		return r.finalize(blk)
	case <-cancelCh:
		canceler()
		return errors.Errorf("sync canceled height=%d hash=%x", blk.Height(), blk.Height())
	}
}
