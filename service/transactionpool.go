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

package service

import (
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	configDefaultMaxTxBytesInABlock = 1024 * 1024
	configDefaultTxSliceCapacity    = 1024
	configDefaultMaxTxCount         = 1500
)

type Monitor interface {
	OnDropTx(n int, user bool)
	OnAddTx(n int, user bool)
	OnRemoveTx(n int, user bool)
	OnCommit(id []byte, ts time.Time, d time.Duration)
}

type TxWaiterManager interface {
	OnTxDrops([]TxDrop)
}

type dummyTxWaiterManager struct{}

func (d dummyTxWaiterManager) OnTxDrops([]TxDrop) {
	// do nothing
}

type PoolCapacityMonitor interface {
	OnPoolCapacityUpdated(group module.TransactionGroup, size, used int)
}

type dummyPoolCapacityMonitor struct{}

func (m dummyPoolCapacityMonitor) OnPoolCapacityUpdated(group module.TransactionGroup, size, used int) {
	// do nothing
}

type TransactionPool struct {
	group module.TransactionGroup

	size int
	tim  TXIDManager

	list *transactionList

	mutex sync.Mutex

	txm     TxWaiterManager
	monitor Monitor
	pcm     PoolCapacityMonitor
	log     log.Logger
}

func NewTransactionPool(group module.TransactionGroup, size int, tim TXIDManager, m Monitor, log log.Logger) *TransactionPool {
	pool := &TransactionPool{
		group:   group,
		size:    size,
		tim:     tim,
		list:    newTransactionList(),
		txm:     dummyTxWaiterManager{},
		monitor: m,
		pcm:     dummyPoolCapacityMonitor{},
		log:     log,
	}
	return pool
}

func (tp *TransactionPool) DropOldTXs(bts int64) {
	lock := common.LockForAutoCall(&tp.mutex)
	defer lock.Unlock()
	// tp.mutex.Lock()
	// defer tp.mutex.Unlock()

	var drops []TxDrop
	iter := tp.list.Front()
	for iter != nil {
		next := iter.Next()
		tx := iter.Value()
		if tx.Timestamp() <= bts {
			tp.list.Remove(iter)
			direct := iter.ts != 0
			if iter.err == nil {
				iter.err = ExpiredTransactionError.Errorf(
					"ExpiredTransaction(diff=%s)", TimestampToDuration(bts-tx.Timestamp()))
			}
			tp.log.Debugf("DROP TX: id=0x%x reason=%v", tx.ID(), iter.err)
			drops = append(drops, TxDrop{tx.ID(), iter.err})
			tp.monitor.OnDropTx(len(tx.Bytes()), direct)
		}
		iter = next
	}
	lock.CallAfterUnlock(func() {
		tp.txm.OnTxDrops(drops)
	})
	// go tp.txm.OnTxDrops(drops)
	tp.pcm.OnPoolCapacityUpdated(tp.group, tp.size, tp.list.Len())
}

// It returns all candidates for a negative integer n.
func (tp *TransactionPool) Candidate(wc state.WorldContext, maxBytes int, maxCount int) (
	[]module.Transaction, int,
) {
	lock := common.Lock(&tp.mutex)
	defer lock.Unlock()

	if tp.list.Len() == 0 {
		return []module.Transaction{}, 0
	}

	startTS := time.Now()

	if maxBytes <= 0 {
		maxBytes = configDefaultMaxTxBytesInABlock
	}
	if maxCount <= 0 {
		maxCount = configDefaultMaxTxCount
	}

	tsr := NewTxTimestampRangeFor(wc, tp.group)
	txs := make([]module.Transaction, 0, configDefaultTxSliceCapacity)
	dropped := make([]*txElement, 0, configDefaultTxSliceCapacity)
	poolSize := tp.list.Len()
	txSize := int(0)
	for e := tp.list.Front(); e != nil && txSize < maxBytes && len(txs) < maxCount; e = e.Next() {
		tx := e.Value()
		if err := tsr.CheckTx(tx); err != nil {
			if ExpiredTransactionError.Equals(err) {
				if e.err == nil {
					e.err = err
				}
				dropped = append(dropped, e)
			}
			continue
		}
		if has, err := tp.tim.HasRecent(tx.ID()); err != nil {
			continue
		} else if has {
			e.err = errors.InvalidStateError.New("AlreadyProcessed")
			dropped = append(dropped, e)
			continue
		}
		if err := tx.PreValidate(wc, true); err != nil {
			if e.err == nil {
				e.err = err
				tp.log.Debugf("PREVALIDATE FAIL: id=%#x from=%s reason=%v",
					tx.ID(), tx.From().String(), err)
			}
			if !transaction.NotEnoughBalanceError.Equals(err) || e.ts == 0 {
				dropped = append(dropped, e)
			}
			continue
		}
		bs := tx.Bytes()
		if txSize+len(bs) > maxBytes {
			break
		}
		txSize += len(bs)
		txs = append(txs, tx)
	}
	lock.Unlock()

	if len(dropped) > 0 {
		go tp.dropTransactions(dropped)
	}

	tp.log.Infof("TransactionPool.Candidate collected=%d removed=%d poolsize=%d duration=%s",
		len(txs), len(dropped), poolSize, time.Since(startTS))

	return txs, txSize
}

func (tp *TransactionPool) CheckTxs(wc state.WorldContext) bool {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	if tp.list.Len() == 0 {
		return false
	}

	t := wc.BlockTimeStamp() - TransactionTimestampThreshold(wc, tp.group)
	for e := tp.list.Front(); e != nil; e = e.Next() {
		tx := e.Value()
		if tx.Timestamp() > t {
			return true
		}
	}
	return false
}

/*
	return nil if tx is nil or tx is added to pool
	return ErrTransactionPoolOverFlow if pool is full
*/
func (tp *TransactionPool) Add(tx transaction.Transaction, direct bool) error {
	if tx == nil {
		return nil
	}
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	if tp.list.Len() >= tp.size {
		return ErrTransactionPoolOverFlow
	}

	err := tp.list.Add(tx, direct)
	if err == nil {
		tp.monitor.OnAddTx(len(tx.Bytes()), direct)
		tp.pcm.OnPoolCapacityUpdated(tp.group, tp.size, tp.list.Len())
	}
	return err
}

// removeList remove transactions when transactions are finalized.
func (tp *TransactionPool) RemoveList(txs module.TransactionList) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	now := time.Now()
	if tp.list.Len() == 0 {
		tp.monitor.OnCommit(txs.Hash(), now, 0)
		return
	}

	var duration time.Duration
	var count int

	for i := txs.Iterator(); i.Has(); i.Next() {
		t, _, err := i.Get()
		if err != nil {
			tp.log.Errorf("Failed to get transaction from iterator\n")
			continue
		}
		if ok, ts := tp.list.RemoveTx(t); ok {
			if ts != 0 {
				duration += now.Sub(time.Unix(0, ts))
				count += 1
			}
			tp.monitor.OnRemoveTx(len(t.Bytes()), ts != 0)
		}
	}

	if count > 0 {
		tp.pcm.OnPoolCapacityUpdated(tp.group, tp.size, tp.list.Len())
		tp.monitor.OnCommit(txs.Hash(), now, duration/time.Duration(count))
	} else {
		tp.monitor.OnCommit(txs.Hash(), now, 0)
	}
}

func (tp *TransactionPool) HasTx(tid []byte) bool {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	return tp.list.HasTx(tid)
}

func (tp *TransactionPool) Size() int {
	return tp.size
}

func (tp *TransactionPool) Used() int {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	return tp.list.Len()
}

func (tp *TransactionPool) SetTxManager(txm TxWaiterManager) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	tp.txm = txm
}

func (tp *TransactionPool) SetPoolCapacityMonitor(pcm PoolCapacityMonitor) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	tp.pcm = pcm
	go tp.pcm.OnPoolCapacityUpdated(tp.group, tp.size, tp.list.Len())
}

func (tp *TransactionPool) GetBloom() *TxBloom {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	return tp.list.GetBloom()
}

func (tp *TransactionPool) dropTransactions(txs []*txElement) {
	lock := common.LockForAutoCall(&tp.mutex)
	defer lock.Unlock()

	var drops []TxDrop
	for _, e := range txs {
		if tp.list.Remove(e) {
			tx := e.Value()
			direct := e.ts != 0
			if e.err == nil {
				tp.log.Panicf("No reason to drop the tx=<%#x>", tx.ID())
			}
			tp.log.Debugf("DROP TX: id=0x%x reason=%v", tx.ID(), e.err)
			drops = append(drops, TxDrop{tx.ID(), e.err})
			tp.monitor.OnDropTx(len(tx.Bytes()), direct)
		}
	}
	lock.CallAfterUnlock(func() {
		tp.txm.OnTxDrops(drops)
	})
}

func (tp *TransactionPool) FilterTransactions(bloom *TxBloom, max int) []module.Transaction {
	txs := make([]module.Transaction, 0, max)
	var invalids []*txElement

	lock := common.Lock(&tp.mutex)
	defer lock.Unlock()

	var working, skip *txBloomElement
	for e := tp.list.Front(); e != nil && len(txs) < max; e = e.Next() {
		if working != e.bloom {
			if e.bloom == skip {
				continue
			}
			if bloom.ContainsAllOf(&e.bloom.bloom) {
				skip = e.bloom
				continue
			} else {
				working = e.bloom
			}
		}
		tx := e.Value()
		id := tx.ID()
		if !bloom.Contains(id) {
			if has, err := tp.tim.HasRecent(id); err == nil && has {
				e.err = errors.InvalidStateError.New("Already processed")
				invalids = append(invalids, e)
				continue
			}
			txs = append(txs, tx)
		}
	}
	go tp.dropTransactions(invalids)
	return txs
}
