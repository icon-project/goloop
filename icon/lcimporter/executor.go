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
	"math"
	"sync"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie/cache"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/blockv0/lcstore"
	"github.com/icon-project/goloop/icon/icdb"
	"github.com/icon-project/goloop/icon/merkle/hexary"
	"github.com/icon-project/goloop/module"
)

const (
	TransactionsPerBlock = 3_000
	TransactionsToStore  = 4_000
)

const (
	terminationMark = -1
)

type OnBlockTransactions func([]*BlockTransaction, error)
type Canceler func()

type IBlockConverter interface {
	Rebase(from, to int64, txs []*BlockTransaction) (<-chan interface{}, error)
	GetBlockVotes(height int64) (*blockv0.BlockVoteList, error)
	Term()
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
	waiter *txWaiter

	consumer consumeID
	pending  *sync.Cond
	bc       IBlockConverter

	acc hexary.Accumulator
}

func (e *Executor) candidateInLock(from int64) ([]*BlockTransaction, error) {
	if e.start > from {
		if err := e.rebaseInLock(from, -1, nil); err != nil {
			return nil, err
		}
		e.cancelWaiterInLock()
		return []*BlockTransaction{}, nil
	}

	if e.end <= from {
		return []*BlockTransaction{}, nil
	}

	txe := e.txs.Front()
	for txe != nil {
		if err, ok := txe.Value.(error); ok {
			return nil, err
		}
		if txe.Value.(*BlockTransaction).Height == from {
			break
		}
		txe = txe.Next()
	}

	var txs []*BlockTransaction
	for cnt := 0 ; txe != nil && cnt < TransactionsPerBlock ; txe = txe.Next() {
		if err, ok := txe.Value.(error); ok {
			if len(txs) == 0 {
				return nil, err
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
func (e *Executor) ProposeTransactions(from int64) ([]*BlockTransaction, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.start == terminationMark {
		return nil, errors.InvalidStateError.New("AlreadyTerminated")
	}

	if txs, err := e.candidateInLock(from); err != nil {
		return nil, err
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
				if err, ok := itr.Value.(error); ok {
					if err == ErrAfterLastBlock {
						return nil, errors.InvalidStateError.New("AlreadyEnded")
					} else {
						return nil, err
					}
				}
				btx := itr.Value.(*BlockTransaction)
				if r := w.addAndCheck(btx); r {
					e.cancelWaiterInLock()
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

	defer func() {
		if e.pending != nil && e.txs.Len() < TransactionsToStore {
			e.pending.Signal()
			e.pending = nil
		}
	}()

	if e.start == terminationMark {
		return errors.InvalidStateError.New("AlreadyTerminated")
	}
	if to < e.start {
		if to+1 < e.start {
			err := e.rebaseInLock(to+1, -1, nil)
			if errors.Is(err, ErrAfterLastBlock) {
				e.txs.Init()
				e.start = to+1
				e.end = e.start+1
				e.txs.PushBack(ErrAfterLastBlock)
			} else {
				return err
			}
		}
		return nil
	}
	if to >= e.end {
		return errors.InvalidStateError.Errorf("NotReachedYet(to=%d,end=%d)", to, e.end)
	}
	accLen := e.acc.Len()
	if accLen < e.start {
		return errors.InvalidStateError.Errorf("AccumulatorIsLackOfTx(len=%d,from=%d)", accLen, e.start)
	}
	for txe := e.txs.Front(); txe!= nil && e.start <= to ; e.start += 1 {
		btx := txe.Value.(*BlockTransaction)
		if accLen == btx.Height {
			if err := e.acc.Add(btx.BlockHash); err != nil {
				return err
			}
			accLen += 1
		}
		txe, _ = txe.Next(), e.txs.Remove(txe)
	}
	return nil
}

func (e *Executor) rebaseInLock(from, to int64, txs []*BlockTransaction) error {
	chn, err := e.bc.Rebase(from, to, txs)
	if err != nil {
		e.log.Errorf("Failure in BlockConverter.Rebase err=%+v", err)
		return err
	}
	if len(txs) > 0 && e.acc.Len() > from {
		if err := e.acc.SetLen(from); err != nil {
			return err
		}
	}
	e.txs.Init()
	e.start = from
	e.end = from
	e.consumer = new(int)
	if e.pending != nil {
		e.pending.Signal()
		e.pending = nil
	}
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
	if err := e.rebaseInLock(from, -1, txs); err != nil {
		return err
	}
	e.cancelWaiterInLock()
	return nil
}

// addTransaction add transaction from block converter
// it's called by consumeBlocks.
func (e *Executor) addTransaction(id consumeID, tx *BlockTransaction) bool {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.consumer == id && e.txs.Len() >= TransactionsToStore {
		e.pending = sync.NewCond(&e.lock)
		e.pending.Wait()
	}

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

func (e *Executor) notifyEnd(id consumeID, err error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.consumer == id {
		e.consumer = nil
		txe := e.txs.Back()
		if txe != nil {
			if _, ok := txe.Value.(error); ok {
				return
			}
		}

		e.txs.PushBack(err)
		e.end += 1
		if w := e.waiter; w != nil {
			w.notifyCanceled(err)
			e.waiter = nil
		}
		return
	}
}

// consumeBlocks consume all data from the channel
// until it's closed. So, if it will not send anything
// through the channel, it should be closed.
func (e *Executor) consumeBlocks(id consumeID, chn <-chan interface{}) {
	var err error
	e.log.Debugf("consumeBlocks START chn=%+v", chn)
	defer func() {
		e.log.Debugf("consumeBlocks END chn=%+v err=%+v", chn, err)
	}()
	for true {
		tx, ok := <-chn
		if !ok {
			if err == nil {
				e.log.Debugf("consumeBlocks NOTIFY END chn=%+v", chn)
				e.notifyEnd(id, ErrAfterLastBlock)
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
			e.log.Errorf("consumeBlocks ERROR chn=%+v err=%+v", chn, err)
			e.notifyEnd(id, obj)
			return
		}
	}
}

func (e *Executor) Start() error {
	// nothing to do
	return nil
}

func (e *Executor) Term() {
	e.bc.Term()

	e.lock.Lock()
	defer e.lock.Unlock()
	e.start = terminationMark
	e.end = terminationMark
	e.consumer = nil
	e.txs.Init()
	e.cancelWaiterInLock()
}

func (e *Executor) GetMerkleHeader(height int64) (*hexary.MerkleHeader, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	return e.getMerkleHeaderInLock(height)
}

func (e *Executor) getMerkleHeaderInLock(height int64) (*hexary.MerkleHeader, error) {
	accLen := e.acc.Len()
	if accLen > height {
		if err := e.acc.SetLen(height); err != nil {
			return nil, err
		}
	} else if accLen < height {
		if accLen < e.start || height > e.end {
			return nil, errors.InvalidStateError.Errorf("FailToMakeMerkle(start=%d,end=%d,current=%d,target=%d)",
				e.start, e.end, accLen, height)
		}
		for txe := e.txs.Front(); txe!= nil ; txe = txe.Next() {
			btx := txe.Value.(*BlockTransaction)
			if accLen == btx.Height {
				if err := e.acc.Add(btx.BlockHash); err != nil {
					return nil, err
				}
				accLen += 1
				if accLen == height {
					break
				}
			}
		}
	}
	if accLen = e.acc.Len() ; accLen != height {
		return nil, errors.InvalidStateError.Errorf("FailToBuildMerkle(size=%d,height=%d)",
			accLen, height)
	}
	return e.acc.GetMerkleHeader(), nil
}

func (e *Executor) FinalizeBlocks(height int64) (*hexary.MerkleHeader, *blockv0.BlockVoteList, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if size := e.acc.Len() ; size != height+1 {
		return nil, nil, errors.InvalidStateError.Errorf("InvalidAccumulatorState(height=%d,size=%d)",
			height, size)
	}
	mh, err :=  e.acc.Finalize()
	if err != nil {
		return nil, nil, err
	}
	votes, err := e.bc.GetBlockVotes(height)
	if err != nil {
		return nil, nil, err
	}
	return mh, votes, nil
}

func newAccumulator(rdb, idb db.Database) (hexary.Accumulator, error) {
	treeBucket, err := rdb.GetBucket(icdb.BlockMerkle)
	if err != nil {
		return nil, err
	}
	tmpBucket, err := idb.GetBucket(icdb.BlockMerkle)
	if err != nil {
		return nil, err
	}
	return hexary.NewAccumulator(treeBucket, tmpBucket, "")
}

func NewExecutorWithBC(rdb, idb db.Database, logger log.Logger, bc IBlockConverter) (*Executor, error) {
	chainBucket, err := idb.GetBucket(db.ChainProperty)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInBucket(bucket=ChainProperty)")
	}

	acc, err := newAccumulator(rdb, idb)
	if err != nil {
		return nil, errors.Wrap(err, "FailInAccumulator")
	}

	ex := &Executor{
		rdb: rdb,
		idb: idb,
		log: logger,

		chainBucket: chainBucket,

		bc: bc,

		acc:   acc,
		start: math.MaxInt64,
		end:   math.MaxInt64,
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
	store, err := lcstore.OpenStore(cfg.StoreURI, cfg.MaxRPS)
	if err != nil {
		return nil, err
	}
	cs := lcstore.NewForwardCache(store, logger, &cfg.CacheConfig)
	cs.SetReceiptParameter(rdb, module.LatestRevision)
	bc, err := NewBlockConverter(chain, cfg.Platform, cfg.ProxyMgr, cs, cfg.BaseDir)
	if err != nil {
		return nil, err
	}

	return NewExecutorWithBC(rdb, idb, logger, bc)
}
