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

package lcimporter

import (
	"sync"

	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

type transitionID struct{ int }

type transitionState int

const (
	stepInitial transitionState = iota
	stepProposed
	stepNeedSync
	stepExecuting
	stepComplete
	stepCanceled
	stepFailed
)

type transitionBase struct {
	sm  *ServiceManager
	ex  *Executor
	log log.Logger
}

type transition struct {
	*transitionBase

	pid    *transitionID
	parent *transition

	lock  sync.Mutex
	state transitionState
	bi    module.BlockInfo
	txs   module.TransactionList

	worldSnapshot  trie.Immutable
	nextValidators module.ValidatorList
	receipts       module.ReceiptList
}

func (t *transition) PatchTransactions() module.TransactionList {
	return t.sm.emptyTransactions
}

func (t *transition) NormalTransactions() module.TransactionList {
	return t.txs
}

func (t *transition) PatchReceipts() module.ReceiptList {
	return t.sm.emptyReceipts
}

func (t *transition) NormalReceipts() module.ReceiptList {
	return t.receipts
}

func newTransactionSliceFromList(txl module.TransactionList) ([]*BlockTransaction, error) {
	var txs []*BlockTransaction
	for itr := txl.Iterator(); itr.Has(); _ = itr.Next() {
		tx, _, err := itr.Get()
		if err != nil {
			return nil, err
		}
		btx := transaction.Unwrap(tx).(*BlockTransaction)
		txs = append(txs, btx)
	}
	return txs, nil
}

func (t *transition) transitState(target transitionState, from ...transitionState) bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	if len(from) > 0 {
		for _, s := range from {
			if t.state == s {
				t.state = target
				return true
			}
		}
		return false
	}
	t.state = target
	return true
}

func makeReceiptList(dbase db.Database, size int, rct txresult.Receipt) module.ReceiptList {
	rcts := make([]txresult.Receipt, size)
	for i := 0; i < size; i++ {
		rcts[i] = rct
	}
	return txresult.NewReceiptListFromSlice(dbase, rcts)
}

func (t *transition) setResult(next int64, txs int, vl module.ValidatorList) {
	t.lock.Lock()
	defer t.lock.Unlock()

	ws := trie_manager.NewMutableFromImmutable(t.parent.worldSnapshot)
	scoredb.NewVarDB(containerdb.NewBytesStoreStateFromRaw(ws), VarNextBlockHeight).Set(next)
	t.worldSnapshot = ws.GetSnapshot()
	t.receipts = makeReceiptList(t.sm.db, txs, t.sm.defaultReceipt)
	if vl != nil {
		t.nextValidators = vl
	} else {
		t.nextValidators = t.parent.nextValidators
	}
	t.state = stepComplete
}

func (t *transition) doExecute(cb module.TransitionCallback, check bool) (ret error) {
	cb.OnValidate(t, nil)

	defer func() {
		if ret != nil {
			t.transitState(stepFailed, stepExecuting)
			cb.OnExecute(t, ret)
		}
	}()

	if t.bi.Height() == 0 {
		vls, err := t.sm.getValidators()
		if err != nil {
			return err
		}
		t.setResult(1, 1, vls)
		cb.OnExecute(t, nil)
		return nil
	}

	txs, err := newTransactionSliceFromList(t.txs)
	if err != nil {
		return err
	}
	if len(txs) < 1 {
		return errors.UnknownError.New("EmptyTransactions")
	}

	if check {
		chn := make(chan interface{}, 1)
		blockCallback := func(txs []*BlockTransaction, err error) {
			if err != nil {
				chn <- err
			} else {
				chn <- txs
			}
		}
		if _, err := t.ex.GetTransactions(txs[0].Height, txs[len(txs)-1].Height, blockCallback); err != nil {
			return err
		}
		var rtxs []*BlockTransaction
		select {
		case result := <-chn:
			if err, ok := result.(error); ok {
				return err
			}
			rtxs = result.([]*BlockTransaction)
		}

		// check length
		if len(rtxs) != len(txs) {
			return errors.InvalidStateError.Errorf(
				"InvalidTxList(rtxs=%d,txs=%d)",
				len(rtxs),
				len(txs),
			)
		}

		// compare each transactions
		for idx, tx := range txs {
			rtx := rtxs[idx]
			if !tx.Equal(rtx) {
				return errors.InvalidStateError.Errorf(
					"HasDifferentResult(exp=%+v,real=%+v)",
					tx,
					rtx,
				)
			}
		}
	}

	t.setResult(txs[len(txs)-1].Height+1, len(txs), nil)
	cb.OnExecute(t, nil)
	return nil
}

func (t *transition) doSync(cb module.TransitionCallback) (ret error) {
	cb.OnValidate(t, nil)

	defer func() {
		if ret != nil {
			t.transitState(stepFailed, stepExecuting)
			cb.OnExecute(t, ret)
		}
	}()

	txs, err := newTransactionSliceFromList(t.txs)
	if err != nil {
		return err
	}
	if len(txs) < 1 {
		return errors.CriticalFormatError.New("NoTransactions")
	}
	if err := t.ex.SyncTransactions(txs); err != nil {
		return err
	}

	t.setResult(txs[len(txs)-1].Height, len(txs), nil)
	cb.OnExecute(t, nil)
	return nil
}

func (t *transition) cancel() bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.state == stepComplete {
		return false
	}
	t.state = stepCanceled
	return true
}

func (t *transition) Execute(cb module.TransitionCallback) (canceler func() bool, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	switch t.state {
	case stepInitial, stepProposed:
		check := t.state == stepInitial
		t.state = stepExecuting
		go t.doExecute(cb, check)
		return t.cancel, nil
	case stepNeedSync:
		t.state = stepExecuting
		go t.doSync(cb)
		return t.cancel, nil
	default:
		return nil, errors.ErrInvalidState
	}
}

func (t *transition) ExecuteForTrace(ti module.TraceInfo) (canceler func() bool, err error) {
	return nil, errors.ErrUnsupported
}

func (t *transition) Result() []byte {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.state != stepComplete {
		return nil
	}
	return t.worldSnapshot.Hash()
}

func (t *transition) NextValidators() module.ValidatorList {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.state != stepComplete {
		return nil
	}
	return t.nextValidators
}

func (t *transition) LogsBloom() module.LogsBloom {
	return new(txresult.LogsBloom)
}

func (t *transition) BlockInfo() module.BlockInfo {
	return t.bi
}

func (t *transition) Equal(t2 module.Transition) bool {
	tr2, _ := t2.(*transition)
	if t == tr2 {
		return true
	}
	if t == nil || tr2 == nil {
		return false
	}
	if t.pid == tr2.pid {
		return true
	}
	// TODO implement
	return false
}

func (t *transition) finalizeTransactions() error {
	if err := t.txs.Flush(); err != nil {
		return err
	}
	return nil
}

func (t *transition) finalizeResult() error {
	if err := t.worldSnapshot.(trie.Snapshot).Flush(); err != nil {
		return err
	}
	if err := t.receipts.Flush(); err != nil {
		return err
	}
	return nil
}

func CreateInitialTransition(dbase db.Database, result []byte, nvl module.ValidatorList, sm *ServiceManager, ex *Executor) *transition {
	return &transition{
		transitionBase: &transitionBase{
			sm:  sm,
			ex:  ex,
			log: sm.log,
		},

		pid: new(transitionID),

		state:          stepComplete,
		worldSnapshot:  trie_manager.NewImmutable(dbase, result),
		nextValidators: nvl,
	}
}

func CreateTransition(parent *transition, bi module.BlockInfo, txs module.TransactionList, executed bool) *transition {
	var state transitionState
	if executed {
		state = stepProposed
	} else {
		state = stepInitial
	}
	tr := &transition{
		transitionBase: parent.transitionBase,

		pid:    new(transitionID),
		parent: parent,

		state: state,
		bi:    bi,
		txs:   txs,
	}
	return tr
}

func CreateSyncTransition(tr *transition) *transition {
	return &transition{
		transitionBase: tr.transitionBase,

		pid:    new(transitionID),
		parent: tr.parent,

		state: stepNeedSync,
		bi:    tr.bi,
		txs:   tr.txs,
	}
}
