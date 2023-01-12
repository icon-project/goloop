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

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
)

const InvalidHeight = -1

type TXIDSet interface {
	GetHeightOf(id string) (int64, error)
	SetPrevious(p TXIDSet)
	Previous() TXIDSet
}

type TXIDManager interface {
	OnThresholdChange()
	HasLocator(id []byte) (bool, error)
	HasRecent(id []byte) (bool, error)
	CheckTXForAdd(tx transaction.Transaction) error
	AddDroppedTX(id []byte, ts int64)
	NewLogger(group module.TransactionGroup, height int64, ts int64) TXIDLogger
}

type TXIDLogger interface {
	Has(id []byte) (bool, error)
	Add(id []byte, force bool) error
	Commit() error
	NewLogger(height int64, ts int64) TXIDLogger
}

type txIDManagerProxy interface {
	GetHeightOf(id string) (int64, error)
	CommitSet(s *txIDMap) (txIDManagerProxy, error)
}

type txIDMap struct {
	group  module.TransactionGroup
	height int64
	ts     int64
	txs    map[string]struct{}

	previous TXIDSet
}

func (m *txIDMap) GetHeightOf(id string) (int64, error) {
	if _, has := m.txs[id]; has {
		return m.height, nil
	} else {
		return InvalidHeight, nil
	}
}

func (m *txIDMap) SetPrevious(p TXIDSet) {
	m.previous = p
}

func (m *txIDMap) Previous() TXIDSet {
	return m.previous
}

func (m *txIDMap) Add(id string) {
	m.txs[id] = struct{}{}
}

func (m *txIDMap) Equal(m2 *txIDMap) bool {
	if m.height != m2.height || m.group != m2.group ||
		m.ts != m2.ts || len(m.txs) != len(m2.txs) {
		return false
	}
	for tid, _ := range m.txs {
		if _, ok := m2.txs[tid]; !ok {
			return false
		}
	}
	return true
}

func (m *txIDMap) IsEmpty() bool {
	return len(m.txs) == 0
}

func newTXIDMap(group module.TransactionGroup, height int64, ts int64) *txIDMap {
	return &txIDMap{
		group:  group,
		height: height,
		ts:     ts,
		txs:    make(map[string]struct{}),
	}
}

type txIDBucket struct {
	tbk   db.Bucket
	cache map[string]int64
}

type transactionLocator struct {
	BlockHeight      int64
	TransactionGroup module.TransactionGroup
	IndexInGroup     int
}

func (s *txIDBucket) GetHeightOf(id string) (int64, error) {
	if height, ok := s.cache[id]; ok {
		return height, nil
	}

	bs, err := s.tbk.Get([]byte(id))
	if err != nil {
		return InvalidHeight, err
	}
	if len(bs) == 0 {
		return InvalidHeight, nil
	}
	var locator transactionLocator
	if _, err := codec.BC.UnmarshalFromBytes(bs, &locator); err != nil {
		return InvalidHeight, err
	}

	s.cache[id] = locator.BlockHeight
	return locator.BlockHeight, nil
}

func (s *txIDBucket) Previous() TXIDSet {
	return nil
}

func (s *txIDBucket) SetPrevious(p TXIDSet) {
	panic("SetParent() isn't allowed")
}

func newTransactionBucket(bk db.Bucket) TXIDSet {
	return &txIDBucket{
		tbk:   bk,
		cache: make(map[string]int64),
	}
}

type txIDManager struct {
	lock       sync.Mutex
	tbk        db.Bucket
	last       TXIDSet
	tsc        *TxTimestampChecker
	droppedTxs TXIDCache
	th         int64
	ts         [2]int64
}

func (mg *txIDManager) NewLogger(group module.TransactionGroup, height int64, ts int64) TXIDLogger {
	return &txIDLogger{
		proxy: mg,
		set:   newTXIDMap(group, height, ts),
	}
}

func (mg *txIDManager) CommitSet(s *txIDMap) (txIDManagerProxy, error) {
	mg.lock.Lock()
	defer mg.lock.Unlock()

	if s.ts != 0 {
		tsMin := s.ts - mg.tsc.Threshold()*2

		for ptr := mg.last; ptr != nil; ptr = ptr.Previous() {
			if m, ok := ptr.(*txIDMap); ok {
				if m.ts == 0 {
					continue
				}
				if m.height < s.height {
					if m.ts < tsMin {
						m.SetPrevious(nil)
						break
					}
					continue
				}
				if m.height > s.height {
					panic("RollbackToOldHeight?")
				}
				if m.height == s.height && m.group == s.group {
					if m.Equal(s) {
						return mg, nil
					} else {
						return nil, errors.InvalidStateError.Errorf("ConflictTXSet(height=%d)", s.height)
					}
				}
			} else {
				break
			}
		}
		mg.ts[s.group] = s.ts
		if s.group == module.TransactionGroupNormal {
			mg.droppedTxs.RemoveOldTXsByTS(s.ts - mg.tsc.Threshold())
		}
	}
	if s.ts != 0 || !s.IsEmpty() {
		s.SetPrevious(mg.last)
		mg.last = s
	}
	return mg, nil
}

func (mg *txIDManager) GetHeightOf(id string) (int64, error) {
	mg.lock.Lock()
	defer mg.lock.Unlock()

	return mg.getHeightOfInLock(id)
}

func (mg *txIDManager) getHeightOfInLock(id string) (int64, error) {
	for ptr := mg.last; ptr != nil; ptr = ptr.Previous() {
		if height, err := ptr.GetHeightOf(id); err != nil {
			return InvalidHeight, err
		} else if height != InvalidHeight {
			return height, nil
		}
	}
	return InvalidHeight, nil
}
func (mg *txIDManager) HasRecent(id []byte) (bool, error) {
	ids := string(id)
	height, err := mg.GetHeightOf(ids)
	return height != InvalidHeight, err
}

func (mg *txIDManager) CheckTXForAdd(tx transaction.Transaction) error {
	mg.lock.Lock()
	defer mg.lock.Unlock()

	minTS := mg.ts[tx.Group()] - mg.tsc.Threshold()
	if err := mg.tsc.CheckWithCurrent(minTS, tx); err != nil {
		return err
	}

	id := tx.ID()
	if mg.droppedTxs.Contains(id, tx.Timestamp()) {
		return InvalidTransactionError.Errorf("AlreadyDropped(id=%#x)", id)
	}

	ids := string(id)
	if height, err := mg.getHeightOfInLock(ids); err != nil {
		return err
	} else if height != InvalidHeight {
		return CommittedTransactionError.Errorf("AlreadyCommitted(height=%d)", height)
	}
	return nil
}

func (mg *txIDManager) HasLocator(id []byte) (bool, error) {
	return mg.tbk.Has(id)
}

func (mg *txIDManager) OnThresholdChange() {
	mg.lock.Lock()
	defer mg.lock.Unlock()

	th := mg.tsc.Threshold()
	if th == mg.th {
		return
	}
	if th > mg.th {
		nbk := newTransactionBucket(mg.tbk)
		ptr := mg.last
		if ptr == nil {
			mg.last = nbk
		} else {
			for ptr.Previous() != nil {
				ptr = ptr.Previous()
			}
			if _, ok := ptr.(*txIDMap); ok {
				ptr.SetPrevious(nbk)
			}
		}
	}
	mg.th = th
}

func (mg *txIDManager) LastTS(group module.TransactionGroup) int64 {
	mg.lock.Lock()
	defer mg.lock.Unlock()
	return mg.ts[group]
}

func (mg *txIDManager) AddDroppedTX(id []byte, ts int64) {
	mg.lock.Lock()
	defer mg.lock.Unlock()
	mg.droppedTxs.Add(id, ts)
}

func NewTXIDManager(dbase db.Database, tsc *TxTimestampChecker, tic TXIDCache) (TXIDManager, error) {
	bk, err := dbase.GetBucket(db.TransactionLocatorByHash)
	if err != nil {
		return nil, err
	}
	if tic == nil {
		tic = newEmptyTxIDCache()
	}
	return &txIDManager{
		tbk:        bk,
		last:       newTransactionBucket(bk),
		th:         tsc.Threshold(),
		tsc:        tsc,
		droppedTxs: tic,
	}, nil
}

type txIDLogger struct {
	proxy txIDManagerProxy
	set   *txIDMap
	done  bool
}

func (l *txIDLogger) GetHeightOf(id string) (int64, error) {
	if height, err := l.set.GetHeightOf(id); err != nil {
		return InvalidHeight, err
	} else if height != InvalidHeight {
		return height, nil
	}
	return l.proxy.GetHeightOf(id)
}

func (l *txIDLogger) NewLogger(height int64, ts int64) TXIDLogger {
	return &txIDLogger{
		proxy: l,
		set:   newTXIDMap(l.set.group, height, ts),
	}
}

func (l *txIDLogger) Has(id []byte) (bool, error) {
	return l.has(string(id))
}

func (l *txIDLogger) has(id string) (bool, error) {
	if height, _ := l.set.GetHeightOf(id); height != InvalidHeight {
		return true, nil
	}
	height, err := l.proxy.GetHeightOf(id)
	return height != InvalidHeight && height < l.set.height, err
}

func (l *txIDLogger) Add(id []byte, force bool) error {
	ids := string(id)
	if has, err := l.has(ids); err != nil {
		return err
	} else if has && !force {
		return CommittedTransactionError.Errorf("Committed(id=%x)", id)
	}
	l.set.Add(ids)
	return nil
}

func (l *txIDLogger) CommitSet(s *txIDMap) (txIDManagerProxy, error) {
	return l.proxy.CommitSet(s)
}

func (l *txIDLogger) Commit() error {
	if l.done {
		return nil
	}
	if proxy, err := l.proxy.CommitSet(l.set); err != nil {
		return err
	} else {
		l.proxy = proxy
		l.done = true
		return nil
	}
}
