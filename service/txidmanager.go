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

package service

import (
	"sync"
	"sync/atomic"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
)

const InvalidHeight = -1

type TXIDManager interface {
	HasLocator(id []byte) (bool, error)
	HasRecent(g module.TransactionGroup, id []byte, ts int64) (bool, error)
	CheckTXForAdd(tx transaction.Transaction) error
	AddDroppedTX(id []byte, ts int64)
	NewLogger(group module.TransactionGroup, height int64, ts int64) TXIDLogger
}

type TXIDLogger interface {
	Has(id []byte, ts int64) (bool, error)
	Add(list module.TransactionList, force bool) (int, error)
	Commit() error
	NewLogger(height int64, ts int64, th int64) TXIDLogger
}

type txIDManager struct {
	lock       sync.Mutex
	lm         module.LocatorManager
	tsc        *TxTimestampChecker
	droppedTXs TXIDCache
	commitTS   [2]int64
}

func (mg *txIDManager) NewLogger(group module.TransactionGroup, height int64, ts int64) TXIDLogger {
	return &txIDLogger{
		lt: mg.lm.NewTracker(group, height, ts, mg.tsc.Threshold()),
		mg: mg,
	}
}

func (mg *txIDManager) HasRecent(g module.TransactionGroup, id []byte, ts int64) (bool, error) {
	return mg.lm.Has(g, id, ts)
}

func (mg *txIDManager) CheckTXForAdd(tx transaction.Transaction) error {
	mg.lock.Lock()
	defer mg.lock.Unlock()

	txg := tx.Group()
	minTS := mg.commitTS[txg] - mg.tsc.TransactionThreshold(txg)
	if err := mg.tsc.CheckWithCurrent(minTS, tx); err != nil {
		return err
	}

	id := tx.ID()
	ts := tx.Timestamp()
	if txg == module.TransactionGroupNormal && mg.droppedTXs.Contains(id, ts) {
		return InvalidTransactionError.Errorf("AlreadyDropped(id=%#x)", id)
	}
	if has, err := mg.lm.Has(txg, id, ts) ; err != nil {
		return err
	} else if has {
		return ErrCommittedTransaction
	}
	return nil
}

func (mg *txIDManager) HasLocator(id []byte) (bool, error) {
	loc, err := mg.lm.GetLocator(id)
	if err != nil {
		return false, err
	} else {
		return loc!=nil, nil
	}
}

func (mg *txIDManager) AddDroppedTX(id []byte, ts int64) {
	mg.lock.Lock()
	defer mg.lock.Unlock()

	mg.droppedTXs.Add(id, ts)
}

func (mg *txIDManager) onCommit(g module.TransactionGroup, ts int64, th int64) {
	mg.lock.Lock()
	defer mg.lock.Unlock()

	if g == module.TransactionGroupNormal {
		mg.droppedTXs.RemoveOldTXsByTS(ts-th)
	}
	mg.commitTS[g] = ts
}

func NewTXIDManager(lm module.LocatorManager, tsc *TxTimestampChecker, tic TXIDCache) (TXIDManager, error) {
	if tic == nil {
		tic = newEmptyTxIDCache()
	}
	return &txIDManager{
		lm:         lm,
		tsc:        tsc,
		droppedTXs: tic,
	}, nil
}

type txIDLogger struct {
	lt     module.LocatorTracker
	mg     *txIDManager
	commit int32
}

func (l *txIDLogger) NewLogger(height int64, ts int64, th int64) TXIDLogger {
	return &txIDLogger{
		lt: l.lt.New(height, ts, th),
		mg: l.mg,
	}
}

func (l *txIDLogger) Has(id []byte, ts int64) (bool, error) {
	return l.lt.Has(id, ts)
}

func (l *txIDLogger) Add(list module.TransactionList, force bool) (int, error) {
	return l.lt.Add(list, force)
}

func (l *txIDLogger) Commit() error {
	if atomic.CompareAndSwapInt32(&l.commit, 0, 1) {
		if err := l.lt.Commit(); err != nil {
			atomic.StoreInt32(&l.commit, 0)
			return err
		}
		l.mg.onCommit(l.lt.Group(), l.lt.Timestamp(), l.lt.Threshold())
	}
	return nil
}
