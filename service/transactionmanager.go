package service

import (
	"sync"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	hashSize = 32
)

type hashValue [hashSize]byte

type TransactionManager struct {
	nid  int
	tsc  *TxTimestampChecker
	log  log.Logger
	lock sync.Mutex

	tim          TXIDManager
	patchTxPool  *TransactionPool
	normalTxPool *TransactionPool

	callback func()

	txWaiters map[hashValue][]chan<- interface{}
}

func (m *TransactionManager) getTxPool(g module.TransactionGroup) *TransactionPool {
	switch g {
	case module.TransactionGroupPatch:
		return m.patchTxPool
	case module.TransactionGroupNormal:
		return m.normalTxPool
	default:
		log.Panicf("Unknown transaction group value=%d", g)
		return nil
	}
}

func (m *TransactionManager) RemoveOldTxByBlockTS(group module.TransactionGroup, bts int64) {
	ts := bts - m.tsc.TransactionThreshold(group)
	m.getTxPool(group).DropOldTXs(ts)
}

func (m *TransactionManager) HasTx(id []byte) bool {
	return m.normalTxPool.HasTx(id) || m.patchTxPool.HasTx(id)
}

func (m *TransactionManager) RemoveTxs(
	g module.TransactionGroup, l module.TransactionList,
) {
	m.getTxPool(g).RemoveList(l)
}

func (m *TransactionManager) Candidate(
	g module.TransactionGroup, wc state.WorldContext, maxBytes, maxCount int,
) ([]module.Transaction, int) {
	return m.getTxPool(g).Candidate(wc, maxBytes, maxCount)
}

func (m *TransactionManager) NotifyFinalized(
	l1 module.TransactionList, r1 module.ReceiptList,
	l2 module.TransactionList, r2 module.ReceiptList,
) {
	m.lock.Lock()
	defer m.lock.Unlock()
	w1 := len(m.txWaiters)
	if w1 > 0 {
		m.notifyFinalizedInLock(l1, r1)
		m.notifyFinalizedInLock(l2, r2)
	}
	w2 := len(m.txWaiters)
	m.log.WithFields(log.Fields{
		"waiters_before": w1,
		"waiters_after":  w2,
	}).Debugf("TM.NotifyFinalized")
	// m.log.Debugf("TM.NotifyFinalized:%5d -> %5d (%5d)", w1, w2, w1-w2)
}

func (m *TransactionManager) notifyFinalizedInLock(l module.TransactionList, r module.ReceiptList) {
	if l == nil || r == nil {
		return
	}
	for itr := l.Iterator(); itr.Has(); itr.Next() {
		tx, idx, err := itr.Get()
		if err != nil {
			m.log.Errorf("Fail to get transaction err=%+v", err)
			return
		}
		rct, err := r.Get(idx)
		if err != nil {
			m.log.Errorf("Fail to get receipt err=%+v", err)
			return
		}
		ws := m.removeWaitersInLock(tx.ID())
		for _, c := range ws {
			c <- rct
			close(c)
		}
	}
}

func (m *TransactionManager) addWaiterInLock(id []byte, rc chan<- interface{}) {
	var hv hashValue
	copy(hv[:], id)
	ws, _ := m.txWaiters[hv]
	m.txWaiters[hv] = append(ws, rc)
}

func (m *TransactionManager) removeWaitersInLock(id []byte) []chan<- interface{} {
	var hv hashValue
	copy(hv[:], id)
	if ws, ok := m.txWaiters[hv]; ok {
		delete(m.txWaiters, hv)
		return ws
	}
	return nil
}

type TxDrop struct {
	ID  []byte
	Err error
}

func (m *TransactionManager) OnTxDrops(drops []TxDrop) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, drop := range drops {
		ws := m.removeWaitersInLock(drop.ID)
		for _, c := range ws {
			c <- drop.Err
			close(c)
		}
	}
}

func (m *TransactionManager) AddAndWait(tx transaction.Transaction) (
	<-chan interface{}, error,
) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if err := m.addInLock(tx, true); err != nil {
		if err != ErrDuplicateTransaction {
			return nil, err
		}
	}
	rc := make(chan interface{}, 1)
	m.addWaiterInLock(tx.ID(), rc)
	return rc, nil
}

func (m *TransactionManager) WaitResult(id []byte) (<-chan interface{}, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.normalTxPool.HasTx(id) || m.patchTxPool.HasTx(id) {
		rc := make(chan interface{}, 1)
		m.addWaiterInLock(id, rc)
		return rc, nil
	}
	if has, err := m.tim.HasRecent(id); err != nil {
		return nil, err
	} else if has {
		return nil, ErrCommittedTransaction
	}
	if has, err := m.tim.HasLocator(id); err != nil {
		return nil, err
	} else if has {
		return nil, ErrCommittedTransaction
	}
	return nil, errors.ErrNotFound
}

func (m *TransactionManager) Add(tx transaction.Transaction, direct bool, verified bool) error {
	if !verified {
		if err := m.VerifyTx(tx); err != nil {
			return err
		}
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.addInLock(tx, direct)
}

func (m *TransactionManager) VerifyTx(tx transaction.Transaction) error {
	if tx == nil {
		return nil
	}
	if !tx.ValidateNetwork(m.nid) {
		return errors.InvalidNetworkError.Errorf(
			"ValidateNetwork(nid=%#x) fail", m.nid)
	}
	if err := tx.Verify(); err != nil {
		return InvalidTransactionError.Wrap(err,
			"Failed to verify transaction")
	}
	return nil
}
func (m *TransactionManager) addInLock(tx transaction.Transaction, direct bool) error {
	if err := m.tim.CheckTXForAdd(tx); err != nil {
		return err
	}

	pool := m.getTxPool(tx.Group())
	if err := pool.Add(tx, direct); err != nil {
		return err
	}
	if m.callback != nil {
		cb := m.callback
		m.callback = nil
		go cb()
	}
	return nil
}

func (m *TransactionManager) Wait(wc state.WorldContext, cb func()) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.patchTxPool.CheckTxs(wc) || m.normalTxPool.CheckTxs(wc) {
		return false
	}
	m.callback = cb
	return true
}

func (m *TransactionManager) GetBloomOf(g module.TransactionGroup) *TxBloom {
	pool := m.getTxPool(g)
	return pool.GetBloom()
}

func (m *TransactionManager) FilterTransactions(g module.TransactionGroup, bloom *TxBloom, max int) []module.Transaction {
	pool := m.getTxPool(g)
	return pool.FilterTransactions(bloom, max)
}

func (m *TransactionManager) Logger() log.Logger {
	return m.log
}

func (m *TransactionManager) SetPoolCapacityMonitor(pcm PoolCapacityMonitor) {
	m.patchTxPool.SetPoolCapacityMonitor(pcm)
	m.normalTxPool.SetPoolCapacityMonitor(pcm)
}

func NewTransactionManager(nid int, tsc *TxTimestampChecker, ptp *TransactionPool, ntp *TransactionPool, tim TXIDManager, logger log.Logger) *TransactionManager {
	txm := &TransactionManager{
		nid:          nid,
		tsc:          tsc,
		patchTxPool:  ptp,
		normalTxPool: ntp,
		tim:          tim,
		log:          logger,
		txWaiters:    map[hashValue][]chan<- interface{}{},
	}
	ptp.SetTxManager(txm)
	ntp.SetTxManager(txm)
	return txm
}
