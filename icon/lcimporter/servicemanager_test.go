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

package lcimporter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/icdb"
	"github.com/icon-project/goloop/icon/merkle/hexary"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

type testChain struct {
	module.Chain
	log log.Logger
	idb db.Database
}

func (c *testChain) Logger() log.Logger {
	return c.log
}

func (c *testChain) Database() db.Database {
	return c.idb
}

type testProofStorage struct {
	Root  []byte
	Size  int64
	Votes *blockv0.BlockVoteList
}

func (t *testProofStorage) GetBlockV1Proof() (*hexary.MerkleHeader, *blockv0.BlockVoteList, error) {
	if len(t.Root) > 0 {
		return &hexary.MerkleHeader{
			RootHash: t.Root,
			Leaves:   t.Size,
		}, t.Votes, nil
	} else {
		return nil, nil, common.ErrNotFound
	}
}

func (t *testProofStorage) SetBlockV1Proof(root []byte, size int64, votes *blockv0.BlockVoteList) error {
	if len(t.Root) == 0 {
		t.Root = root
		t.Size = size
		t.Votes = votes
		return nil
	} else {
		return common.ErrInvalidState
	}
}

type testTransitionCallback chan error

func (tcb testTransitionCallback) OnValidate(_ module.Transition, err error) {
	tcb <- err
}

func (tcb testTransitionCallback) OnExecute(_ module.Transition, err error) {
	tcb <- err
}

type testResultCallback chan error

func (trb testResultCallback) OnResult(err error) {
	trb <- err
}

func TestServiceManager_Basic(t *testing.T) {
	rdb := db.NewMapDB()
	idb := db.NewMapDB()
	logger := log.GlobalLogger()
	bc := newTestBlockConverter(rdb)

	ex, err := NewExecutorWithBC(rdb, idb, logger, bc)
	assert.NoError(t, err)
	chain := &testChain {
		log: logger,
		idb: idb,
	}
	vls := []*common.Address{
		common.MustNewAddressFromString("hx01"),
		common.MustNewAddressFromString("hx02"),
		common.MustNewAddressFromString("hx03"),
	}
	ps := &testProofStorage{}
	sm, err := NewServiceManagerWithExecutor(chain, ex, ps, vls, nil)
	assert.NoError(t, err)

	vl, err := newValidatorListFromSlice(idb, vls)
	assert.NoError(t, err)

	sm.Start()

	tr0, err := sm.CreateInitialTransition(nil, nil)
	assert.NoError(t, err)

	t.Log("start and finalize block0 (genesis)")
	//  GENESIS Transition
	height := int64(0)
	ts := int64(0)
	gtx := buildTestTx(0,  "GENESIS")
	gtxs := transaction.NewTransactionListFromSlice(idb, []module.Transaction{gtx})
	tr1, err := sm.CreateTransition(tr0, gtxs, common.NewBlockInfo(height, ts), nil, true)
	assert.NotNil(t, tr1)
	assert.NoError(t, err)
	tcb := testTransitionCallback(make(chan error, 1))
	_, err = tr1.Execute(tcb)
	assert.NoError(t, err)
	assert.NoError(t, <-tcb)
	assert.NoError(t, <-tcb)
	err = sm.Finalize(tr1, module.FinalizeResult|module.FinalizeNormalTransaction|module.FinalizePatchTransaction)
	assert.NoError(t, err)

	assert.Equal(t, vl.Hash(), tr1.NextValidators().Hash())

	toBC := make(chan string, 3)
	toTC := make(chan string, 3)

	t.Log("prepare txs for block1")
	txs1 := buildTestTxs(0, 9, "OK")
	go func() {
		req := <-bc.channel
		assert.Equal(t, int64(0), req.from)
		assert.Equal(t, int64(-1), req.to)
		req.sendTxs(txs1[:5])

		time.Sleep(delayForConfirm)
		toTC<-"on_send_5"

		assert.Equal(t, "send_remain", <-toBC)
		req.sendTxs(txs1[5:])

		time.Sleep(delayForConfirm)
		toTC<-"on_send_10"

		assert.Equal(t, "interrupt", <-toBC)
		req.interrupt()
	}()

	assert.Equal(t, "on_send_5", <-toTC)

	t.Log("propose block1")
	// propose 0~4 transactions
	height += 1
	ts += 10
	tr2, err := sm.ProposeTransition(tr1, common.NewBlockInfo(height, ts), nil)
	assert.NotNil(t, tr2)
	assert.NoError(t, err)
	tcb = testTransitionCallback(make(chan error, 1))
	_, err = tr2.Execute(tcb)
	assert.NoError(t, err)

	t.Log("finalize block1")
	// pre validation success
	assert.NoError(t, <-tcb)
	err = sm.Finalize(tr1, module.FinalizeResult)
	assert.NoError(t, err)
	err = sm.Finalize(tr2, module.FinalizeNormalTransaction|module.FinalizePatchTransaction)
	assert.NoError(t, err)

	t.Log("check block1")
	// check result & transactions
	assert.Equal(t, vl.Hash(), tr1.NextValidators().Hash())
	tls1 := tr2.NormalTransactions()
	for itr, idx := tls1.Iterator(), 0 ; itr.Has() ; idx, _ = idx+1, itr.Next() {
		tx, i, err := itr.Get()
		assert.NoError(t, err)
		assert.Equal(t, idx, i)
		assert.Equal(t, txs1[idx], transaction.Unwrap(tx))
	}

	// execution success
	assert.NoError(t, <-tcb)

	t.Log("prepare txs for block2")
	toBC <- "send_remain"
	assert.Equal(t, "on_send_10", <-toTC)

	t.Log("propose block2")
	// propose 5~9 transactions
	height += 1
	ts += 10
	tr3, err := sm.ProposeTransition(tr2, common.NewBlockInfo(height, ts), nil)
	assert.NotNil(t, tr3)
	assert.NoError(t, err)
	tcb = testTransitionCallback(make(chan error, 1))
	_, err = tr3.Execute(tcb)
	assert.NoError(t, err)

	t.Log("finalize block2")
	// pre validation success
	assert.NoError(t, <-tcb)
	err = sm.Finalize(tr2, module.FinalizeResult)
	assert.NoError(t, err)
	err = sm.Finalize(tr3, module.FinalizeNormalTransaction|module.FinalizePatchTransaction)
	assert.NoError(t, err)

	t.Log("check block2")
	// check result & transactions
	assert.Equal(t, vl.Hash(), tr1.NextValidators().Hash())
	tls2 := tr3.NormalTransactions()
	for itr, idx := tls2.Iterator(), 0 ; itr.Has() ; idx, _ = idx+1, itr.Next() {
		tx, i, err := itr.Get()
		assert.NoError(t, err)
		assert.Equal(t, idx, i)
		assert.Equal(t, txs1[idx+5], transaction.Unwrap(tx))
	}

	// execution success
	assert.NoError(t, <-tcb)

	result1 := tr2.Result()
	vh1 := tr2.NextValidators().Hash()
	t.Logf("TXH=%#x", tls2.Hash())
	txh1 := tr3.NormalTransactions().Hash()
	t.Logf("Finalized: result=%#x, vh=%#x, txh=%#x", result1, vh1, txh1)

	toBC <- "interrupt"
	sm.Term()

	t.Log("Restart chain")

	bc = newTestBlockConverter(rdb)
	ex, err = NewExecutorWithBC(rdb, idb, logger, bc)
	assert.NoError(t, err)

	trb := testResultCallback(make(chan error, 1))
	sm, err = NewServiceManagerWithExecutor(chain, ex, ps, vls, trb)
	assert.NoError(t, err)

	txs2 := buildTestTxs(10, 19, "OK")
	go func() {
		// req := <-bc.channel
		// t.Log("BC reload from finalized")
		// assert.Equal(t, int64(10), req.from)
		// assert.Equal(t, int64(-1), req.to)
		//
		// req.interrupt()
		toTC <- "confirm_start"

		req := <-bc.channel
		t.Log("BC reload for last transition")
		assert.Equal(t, int64(5), req.from)
		assert.Equal(t, int64(-1), req.to)
		req.sendTxs(txs1[5:])

		t.Log("BC send new transactions")
		req.sendTxs(txs2)
		time.Sleep(delayForConfirm)
		toTC <- "confirm_send_new"
		req.end(19)
	}()

	sm.Start()

	t.Log("transition from block2")
	vl1, err := state.ValidatorSnapshotFromHash(idb, vh1)
	assert.NoError(t, err)
	tr2, err = sm.CreateInitialTransition(result1, vl1)
	assert.NoError(t, err)

	tls2 = transaction.NewTransactionListFromHash(idb, txh1)
	err = sm.Finalize(tr2, module.FinalizeResult)
	assert.NoError(t, err)

	assert.Equal(t, "confirm_start", <-toTC)

	tr3, err = sm.CreateTransition(tr2, tls2, common.NewBlockInfo(height, ts), nil, true)
	assert.NoError(t, err)

	tcb = testTransitionCallback(make(chan error, 1))
	_, err = tr3.Execute(tcb)
	assert.NoError(t, err)

	// pre validation success
	assert.NoError(t, <-tcb)

	// execution success
	assert.NoError(t, <-tcb)

	assert.Equal(t, "confirm_send_new", <-toTC)

	// process 10~19 transactions
	txo := 0

	trp := tr3
	var trc module.Transition
	for txo < 10 {
		height += 1
		ts += 10
		t.Logf("propose block height=%d offset=%d", height, txo)
		trc, err = sm.ProposeTransition(trp, common.NewBlockInfo(height, ts), nil)
		assert.NoError(t, err)

		tcb = testTransitionCallback(make(chan error, 1))
		_, err = trc.Execute(tcb)
		assert.NoError(t, err)

		t.Log("finalize block", height)
		// pre validation success
		assert.NoError(t, <-tcb)
		err = sm.Finalize(trp, module.FinalizeResult)
		assert.NoError(t, err)
		err = sm.Finalize(trc, module.FinalizeNormalTransaction|module.FinalizePatchTransaction)
		assert.NoError(t, err)

		t.Log("check block", height)
		// check result & transactions
		assert.Equal(t, vl.Hash(), trc.NextValidators().Hash())
		tls3 := trc.NormalTransactions()
		txsum := 0
		for itr := tls3.Iterator() ; itr.Has() ; txo, _ = txo+1, itr.Next() {
			tx, _, err := itr.Get()
			assert.NoError(t, err)
			assert.Equal(t, txs2[txo], transaction.Unwrap(tx))
			txsum += int(txs2[txo].TXCount)
		}
		assert.LessOrEqual(t, txsum, TransactionsPerBlock)

		// execution success
		assert.NoError(t, <-tcb)

		trp = trc
	}

	for i := 0 ; i<2 ; i++ {
		assert.False(t, sm.Finished())

		height += 1
		ts += 10
		t.Log("propose block", height)
		trc, err = sm.ProposeTransition(trp, common.NewBlockInfo(height, ts), nil)
		assert.NoError(t, err)

		tcb = testTransitionCallback(make(chan error, 1))
		_, err = trc.Execute(tcb)
		assert.NoError(t, err)

		t.Log("finalize block", height)
		assert.NoError(t, <-tcb)
		err = sm.Finalize(trp, module.FinalizeResult)
		assert.NoError(t, err)
		err = sm.Finalize(trc, module.FinalizeNormalTransaction|module.FinalizePatchTransaction)
		assert.NoError(t, err)

		trp = trc
	}

	assert.True(t, sm.Finished())

	select {
	case err := <- trb:
		assert.Failf(t, "Got error", "err=%+v", err)
	default:
		// do nothing
	}

	tmp_db := db.NewMapDB()
	tmp, err := tmp_db.GetBucket(db.MerkleTrie)
	assert.NoError(t, err)
	real, err := tmp_db.GetBucket(icdb.BlockMerkle)
	assert.NoError(t, err)
	acc, err := hexary.NewAccumulator(tmp, real, "")
	assert.NoError(t, err)
	for _, tx := range txs1 {
		err = acc.Add(tx.BlockHash)
		assert.NoError(t, err)
	}
	for _, tx := range txs2 {
		err = acc.Add(tx.BlockHash)
		assert.NoError(t, err)
	}
	mh, err := acc.Finalize()
	assert.NoError(t, err)

	assert.Equal(t, mh.RootHash, ps.Root)
	assert.Equal(t, mh.Leaves, ps.Size)
	assert.True(t, bc.votes == ps.Votes)

	sm.Term()
}
