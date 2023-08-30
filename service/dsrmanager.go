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

package service

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

func keyForDSR(height int64, signer module.Address) string {
	return fmt.Sprintf("%d-%s", height, signer.String())
}

type DSRLocator struct {
	Height int64
	Signer module.Address
}

type DSRTracker interface {
	Has(height int64, signer module.Address) bool
	Add(height int64, signer module.Address)
	Commit()
	New() DSRTracker
}

type DSRManager interface {
	NewTracker() DSRTracker
	OnFinalizeState(ass state.AccountSnapshot)
}

type dsrTracker struct {
	lock    sync.Mutex
	reports map[string]DSRLocator
	parent  DSRTracker
	manager *dsrManager
}

func (t *dsrTracker) Has(height int64, signer module.Address) bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	key := keyForDSR(height, signer)
	if len(t.reports) > 0 {
		if _, ok := t.reports[key]; ok {
			return true
		}
	}
	if t.parent != nil {
		return t.parent.Has(height, signer)
	} else {
		return t.manager.Has(height, signer)
	}
}

func (t *dsrTracker) Add(height int64, signer module.Address) {
	if t.reports == nil {
		t.reports = make(map[string]DSRLocator)
	}
	t.reports[keyForDSR(height, signer)] = DSRLocator{height, signer }
}

func (t *dsrTracker) Commit() {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.parent != nil {
		t.parent.Commit()
		t.parent = nil
	}
	if t.reports != nil {
		items := make([]DSRLocator,0,len(t.reports))
		for _, loc := range t.reports {
			items = append(items, loc)
		}
		t.manager.Commit(items)
		t.reports = nil
	}
}

func (t *dsrTracker) New() DSRTracker {
	return &dsrTracker{
		parent:  t,
		manager: t.manager,
	}
}

type doubleSignReport struct {
	Key     string
	Height  int64
	Signer  module.Address
	Type    string
	Data    []module.DoubleSignData
	Context module.DoubleSignContext
}

const InvalidFirstHeight = -1
type dsrManager struct {
	lock        sync.Mutex
	log         log.Logger
	todo        list.List
	done        list.List
	reports     map[string]*list.Element
	firstHeight int64
}

func addToListInOrder(lst *list.List, dsr *doubleSignReport) *list.Element {
	for itr := lst.Back() ; itr != nil ; itr = itr.Prev() {
		r := itr.Value.(*doubleSignReport)
		if r.Height <= dsr.Height {
			return lst.InsertAfter(dsr, itr)
		}
	}
	return lst.PushFront(dsr)
}

func (m *dsrManager) Add(data []module.DoubleSignData, ctx module.DoubleSignContext) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if len(data)!=2 {
		return errors.IllegalArgumentError.Errorf("InvalidDataLength(len=%d)", len(data))
	}

	height := data[0].Height()
	if m.firstHeight == InvalidFirstHeight || height < m.firstHeight {
		m.log.Infof("DROP DSR: feature is not enabled or out of history first=%d, height=%d",
			m.firstHeight, height)
		return nil
	}

	if !data[0].IsConflictWith(data[1]) {
		return errors.IllegalArgumentError.New("InvalidDoubleSignData")
	}

	signer := ctx.AddressOf(data[0].Signer())
	if signer == nil {
		return errors.IllegalArgumentError.New("SignerNotFoundInContext")
	}


	key := keyForDSR(data[0].Height(), signer)

	if _, ok := m.reports[key] ; ok {
		m.log.Infof("DROP DSR: already reported key=%s", key)
		return nil
	}

	report := &doubleSignReport{
		Key:     key,
		Height:  height,
		Signer:  signer,
		Data:    data,
		Context: ctx,
	}
	m.reports[report.Key] = addToListInOrder(&m.todo, report)
	return nil
}

func (m *dsrManager) Candidate(tracker DSRTracker, wc state.WorldContext) ([]module.Transaction, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	as := wc.GetAccountState(state.SystemID)
	ch, err := contract.NewDSContextHistoryDB(as)
	if err != nil {
		return nil, err
	}
	fh, ok := ch.FirstHeight()
	if !ok {
		return nil, nil
	}
	bh := wc.BlockHeight()

	var txs []module.Transaction
	for itr := m.todo.Front() ; itr != nil ; itr = itr.Next() {
		r := itr.Value.(*doubleSignReport)
		if r.Height < fh || r.Height > bh {
			continue
		}
		if tracker.Has(r.Height, r.Signer) {
			continue
		}
		tx := transaction.NewDoubleSignReportTx(r.Data, r.Context, wc.BlockTimeStamp())
		txs = append(txs, tx)
	}
	return txs, nil
}

func (m *dsrManager) setFirstHeightInLock(h int64) {
	if m.firstHeight >= h {
		return
	}
	m.firstHeight = h
	for itr := m.done.Front() ; itr != nil ; {
		dsr := itr.Value.(*doubleSignReport)
		if dsr.Height < h {
			ptr := itr
			itr = itr.Next()
			m.done.Remove(ptr)
			delete(m.reports, dsr.Key)
		} else {
			itr = itr.Next()
		}
	}
	for itr := m.todo.Front() ; itr != nil ; {
		dsr := itr.Value.(*doubleSignReport)
		if dsr.Height < h {
			ptr := itr
			itr = itr.Next()
			m.todo.Remove(ptr)
			delete(m.reports, dsr.Key)
		} else {
			itr = itr.Next()
		}
	}
}

func (m *dsrManager) Commit(rs []DSRLocator) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, r := range rs {
		key := keyForDSR(r.Height, r.Signer)
		if e, ok := m.reports[key]; ok {
			dsr := e.Value.(*doubleSignReport)
			if dsr.Data != nil {
				m.todo.Remove(e)
				dsr.Data = nil
				dsr.Context = nil
				m.reports[dsr.Key] = addToListInOrder(&m.done, dsr)
			}
		} else {
			m.reports[key] = addToListInOrder(&m.done, &doubleSignReport{
				Key:    key,
				Height: r.Height,
				Signer: r.Signer,
			})
		}
	}
}

func (m *dsrManager) Has(height int64, signer module.Address) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	key := keyForDSR(height, signer)
	if e, ok := m.reports[key]; ok {
		dsr := e.Value.(*doubleSignReport)
		return dsr.Data == nil
	} else {
		return false
	}
}

func (m *dsrManager) OnFinalizeState(ass state.AccountSnapshot) {
	m.lock.Lock()
	defer m.lock.Unlock()

	as :=  scoredb.NewStateStoreWith(ass)
	cdb, err := contract.NewDSContextHistoryDB(as)
	if err != nil {
		m.log.Warnf("Fail to build DSContextHistoryDB err=%+v", err)
		return
	}
	if height, ok := cdb.FirstHeight(); ok {
		m.setFirstHeightInLock(height)
	}
}

func (m *dsrManager) NewTracker() DSRTracker {
	return &dsrTracker{
		manager: m,
	}
}

func newDSRManager(logger log.Logger) *dsrManager {
	s := &dsrManager{
		log:         logger,
		reports:     make(map[string]*list.Element),
		firstHeight: InvalidFirstHeight,
	}
	s.todo.Init()
	s.done.Init()
	return s
}