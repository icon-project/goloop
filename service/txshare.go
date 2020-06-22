package service

import (
	"container/list"
	"math/big"
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const (
	taskBufferSize                 = 5
	handleTxRequestIntervalBase    = time.Millisecond * 100
	handleTxRequestIntervalMin     = time.Millisecond * 10
	sendTxRequestIntervalBase      = time.Millisecond * 1000
	sendTxRequestIntervalMin       = time.Millisecond * 900
	maxNumberOfTransactionsForSend = 10
	txRequestActivateWatermark     = 0.1
	txRequestDeactivateWatermark   = 0.3
)

type msgTransactionRequest struct {
	Peer      common.Address
	Bits      uint
	Bloom     []byte
	Timestamp int64
}

func (r *msgTransactionRequest) GetBloom() (uint, *big.Int) {
	return r.Bits, new(big.Int).SetBytes(common.Decompress(r.Bloom))
}

func (r *msgTransactionRequest) SetBloom(self module.Address, bits uint, value *big.Int) {
	r.Peer.SetBytes(self.Bytes())
	r.Bits = bits
	r.Bloom = common.Compress(value.Bytes())
	r.Timestamp = common.UnixMicroFromTime(time.Now())
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

func (r *transactionRequest) GetBloom() (uint, *big.Int) {
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
	tr, next := ts.popRequest()
	current := time.Now()
	if tr != nil {
		bits, bloom := tr.GetBloom()
		ts.log.Debugf("handleTxRequest(bits=%d,peer=%s)", bits, tr.peer)
		txs := ts.tm.FilterTransactions(module.TransactionGroupNormal,
			bits, bloom, maxNumberOfTransactionsForSend)
		if len(txs) > 0 {
			ts.log.Infof("ShareTransactions(txs=%d,to=%s)", len(txs), tr.peer)
			for i, tx := range txs {
				if err := ts.ph.Unicast(protoPropagateTransaction, tx.Bytes(), tr.peer); err != nil {
					ts.log.Debugf("Fail to send transaction at=%d err=%+v", i, err)
					break
				}
			}
		}
	}
	if next {
		delay := handleTxRequestIntervalBase - time.Now().Sub(current)
		ts.handlerTimer.Reset(durationMax(delay, handleTxRequestIntervalMin))
	}
}

func (ts *TransactionShare) sendTxRequest() {
	ts.lock.Lock()
	defer ts.lock.Unlock()

	if ts.requestTimerEnabled {
		bits, bloom := ts.tm.GetBloomOf(module.TransactionGroupNormal)
		ts.log.Debugf("sendTxRequest(bits=%d)", bits)
		var msg msgTransactionRequest
		msg.SetBloom(ts.self, bits, bloom)
		ts.ph.Broadcast(protoRequestTransaction, msg.Bytes(), module.BROADCAST_ALL)

		ts.requestTimer.Reset(sendTxRequestIntervalBase)
	}
	ts.requestTimerActive = ts.requestTimerEnabled
}

func (ts *TransactionShare) updateRequestTimer() {
	ts.lock.Lock()
	defer ts.lock.Unlock()

	if ts.requestTimerEnabled && !ts.requestTimerActive {
		ts.requestTimer.Reset(0)
		ts.requestTimerActive = true
	}
	ts.updatingRequestTimer = false
}

func (ts *TransactionShare) taskLoop() {
	for {
		select {
		case task := <-ts.tasks:
			if task == nil {
				break
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

func (ts *TransactionShare) EnableTxRequest(yn bool) {
	ts.lock.Lock()
	defer ts.lock.Unlock()

	if ts.requestTimerEnabled != yn {
		ts.log.Infof("EnableTxRequest(%v)", yn)
		ts.requestTimerEnabled = yn
		if ts.requestTimerEnabled && !ts.requestTimerActive && !ts.updatingRequestTimer {
			ts.updatingRequestTimer = true
			ts.tasks <- ts.updateRequestTimer
		}
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
	ts.requests.Init()
	return ts
}
