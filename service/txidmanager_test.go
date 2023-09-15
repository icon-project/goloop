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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/txlocator"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	SecondInMicro = 1000000
	MinuteInMicro = SecondInMicro * 60
)

func tidOf(idx int) []byte {
	return []byte(fmt.Sprintf("tx%d", idx))
}

type testDummyTx struct {
	id []byte
	ts int64
	g  module.TransactionGroup
	transaction.Transaction
}

func (t *testDummyTx) Group() module.TransactionGroup {
	return t.g
}

func (t *testDummyTx) ID() []byte {
	return t.id
}

func (t *testDummyTx) Hash() []byte {
	return t.id
}

func (t *testDummyTx) Timestamp() int64 {
	return t.ts
}

type testTxList struct {
	txs []*testDummyTx
	module.TransactionList
}

func (t *testTxList) Get(i int) (module.Transaction, error) {
	return t.txs[i], nil
}

func (t *testTxList) Iterator() module.TransactionIterator {
	return &testTxIterator{ l: t, idx: 0 }
}

type testTxIterator struct {
	l   *testTxList
	idx int
}

func (t *testTxIterator) Has() bool {
	return t.idx < len(t.l.txs)
}

func (t *testTxIterator) Next() error {
	t.idx += 1
	return nil
}

func (t *testTxIterator) Get() (module.Transaction, int, error) {
	return t.l.txs[t.idx], t.idx, nil
}

func newTestTXList(g module.TransactionGroup, start, cnt int, ts, delta int64) *testTxList {
	txs := make([]*testDummyTx, cnt)
	for i := 0 ; i<cnt ; i++ {
		txs[i] = &testDummyTx{
			id: tidOf(i + start),
			ts: ts + delta*int64(i),
			g:  g,
		}
	}
	return &testTxList{ txs: txs }
}

func TestTXIDManager_RecordedLocator(t *testing.T) {
	baseHeight := int64(10)
	baseTS := int64(100)
	tid1 := tidOf(1)
	tid2 := tidOf(2)
	tid3 := tidOf(3)
	tid4 := tidOf(4)

	dbase := db.NewMapDB()
	bk, err := dbase.GetBucket(db.TransactionLocatorByHash)
	assert.NoError(t, err)

	var locator module.TransactionLocator
	locator.BlockHeight = baseHeight
	locator.TransactionGroup = module.TransactionGroupNormal
	err = bk.Set([]byte(tid1), codec.MustMarshalToBytes(&locator))
	assert.NoError(t, err)

	tsc := NewTimestampChecker()
	tsc.SetThreshold(5 * time.Minute)

	t.Run("dummy_logger", func(t *testing.T) {
		lm, err := txlocator.NewManager(db.NewLayerDB(dbase), log.GlobalLogger())
		assert.NoError(t, err)

		mgr, err := NewTXIDManager(lm, tsc, nil)
		assert.NoError(t, err)

		logger := mgr.NewLogger(module.TransactionGroupNormal, 0, 0)

		logger1 := logger.NewLogger(baseHeight+1, baseTS+MinuteInMicro, tsc.Threshold())

		tx2 := newTestTXList(module.TransactionGroupNormal, 2, 1, baseTS+MinuteInMicro, 0)
		_, err = logger1.Add(tx2, false)
		assert.NoError(t, err)

		err = logger.Commit()
		assert.NoError(t, err)

		logger2 := logger1.NewLogger(baseHeight+2, baseTS+2*MinuteInMicro, tsc.Threshold())

		has, err := logger2.Has(tid1, baseTS)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = logger2.Has(tid2, baseTS+MinuteInMicro)
		assert.NoError(t, err)
		assert.True(t, has)

		tx34 := newTestTXList(module.TransactionGroupNormal, 3, 2, baseTS+MinuteInMicro, 0)
		_, err = logger2.Add(tx34, false)
		assert.NoError(t, err)

		err = logger1.Commit()
		assert.NoError(t, err)

		has, err = logger2.Has(tid1, baseTS)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = logger2.Has(tid2, baseTS+MinuteInMicro)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = mgr.HasRecent(module.TransactionGroupNormal, tid1, baseTS)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = mgr.HasRecent(module.TransactionGroupNormal, tid2, baseTS+MinuteInMicro)
		assert.NoError(t, err)
		assert.True(t, has)

		logger3 := logger2.NewLogger(baseHeight+3, baseTS+3*MinuteInMicro, tsc.Threshold())

		err = logger2.Commit()
		assert.NoError(t, err)

		tx5 := newTestTXList(module.TransactionGroupNormal, 5, 1, baseTS+3*MinuteInMicro, 0)
		_, err = logger3.Add(tx5, false)
		assert.NoError(t, err)

		err = logger3.Commit()
		assert.NoError(t, err)

		// duplicated commit (should be silently ignored)
		err = logger3.Commit()
		assert.NoError(t, err)

		has, err = mgr.HasRecent(module.TransactionGroupNormal, tid1, baseTS)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = mgr.HasLocator(tid1)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = mgr.HasRecent(module.TransactionGroupNormal, tid2, baseTS+MinuteInMicro)
		assert.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("same height same tx", func(t *testing.T) {
		lm, err := txlocator.NewManager(db.NewLayerDB(dbase), log.GlobalLogger())
		assert.NoError(t, err)

		mgr, err := NewTXIDManager(lm, tsc, nil)
		assert.NoError(t, err)

		has, err := mgr.HasRecent(module.TransactionGroupNormal, tid1, 0)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = mgr.HasRecent(module.TransactionGroupNormal, tid2, baseTS+MinuteInMicro)
		assert.NoError(t, err)
		assert.False(t, has)

		logger := mgr.NewLogger(module.TransactionGroupNormal, baseHeight, baseTS)

		has, err = logger.Has(tid1, 0)
		assert.NoError(t, err)
		assert.True(t, has)


		tx2 := newTestTXList(module.TransactionGroupNormal, 2, 1, baseTS, 0)
		assert.NoError(t, mgr.CheckTXForAdd(tx2.txs[0]))
		_, err = logger.Add(tx2, false)
		assert.NoError(t, err)

		logger2 := logger.NewLogger(baseHeight+1, baseTS+5*MinuteInMicro, tsc.Threshold())

		has, err = logger2.Has(tid1, 0)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = logger2.Has(tid2, 0)
		assert.NoError(t, err)
		assert.True(t, has)

		tx34 := newTestTXList(module.TransactionGroupNormal, 3, 2, baseTS+5*MinuteInMicro, 0)
		_, err = logger2.Add(tx34, false)
		assert.NoError(t, err)

		// first block is committed
		err = logger.Commit()
		assert.NoError(t, err)

		// check first block(tid1:OK, tid2:OK, tid3:NG, tid4:NG)
		has, err = mgr.HasRecent(module.TransactionGroupNormal, tid1, 0)
		assert.NoError(t, err)
		assert.True(t, has)
		has, err = mgr.HasRecent(module.TransactionGroupNormal, tid2, baseTS)
		assert.NoError(t, err)
		assert.True(t, has)
		has, err = mgr.HasRecent(module.TransactionGroupNormal, tid3, baseTS+5*MinuteInMicro)
		assert.NoError(t, err)
		assert.False(t, has)
		has, err = mgr.HasRecent(module.TransactionGroupNormal, tid4, baseTS+5+MinuteInMicro)
		assert.NoError(t, err)
		assert.False(t, has)

		// second block is committed
		err = logger2.Commit()
		assert.NoError(t, err)

		// duplicate commit (should be silently ignored)
		err = logger2.Commit()
		assert.NoError(t, err)

		// check second items (tid3, tid4)
		has, err = mgr.HasRecent(module.TransactionGroupNormal, tid3, baseTS+5*MinuteInMicro)
		assert.NoError(t, err)
		assert.True(t, has)
		has, err = mgr.HasRecent(module.TransactionGroupNormal, tid4, baseTS+5*MinuteInMicro)
		assert.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("old_block_pruning", func(t *testing.T) {
		layer := db.NewLayerDB(dbase)
		lm, err := txlocator.NewManager(layer, log.GlobalLogger())
		assert.NoError(t, err)

		mgr, err := NewTXIDManager(lm, tsc, nil)
		assert.NoError(t, err)

		height := baseHeight
		ts := baseTS

		logger := mgr.NewLogger(module.TransactionGroupNormal, height, ts)
		tx1 := newTestTXList(module.TransactionGroupNormal, 1, 1, 0, 0)
		_, err = logger.Add(tx1, false)
		assert.Error(t, err)
		height += 1
		ts += MinuteInMicro

		for i := 2; i <= 20; i++ {
			nlogger := logger.NewLogger(height, ts, tsc.Threshold())

			txn := newTestTXList(module.TransactionGroupNormal, i, 1, ts, 0)
			_, err := nlogger.Add(txn, false)
			assert.NoError(t, err)

			err = logger.Commit()
			assert.NoError(t, err)

			logger = nlogger
			height += 1
			ts += MinuteInMicro
		}

		// purge database changes then check it with cache
		err = layer.Flush(false)
		assert.NoError(t, err)

		has, err := logger.Has(tid2, baseTS)
		assert.NoError(t, err)
		assert.False(t, has)

		has, err = mgr.HasRecent(module.TransactionGroupNormal, tid2, baseTS)
		assert.NoError(t, err)
		assert.False(t, has)

		has, err = logger.Has(tidOf(20), ts)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = mgr.HasRecent(module.TransactionGroupNormal, tidOf(20), ts)
		assert.NoError(t, err)
		assert.False(t, has)

		has, err = mgr.HasRecent(module.TransactionGroupNormal, tidOf(19), ts)
		assert.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("new height same tx", func(t *testing.T) {
		lm, err := txlocator.NewManager(db.NewLayerDB(dbase), log.GlobalLogger())
		assert.NoError(t, err)
		history, err := NewTXIDManager(lm, tsc, nil)
		assert.NoError(t, err)
		logger := history.NewLogger(module.TransactionGroupNormal, baseHeight+1, baseTS)
		tx1 := newTestTXList(module.TransactionGroupNormal, 1, 1, baseTS, 0)

		assert.Error(t, history.CheckTXForAdd(tx1.txs[0]))

		_, err = logger.Add(tx1, false)
		assert.Error(t, err)
	})

	t.Run("CheckTxForAdd", func(t *testing.T) {
		lm, err := txlocator.NewManager(db.NewLayerDB(dbase), log.GlobalLogger())
		assert.NoError(t, err)

		tic := NewTxIDCache(
			ConfigDroppedTxSlotDuration,
			ConfigMaxDroppedTxSlotSize,
			log.GlobalLogger())

		mgr, err := NewTXIDManager(lm, tsc, tic)
		assert.NoError(t, err)

		logger := mgr.NewLogger(module.TransactionGroupNormal, baseHeight+1, baseTS)
		tx1 := newTestTXList(module.TransactionGroupNormal, 1, 1, baseTS, 0)

		assert.Error(t, mgr.CheckTXForAdd(tx1.txs[0]))

		tx2 := newTestTXList(module.TransactionGroupNormal, 2, 1, baseTS+MinuteInMicro, 0)

		assert.NoError(t, mgr.CheckTXForAdd(tx2.txs[0]))

		mgr.AddDroppedTX(tx2.txs[0].ID(), baseTS+MinuteInMicro)

		assert.Error(t, mgr.CheckTXForAdd(tx2.txs[0]))

		tx3 := newTestTXList(module.TransactionGroupNormal, 3, 1, baseTS+MinuteInMicro, 0)

		assert.NoError(t, mgr.CheckTXForAdd(tx3.txs[0]))
		_, err = logger.Add(tx3, false)
		assert.NoError(t, err)

		assert.NoError(t, logger.Commit())

		assert.Error(t, mgr.CheckTXForAdd(tx3.txs[0]))
	})
}
