package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	configTxPoolSize             = 5000
	configDefaultTxSliceCapacity = 1024
	configMaxTxCount             = 1500
)

type TransactionPool struct {
	txdb db.Bucket

	list *transactionList

	mutex sync.Mutex

	metric context.Context
}

func NewTransactionPool(txdb db.Bucket, ctx context.Context) *TransactionPool {
	pool := &TransactionPool{
		txdb:   txdb,
		list:   newTransactionList(),
		metric: ctx,
	}
	return pool
}

func (tp *TransactionPool) RemoveOldTXs(bts int64) {
	if !transaction.ConfigOnCheckingTimestamp {
		return
	}
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	iter := tp.list.Front()
	for iter != nil {
		next := iter.Next()
		tx := iter.Value()
		if tx.Timestamp() <= bts {
			tp.list.Remove(iter)
			metric.RecordOnDropTx(tp.metric, len(tx.Bytes()))
		}
		iter = next
	}
}

// It returns all candidates for a negative integer n.
func (tp *TransactionPool) Candidate(wc state.WorldContext, maxBytes int, maxCount int) ([]module.Transaction, int) {
	tp.mutex.Lock()
	if tp.list.Len() == 0 {
		tp.mutex.Unlock()
		return []module.Transaction{}, 0
	}

	startTS := time.Now()

	if maxBytes <= 0 {
		maxBytes = ConfigMaxTxBytesInABlock
	}
	if maxCount <= 0 {
		maxCount = configMaxTxCount
	}

	txs := make([]transaction.Transaction, 0, configDefaultTxSliceCapacity)
	poolSize := tp.list.Len()
	txSize := int(0)
	for e := tp.list.Front(); e != nil && txSize < maxBytes && len(txs) < maxCount; e = e.Next() {
		tx := e.Value()
		bs := tx.Bytes()
		if txSize+len(bs) > maxBytes {
			break
		}
		txSize += len(bs)
		txs = append(txs, tx)
	}
	tp.mutex.Unlock()

	// make list of valid transactions
	validTxs := make([]module.Transaction, len(txs))
	valNum := 0
	invalidNum := 0
	txSize = 0
	for _, tx := range txs {
		// TODO need to check transaction in parent transitions.
		if v, err := tp.txdb.Get(tx.ID()); err == nil && v != nil {
			txs[invalidNum] = tx
			invalidNum += 1
			continue
		}
		if err := tx.PreValidate(wc, true); err != nil {
			// If returned error is critical(not usable in the future)
			// then it should removed from the pool
			// Otherwise, it remains in the pool
			if err == state.ErrTimeOut || err == state.ErrNotEnoughStep {
				txs[invalidNum] = tx
				invalidNum += 1
			}
			continue
		}
		validTxs[valNum] = tx
		txSize += len(tx.Bytes())
		valNum++
	}

	if invalidNum > 0 {
		go func(txs []transaction.Transaction) {
			tp.mutex.Lock()
			defer tp.mutex.Unlock()
			for _, tx := range txs {
				if tx != nil {
					tp.list.RemoveTx(tx)
					metric.RecordOnDropTx(tp.metric, len(tx.Bytes()))
				}
			}
		}(txs[0:invalidNum])
	}

	log.Printf("TransactionPool.Candidate collected=%d removed=%d poolsize=%d duration=%s",
		valNum, invalidNum, poolSize, time.Now().Sub(startTS))

	return validTxs[:valNum], txSize
}

/*
	return nil if tx is nil or tx is added to pool
	return ErrTransactionPoolOverFlow if pool is full
	return ErrDuplicateTransaction if tx exists in pool
	return ErrExpiredTransaction if Timestamp of tx is expired
*/
func (tp *TransactionPool) Add(tx transaction.Transaction) error {
	if tx == nil {
		return nil
	}
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	if tp.list.Len() >= configTxPoolSize {
		return ErrTransactionPoolOverFlow
	}

	err := tp.list.Add(tx)
	if err == nil {
		metric.RecordOnAddTx(tp.metric, len(tx.Bytes()))
	}
	return err
}

// removeList remove transactions when transactions are finalized.
func (tp *TransactionPool) RemoveList(txs module.TransactionList) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	if tp.list.Len() == 0 {
		return
	}

	for i := txs.Iterator(); i.Has(); i.Next() {
		t, _, err := i.Get()
		if err != nil {
			log.Printf("Failed to get transaction from iterator\n")
			continue
		}
		tp.list.RemoveTx(t)
		metric.RecordOnRemoveTx(tp.metric, len(t.Bytes()))
	}
}
