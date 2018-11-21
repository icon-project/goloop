package service

import (
	"bytes"
	"errors"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/codec"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
)

const (
	stepInited    = iota // parent, patch/normalTxes and state are ready.
	stepValidated        // Upon inited state, Txes are validated.
	stepValidating
	stepExecuting
	stepComplete // all information is ready. REMARK: InitTransition only has some result parts - result and nextValidators
	stepError    // fails validation or execution
	stepCanceled // canceled. requested to cancel after complete executione, just remain stepFinished
)

// TODO temporary; remove
var Zero32 = make([]byte, 32)

type transition struct {
	parent    *transition
	height    int64
	timestamp int64

	patchTransactions  module.TransactionList
	normalTransactions module.TransactionList

	db db.Database

	cb module.TransitionCallback

	// internal processing state
	step  int
	mutex sync.Mutex

	// TODO add receipt list
	result        resultBytes
	worldSnapshot WorldSnapshot
	logBloom      LogBloom

	patchReceipts []Receipt
}

func newTransition(parent *transition, patchtxs module.TransactionList, normaltxs module.TransactionList, alreadyValidated bool) *transition {
	var step int
	if alreadyValidated {
		step = stepValidated
	} else {
		step = stepInited
	}
	if patchtxs == nil {
		patchtxs = newTransactionListFromList(parent.db, nil)
	}
	if normaltxs == nil {
		normaltxs = newTransactionListFromList(parent.db, nil)
	}
	return &transition{
		parent:             parent,
		height:             parent.height + 1,
		timestamp:          time.Now().UnixNano() / 1000,
		patchTransactions:  patchtxs,
		normalTransactions: normaltxs,
		db:                 parent.db,
		step:               step,
	}
}

// all parameters should be valid.
func newInitTransition(db db.Database, result []byte, validatorList module.ValidatorList, height int64) (*transition, error) {
	hashes := [][]byte{nil, nil, nil}
	if len(result) > 0 {
		if _, err := codec.UnmarshalFromBytes(result, &hashes); err != nil {
			return nil, err
		}
	}
	ws := NewWorldState(db, hashes[0], validatorList)

	// TODO also need to recover receipts.

	return &transition{
		height:             height,
		db:                 db,
		patchTransactions:  newTransactionListFromList(db, nil),
		normalTransactions: newTransactionListFromList(db, nil),
		worldSnapshot:      ws.GetSnapshot(),
		step:               stepComplete,
	}, nil
}

func (t *transition) PatchTransactions() module.TransactionList {
	return t.patchTransactions
}
func (t *transition) NormalTransactions() module.TransactionList {
	return t.normalTransactions
}

// Execute executes this transition.
// The result is asynchronously notified by cb. canceler can be used
// to cancel it after calling Execute. After canceler returns true,
// all succeeding cb functions may not be called back.
// REMARK: It is assumed to be called once. Any additional call returns
// error.
func (t *transition) Execute(cb module.TransitionCallback) (canceler func() bool, err error) {
	t.mutex.Lock()

	switch t.step {
	case stepInited:
		t.step = stepValidating
	case stepValidated:
		// when this transition created by this node
		t.step = stepExecuting
	default:
		return nil, errors.New("Invalid transition state: " + t.stepString())
	}
	t.cb = cb
	go t.executeSync(t.step == stepExecuting)

	t.mutex.Unlock()

	return t.cancelExecution, nil
}

// Result returns service manager defined result bytes.
func (t *transition) Result() []byte {
	r := make([]byte, len(t.result))
	copy(r, t.result)
	return r
}

// NextValidators returns the addresses of validators as a result of
// transaction processing.
// It may return nil before cb.OnExecute is called back by Execute.
func (t *transition) NextValidators() module.ValidatorList {
	if t.worldSnapshot != nil {
		return t.worldSnapshot.GetValidators()
	}
	log.Printf("Fail to get valid Validators")
	return nil
}

// LogBloom returns log bloom filter for this transition.
// It may return nil before cb.OnExecute is called back by Execute.
func (t *transition) LogBloom() []byte {
	if t.step != stepComplete {
		return nil
	}
	return t.logBloom.Bytes()
}

func (t *transition) executeSync(alreadyValidated bool) {
	var normalCount, patchCount int
	if !alreadyValidated {
		var ok bool
		ws, err := WorldStateFromSnapshot(t.parent.worldSnapshot)
		if err != nil {
			log.Panicf("Fail to build world state from snapshot err=%+v", err)
		}
		wc := NewWorldContext(ws, t.timestamp, uint64(t.height))
		ok, patchCount = t.validateTxs(t.patchTransactions, wc)
		if !ok {
			return
		}
		ok, normalCount = t.validateTxs(t.normalTransactions, wc)
		if !ok {
			return
		}
		if t.cb != nil {
			t.cb.OnValidate(t, nil)
		}
	} else {
		for i := t.patchTransactions.Iterator(); i.Has(); i.Next() {
			patchCount++
		}
		for i := t.normalTransactions.Iterator(); i.Has(); i.Next() {
			normalCount++
		}
		if t.cb != nil {
			t.cb.OnValidate(t, nil)
		}
	}

	t.mutex.Lock()
	t.step = stepExecuting
	t.mutex.Unlock()

	ws, err := WorldStateFromSnapshot(t.parent.worldSnapshot)
	if err != nil {
		log.Panicf("Fail to make WorldState from snapshot err=%+v", err)
	}
	wc := NewWorldContext(ws, t.timestamp, uint64(t.height))
	patchReceipts := make([]Receipt, patchCount)
	t.executeTxs(t.patchTransactions, wc, patchReceipts)
	normalReceipts := make([]Receipt, normalCount)
	t.executeTxs(t.normalTransactions, wc, normalReceipts)

	cumulativeSteps := big.NewInt(0)
	gatheredFee := big.NewInt(0)
	fee := big.NewInt(0)

	// TODO we need to use ReceiptList implementation to store it.
	for _, r := range patchReceipts {
		used := r.StepUsed()
		cumulativeSteps.Add(cumulativeSteps, used)
		r.SetCumulativeStepUsed(cumulativeSteps)

		fee.Set(r.StepPrice())
		fee.Mul(fee, used)
		gatheredFee.Add(gatheredFee, fee)
	}

	// TODO we need to use ReceiptList implementation to store it.
	for _, r := range patchReceipts {
		used := r.StepUsed()
		cumulativeSteps.Add(cumulativeSteps, used)
		r.SetCumulativeStepUsed(cumulativeSteps)

		fee.Set(r.StepPrice())
		fee.Mul(fee, used)
		gatheredFee.Add(gatheredFee, fee)
	}

	// save gathered fee to treasury
	tr := wc.GetAccountState(wc.Treasury().ID())
	trbal := tr.GetBalance()
	trbal.Add(trbal, gatheredFee)
	tr.SetBalance(trbal)

	t.worldSnapshot = wc.GetSnapshot()

	t.result, _ = codec.MarshalToBytes([][]byte{
		t.worldSnapshot.StateHash(),
		nil,
		nil,
	})

	t.mutex.Lock()
	t.step = stepComplete
	t.mutex.Unlock()
	if t.cb != nil {
		t.cb.OnExecute(t, nil)
	}
}

func (t *transition) validateTxs(l module.TransactionList, wc WorldContext) (bool, int) {
	if l == nil {
		return true, 0
	}
	cnt := 0
	for i := l.Iterator(); i.Has(); i.Next() {
		if t.step == stepCanceled {
			return false, 0
		}

		tx, _, err := i.Get()
		if err != nil {
			log.Panicf("Fail to iterate transaction list err=%+v", err)
		}

		if err := tx.(Transaction).PreValidate(wc, true); err != nil {
			t.mutex.Lock()
			t.step = stepError
			t.mutex.Unlock()
			t.cb.OnValidate(t, err)
			return false, 0
		}
		cnt += 1
	}
	return true, cnt
}

func (t *transition) executeTxs(l module.TransactionList, wc WorldContext, rctBuf []Receipt) (bool, int) {
	if l == nil {
		return true, 0
	}
	cnt := 0
	for i := l.Iterator(); i.Has(); i.Next() {
		if t.step == stepCanceled {
			return false, 0
		}
		tx, _, err := i.Get()
		if err != nil {
			log.Panicf("Fail to iterate transaction list err=%+v", err)
		}
		if rct, err := tx.(Transaction).Execute(wc); err != nil {
			t.mutex.Lock()
			t.step = stepError
			t.mutex.Unlock()
			t.cb.OnExecute(t, err)
			return false, 0
		} else {
			rctBuf[cnt] = rct
		}
	}
	return true, cnt
}

func (t *transition) finalizeNormalTransaction() {
	t.normalTransactions.Flush()
}

func (t *transition) finalizePatchTransaction() {
	t.patchTransactions.Flush()
}

func (t *transition) finalizeResult() {
	t.worldSnapshot.Flush()
	t.parent = nil
}

func (t *transition) hasValidResult() bool {
	if t.result != nil && t.worldSnapshot != nil {
		return true
	}
	return false
}

func (t *transition) cancelExecution() bool {
	t.mutex.Lock()
	if t.step != stepComplete && t.step != stepError {
		t.step = stepCanceled
	}
	t.mutex.Unlock()
	return true
}

func (t *transition) stepString() string {
	switch t.step {
	case stepInited:
		return "Inited"
	case stepValidated:
		return "Validated"
	case stepValidating:
		return "Validating"
	case stepExecuting:
		return "Executing"
	case stepComplete:
		return "Executed"
	case stepCanceled:
		return "Canceled"
	default:
		return ""
	}
}

// TODO store a serialized form to []byte and remove the concept of zero bytes
type resultBytes []byte

func newEmptyResultBytes() resultBytes {
	b := make([]byte, 96)
	return resultBytes(b)
}
func newResultBytes(result []byte) (resultBytes, error) {
	if len(result) != 96 {
		return nil, common.ErrIllegalArgument
	}
	bytes := make([]byte, len(result))
	copy(bytes, result)
	return resultBytes(bytes), nil
}

func newResultBytesFromHashes(state trie.Mutable, patchRcList *receiptList, normalRcList *receiptList) resultBytes {
	bytes := make([]byte, 0, 96)
	var h []byte
	if state != nil {
		h = state.GetSnapshot().Hash()
	}
	if h == nil {
		h = Zero32
	}
	bytes = append(bytes, h...)
	if patchRcList == nil || patchRcList.Hash() == nil {
		h = Zero32
	} else {
		h = patchRcList.Hash()
	}
	bytes = append(bytes, h...)
	if normalRcList == nil || normalRcList.Hash() == nil {
		h = Zero32
	} else {
		h = normalRcList.Hash()
	}
	bytes = append(bytes, h...)
	return resultBytes(bytes)
}

func (r resultBytes) stateHash() []byte {
	// assumes bytes are already valid
	if bytes.Equal(r[0:32], Zero32) {
		return nil
	}
	return r[0:32]
}

// It returns nil for no patch receipt
func (r resultBytes) patchReceiptHash() []byte {
	// assumes bytes are already valid
	if bytes.Equal(r[32:64], Zero32) {
		return nil
	}
	return r[32:64]
}

func (r resultBytes) normalReceiptHash() []byte {
	// assumes bytes are already valid
	if bytes.Equal(r[64:96], Zero32) {
		return nil
	}
	return r[64:96]
}
