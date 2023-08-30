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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
)

func txIDFromInt(i int) []byte {
	return crypto.SHA3Sum256([]byte(fmt.Sprintf("test transaction %d", i)))
}

func TestManager_HasInDatabase(t *testing.T) {
	dbase := db.NewMapDB()
	logger := log.GlobalLogger()
	mgr, err := NewManager(dbase, logger)
	assert.NoError(t, err)

	tid0 := txIDFromInt(0)
	has, err := mgr.Has(module.TransactionGroupNormal, tid0, 100)
	assert.NoError(t, err)
	assert.False(t, has)

	bk, err := dbase.GetBucket(db.TransactionLocatorByHash)
	assert.NoError(t, err)

	assert.NoError(t, bk.Set(tid0, tid0))
	has, err = mgr.Has(module.TransactionGroupNormal, tid0, 100)
	assert.NoError(t, err)
	assert.True(t, has)
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
			id: txIDFromInt(i + start),
			ts: ts + delta*int64(i),
			g:  g,
		}
	}
	return &testTxList{ txs: txs }
}

func TestTracker_Basic(t *testing.T) {
	dbase := db.NewMapDB()
	logger := log.GlobalLogger()
	mgr, err := NewManager(dbase, logger)
	assert.NoError(t, err)
	mgr.Start()
	defer mgr.Term()

	var height int64 = 100
	var ts int64 = 2000
	var th int64 = 1000

	tid0 := txIDFromInt(0)
	bk, err := dbase.GetBucket(db.TransactionLocatorByHash)
	assert.NoError(t, err)
	assert.NoError(t, bk.Set(tid0, codec.BC.MustMarshalToBytes(module.TransactionLocator{
		height,
		module.TransactionGroupNormal,
		0,
	})))

	// test lookup the id with database
	has, err := mgr.Has(module.TransactionGroupNormal, tid0, 2000)
	assert.NoError(t, err)
	assert.True(t, has)

	tr0 := mgr.NewTracker(module.TransactionGroupNormal, height, ts-1, th)
	assert.NoError(t, tr0.Commit())


	tr1 := tr0.New(height, ts, th)

	has, err = tr1.Has(tid0, 2000)
	assert.NoError(t, err)
	assert.True(t, has)

	loc, err := mgr.GetLocator(tid0)
	assert.NoError(t, err)
	assert.EqualValues(t, height, loc.BlockHeight)
	assert.EqualValues(t, module.TransactionGroupNormal, loc.TransactionGroup)
	assert.EqualValues(t, 0, loc.IndexInGroup)

	txs1 := newTestTXList(module.TransactionGroupNormal, 1, 10, ts, 100)
	cnt, err := tr1.Add(txs1, false)
	assert.NoError(t, err)
	assert.EqualValues(t, len(txs1.txs), cnt)

	for _, tx := range txs1.txs {
		has, err := tr1.Has(tx.ID(), tx.Timestamp())
		assert.NoError(t, err)
		assert.True(t, has)
	}

	tr2 := tr1.New(height, ts+th, th)

	// transactions with invalid timestamp (out of scope)
	for _, tx := range txs1.txs {
		has, err := tr2.Has(tx.ID(), tx.Timestamp()+th)
		assert.NoError(t, err)
		assert.False(t, has)
	}

	txs2 := newTestTXList(module.TransactionGroupNormal, 11, 10, ts+th, 100)
	cnt, err = tr2.Add(txs2, false)
	assert.NoError(t, err)
	assert.EqualValues(t, len(txs2.txs), cnt)

	assert.NoError(t, tr1.Commit())

	time.Sleep(10*time.Millisecond)

	for _, tx := range txs1.txs {
		has, err = tr2.Has(tx.ID(), tx.Timestamp())
		assert.NoError(t, err)
		assert.True(t, has)
		has, err =  mgr.Has(tx.Group(), tx.ID(), tx.Timestamp())
		assert.NoError(t, err)
		assert.True(t, has)
	}
}

func TestManager_GetLocator(t *testing.T) {
	dbase := db.NewMapDB()
	layer := db.NewLayerDB(dbase)
	logger := log.GlobalLogger()
	mgr, err := NewManager(layer, logger)
	assert.NoError(t, err)
	mgr.Start()
	defer mgr.Term()

	var height int64 = 100
	var ts int64 = 2000
	var th int64 = 1000
	const PatchOffset = 1000_000

	ntr := mgr.NewTracker(module.TransactionGroupNormal, height, ts, th)
	ptr := mgr.NewTracker(module.TransactionGroupPatch, height, ts, th)

	heightBase := height
	tsBase := ts
	for i := 0 ; i<100 ; i++ {
		height = height+1
		ts = ts+100

		nntr := ntr.New(height, ts, th)
		nptr := ptr.New(height, ts, th)

		ntxs := newTestTXList(module.TransactionGroupNormal, i*10, 10, ts, 100)
		cnt, err := nntr.Add(ntxs, false)
		assert.NoError(t, err)
		assert.EqualValues(t, len(ntxs.txs), cnt)
		assert.NoError(t, nntr.Commit())
		ntr = nntr

		ptxs := newTestTXList(module.TransactionGroupPatch, PatchOffset + i*10, 10, ts, 100)
		cnt, err = nptr.Add(ptxs, false)
		assert.NoError(t, err)
		assert.EqualValues(t, len(ptxs.txs), cnt)
		assert.NoError(t, nptr.Commit())
		ptr = nptr
	}

	height = heightBase
	ts = tsBase
	for i := 0 ; i<100 ; i++ {
		height = height+1
		ts = ts+100

		ntxs := newTestTXList(module.TransactionGroupNormal, i*10, 10, ts, 100)
		for idx, tx := range ntxs.txs {
			loc, err := mgr.GetLocator(tx.ID())
			assert.NoError(t, err)
			assert.NotNilf(t, loc, "Height Offset:%d Index Offset:%d", i, idx)
			assert.EqualValues(t, module.TransactionGroupNormal, loc.TransactionGroup)
			assert.EqualValues(t, height, loc.BlockHeight)
			assert.EqualValues(t, idx, loc.IndexInGroup)
		}
		ptxs := newTestTXList(module.TransactionGroupPatch, PatchOffset + i*10, 10, ts, 100)
		for idx, tx := range ptxs.txs {
			loc, err := mgr.GetLocator(tx.ID())
			assert.NoError(t, err)
			assert.NotNilf(t, loc, "Height Offset:%d Index Offset:%d", i, idx)
			assert.EqualValues(t, module.TransactionGroupPatch, loc.TransactionGroup)
			assert.EqualValues(t, height, loc.BlockHeight)
			assert.EqualValues(t, idx, loc.IndexInGroup)
		}
	}

	assert.NoError(t, layer.Flush(false))

	minTS := ts-th

	height = heightBase
	ts = tsBase
	for i := 0 ; i<100 ; i++ {
		height = height+1
		ts = ts+100

		ntxs := newTestTXList(module.TransactionGroupNormal, i*10, 10, ts, 100)
		for idx, tx := range ntxs.txs {
			loc, err := mgr.GetLocator(tx.ID())
			assert.NoError(t, err)
			if ts+th <= minTS {
				assert.Nil(t, loc)
			} else {
				assert.NotNilf(t, loc, "Height Offset:%d Index Offset:%d", i, idx)
				assert.EqualValues(t, module.TransactionGroupNormal, loc.TransactionGroup)
				assert.EqualValues(t, height, loc.BlockHeight)
				assert.EqualValues(t, idx, loc.IndexInGroup)
			}
		}
		ptxs := newTestTXList(module.TransactionGroupPatch, PatchOffset + i*10, 10, ts, 100)
		for idx, tx := range ptxs.txs {
			loc, err := mgr.GetLocator(tx.ID())
			assert.NoError(t, err)
			if ts+th <= minTS {
				assert.Nil(t, loc)
			} else {
				assert.NotNilf(t, loc, "Height Offset:%d Index Offset:%d", i, idx)
				assert.EqualValues(t, module.TransactionGroupPatch, loc.TransactionGroup)
				assert.EqualValues(t, height, loc.BlockHeight)
				assert.EqualValues(t, idx, loc.IndexInGroup)
			}
		}
	}
}

func TestTracker_Add(t *testing.T) {
	dbase := db.NewMapDB()
	layer := db.NewLayerDB(dbase)
	logger := log.GlobalLogger()
	mgr, err := NewManager(layer, logger)
	assert.NoError(t, err)
	mgr.Start()
	defer mgr.Term()

	var height int64 = 100
	var ts int64 = 2000
	var th int64 = 1000

	ntr1 := mgr.NewTracker(module.TransactionGroupNormal, height, ts, th)

	// invalid timestamp range
	txs1 := newTestTXList(module.TransactionGroupNormal, 0, 10, ts+th, 100)
	_, err = ntr1.Add(txs1, false)
	assert.Error(t, err)

	txs2 := newTestTXList(module.TransactionGroupNormal, 0, 10, ts, 100)

	// normal addition
	cnt, err := ntr1.Add(txs2, false)
	assert.NoError(t, err)
	assert.EqualValues(t, len(txs2.txs), cnt)

	// duplicate addition 1
	_, err = ntr1.Add(txs2, false)
	assert.Error(t, err)

	height += 1
	ts += 100
	ntr2 := ntr1.New(height, ts, th)
	assert.NoError(t, ntr1.Commit())

	// duplicates addition 2
	_, err = ntr2.Add(txs2, false)
	assert.Error(t, err)

	txs3 := newTestTXList(module.TransactionGroupNormal, 10, 10, ts, 100)

	// addition to committed
	_, err = ntr1.Add(txs3, false)
	assert.Error(t, err)
}