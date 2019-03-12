package service

import (
	"container/list"
	"log"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	configTxPoolSize             = 5000
	configDefaultTxSliceCapacity = 1024
)

const (
	txBucketCount = 256
)

func indexAndBucketKeyFromKey(k string) (int, string) {
	return int(k[0]), k[1:]
}

type transactionMap struct {
	buckets []map[string]*list.Element
}

func (m *transactionMap) Get(k string) (*list.Element, bool) {
	idx, bkk := indexAndBucketKeyFromKey(k)
	obj, ok := m.buckets[idx][bkk]
	return obj, ok
}

func (m *transactionMap) Put(k string, v *list.Element) {
	idx, bkk := indexAndBucketKeyFromKey(k)
	m.buckets[idx][bkk] = v
}

func (m *transactionMap) Remove(k string) (*list.Element, bool) {
	idx, bkk := indexAndBucketKeyFromKey(k)
	obj, ok := m.buckets[idx][bkk]
	if ok {
		delete(m.buckets[idx], bkk)
	}
	return obj, ok
}

func (m *transactionMap) Delete(k string) {
	idx, bkk := indexAndBucketKeyFromKey(k)
	delete(m.buckets[idx], bkk)
}

func newTransactionMap() *transactionMap {
	m := new(transactionMap)
	m.buckets = make([]map[string]*list.Element, txBucketCount)
	for i := 0; i < txBucketCount; i++ {
		m.buckets[i] = make(map[string]*list.Element)
	}
	return m
}

type TransactionPool struct {
	txdb db.Bucket

	list  *list.List
	txMap *transactionMap

	mutex sync.Mutex
}

func NewTransactionPool(txdb db.Bucket) *TransactionPool {
	pool := &TransactionPool{
		txdb:  txdb,
		list:  list.New(),
		txMap: newTransactionMap(),
	}
	return pool
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Microsecond)
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
		if iter.Value.(transaction.Transaction).Timestamp() <= bts {
			tp.list.Remove(iter)
		}
		iter = next
	}
}

// It returns all candidates for a negative integer n.
func (tp *TransactionPool) Candidate(wc state.WorldContext, max int) ([]module.Transaction, int) {
	tp.mutex.Lock()
	if tp.list.Len() == 0 {
		tp.mutex.Unlock()
		return []module.Transaction{}, 0
	}

	if max <= 0 {
		max = ConfigMaxTxBytesInABlock
	}

	txs := make([]transaction.Transaction, 0, configDefaultTxSliceCapacity)
	poolSize := tp.list.Len()
	txSize := int(0)
	for e := tp.list.Front(); e != nil && txSize < max; e = e.Next() {
		tx := e.Value.(transaction.Transaction)
		bs := tx.Bytes()
		if txSize+len(bs) > max {
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
					if v, ok := tp.txMap.Remove(string(tx.ID())); ok {
						tp.list.Remove(v)
					}
				}
			}
		}(txs[0:invalidNum])
	}

	log.Printf("TransactionPool.candidate collected=%d removed=%d poolsize=%d",
		valNum, invalidNum, poolSize)

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
	var err error
	txList := tp.list
	if txList.Len() >= configTxPoolSize {
		return ErrTransactionPoolOverFlow
	}

	txid := string(tx.ID())

	// check whether this transaction is already in txPool
	if _, ok := tp.txMap.Get(txid); ok {
		return ErrDuplicateTransaction
	}

	element := txList.PushBack(tx)
	tp.txMap.Put(txid, element)

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

		if v, ok := tp.txMap.Remove(string(t.ID())); ok {
			tp.list.Remove(v)
		}
	}
}
