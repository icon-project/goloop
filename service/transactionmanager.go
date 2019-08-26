package service

import (
	"sync"
	"sync/atomic"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

type TransactionManager struct {
	nid  int
	tsc  *TxTimestampChecker
	log  log.Logger
	lock sync.Mutex

	patchTxPool  *TransactionPool
	normalTxPool *TransactionPool

	callback func()

	lastTS int64
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

func (m *TransactionManager) RemoveOldTxByBlockTS(bts int64) {
	ts := bts - m.tsc.Threshold()
	atomic.StoreInt64(&m.lastTS, ts)
	m.patchTxPool.RemoveOldTXs(ts)
	m.normalTxPool.RemoveOldTXs(ts)
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

func (m *TransactionManager) Add(tx transaction.Transaction, direct bool) error {
	if tx == nil {
		return nil
	}
	if !tx.ValidateNetwork(m.nid) {
		return errors.InvalidNetworkError.Errorf(
			"ValidateNetwork(nid=%#x) fail", m.nid)
	}
	lastTS := atomic.LoadInt64(&m.lastTS)
	if err := m.tsc.CheckWithCurrent(lastTS, tx); err != nil {
		return err
	}
	if err := tx.Verify(); err != nil {
		return InvalidTransactionError.Wrap(err,
			"Failed to verify transaction")
	}

	m.lock.Lock()
	defer m.lock.Unlock()
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

func NewTransactionManager(nid int, tsc *TxTimestampChecker, ptp *TransactionPool, ntp *TransactionPool, logger log.Logger) *TransactionManager {
	return &TransactionManager{
		nid:          nid,
		tsc:          tsc,
		patchTxPool:  ptp,
		normalTxPool: ntp,
		log:          logger,
	}
}
