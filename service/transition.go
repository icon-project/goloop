package service

import (
	"errors"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/codec"

	"github.com/icon-project/goloop/common/db"
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

	result         []byte
	worldSnapshot  WorldSnapshot
	patchReceipts  module.ReceiptList
	normalReceipts module.ReceiptList
	logBloom       logBloom
}

type transitionResult struct {
	StateHash         []byte
	PatchReceiptHash  []byte
	NormalReceiptHash []byte
}

func newTransitionResultFromBytes(bs []byte) (*transitionResult, error) {
	tresult := new(transitionResult)
	if _, err := codec.UnmarshalFromBytes(bs, tresult); err != nil {
		return nil, err
	}
	return tresult, nil
}

func (tr *transitionResult) Bytes() []byte {
	if bs, err := codec.MarshalToBytes(tr); err != nil {
		log.Panicf("Fail to marshal transitionResult")
		return nil
	} else {
		return bs
	}
}

func newTransition(parent *transition, patchtxs module.TransactionList, normaltxs module.TransactionList, alreadyValidated bool) *transition {
	var step int
	if alreadyValidated {
		step = stepValidated
	} else {
		step = stepInited
	}

	var height int64
	var timestamp int64
	if parent != nil {
		height = parent.height + 1
		// TODO set a correct timestamp.
		timestamp = time.Now().UnixNano() / 1000
	}

	if patchtxs == nil {
		patchtxs = NewTransactionListFromSlice(parent.db, nil)
	}
	if normaltxs == nil {
		normaltxs = NewTransactionListFromSlice(parent.db, nil)
	}
	return &transition{
		parent:             parent,
		height:             height,
		timestamp:          timestamp,
		patchTransactions:  patchtxs,
		normalTransactions: normaltxs,
		db:                 parent.db,
		step:               step,
	}
}

// all parameters should be valid.
func newInitTransition(db db.Database, result []byte, validatorList module.ValidatorList, height int64) (*transition, error) {
	var tresult transitionResult
	if len(result) > 0 {
		if _, err := codec.UnmarshalFromBytes(result, &tresult); err != nil {
			return nil, err
		}
	}
	ws := NewWorldState(db, tresult.StateHash, validatorList)

	return &transition{
		height:             height,
		patchTransactions:  NewTransactionListFromSlice(db, nil),
		normalTransactions: NewTransactionListFromSlice(db, nil),
		db:                 db,
		step:               stepComplete,
		worldSnapshot:      ws.GetSnapshot(),
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
	if t.step != stepComplete {
		return nil
	}
	return t.result
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

func (t *transition) newWorldContext() WorldContext {
	var ws WorldState
	if t.parent != nil {
		var err error
		ws, err = WorldStateFromSnapshot(t.parent.worldSnapshot)
		if err != nil {
			log.Panicf("Fail to build world state from snapshot err=%+v", err)
		}
	} else {
		ws = NewWorldState(t.db, nil, nil)
	}
	return NewWorldContext(ws, t.timestamp, t.height)
}

func (t *transition) executeSync(alreadyValidated bool) {
	var normalCount, patchCount int
	if !alreadyValidated {
		var ok bool
		wc := t.newWorldContext()
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

	wc := t.newWorldContext()
	patchReceipts := make([]Receipt, patchCount)
	t.executeTxs(t.patchTransactions, wc, patchReceipts)
	normalReceipts := make([]Receipt, normalCount)
	t.executeTxs(t.normalTransactions, wc, normalReceipts)

	cumulativeSteps := big.NewInt(0)
	gatheredFee := big.NewInt(0)
	fee := big.NewInt(0)

	for _, receipts := range [][]Receipt{patchReceipts, normalReceipts} {
		for _, r := range receipts {
			used := r.StepUsed()
			cumulativeSteps.Add(cumulativeSteps, used)
			r.SetCumulativeStepUsed(cumulativeSteps)

			fee.Set(r.StepPrice())
			fee.Mul(fee, used)
			gatheredFee.Add(gatheredFee, fee)
		}
	}
	t.patchReceipts = NewReceiptListFromSlice(t.db, patchReceipts)
	t.normalReceipts = NewReceiptListFromSlice(t.db, normalReceipts)

	// save gathered fee to treasury
	tr := wc.GetAccountState(wc.Treasury().ID())
	trbal := tr.GetBalance()
	trbal.Add(trbal, gatheredFee)
	tr.SetBalance(trbal)

	t.worldSnapshot = wc.GetSnapshot()

	t.result = transitionResult{
		t.worldSnapshot.StateHash(),
		t.patchReceipts.Hash(),
		t.normalReceipts.Hash(),
	}.Bytes()

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
			cnt++
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
	t.patchReceipts.Flush()
	t.normalReceipts.Flush()
	t.parent = nil
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
