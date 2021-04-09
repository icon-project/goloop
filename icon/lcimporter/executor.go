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
	"sync"

	"github.com/icon-project/goloop/common"
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
)

type GetBlockTxCallback func([]*BlockTransaction, error)
type Canceler func()

type Executor struct {
	cdb db.Database
	rdb db.Database
	log log.Logger

	chainBucket db.Bucket

	lock    sync.Mutex
	txs     list.List
	offset  int64
	next    int64
	getters map[int64][]*txGetter

	bc *BlockConverter
}

// ProposeTransactions propose transactions for blocks to be consensus
// after finalized.
func (e *Executor) ProposeTransactions() ([]*BlockTransaction, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	var txs []*BlockTransaction
	cnt := 0
	for itr := e.txs.Front(); itr != nil && cnt < 300 ; itr = itr.Next() {
		tx := itr.Value.(*BlockTransaction)
		txs = append(txs, tx)
		cnt += 1
	}
	return txs, nil
}

type txGetter struct {
	ex   *Executor
	from int64
	to   int64
	cb   GetBlockTxCallback
}

func (g *txGetter) cancel() {
	g.ex.removeGetter(g.to)
}

func (g *txGetter) OnAdded() {
	if txs, err := g.ex.getTransactions(g.from, g.to); err != nil {
		g.cb(nil, err)
	} else {
		g.cb(txs, nil)
	}
}

// GetTransactions get already processed transactions in the range.
func (e *Executor) GetTransactions(from, to int64, callback GetBlockTxCallback) (Canceler, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if to < from {
		return nil, errors.IllegalArgumentError.Errorf(
			"InvalidRequest(from=%d,to=%d)",
			from,
			to,
		)
	}
	if from != e.offset {
		return nil, errors.InvalidStateError.Errorf(
			"GetFinalizedTransactions(from=%d,offset=%d)",
			from,
			e.offset,
		)
	}
	if to < e.next {
		if txs, err := e.getTransactionsInLock(from, to); err != nil {
			return nil, err
		} else {
			go callback(txs, nil)
			canceler := func() {
				return
			}
			return canceler, nil
		}
	} else {
	}

	return nil, nil
}

func (e *Executor) getTransactions(from, to int64) ([]*BlockTransaction, error) {
	e.lock.Lock()
	defer e.lock.Unlock()

	return e.getTransactionsInLock(from, to)
}

func (e *Executor) getTransactionsInLock(from, to int64) ([]*BlockTransaction, error) {
	if from != e.offset {
		return nil, errors.InvalidStateError.Errorf(
			"GetFinalizedTransactions(from=%d,offset=%d)",
			from,
			e.offset,
		)
	}
	cnt := int(to-from+1)
	txs := make([]*BlockTransaction, 0, cnt)
	for itr := e.txs.Front(); itr != nil ; itr = itr.Next() {
		tx := itr.Value.(*BlockTransaction)
		if tx.Height <= to {
			txs = append(txs, tx)
		} else {
			break
		}
	}
	if len(txs) != cnt {
		return nil, errors.InvalidStateError.New("WeiredTransactions")
	}
	return txs, nil
}

func (e *Executor) removeGetter(h int64) {
	e.lock.Lock()
	defer e.lock.Unlock()

	delete(e.getters, h)
}

// FinalizeTransactions finalize transactions by specific range.
func (e *Executor) FinalizeTransactions(to int64) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	if to < e.offset || to >= e.next {
		return errors.InvalidStateError.Errorf("FailToFinalize(height=%d,offset=%d,next=%d)",
			to, e.offset, e.next)
	}
	itr := e.txs.Front()
	for itr.Value.(*BlockTransaction).Height <= to {
		next := itr.Next()
		e.txs.Remove(itr)
		itr = next
	}
	e.offset = to+1
	return nil
}

// SyncTransactions sync transactions
func (e *Executor) SyncTransactions([]*BlockTransaction) error {
	return errors.ErrUnsupported
}

func (e *Executor) setNextBlockHeight(h int64) error {
	return e.chainBucket.Set([]byte(KeyNextBlockHeight), codec.BC.MustMarshalToBytes(h))
}

func (e *Executor) getNextBlockHeight() int64 {
	if bs, err := e.chainBucket.Get([]byte(KeyNextBlockHeight)); err == nil {
		var height int64
		codec.BC.MustUnmarshalFromBytes(bs, &height)
		return height
	} else {
		return 0
	}
}

func (e *Executor) addTransaction(tx *BlockTransaction) bool {
	locker := common.LockForAutoCall(&e.lock)
	defer locker.Unlock()

	if e.next == tx.Height {
		e.txs.PushBack(tx)
		e.next += 1
		if getters, ok := e.getters[tx.Height]; ok {
			delete(e.getters, tx.Height)
			locker.CallAfterUnlock(func() {
				for _, getter := range getters {
					getter.OnAdded()
				}
			})
		}
		return true
	}
	return false
}

func (e *Executor) consumeBlocks(chn <-chan interface{}) {
	for true {
		tx, ok := <- chn
		if !ok {
			return
		}
		switch obj := tx.(type) {
		case *BlockTransaction:
			if ok := e.addTransaction(obj); !ok {
				return
			}
		default:
			return
		}
	}
}

func (e *Executor) Start() error {
	e.lock.Lock()
	defer e.lock.Unlock()

	next := e.getNextBlockHeight()
	if chn, err := e.bc.Start(e.next, 0); err != nil {
		return err
	} else {
		e.next = next
		e.offset = next
		go e.consumeBlocks(chn)
	}

	return nil
}

func NewExecutor(chain module.Chain, dbase db.Database, cfg *Config) (*Executor, error) {
	logger := chain.Logger()
	cdb := chain.Database()
	chainBucket, err := cdb.GetBucket(db.ChainProperty)
	if err != nil {
		return nil, errors.Wrap(err, "FailureInBucket(bucket=ChainProperty)")
	}

	// build converter
	rdb := cache.AttachManager(dbase, "", 0, 0)
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

	ex := &Executor{
		rdb: rdb,
		cdb: cdb,
		log: logger,

		chainBucket: chainBucket,

		getters: make(map[int64][]*txGetter),

		bc: bc,
	}
	ex.txs.Init()
	return ex, nil
}
