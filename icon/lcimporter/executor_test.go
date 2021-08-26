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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/icdb"
	"github.com/icon-project/goloop/icon/merkle/hexary"
)

const delayForConfirm = 10*time.Millisecond

func buildTestTx(height int64, suffix string) *BlockTransaction {
	return &BlockTransaction{
		Height:        height,
		BlockHash:     crypto.SHA3Sum256([]byte(fmt.Sprintf("BLOCKID[%d,%s]", height, suffix))),
		Result:        []byte(fmt.Sprintf("RESULT[%d,%s]", height, suffix)),
		ValidatorHash: []byte(fmt.Sprintf("VALIDATOR[%d,%s]", height, suffix)),
		TXCount:       TransactionsPerBlock/6,
	}
}

func buildTestTxs(from, to int64, suffix string) []*BlockTransaction {
	txs := make([]*BlockTransaction, 0, int(to-from+1))
	for height := from; height <= to; height += 1 {
		tx := buildTestTx(height, suffix)
		txs = append(txs, tx)
	}
	return txs
}

type testBCRequest struct {
	from, to int64
	txs      []*BlockTransaction
	bc       *testBlockConverter
	channel  chan interface{}
}

func (r *testBCRequest) sendTxs(txs []*BlockTransaction) {
	for _, tx := range txs {
		r.channel <- tx
	}
}

func (r *testBCRequest) generateTxs(from, to int64, suffix string) {
	for height := from; height <= to; height += 1 {
		tx := buildTestTx(height, suffix)
		r.channel <- tx
	}
}

func (r *testBCRequest) interrupt() {
	r.channel <- errors.ErrInterrupted
	close(r.channel)
}

func (r *testBCRequest) end(h int64) {
	r.bc.setLastHeight(h)
	close(r.channel)
}

type testBlockConverter struct {
	channel chan *testBCRequest
	last    int64
	votes   *blockv0.BlockVoteList
}

func (t *testBlockConverter) Rebase(from, to int64, txs []*BlockTransaction) (<-chan interface{}, error) {
	if t.last > 0 && from >= t.last {
		return nil, ErrAfterLastBlock
	}
	req := &testBCRequest{
		bc:      t,
		from:    from,
		to:      to,
		txs:     txs,
		channel: make(chan interface{}, 1),
	}
	t.channel <- req
	return req.channel, nil
}

func (t *testBlockConverter) Term() {
	close(t.channel)
}

func (t *testBlockConverter) GetBlockVotes(h int64) (*blockv0.BlockVoteList, error) {
	return t.votes, nil
}

func (t *testBlockConverter) setLastHeight(h int64) {
	t.last = h
}

func newTestBlockConverter(rdb db.Database) *testBlockConverter {
	return &testBlockConverter{
		channel: make(chan *testBCRequest, 1),
		votes:   new(blockv0.BlockVoteList),
	}
}

func newTestAcc(t *testing.T) hexary.Accumulator {
	dbase := db.NewMapDB()
	bmk, err := dbase.GetBucket(icdb.BlockMerkle)
	assert.NoError(t, err)
	tmp, err := dbase.GetBucket(icdb.IDToHash)
	acc, err := hexary.NewAccumulator(tmp, bmk, "")
	assert.NoError(t, err)
	return acc
}

func applyTxsOnAcc(t *testing.T, acc hexary.Accumulator, txs []*BlockTransaction) {
	for _, tx := range txs {
		err := acc.Add(tx.BlockHash)
		assert.NoError(t, err)
	}
}

func TestExecutor_Basic(t *testing.T) {
	rdb := db.NewMapDB()
	idb := db.NewMapDB()
	logger := log.GlobalLogger()
	bc := newTestBlockConverter(rdb)
	ex, err := NewExecutorWithBC(rdb, idb, logger, bc)
	assert.NoError(t, err)

	t.Log("start executor")
	err = ex.Start()
	assert.NoError(t, err)
	t.Log("executor started")

	toTC := make(chan string, 3)
	toBC := make(chan string, 3)

	txs1 := buildTestTxs(0, 9, "OK")
	go func() {
		req := <- bc.channel
		t.Log("request received")
		assert.Equal(t, int64(0), req.from)
		assert.Equal(t, int64(-1), req.to)
		assert.Nil(t, req.txs)
		t.Log("sending 0~4")
		req.sendTxs(txs1[0:5])

		time.Sleep(delayForConfirm)
		toTC <- "on_send5"
		assert.Equal(t, "on_timeout", <-toBC)

		t.Log("sending 5~9")
		req.sendTxs(txs1[5:])

		assert.Equal(t, "quit", <-toBC)
		req.interrupt()
	}()
	_, err = ex.GetTransactions(0, 9, func(txs []*BlockTransaction, err error) {
		t.Logf("transaction arrives size=%d", len(txs))
		assert.NoError(t, err)
		assert.Equal(t, txs1, txs)

		toTC <- "on_receive_10"
	})
	assert.NoError(t, err)
	assert.Equal(t, "on_send5", <-toTC)
	select {
	case <-time.After(delayForConfirm):
		// do nothing
		toBC <- "on_timeout"
	case msg := <-toTC:
		assert.Failf(t, "unexpected msg", "msg=%s", msg)
	}
	assert.Equal(t, "on_receive_10", <-toTC)

	toBC <- "quit"
	time.Sleep(delayForConfirm)
	ex.Term()
}

func TestExecutor_Propose(t *testing.T) {
	rdb := db.NewMapDB()
	idb := db.NewMapDB()
	logger := log.GlobalLogger()
	bc := newTestBlockConverter(rdb)
	ex, err := NewExecutorWithBC(rdb, idb, logger, bc)
	assert.NoError(t, err)

	t.Log("start executor")
	err = ex.Start()
	assert.NoError(t, err)
	err = ex.FinalizeTransactions(-1)
	assert.NoError(t, err)
	t.Log("executor started")

	toTC := make(chan string, 3)
	toBC := make(chan string, 3)

	txs1 := buildTestTxs(0, 9, "OK")
	go func() {
		req := <- bc.channel
		t.Log("request received")
		assert.Equal(t, int64(0), req.from)
		assert.Equal(t, int64(-1), req.to)
		assert.Nil(t, req.txs)
		t.Log("sending 0~4")
		req.sendTxs(txs1[0:5])

		time.Sleep(delayForConfirm)
		toTC <- "on_send_5"
		assert.Equal(t, "send_remain", <-toBC)

		t.Log("sending 5~9")
		req.sendTxs(txs1[5:])

		time.Sleep(delayForConfirm)
		toTC <- "on_send_10"
		req.interrupt()
	}()

	height := 0
	assert.Equal(t, "on_send_5", <-toTC)
	txs, err := ex.ProposeTransactions(int64(height))
	assert.NoError(t, err)
	assert.Equal(t, txs1[0:5], txs)
	height += len(txs)
	toBC <- "send_remain"

	err = ex.FinalizeTransactions(int64(height-1))
	assert.NoError(t, err)

	assert.Equal(t, "on_send_10", <-toTC)

	txs, err = ex.ProposeTransactions(int64(height))
	assert.NoError(t, err)
	assert.Equal(t, txs1[5:], txs)
	height += len(txs)

	err = ex.FinalizeTransactions(int64(height-1))
	assert.NoError(t, err)

	ex.Term()

	t.Log("continue from 10")
	bc = newTestBlockConverter(rdb)
	ex, err = NewExecutorWithBC(rdb, idb, logger, bc)
	assert.NoError(t, err)
	err = ex.Start()
	assert.NoError(t, err)
	err = ex.FinalizeTransactions(int64(height-1))
	assert.NoError(t, err)

	go func() {
		req := <-bc.channel
		assert.Equal(t, int64(10), req.from)
		assert.Equal(t, int64(-1), req.to)
		assert.Nil(t, req.txs)

		t.Log("sending 5 more")
		txs := buildTestTxs(req.from, req.from+4, "OK")
		req.sendTxs(txs)
		time.Sleep(delayForConfirm)
		toTC <- "on_send_15"

		assert.Equal(t, "quit", <-toBC)
		req.interrupt()
	}()

	assert.Equal(t, "on_send_15", <-toTC)
	t.Log("propose and finalize to=14")
	txs2, err := ex.ProposeTransactions(int64(height))
	assert.NoError(t, err)
	assert.Equal(t, 5, len(txs2))
	height += len(txs2)
	err = ex.FinalizeTransactions(int64(height-1))
	assert.NoError(t, err)

	t.Log("cleanup")
	toBC <- "quit"
	ex.Term()
}

func TestExecutor_SyncTransactions(t *testing.T) {
	rdb := db.NewMapDB()
	idb := db.NewMapDB()
	logger := log.GlobalLogger()
	bc := newTestBlockConverter(rdb)
	ex, err := NewExecutorWithBC(rdb, idb, logger, bc)
	assert.NoError(t, err)

	t.Log("start executor")
	err = ex.Start()
	assert.NoError(t, err)
	err = ex.FinalizeTransactions(-1)
	assert.NoError(t, err)
	t.Log("executor started")

	acc := newTestAcc(t)

	toTest := make(chan string, 3)
	toBC := make(chan string, 3)

	txs1 := buildTestTxs(0, 9, "OK")
	txs2 := buildTestTxs(0, 9, "OTHER")
	go func() {
		req := <- bc.channel
		t.Log("request received")
		assert.Equal(t, int64(0), req.from)
		assert.Equal(t, int64(-1), req.to)
		assert.Nil(t, req.txs)
		t.Log("sending 0~9")
		req.sendTxs(txs1[0:5])

		assert.Equal(t, "send_old_remain", <-toBC)

		req.sendTxs(txs1[5:])

		toTest <- "after_send_old_remain"

		assert.Equal(t, "interrupt1", <-toBC)
		req.interrupt()
	}()

	t.Log("try to get 0~9 (should block)")

	_, err = ex.GetTransactions(0, 9, func(txs []*BlockTransaction, err error) {
		assert.Error(t, err)

		toTest <- "on_failure_for_previous"
	})
	assert.NoError(t, err)

	t.Log("try to get 0~1 (trigger failure on previous request)")

	_, err = ex.GetTransactions(0, 1, func(txs []*BlockTransaction, err error) {
		assert.NoError(t, err)
		assert.Equal(t, txs1[0:2], txs)

		toTest <- "on_receive_old_2"
	})
	assert.NoError(t, err)

	t.Log("wait for results of both GetTransactions()")

	msgs := append([]string{}, <-toTest)
	msgs = append(msgs, <-toTest)
	t.Log("check failure of get(0~9)")
	assert.Contains(t, msgs, "on_failure_for_previous")

	t.Log("check result of get(0~1)")
	assert.Contains(t, msgs, "on_receive_old_2")

	mh, err := ex.GetMerkleHeader(0)
	assert.NoError(t, err)
	assert.Equal(t, &hexary.MerkleHeader{nil, 0}, mh)

	err = ex.FinalizeTransactions(1)
	assert.NoError(t, err)

	t.Log("try to get 2~4 (should success)")
	_, err = ex.GetTransactions(2, 4, func(txs []*BlockTransaction, err error) {
		assert.NoError(t, err)
		assert.Equal(t, txs1[2:5], txs)

		toTest <- "confirm_2~4"
	})
	mh, err = ex.GetMerkleHeader(2)
	assert.NoError(t, err)
	applyTxsOnAcc(t, acc, txs1[0:2])
	assert.Equal(t, acc.GetMerkleHeader(), mh)

	t.Log("check result of get(2~4)")
	assert.Equal(t, "confirm_2~4", <-toTest)

	err = ex.SyncTransactions(txs2[2:5])
	assert.NoError(t, err)

	toBC <- "send_old_remain"
	assert.Equal(t, "after_send_old_remain", <-toTest)

	go func() {
		req := <- bc.channel
		toBC <- "interrupt1"
		t.Log("sync request received")
		assert.Equal(t, int64(2), req.from)
		assert.Equal(t, int64(-1), req.to)
		assert.Equal(t, txs2[2:5], req.txs)

		req.sendTxs(txs2[2:])

		assert.Equal(t, "interrupt2", <-toBC)
		req.interrupt()
	}()

	t.Log("wait for new 2~4")
	_, err = ex.GetTransactions(2, 4, func(txs []*BlockTransaction, err error) {
		t.Log("receive new 3")
		assert.NoError(t, err)
		assert.Equal(t, txs2[2:5], txs)

		toTest <- "on_receive_new_3"
	})
	t.Log("check result for new 3")
	assert.Equal(t, "on_receive_new_3", <-toTest)

	mh, err = ex.GetMerkleHeader(2)
	assert.NoError(t, err)
	assert.Equal(t, acc.GetMerkleHeader(), mh)

	t.Log("finalize to=4")
	err = ex.FinalizeTransactions(4)
	assert.NoError(t, err)

	canceler, err := ex.GetTransactions(5, 9, func(txs []*BlockTransaction, err error) {
		t.Log("receive new 5")
		assert.NoError(t, err)
		assert.Equal(t, txs2[5:], txs)

		toTest <- "on_success"
	})
	assert.NoError(t, err)

	assert.Equal(t, "on_success", <-toTest)

	mh, err = ex.GetMerkleHeader(5)
	applyTxsOnAcc(t, acc, txs2[2:5])
	assert.NoError(t, err)
	assert.Equal(t, acc.GetMerkleHeader(), mh)

	canceler, err = ex.GetTransactions(10, 10, func(txs []*BlockTransaction, err error) {
		t.Log("expected failure from cancelation")
		assert.Error(t, err)

		toTest <- "on_expected_failure"
	})
	assert.NoError(t, err)

	canceler()

	select {
	case <- time.After(time.Millisecond*100):
		assert.Fail(t, "Timeout to receive result")
	case v := <-toTest:
		assert.Equal(t, "on_expected_failure", v)
	}
	toBC <- "interrupt2"
	ex.Term()
}

func TestExecutor_Term(t *testing.T) {
	rdb := db.NewMapDB()
	idb := db.NewMapDB()
	logger := log.GlobalLogger()
	bc := newTestBlockConverter(rdb)
	ex, err := NewExecutorWithBC(rdb, idb, logger, bc)
	assert.NoError(t, err)

	t.Log("start executor")
	err = ex.Start()
	assert.NoError(t, err)
	t.Log("executor started")

	toTest := make(chan string, 3)
	toBC := make(chan string, 3)

	txs1 := buildTestTxs(0, 9, "OK")
	go func() {
		req := <- bc.channel
		t.Log("request received")
		assert.Equal(t, int64(0), req.from)
		assert.Equal(t, int64(-1), req.to)
		assert.Nil(t, req.txs)
		t.Log("sending 0~8")
		req.sendTxs(txs1[:9])

		assert.Equal(t, "send_after_term", <-toBC)

		t.Log("sending 9")
		req.sendTxs(txs1[9:])
		req.interrupt()

		toTest<-"closed"
	}()

	_, err = ex.GetTransactions(0, 9, func(txs []*BlockTransaction, err error) {
		assert.Error(t, err)
		toTest <- "got_error"
	})

	ex.Term()
	assert.Equal(t, "got_error", <-toTest)

	toBC <- "send_after_term"

	assert.Equal(t, "closed", <-toTest)
	time.Sleep(delayForConfirm)
}

func TestExecutor_LastBlock(t *testing.T) {
	rdb := db.NewMapDB()
	idb := db.NewMapDB()
	logger := log.GlobalLogger()
	bc := newTestBlockConverter(rdb)
	ex, err := NewExecutorWithBC(rdb, idb, logger, bc)
	assert.NoError(t, err)

	err = ex.Start()
	assert.NoError(t, err)
	err = ex.FinalizeTransactions(-1)
	assert.NoError(t, err)

	toTC := make(chan string, 3)
	toBC := make(chan string, 3)

	txs1 := buildTestTxs(0, 9, "OK")
	go func() {
		req := <-bc.channel
		t.Log("request received")
		assert.Equal(t, int64(0), req.from)
		assert.Equal(t, int64(-1), req.to)
		assert.Nil(t, req.txs)

		t.Log("sending 0~8")
		req.sendTxs(txs1[:9])
		time.Sleep(100*time.Millisecond)
		toTC <- "sent 0~8"

		assert.Equal(t, "send_last", <-toBC)

		t.Log("sending 9")
		req.sendTxs(txs1[9:])
		time.Sleep(delayForConfirm)
		toTC <- "send 9"

		req.end(9)

		toTC <- "closed"
	}()

	height := 0

	assert.Equal(t, "sent 0~8", <-toTC)
	txs, err := ex.ProposeTransactions(int64(height))
	assert.NoError(t, err)
	assert.Equal(t, txs1[:len(txs)], txs)
	height += len(txs)
	t.Logf("set height=%d", height)

	err = ex.FinalizeTransactions(int64(height-1))
	assert.NoError(t, err)

	toBC <- "send_last"
	assert.Equal(t, "send 9", <-toTC)
	for height < 10 {
		txs, err = ex.ProposeTransactions(int64(height))
		assert.NoError(t, err)
		assert.Equal(t, txs1[height:height+len(txs)], txs)
		height += len(txs)

		err = ex.FinalizeTransactions(int64(height-1))
		assert.NoError(t, err)
		t.Logf("set height=%d", height)
	}

	assert.Equal(t, "closed", <-toTC)
	time.Sleep(delayForConfirm)

	t.Log("start last propose")
	txs, err = ex.ProposeTransactions(int64(height))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrAfterLastBlock))

	ex.Term()
}