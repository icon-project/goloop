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

package lcstore

import (
	"container/list"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/module"
)

const (
	MaxTrials        = 5
	DelayBeforeRetry = 500 * time.Millisecond
)

type CacheConfig struct {
	MaxWorkers int `json:"max_workers"`
	MaxBlocks  int `json:"max_blocks"`
}

type blockTask struct {
	height int64
	chn    chan interface{}
}

func (t *blockTask) Do(cs *ForwardCache) {
	block, err := cs.doGetBlockByHeight(int(t.height))
	if err != nil {
		t.chn <- err
	} else {
		t.chn <- block
	}
}

type receiptTask struct {
	id  []byte
	chn chan interface{}
}

func (t *receiptTask) Do(cs *ForwardCache) {
	receipt, err := cs.doGetReceipt(t.id)
	if err != nil {
		t.chn <- err
	} else {
		t.chn <- receipt
	}
}

type task interface {
	Do(cs *ForwardCache)
}

type ForwardCache struct {
	*Store
	lock sync.Mutex
	log  log.Logger

	workers int
	config  CacheConfig

	tasks       list.List
	blockInfo   map[int64]*blockTask
	receiptInfo map[string]*receiptTask
}

func (cs *ForwardCache) workLoop() {
	fetchTask := func() task {
		cs.lock.Lock()
		defer cs.lock.Unlock()
		e := cs.tasks.Front()
		if e == nil {
			cs.workers -= 1
			return nil
		} else {
			cs.tasks.Remove(e)
			return e.Value.(task)
		}
	}

	for {
		t := fetchTask()
		if t != nil {
			t.Do(cs)
		} else {
			break
		}
	}
}

func (cs *ForwardCache) getBlockTask(height int64) *blockTask {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	if t, ok := cs.blockInfo[height]; ok {
		delete(cs.blockInfo, height)
		return t
	} else {
		return nil
	}
}

func (cs *ForwardCache) addWorkerInLock() {
	if cs.workers < cs.config.MaxWorkers {
		cs.workers += 1
		go cs.workLoop()
	}
}

func (cs *ForwardCache) scheduleReceiptInLock(id []byte) {
	ids := string(id)
	if t, ok := cs.receiptInfo[ids]; !ok {
		t = &receiptTask{
			id:  id,
			chn: make(chan interface{}, 1),
		}
		cs.tasks.PushBack(t)
		cs.receiptInfo[ids] = t
		cs.addWorkerInLock()
	}
}

func (cs *ForwardCache) scheduleBlockInLock(height int64) {
	if t, ok := cs.blockInfo[height]; !ok {
		t = &blockTask{
			height: height,
			chn:    make(chan interface{}, 1),
		}
		cs.tasks.PushBack(t)
		cs.blockInfo[height] = t
		cs.addWorkerInLock()
	}
}

func (cs *ForwardCache) scheduleFollowings(b blockv0.Block) {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	txs := b.NormalTransactions()
	for _, tx := range txs {
		cs.scheduleReceiptInLock(tx.ID())
	}
	for h := b.Height() + 1; len(cs.blockInfo) < cs.config.MaxBlocks; h += 1 {
		cs.scheduleBlockInLock(int64(h))
	}
}

func (cs *ForwardCache) doGetBlockByHeight(height int) (blockv0.Block, error) {
	trial := 1
	for {
		block, err := cs.Store.GetBlockByHeight(height)
		if err == nil {
			cs.scheduleFollowings(block)
			return block, nil
		} else if errors.NotFoundError.Equals(err) {
			return nil, err
		} else {
			trial += 1
			if trial > MaxTrials {
				cs.log.Warnf("BLOCK failed height=%d err=%+v",
					height, err)
				return nil, err
			} else {
				cs.log.Debugf("BLOCK retry height=%d trial=[%d/%d] err=%v",
					height, trial, MaxTrials, err)
				time.Sleep(DelayBeforeRetry)
			}
		}
	}
}

const GetBlockMaxDuration = time.Minute*2
const GetBlockRetryDelay = time.Second*2

func (cs *ForwardCache) GetBlockByHeight(height int) (blockv0.Block, error) {
	ts := time.Now()
	if bt := cs.getBlockTask(int64(height)); bt != nil {
		r := <-bt.chn
		close(bt.chn)
		switch obj := r.(type) {
		case blockv0.Block:
			cs.scheduleFollowings(obj)
			return obj, nil
		case error:
			if errors.NotFoundError.Equals(obj) {
				break
			}
			return nil, obj
		default:
			panic("UnknownType")
		}
	}
	for {
		blk, err := cs.doGetBlockByHeight(height)
		if err == nil {
			return blk, nil
		}
		dur := time.Now().Sub(ts)
		if dur >= GetBlockMaxDuration {
			return blk, err
		}
		if !errors.NotFoundError.Equals(err) {
			return nil, err
		}
		cs.log.Debugf("GetBlockByHeight(height=%d) elapsed=%s TRY AGAIN", height, dur)
		time.Sleep(GetBlockRetryDelay)
	}
}

func (cs *ForwardCache) getReceiptTask(id []byte) *receiptTask {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	ids := string(id)
	if rt, ok := cs.receiptInfo[ids]; ok {
		delete(cs.receiptInfo, ids)
		return rt
	} else {
		return nil
	}
}

func (cs *ForwardCache) doGetReceipt(id []byte) (module.Receipt, error) {
	trial := 0
	for {
		if rct, err := cs.Store.GetReceipt(id); err == nil {
			return rct, nil
		} else {
			trial += 1
			if trial >= MaxTrials {
				cs.log.Warnf("RECEIPT failure id=%#x", id)
				return nil, err
			} else {
				cs.log.Debugf("RECEIPT retry tid=%#x trial=%d err=%+v", id, trial, err)
				time.Sleep(DelayBeforeRetry)
			}
		}
	}
}

func (cs *ForwardCache) GetReceipt(id []byte) (module.Receipt, error) {
	if rt := cs.getReceiptTask(id); rt != nil {
		r := <-rt.chn
		close(rt.chn)
		switch obj := r.(type) {
		case module.Receipt:
			return obj, nil
		case error:
			return nil, obj
		default:
			panic("UnknownType")
		}
	}
	return cs.doGetReceipt(id)
}

func (cs *ForwardCache) GetTPS() float32 {
	return cs.Database.GetTPS()
}

var defaultCacheConfig = CacheConfig{
	MaxBlocks:  32,
	MaxWorkers: 8,
}

func NewForwardCache(store *Store, logger log.Logger, config *CacheConfig) *ForwardCache {
	if config == nil {
		config = &defaultCacheConfig
	}
	cs := &ForwardCache{
		Store:       store,
		log:         logger,
		config:      *config,
		blockInfo:   make(map[int64]*blockTask),
		receiptInfo: make(map[string]*receiptTask),
	}
	cs.tasks.Init()
	return cs
}
