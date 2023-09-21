/*
 * Copyright 2023 ICON Foundation
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
 *
 */

package txlocator

import (
	"fmt"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
)

type txList struct {
	group  module.TransactionGroup
	height int64
	ts     int64
	th     int64
	head   *locator
	next   *txList
}

func (l *txList) String() string {
	return fmt.Sprintf("txList{group=%d,height=%d,ts=%d,th=%d}",
		l.group, l.height, l.ts, l.th)
}

type locator struct {
	list   *txList
	id     string
	offset int
	next   *locator
}

func (l *locator) AsTransactionLocator() *module.TransactionLocator {
	return &module.TransactionLocator{
		BlockHeight:      l.list.height,
		TransactionGroup: l.list.group,
		IndexInGroup:     l.offset,
	}
}

var locatorPool = sync.Pool {
	New: func() interface{} {
		return new(locator)
	},
}

func allocLocator(list *txList, id string, idx int) *locator {
	loc := locatorPool.Get().(*locator)
	loc.list = list
	loc.id = id
	loc.offset = idx
	return loc
}

func freeLocator(loc *locator) {
	loc.list = nil
	loc.next = nil
	locatorPool.Put(loc)
}

type locatorFlushJob struct {
	wg   sync.WaitGroup
	list *txList
	next *locatorFlushJob
}

func newLocatorFlushJob(l *txList) *locatorFlushJob {
	job := &locatorFlushJob{ list: l }
	job.wg.Add(1)
	return job
}

func (j *locatorFlushJob) WaitForFetching() {
	j.wg.Wait()
}

func (j *locatorFlushJob) Fetch() *txList {
	j.wg.Done()
	return j.list
}

type txListCache struct {
	// linked list of cached tx lists.
	head   *txList
	lastP  **txList

	// maximum timestamp value of transactions in database
	// 0 means that it doesn't know the maximum.
	// In that case, it should look up database always
	maxTSInDB int64
}

type manager struct {
	lock     sync.Mutex
	lbk      db.Bucket
	log      log.Logger

	locators map[string]*locator

	// tx list to flush
	flushWG   sync.WaitGroup
	flushHead   *locatorFlushJob
	flushLastP  **locatorFlushJob
	flushWorker int


	// linked list of cached tx lists.
	cache [2]txListCache
}

func (m *manager) Has(group module.TransactionGroup, id []byte, ts int64) (bool, error) {
	if has, ok := m.hasLocatorInCache(group, id, ts); ok {
		return has, nil
	} else {
		return m.hasLocatorInDB(id)
	}
}

func (m *manager) GetLocator(id []byte) (*module.TransactionLocator, error) {
	if loc, ok := m.getLocatorFromCache(id) ; ok {
		return loc, nil
	}
	return  m.getLocatorFromDB(id)
}

func (m *manager) hasLocatorInCache(group module.TransactionGroup, id []byte, ts int64) (bool, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.locators == nil {
		return false, true
	}
	if _, ok := m.locators[string(id)]; ok {
		return ok, true
	}
	if l := m.cache[group].maxTSInDB; l != 0 && l <= ts {
		return false, true
	}
	return false, false
}

func (m *manager) getLocatorFromCache(id []byte) (*module.TransactionLocator, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.locators == nil {
		return nil, true
	}
	if loc, ok := m.locators[string(id)]; ok {
		return loc.AsTransactionLocator(), true
	}
	return nil, false
}

func (m *manager) hasLocatorInDB(id []byte) (bool, error) {
	bs, err := m.lbk.Get(id)
	if err != nil {
		return false, err
	} else {
		return len(bs)>0, nil
	}
}

func (m *manager) getLocatorFromDB(id []byte) (*module.TransactionLocator, error) {
	bs, err := m.lbk.Get(id)
	if err != nil {
		return nil, err
	}
	if bs == nil {
		return nil, nil
	}
	var loc module.TransactionLocator
	if remain, err := codec.BC.UnmarshalFromBytes(bs, &loc); err != nil {
		return nil, errors.CriticalFormatError.Wrap(err, "InvalidLocatorData")
	} else if len(remain) > 0 {
		return nil, errors.CriticalFormatError.New("RemainingLocatorBytes")
	}
	return &loc, nil
}

func (m *manager) NewTracker(group module.TransactionGroup, height int64, ts int64, th int64) module.LocatorTracker {
	return &tracker{
		list: &txList{
			group:  group,
			height: height,
			ts:     ts,
			th:     th,
		},
		locators: make(map[string]*locator),
		manager:  m,
	}
}

func (m *manager) commitTracker(t *tracker) error {
	lock := common.LockForAutoCall(&m.lock)
	defer lock.Unlock()

	if m.locators == nil {
		return errors.InvalidStateError.New("AlreadyTerminated")
	}
	for k, l := range t.locators {
		if loc, ok := m.locators[k] ; ok {
			loc.id = ""
		}
		m.locators[k] = l
	}
	if t.list.group == module.TransactionGroupNormal {
		job := m.pushFlushJobInLock(t.list)
		lock.CallAfterUnlock(func() {
			job.WaitForFetching()
		})
	} else {
		if err := m.flushList(t.list) ; err != nil {
			return err
		}
		m.addListAndClearOldInLock(t.list)
	}
	return nil
}
func (m *manager) addListAndClearOld(list *txList) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.addListAndClearOldInLock(list)
}

func (m *manager) addListAndClearOldInLock(list *txList) {
	if m.locators == nil {
		return
	}
	m.log.Tracef("LM: cache.addListAndClearOldInLock add %s", list)
	c := &m.cache[list.group]
	listMin := list.ts-list.th
	for ptr := c.head; ptr != nil; ptr = ptr.next {
		ptrMax := ptr.ts+ptr.th
		if ptrMax > listMin {
			break
		}
		m.log.Tracef("LM: cache.addListAndClearOldInLock remove %s", ptr)
		var next *locator
		for itr := ptr.head ; itr != nil ; itr = next {
			next = itr.next
			delete(m.locators, itr.id)
			freeLocator(itr)
		}
		if ptr.ts != 0 && c.maxTSInDB < ptrMax {
			m.log.Tracef("LM: cache[%d].maxTSInDB %d -> %d",
				list.group, c.maxTSInDB, ptrMax)
			c.maxTSInDB = ptrMax
		}
		c.head = ptr.next
	}
	if c.head == nil {
		c.lastP = &c.head
	}
	*c.lastP = list
	c.lastP = &list.next
}

func (m *manager) flushList(l *txList) error {
	for ptr := l.head ; ptr != nil ; ptr = ptr.next {
		bs := codec.BC.MustMarshalToBytes(module.TransactionLocator{
			BlockHeight:      ptr.list.height,
			IndexInGroup:     ptr.offset,
			TransactionGroup: ptr.list.group,
		})
		if err := m.lbk.Set([]byte(ptr.id), bs); err != nil {
			return err
		}
	}
	return nil
}

func (m *manager) pushFlushJobInLock(l *txList) *locatorFlushJob {
	m.flushWG.Add(1)

	job := newLocatorFlushJob(l)
	*m.flushLastP = job
	m.flushLastP = &job.next

	if m.flushWorker < 1 {
		m.flushWorker += 1
		go m.handleFlushJobs()
	}
	return job
}

func (m *manager) fetchFlushJob() *locatorFlushJob {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.flushHead == nil {
		m.flushWorker -= 1
		return nil
	}
	job := m.flushHead
	m.flushHead = job.next
	if m.flushHead == nil {
		m.flushLastP = &m.flushHead
	}
	return job
}

func (m *manager) handleFlushJobs() {
	for {
		job := m.fetchFlushJob()
		if job == nil {
			return
		}
		l := job.Fetch()
		if err := m.flushList(l); err != nil {
			func() {
				m.lock.Lock()
				defer m.lock.Unlock()
				m.locators = nil
			}()
			m.log.Errorf("FAIL to flush locators err=%+v", err)

			m.flushWG.Done()

			// cleanup pending jobs to wake up others
			for {
				job = m.fetchFlushJob()
				if job == nil {
					break
				}
				job.Fetch()
				m.flushWG.Done()
			}
			return
		}
		m.addListAndClearOld(l)
		m.flushWG.Done()
	}
}

func (m *manager) Start() {
}

func (m *manager) Term() {
	locker := common.LockForAutoCall(&m.lock)
	defer locker.Unlock()

	if m.locators != nil {
		m.locators = nil
		locker.CallAfterUnlock(func() {
			m.flushWG.Wait()
		})
	}
}

func NewManager(dbase db.Database, logger log.Logger) (module.LocatorManager, error) {
	lbk, err := dbase.GetBucket(db.TransactionLocatorByHash)
	if err != nil {
		return nil, err
	}
	mgr := &manager{
		lbk:      lbk,
		log:      logger,
		locators: make(map[string]*locator),
	}
	mgr.flushLastP = &mgr.flushHead
	return mgr, nil
}

type tracker struct {
	lock     sync.Mutex
	list     *txList
	locators map[string]*locator
	parent   *tracker
	manager  *manager
}

func (t *tracker) Has(id []byte, ts int64) (bool, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if ts >= t.list.ts+t.list.th {
		return false, nil
	}
	if t.locators != nil {
		if _, ok := t.locators[string(id)] ; ok {
			return true, nil
		}
	}
	return t.parentHasInLock(id, ts)
}

func (t *tracker) parentHasInLock(id []byte, ts int64) (bool, error) {
	if t.parent != nil {
		return t.parent.Has(id, ts)
	} else {
		return t.manager.Has(t.list.group, id, ts)
	}
}

func (t *tracker) Add(list module.TransactionList, force bool) (int, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	locators := t.locators
	if len(locators) > 0 {
		return 0, errors.InvalidStateError.New("AlreadyAdded")
	} else if locators == nil {
		return 0, errors.InvalidStateError.New("AlreadyCommitted")
	}
	prevP := &t.list.head
	cnt := 0
	for itr := list.Iterator(); itr.Has() ; _ = itr.Next() {
		if tx, idx, err := itr.Get(); err != nil {
			return cnt, err
		} else {
			txi := transaction.Unwrap(tx).(transaction.Transaction)
			id := txi.ID()
			if _, ok := locators[string(id)]; ok {
				return cnt, errors.IllegalArgumentError.Errorf("DuplicateTx(id=%#x)", id)
			}
			if !force {
				if has, err := t.parentHasInLock(id, txi.Timestamp()) ; err != nil {
					return cnt, err
				} else if has {
					return cnt, errors.IllegalArgumentError.Errorf("DuplicateTx(id=%#x)", id)
				}
			}
			loc := allocLocator(t.list, string(id), idx)
			*prevP = loc
			prevP = &loc.next
			locators[loc.id] = loc
			cnt += 1
		}
	}
	*prevP = nil
	t.locators = locators
	return cnt, nil
}

func (t *tracker) New(height int64, ts int64, th int64) module.LocatorTracker {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.locators == nil && t.parent == nil {
		return t.manager.NewTracker(t.list.group, height, ts, th)
	}
	return &tracker{
		list: &txList{
			group:  t.list.group,
			height: height,
			ts:     ts,
			th:     th,
		},
		locators: make(map[string]*locator),
		parent:   t,
		manager:  t.manager,
	}
}

func (t *tracker) Commit() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.parent != nil {
		if err := t.parent.Commit(); err != nil {
			return err
		}
		t.parent = nil
	}
	if t.locators != nil {
		if err := t.manager.commitTracker(t); err != nil {
			return err
		}
		t.locators = nil
	}
	return nil
}

func (t *tracker) Group() module.TransactionGroup {
	return t.list.group
}
func (t *tracker) Timestamp() int64 {
	return t.list.ts
}

func (t *tracker) Threshold() int64 {
	return t.list.th
}

func WriteTransactionLocators(
	dbase db.Database,
	height int64,
	ptl module.TransactionList,
	ntl module.TransactionList,
) error {
	bk, err := db.NewCodedBucket(dbase, db.TransactionLocatorByHash, nil)
	if err != nil {
		return err
	}
	for it := ptl.Iterator(); it.Has(); log.Must(it.Next()) {
		tr, i, err := it.Get()
		if err != nil {
			return err
		}
		trLoc := module.TransactionLocator{
			BlockHeight:      height,
			TransactionGroup: module.TransactionGroupPatch,
			IndexInGroup:     i,
		}
		if err = bk.Set(db.Raw(tr.ID()), trLoc); err != nil {
			return err
		}
	}
	for it := ntl.Iterator(); it.Has(); log.Must(it.Next()) {
		tr, i, err := it.Get()
		if err != nil {
			return err
		}
		trLoc := module.TransactionLocator{
			BlockHeight:      height,
			TransactionGroup: module.TransactionGroupNormal,
			IndexInGroup:     i,
		}
		if err = bk.Set(db.Raw(tr.ID()), trLoc); err != nil {
			return err
		}
	}
	return nil
}

