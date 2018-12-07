package service

import (
	"container/list"
	"log"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

const (
	txPoolSize = 5000
)

type transactionPool struct {
	txdb db.Bucket

	list      *list.List
	txHashMap map[string]*list.Element

	mutex sync.Mutex
}

func NewTransactionPool(txdb db.Bucket) *transactionPool {
	pool := &transactionPool{
		txdb:      txdb,
		list:      list.New(),
		txHashMap: make(map[string]*list.Element),
	}
	return pool
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Microsecond)
}

// TODO: check thread safe below
func (tp *transactionPool) runGc(expired int64) error {
	txList := tp.list

	for iter := txList.Front(); iter != nil; {
		if iter.Value.(*transaction).Timestamp() >= expired {
			break
		}
		tmp := iter.Next()
		delete(tp.txHashMap, string(iter.Value.(*transaction).ID()))
		txList.Remove(iter)
		iter = tmp
	}
	return nil
}

/*
	return nil if tx is nil or tx is added to pool
	return ErrTransactionPoolOverFlow if pool is full
	return ErrDuplicateTransaction if tx exists in pool
	return ErrExpiredTransaction if Timestamp of tx is expired
*/
func (tp *transactionPool) add(tx *transaction) error {
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

	// check whether this transaction is already in txPool
	if _, ok := tp.txHashMap[string(tx.ID())]; ok {
		return ErrDuplicateTransaction
	}

	element := txList.PushBack(tx)
	tp.txHashMap[string(tx.ID())] = element

	return err
}

// It returns all candidates for a negative integer n.
func (tp *transactionPool) candidate(wc WorldContext, max int) []module.Transaction {
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
					if v, ok := tp.txHashMap[string(tx.ID())]; ok {
						tp.list.Remove(v)
						delete(tp.txHashMap, string(tx.ID()))
					}
				}
			}
		}()
	}

	log.Printf("transactionPool.candidate collected=%d removed=%d poolsize=%d",
		valNum, len(txs)-valNum, poolSize)

	return validTxs[:valNum]
}

// removeList remove transactions when transactions are finalized.
func (tp *transactionPool) removeList(txs module.TransactionList) {
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

		id := string(t.ID())
		if v, ok := tp.txHashMap[id]; ok {
			tp.list.Remove(v)
			delete(tp.txHashMap, id)
		}
	}
}
