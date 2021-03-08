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

package main

import (
	"container/list"
	"sync"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/blockv0/lcstore"
	"github.com/icon-project/goloop/module"
)

type blockTask struct {
	height int64
	chn    chan interface{}
}

const (
	MaxTrials = 5
)

func (t *blockTask) Do(cs *CacheStore) {
	trial := 0
	for {
		block, err := cs.Store.GetBlockByHeight(int(t.height))
		if err == nil {
			t.chn <- block
			return
		} else {
			trial += 1
			if trial >= MaxTrials {
				t.chn <- err
				return
			} else {
				log.Warnf("Re-try BLOCK for height=%d trial=%d", t.height, trial)
			}
		}
	}
}

type receiptTask struct {
	id  []byte
	chn chan interface{}
}

func (t *receiptTask) Do(cs *CacheStore) {
	trial := 0
	for {
		receipt, err := cs.Store.GetReceiptByTransaction(t.id)
		if err == nil {
			t.chn <- receipt
			return
		} else {
			trial += 1
			if trial >= MaxTrials {
				t.chn <- err
				return
			} else {
				log.Warnf("Re-try RECEIPT for tx=%#x trial=%d", t.id, trial)
			}
		}
	}
}

type task interface {
	Do(cs *CacheStore)
}

type CacheStore struct {
	*lcstore.Store
	lock sync.Mutex
	log  log.Logger

	blockWorkers      int
	maxBlocks         int
	receiptWorkers    int
	maxReceiptWorkers int

	blockTasks   list.List
	blockInfo    map[int64]*blockTask
	receiptTasks list.List
	receiptInfo  map[string]*receiptTask
}

func (cs *CacheStore) receiptLoop() {
	fetchTask := func() *receiptTask {
		cs.lock.Lock()
		defer cs.lock.Unlock()
		e := cs.receiptTasks.Front()
		if e == nil {
			cs.receiptWorkers -= 1
			return nil
		} else {
			cs.receiptTasks.Remove(e)
			return e.Value.(*receiptTask)
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

func (cs *CacheStore) blockLoop() {
	fetchTask := func() *blockTask {
		cs.lock.Lock()
		defer cs.lock.Unlock()
		e := cs.blockTasks.Front()
		if e == nil {
			cs.blockWorkers -= 1
			return nil
		} else {
			cs.blockTasks.Remove(e)
			return e.Value.(*blockTask)
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

func (cs *CacheStore) getBlockTask(height int64) *blockTask {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	if t, ok := cs.blockInfo[height]; ok {
		delete(cs.blockInfo, height)
		return t
	} else {
		return nil
	}
}

func (cs *CacheStore) scheduleReceipt(id []byte) {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	ids := string(id)
	if t, ok := cs.receiptInfo[ids]; !ok {
		t = &receiptTask{
			id:  id,
			chn: make(chan interface{}, 1),
		}
		cs.receiptTasks.PushBack(t)
		cs.receiptInfo[ids] = t
		if cs.receiptWorkers < cs.maxReceiptWorkers {
			cs.receiptWorkers += 1
			go cs.receiptLoop()
		}
	}
}

func (cs *CacheStore) scheduleBlock(height int64) {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	if t, ok := cs.blockInfo[height]; !ok {
		t = &blockTask{
			height: height,
			chn:    make(chan interface{}, 1),
		}
		cs.blockTasks.PushBack(t)
		cs.blockInfo[height] = t
		if cs.blockWorkers < 2 {
			cs.blockWorkers += 1
			go cs.blockLoop()
		}
	}
}

func (cs *CacheStore) scheduleFollowings(b blockv0.Block) {
	txs := b.NormalTransactions()
	for _, tx := range txs {
		cs.scheduleReceipt(tx.ID())
	}
	height := int(b.Height())
	for h := height + 1; h <= height+cs.maxBlocks; h += 1 {
		cs.scheduleBlock(int64(h))
	}
}

func (cs *CacheStore) GetBlockByHeight(height int) (blockv0.Block, error) {
	if bt := cs.getBlockTask(int64(height)); bt != nil {
		r := <-bt.chn
		close(bt.chn)
		switch obj := r.(type) {
		case blockv0.Block:
			cs.scheduleFollowings(obj)
			return obj, nil
		case error:
			return nil, obj
		default:
			panic("UnknownType")
		}
	}
	if blk, err := cs.Store.GetBlockByHeight(height); err != nil {
		return nil, err
	} else {
		cs.scheduleFollowings(blk)
		return blk, nil
	}
}

func (cs *CacheStore) getReceiptTask(id []byte) *receiptTask {
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

func (cs *CacheStore) GetReceiptByTransaction(id []byte) (module.Receipt, error) {
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
	trial := 0
	for {
		if rct, err := cs.Store.GetReceiptByTransaction(id); err == nil {
			return rct, nil
		} else {
			if trial >= MaxTrials {
				return nil, err
			} else {
				trial += 1
				cs.log.Debugf("Try RECEIPT tid=%#x again err=%+v", id, err)
			}
		}
	}
	return cs.Store.GetReceiptByTransaction(id)
}

func NewCacheStore(logger log.Logger, store *lcstore.Store) *CacheStore {
	cs := &CacheStore{
		Store:             store,
		log:               logger,
		maxBlocks:         32,
		maxReceiptWorkers: 64,
		blockInfo:         make(map[int64]*blockTask),
		receiptInfo:       make(map[string]*receiptTask),
	}
	cs.receiptTasks.Init()
	cs.blockTasks.Init()
	return cs
}
