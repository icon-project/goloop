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
	"container/list"
	"fmt"
	"io"
	"sync"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie/cache"
	"github.com/icon-project/goloop/icon/blockv0/lcstore"
	"github.com/icon-project/goloop/module"
)

const (
	KeyNextBlockHeight = "block.lastFinalizedHeight"
	TransactionsPerBlock = 3_000
)

const (
	terminationMark = -1
)

type OnBlockTransactions func([]*BlockTransaction, error)
type Canceler func()

type IBlockConverter interface {
	Rebase(from, to int64, txs []*BlockTransaction) (<-chan interface{}, error)
	// Term()
}

type consumeID *int

type Executor struct {
	idb db.Database // database for import chain
	rdb db.Database // database for real chain
	log log.Logger

	chainBucket db.Bucket

	lock   sync.Mutex
	txs    list.List // cached transactions (start <= tx.Height < end)
	start  int64
	end    int64
	next   int64
	waiter *txWaiter

	consumer consumeID
	bc       IBlockConverter
}

func (e *Executor) candidateInLock(from int64) ([]*BlockTransaction, error) {
	if e.start > from {
		return nil, errors.InvalidStateError.New("NeedToRebase")
	}
	if e.end <= from {
		return []*BlockTransaction{}, nil
	}

	txe := e.txs.Front()
	for txe != nil {
		if txe.Value == io.EOF {
			return nil, errors.InvalidStateError.New("AlreadyEnded")
		}
		if txe.Value.(*BlockTransaction).Height == from {
			break
		}
		txe = txe.Next()
	}

	var txs []*BlockTransaction
	for cnt := 0 ; txe != nil && cnt < TransactionsPerBlock ; txe = txe.Next() {
		if txe.Value == io.EOF {
			if len(txs) == 0 {
				return nil, errors.InvalidStateError.New("AlreadyEnded")
			}
			break
		}
		tx := txe.Value.(*BlockTransaction)
		txs = append(txs, tx)
		cnt += int(tx.TXCount)
	}
	return txs, nil
}

// ProposeTransactions propose transactions for blocks to be consensus
// after finalized.
func (e *Executor) ProposeTransactions() ([]*BlockTransaction, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.start == terminationMark {
		return nil, errors.InvalidStateError.New("AlreadyTerminated")
	}

	if txs, err := e.candidateInLock(e.next); err != nil {
		if err := e.rebaseInLock(e.next, -1, nil); err != nil {
			return nil, err
		}
		e.cancelWaiterInLock()
		return []*BlockTransaction{}, nil
	} else {
		return txs, nil
	}
}

type txWaiter struct {
	ex       *Executor
	cb       OnBlockTransactions
	from, to int64
	txs      []*BlockTransaction
}

func (w *txWaiter) addAndCheck(tx *BlockTransaction) bool {
	if tx.Height == w.from && w.from <= w.to {
		w.txs = append(w.txs, tx)
		w.from += 1
		if tx.Height == w.to {
			go w.cb(w.txs, nil)
			return true
		}
	}
	return false
}

func (w *txWaiter) notifyCanceled(err error) {
	go w.cb(nil, err)
}

func (w *txWaiter) cancel() {
	 if w.ex.removeWaiter(w) {
		 w.notifyCanceled(errors.InterruptedError.New("Canceled"))
	 }
}

func (w *txWaiter) String() string {
	return fmt.Sprintf("GetTransaction(from=%d,to=%d)", w.from, w.to)
}

func (e *Executor) dummyCanceler() {
	e.log.Debugln("RESULT already sent")
}

// GetTransactions get transactions in the range.
func (e *Executor) GetTransactions(from, to int64, callback OnBlockTransactions) (Canceler, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.start == terminationMark {
		return nil, errors.InvalidStateError.New("AlreadyTerminated")
	}
	if to < from {
		return nil, errors.IllegalArgumentError.Errorf("InvalidRequest(from=%d,to=%d)", from, to)
	}
	w := &txWaiter{
		ex:   e,
		cb:   callback,
		from: from,
		to:   to,
	}
	if from < e.start {
		if err := e.rebaseInLock(from, -1, nil); err != nil {
			return nil, err
		}
	} else {
		if from < e.end {
			for itr := e.txs.Front() ; itr != nil ; itr = itr.Next() {
				if itr.Value == io.EOF {
					return nil,  errors.InvalidStateError.New("AlreadyEnded")
				}
				if r := w.addAndCheck(itr.Value.(*BlockTransaction)); r {
					return e.dummyCanceler, nil
				}
			}
		}
	}
	e.resetWaiterInLock(w)
	return w.cancel, nil
}

func (e *Executor) removeWaiter(w *txWaiter) bool {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.waiter == w {
		e.waiter = nil
		return true
	}
	return false
}

func (e *Executor) resetWaiterInLock(nw *txWaiter) {
	if w := e.waiter ; w != nil {
		w.notifyCanceled(errors.ErrInterrupted)
	}
	e.waiter = nw
}

func (e *Executor) cancelWaiterInLock() {
	e.resetWaiterInLock(nil)
}

// FinalizeTransactions finalize transactions to the height
func (e *Executor) FinalizeTransactions(to int64) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.start == terminationMark {
		return errors.InvalidStateError.New("AlreadyTerminated")
	}

	if to < e.start {
		// already it's finalized, so nothing to do
		return nil
	}
	if to >= e.end {
		return errors.InvalidStateError.Errorf("NotReachedYet(to=%d,end=%d)", to, e.end)
	}
	for txe := e.txs.Front(); txe!= nil && e.start <= to ; e.start += 1 {
		txe, _ = txe.Next(), e.txs.Remove(txe)
	}
	e.next = to+1
	return storeNextBlockHeight(e.chainBucket, e.next)
}

func (e *Executor) rebaseInLock(from, to int64, txs []*BlockTransaction) error {
	chn, err := e.bc.Rebase(from, to, txs)
	if err != nil {
		e.log.Errorf("Failure in BlockConverter.Rebase err=%+v", err)
		return err
	}
	e.txs.Init()
	e.start = from
	e.end = from
	e.consumer = new(int)
	go e.consumeBlocks(e.consumer, chn)
	return nil
}

// SyncTransactions sync transactions
func (e *Executor) SyncTransactions(txs []*BlockTransaction) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	if len(txs) == 0 {
		return errors.IllegalArgumentError.New("EmptyTransactions")
	}
	if e.start == terminationMark {
		return errors.InvalidStateError.New("AlreadyTerminated")
	}

	from := txs[0].Height
	if from != e.next {
		return errors.IllegalArgumentError.Errorf("InvalidSync(height=%d,next=%d)", from, e.next)
	}

	e.log.Debugf("SyncTransactions(from=%d,cnt=%d)", from, len(txs))
	if err := e.rebaseInLock(from, -1, txs); err != nil {
		return err
	}
	e.cancelWaiterInLock()
	return nil
}

func storeNextBlockHeight(bk db.Bucket, h int64) error {
	return bk.Set([]byte(KeyNextBlockHeight), codec.BC.MustMarshalToBytes(h))
}

func loadNextBlockHeight(bk db.Bucket) (int64, error) {
	bs, err := bk.Get([]byte(KeyNextBlockHeight))
	if err != nil {
		return 0, err
	}
	if len(bs) > 0 {
		var height int64
		if _, err := codec.BC.UnmarshalFromBytes(bs, &height); err == nil {
			return height, nil
		}
	}
	return 0, nil
}

func (e *Executor) GetImportedBlocks() int64 {
	e.lock.Lock()
	defer e.lock.Unlock()
	return e.next
}

// addTransaction add transaction from block converter
// it's called by consumeBlocks.
func (e *Executor) addTransaction(id consumeID, tx *BlockTransaction) bool {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.consumer == id && e.end == tx.Height {
		e.log.Tracef("addTransaction height=%d", tx.Height)
		e.txs.PushBack(tx)
		e.end = tx.Height+1
		if w := e.waiter ; w != nil {
			if ok := w.addAndCheck(tx); ok {
				e.waiter = nil
			}
		}
		return true
	}
	return false
}

func (e *Executor) notifyEnd(id consumeID) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.consumer == id {
		txe := e.txs.Back()
		if txe != nil && txe.Value == io.EOF {
			e.log.Tracef("notifyEnd already notified")
			return
		}
		e.log.Tracef("notifyEnd height=%d", e.end)
		e.txs.PushBack(io.EOF)
		e.end += 1
		if w := e.waiter; w != nil {
			w.notifyCanceled(errors.InterruptedError.New("EndedTransaction"))
			e.waiter = nil
		}
	}
}

// consumeBlocks consume all data from the channel
// until it's closed. So, if it will not send anything
// through the channel, it should be closed.
func (e *Executor) consumeBlocks(id consumeID, chn <-chan interface{}) {
	var err error
	e.log.Debugf("consumeBlocks START chn=%+v", chn)
	defer e.log.Debugf("consumeBlocks STOP chn=%+v err=%v", chn, err)
	for true {
		tx, ok := <-chn
		if !ok {
			if err == nil {
				e.notifyEnd(id)
			}
			return
		}
		if err != nil {
			continue
		}
		switch obj := tx.(type) {
		case *BlockTransaction:
			if ok := e.addTransaction(id, obj); !ok {
				err = errors.ErrInvalidState
			}
		case error:
			err = obj
		}
	}
}

func (e *Executor) Start() error {
	e.lock.Lock()
	defer e.lock.Unlock()

	return e.rebaseInLock(e.next, -1, nil)
}

func (e *Executor) Term() {
	// TODO terminate BC
	// e.bc.Term()

	e.lock.Lock()
	defer e.lock.Unlock()
	e.start = terminationMark
	e.end = terminationMark
	e.txs.Init()
	e.cancelWaiterInLock()
}

func NewExecutorWithBC(rdb, idb db.Database, logger log.Logger, bc IBlockConverter) (*Executor, error) {
	chainBucket, err := idb.GetBucket(db.ChainProperty)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInBucket(bucket=ChainProperty)")
	}

	next, err := loadNextBlockHeight(chainBucket)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInLoadNextBlockHeight")
	}

	ex := &Executor{
		rdb: rdb,
		idb: idb,
		log: logger,

		next: next,

		chainBucket: chainBucket,

		bc: bc,
	}
	ex.txs.Init()
	return ex, nil
}

func NewExecutor(chain module.Chain, dbase db.Database, cfg *Config) (*Executor, error) {
	logger := chain.Logger()
	idb := chain.Database()

	// build converter
	rdb := cache.AttachManager(dbase, "", 5, 0, 0)
	chain = NewChain(chain, rdb)
	store, err := lcstore.OpenStore(cfg.StoreURI)
	if err != nil {
		return nil, err
	}
	cs := lcstore.NewForwardCache(store, logger, &cfg.CacheConfig)
	cs.SetReceiptParameter(rdb, module.LatestRevision)
	bc, err := NewBlockConverter(chain, cfg.Platform, cs, cfg.BaseDir)
	if err != nil {
		return nil, err
	}

	return NewExecutorWithBC(rdb, idb, logger, bc)
}
