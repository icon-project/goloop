package service

import (
	"github.com/icon-project/goloop/common/log"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	configDefaultTxSliceCapacity = 1024
	configMaxTxCount             = 1500
)

type Monitor interface {
	OnDropTx(n int, user bool)
	OnAddTx(n int, user bool)
	OnRemoveTx(n int, user bool)
	OnCommit(id []byte, ts time.Time, d time.Duration)
}

type TransactionPool struct {
	nid  int
	size int
	txdb db.Bucket

	list *transactionList

	mutex sync.Mutex

	monitor Monitor
	log     log.Logger
}

func NewTransactionPool(nid int, size int, txdb db.Bucket, m Monitor, log log.Logger) *TransactionPool {
	pool := &TransactionPool{
		nid:     nid,
		size:    size,
		txdb:    txdb,
		list:    newTransactionList(),
		monitor: m,
		log:     log,
	}
	return pool
}

func (tp *TransactionPool) RemoveOldTXs(bts int64) {
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	iter := tp.list.Front()
	for iter != nil {
		next := iter.Next()
		tx := iter.Value()
		if tx.Timestamp() <= bts {
			tp.list.Remove(iter)
			direct := iter.ts != 0
			if iter.err == nil {
				tp.log.Debugf("DROP TX: id=0x%x reason=%v", tx.ID(), iter.err)
			} else {
				tp.log.Debugf("DROP TX: id=0x%x timeout %d <= %d",
					tx.ID(), tx.Timestamp(), bts)
			}
			tp.monitor.OnDropTx(len(tx.Bytes()), direct)
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

	txs := make([]*txElement, 0, configDefaultTxSliceCapacity)
	poolSize := tp.list.Len()
	txSize := int(0)
	for e := tp.list.Front(); e != nil && txSize < maxBytes && len(txs) < maxCount; e = e.Next() {
		tx := e.Value()
		bs := tx.Bytes()
		if txSize+len(bs) > maxBytes {
			break
		}
		txSize += len(bs)
		txs = append(txs, e)
	}
	tp.mutex.Unlock()

	// make list of valid transactions
	validTxs := make([]module.Transaction, len(txs))
	valNum := 0
	invalidNum := 0
	txSize = 0
	for _, e := range txs {
		tx := e.Value()
		// TODO need to check transaction in parent transitions.
		if v, err := tp.txdb.Get(tx.ID()); err == nil && v != nil {
			e.err = errors.InvalidStateError.New("Already processed")
			txs[invalidNum] = e
			invalidNum += 1
			continue
		}
		if err := CheckTxTimestamp(wc, tx); err != nil {
			if e.err == nil {
				e.err = err
				tp.log.Debugf("CHECKTS FAIL: id=%#x reason=%v",
					tx.ID(), err)
			}
			txs[invalidNum] = e
			invalidNum += 1
			continue
		}
		if err := tx.PreValidate(wc, true); err != nil {
			// If returned error is critical(not usable in the future)
			// then it should removed from the pool
			// Otherwise, it remains in the pool
			if e.err == nil {
				e.err = err
				tp.log.Debugf("PREVALIDATE FAIL: id=%#x reason=%v",
					tx.ID(), err)
			}
			if state.NotEnoughStepError.Equals(err) {
				txs[invalidNum] = e
				invalidNum += 1
			}
			continue
		}
		validTxs[valNum] = tx
		txSize += len(tx.Bytes())
		valNum++
	}

	if invalidNum > 0 {
		go func(txs []*txElement) {
			tp.mutex.Lock()
			defer tp.mutex.Unlock()
			for _, e := range txs {
				if tp.list.Remove(e) {
					tx := e.Value()
					direct := e.ts != 0
					if e.err != nil {
						tp.log.Debugf("DROP TX: id=0x%x reason=%v",
							tx.ID(), e.err)
					}
					tp.monitor.OnDropTx(len(tx.Bytes()), direct)
				}
			}
		}(txs[0:invalidNum])
	}

	tp.log.Infof("TransactionPool.Candidate collected=%d removed=%d poolsize=%d duration=%s",
		valNum, invalidNum, poolSize, time.Now().Sub(startTS))

	return validTxs[:valNum], txSize
}

/*
	return nil if tx is nil or tx is added to pool
	return ErrTransactionPoolOverFlow if pool is full
	return ErrDuplicateTransaction if tx exists in pool
	return ErrExpiredTransaction if Timestamp of tx is expired
*/
func (tp *TransactionPool) Add(tx transaction.Transaction, direct bool) error {
	if tx == nil {
		return nil
	}
	if !tx.ValidateNetwork(tp.nid) {
		return errors.InvalidNetworkError.Errorf("Invalid Network ID(%d)", tp.nid)
	}
	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	if tp.list.Len() >= tp.size {
		return ErrTransactionPoolOverFlow
	}

	err := tp.list.Add(tx, direct)
	if err == nil {
		tp.monitor.OnAddTx(len(tx.Bytes()), direct)
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

	now := time.Now()
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
		tp.monitor.OnCommit(txs.Hash(), now, duration/time.Duration(count))
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
