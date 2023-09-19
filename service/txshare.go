/*
 * Copyright 2020 ICON Foundation
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
	"container/list"
	"encoding/binary"
	"math/big"
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
)

const (
	taskBufferSize              = 5
	handleTxRequestIntervalBase = time.Millisecond * 100
	handleTxRequestIntervalMin  = time.Millisecond * 10
	sendTxRequestIntervalBase   = time.Millisecond * 1000
	sendTxRequestIntervalMin    = time.Millisecond * 950

	maxNumberOfTransactionsToSend = 50
	txRequestActivateWatermark    = 0.1
	txRequestDeactivateWatermark  = 0.3
	maxTxCountForBloomElement     = 50
	txBloomBits                   = 12
)

type TxBloom struct {
	Bits  uint
	Bloom big.Int
}

func (b *TxBloom) Contains(id []byte) bool {
	var filter big.Int
	var result big.Int
	if b.Bits == 0 {
		return false
	}
	if len(id) < crypto.HashLen {
		id = crypto.SHA3Sum256(id)
	}
	mask := (1 << b.Bits) - 1
	idx1 := int(binary.BigEndian.Uint16(id[:])) & mask
	idx2 := int(binary.BigEndian.Uint16(id[2:])) & mask
	idx3 := int(binary.BigEndian.Uint16(id[4:])) & mask
	filter.SetBit(&filter, idx1, 1)
	filter.SetBit(&filter, idx2, 1)
	filter.SetBit(&filter, idx3, 1)
	result.And(&filter, &b.Bloom)
	return result.Cmp(&filter) == 0
}

func (b *TxBloom) Add(id []byte) {
	if b.Bits == 0 {
		b.Bits = txBloomBits
	}
	if len(id) < crypto.HashLen {
		id = crypto.SHA3Sum256(id)
	}
	idx1 := int(binary.BigEndian.Uint16(id[:])) & ((1 << b.Bits) - 1)
	idx2 := int(binary.BigEndian.Uint16(id[2:])) & ((1 << b.Bits) - 1)
	idx3 := int(binary.BigEndian.Uint16(id[4:])) & ((1 << b.Bits) - 1)
	b.Bloom.SetBit(&b.Bloom, idx1, 1)
	b.Bloom.SetBit(&b.Bloom, idx2, 1)
	b.Bloom.SetBit(&b.Bloom, idx3, 1)
}

func (b *TxBloom) Merge(b2 *TxBloom) {
	if b2.Bits == 0 {
		return
	}
	if b.Bits != b2.Bits {
		if b.Bits != 0 {
			panic("InvalidMerge")
		}
		b.Bits = b2.Bits
	}
	b.Bloom.Or(&b.Bloom, &b2.Bloom)
}

func (b *TxBloom) ContainsAllOf(b2 *TxBloom) bool {
	if b2.Bits == 0 {
		return true
	}
	if b.Bits != b2.Bits {
		return false
	}
	var tmp big.Int
	tmp.And(&b.Bloom, &b2.Bloom)
	return tmp.Cmp(&b2.Bloom) == 0
}

type msgTransactionRequest struct {
	Bits  uint
	Bloom []byte
}

func (r *msgTransactionRequest) GetBloom() *TxBloom {
	bloom := new(TxBloom)
	bloom.Bits = r.Bits
	bloom.Bloom.SetBytes(common.Decompress(r.Bloom))
	return bloom
}

func (r *msgTransactionRequest) SetBloom(bloom *TxBloom) {
	r.Bits = bloom.Bits
	r.Bloom = common.Compress(bloom.Bloom.Bytes())
}

func (r *msgTransactionRequest) SetBytes(bs []byte) error {
	_, err := codec.BC.UnmarshalFromBytes(bs, r)
	return err
}

func (r *msgTransactionRequest) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(r)
}

type transactionRequest struct {
	lock sync.Mutex
	peer module.PeerID
	ts   time.Time
	msg  *msgTransactionRequest
	elem *list.Element
}

func (r *transactionRequest) GetBloom() *TxBloom {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.msg.GetBloom()
}

func (r *transactionRequest) SetRequest(msg *msgTransactionRequest) bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	now := time.Now()
	if now.Sub(r.ts) < sendTxRequestIntervalMin && r.elem == nil {
		return false
	}
	r.ts = now
	r.msg = msg
	return true
}

func newTransactionRequest(peer module.PeerID, msg *msgTransactionRequest) *transactionRequest {
	return &transactionRequest{
		peer: peer,
		msg:  msg,
		ts:   time.Now(),
	}
}

type TransactionShare struct {
	self module.Address
	tm   *TransactionManager
	ph   module.ProtocolHandler
	log  log.Logger

	lock       sync.Mutex
	requestMap map[string]*transactionRequest
	requests   list.List

	handlerTimer *time.Timer
	requestTimer *time.Timer

	updatingRequestTimer bool
	requestTimerActive   bool
	requestTimerEnabled  bool
	emptyHandlers        bool

	tasks chan func()
}

func (ts *TransactionShare) popRequest() (*transactionRequest, bool) {
	ts.lock.Lock()
	defer ts.lock.Unlock()

	e := ts.requests.Front()
	if e == nil {
		return nil, false
	}

	tr := ts.requests.Remove(e).(*transactionRequest)
	tr.elem = nil

	return tr, ts.requests.Len() > 0
}

func durationMax(a, b time.Duration) time.Duration {
	if a < b {
		return b
	} else {
		return a
	}
}

func (ts *TransactionShare) handleTxRequest() {
	current := time.Now()
	sentSum := 0
	tr, next := ts.popRequest()
	for tr != nil {
		bloom := tr.GetBloom()
		ts.log.Debugf("handleTxRequest(bits=%d,peer=%s)", bloom.Bits, tr.peer)
		txs := ts.tm.FilterTransactions(module.TransactionGroupNormal, bloom, maxNumberOfTransactionsToSend)
		sent := 0
		if len(txs) > 0 {
			ts.log.Infof("ShareTransactions(txs=%d,to=%s)", len(txs), tr.peer)
			for i, tx := range txs {
				if err := ts.ph.Unicast(protoResponseTransaction, tx.Bytes(), tr.peer); err != nil {
					ts.log.Debugf("Fail to send transaction at=%d err=%+v", i, err)
					break
				}
				sent += 1
			}
		}
		sentSum += sent
		if sent < maxNumberOfTransactionsToSend && next {
			tr, next = ts.popRequest()
		} else {
			tr = nil
		}
	}
	if next {
		delay := handleTxRequestIntervalBase - time.Since(current)
		ts.handlerTimer.Reset(durationMax(delay, handleTxRequestIntervalMin))
	}
}

func (ts *TransactionShare) sendTxRequest() {
	ts.lock.Lock()
	defer ts.lock.Unlock()

	enabled := ts.requestTimerEnabled && !ts.emptyHandlers
	if enabled {
		bloom := ts.tm.GetBloomOf(module.TransactionGroupNormal)
		ts.log.Debugf("sendTxRequest(bits=%d)", bloom.Bits)
		var msg msgTransactionRequest
		msg.SetBloom(bloom)
		if err := ts.ph.Broadcast(protoRequestTransaction, msg.Bytes(), module.BroadcastChildren); err != nil {
			if network.NotAvailableError.Equals(err) {
				ts.emptyHandlers = true
				enabled = false
			} else {
				ts.log.Debugf("Fail to broadcast TxRequest (err=%+v)", err)
			}
		}
		if enabled {
			ts.requestTimer.Reset(sendTxRequestIntervalBase)
		}
	}
	ts.requestTimerActive = enabled
}

func (ts *TransactionShare) handleUpdateRequestTimer() {
	ts.lock.Lock()
	defer ts.lock.Unlock()

	if (ts.requestTimerEnabled && !ts.emptyHandlers) && !ts.requestTimerActive {
		ts.requestTimer.Reset(0)
		ts.requestTimerActive = true
	}
	ts.updatingRequestTimer = false
}

func (ts *TransactionShare) taskLoop() {
loop:
	for {
		select {
		case task := <-ts.tasks:
			if task == nil {
				break loop
			}
			task()
		case <-ts.handlerTimer.C:
			ts.handleTxRequest()
		case <-ts.requestTimer.C:
			ts.sendTxRequest()
		}
	}
	ts.handlerTimer.Stop()
	ts.requestTimer.Stop()
}

func (ts *TransactionShare) Start(handler module.ProtocolHandler, wallet module.Wallet) {
	ts.ph = handler
	ts.self = wallet.Address()
	go ts.taskLoop()
}

func (ts *TransactionShare) Stop() {
	ts.tasks <- nil
}

func resetTimer(timer *time.Timer, dur time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(dur)
}

func (ts *TransactionShare) HandleRequestTransaction(buf []byte, peer module.PeerID) (bool, error) {
	req := new(msgTransactionRequest)
	if err := req.SetBytes(buf); err != nil {
		ts.log.Warn("InvalidPacket(TransactionRequest)")
		ts.log.Debugf("Failed to unmarshal msgTransactionRequest. buf=%x, err=%+v\n", buf, err)
		return false, err
	}
	ts.lock.Lock()
	defer ts.lock.Unlock()

	peerName := peer.String()

	tr, ok := ts.requestMap[peerName]
	if !ok {
		tr = newTransactionRequest(peer, req)
		ts.requestMap[peerName] = tr
	} else {
		if !tr.SetRequest(req) {
			return false, nil
		}
	}

	if tr.elem == nil {
		tr.elem = ts.requests.PushBack(tr)
		if ts.requests.Len() == 1 {
			ts.tasks <- func() {
				resetTimer(ts.handlerTimer, 0)
			}
		}
	}
	return false, nil
}

func (ts *TransactionShare) HandleJoin(peer module.PeerID) {
	ts.lock.Lock()
	defer ts.lock.Unlock()

	if ts.emptyHandlers {
		ts.emptyHandlers = false
		ts.updateRequestTimerInLock()
	}
}

func (ts *TransactionShare) HandleLeave(peer module.PeerID) {
	ts.lock.Lock()
	defer ts.lock.Unlock()

	name := peer.String()
	if tr, ok := ts.requestMap[name]; ok {
		delete(ts.requestMap, name)
		if tr.elem != nil {
			ts.requests.Remove(tr.elem)
			tr.elem = nil
		}
	}
}

func (ts *TransactionShare) updateRequestTimerInLock() {
	if ts.requestTimerEnabled && !ts.emptyHandlers && !ts.requestTimerActive && !ts.updatingRequestTimer {
		ts.updatingRequestTimer = true
		ts.tasks <- ts.handleUpdateRequestTimer
	}
}

func (ts *TransactionShare) EnableTxRequest(yn bool) {
	ts.lock.Lock()
	defer ts.lock.Unlock()

	if ts.requestTimerEnabled != yn {
		ts.log.Infof("EnableTxRequest(%v)", yn)
		ts.requestTimerEnabled = yn
		ts.updateRequestTimerInLock()
	}
}

func watermarkToCount(mark float32, size int) int {
	return int(mark * float32(size))
}

func (ts *TransactionShare) OnPoolCapacityUpdated(group module.TransactionGroup, size, used int) {
	if group != module.TransactionGroupNormal {
		return
	}
	if used < watermarkToCount(txRequestActivateWatermark, size) {
		go ts.EnableTxRequest(true)
	} else if used > watermarkToCount(txRequestDeactivateWatermark, size) {
		go ts.EnableTxRequest(false)
	}
}

func NewTransactionShare(tm *TransactionManager) *TransactionShare {
	ts := &TransactionShare{
		tm:           tm,
		log:          tm.Logger(),
		tasks:        make(chan func(), taskBufferSize),
		requestMap:   make(map[string]*transactionRequest),
		handlerTimer: time.NewTimer(handleTxRequestIntervalBase),
		requestTimer: time.NewTimer(handleTxRequestIntervalBase),
	}
	return ts
}
