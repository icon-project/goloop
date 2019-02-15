package tx

import (
	"container/list"
	"log"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

const (
	txPoolSize = 5000
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
	if !ConfigOnCheckingTimestamp {
		return
	}
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	iter := tp.list.Front()
	for iter != nil {
		next := iter.Next()
		if iter.Value.(*transaction).Timestamp() <= bts {
			tp.list.Remove(iter)
		}
		iter = next
	}
}

// It returns all candidates for a negative integer n.
func (tp *TransactionPool) Candidate(wc state.WorldContext, max int) []module.Transaction {
	tp.mutex.Lock()
	if tp.list.Len() == 0 {
		tp.mutex.Unlock()
		return []module.Transaction{}
	}

	poolSize := tp.list.Len()

	// txNum is number of transactions for pre-validate
	txNum := poolSize
	if max >= 0 && txNum > (max*13/10) {
		txNum = max * 13 / 10
	}

	// make list to be used for pre-validate.
	txs := make([]Transaction, txNum)
	txIdx := 0
	for e := tp.list.Front(); txIdx < txNum; e = e.Next() {
		txs[txIdx] = e.Value.(Transaction)
		txIdx++
	}

	tp.mutex.Unlock()

	// txNum is number of transactions actually returned
	if max >= 0 && txNum > max {
		txNum = max
	}

	// make list of valid transactions
	validTxs := make([]module.Transaction, txNum)
	valNum := 0
	for i, tx := range txs {
		// TODO need to check transaction in parent transitions.
		if v, err := tp.txdb.Get(tx.ID()); err == nil && v != nil {
			continue
		}
		if err := tx.PreValidate(wc, true); err != nil {
			// If returned error is critical(not usable in the future)
			// then it should removed from the pool
			// Otherwise, it remains in the pool
			if err != state.ErrTimeOut && err != state.ErrNotEnoughStep {
				txs[i] = nil
			}
			continue
		}
		validTxs[valNum] = tx
		txs[i] = nil
		valNum++
		if valNum == txNum {
			txs = txs[:i+1]
			validTxs = validTxs[:valNum]
			break
		}
	}

	if valNum != len(txs) {
		go func() {
			tp.mutex.Lock()
			defer tp.mutex.Unlock()
			for _, tx := range txs {
				if tx != nil {
					if v, ok := tp.txMap.Remove(string(tx.ID())); ok {
						tp.list.Remove(v)
					}
				}
			}
		}()
	}

	log.Printf("TransactionPool.candidate collected=%d removed=%d poolsize=%d",
		valNum, len(txs)-valNum, poolSize)

	return validTxs[:valNum]
}

/*
	return nil if tx is nil or tx is added to pool
	return ErrTransactionPoolOverFlow if pool is full
	return ErrDuplicateTransaction if tx exists in pool
	return ErrExpiredTransaction if Timestamp of tx is expired
*/
func (tp *TransactionPool) Add(tx Transaction) error {
	if tx == nil {
		return nil
	}
	tp.mutex.Lock()
	defer tp.mutex.Unlock()
	var err error
	txList := tp.list
	if txList.Len() >= txPoolSize {
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
