package service

import (
	"bytes"
	"errors"
	"sync"

	"github.com/icon-project/goloop/common/trie/trie_manager"

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

// TODO Need to define Validator struct
type transitionState struct {
	// state always stores the initial state at the beginning and changes
	// according to transaction executions of this transition.
	// It will be initiated from parent transition at the top of Execute()
	// to set the base of transaction execution.
	// It can't be modified out of this package, so use the pointer directly
	// without copying.
	state             trie.Mutable
	nextValidatorList module.ValidatorList

	normalReceipts *receiptList
	patchReceipts  *receiptList
}

func newInitialTransitionState(state trie.Mutable) *transitionState {
	return &transitionState{state: state}
}

func newFinalTransitionState(validatorList module.ValidatorList) *transitionState {
	return &transitionState{nextValidatorList: validatorList}
}

type transition struct {
	parent *transition

	patchTransactions  *transactionlist
	normalTransactions *transactionlist

	db db.Database

	result resultBytes
	*transitionState
	// TODO logBloom은 개별 handler가 제공해 주는 게 맞는가? 아니면 여기서 일괄적으로
	// 계산하는 게 맞는가?
	logBloom []byte

	cb module.TransitionCallback

	// internal processing state
	step  int
	mutex sync.Mutex
}

// all parameters should be valid
func newTransition(parent *transition, patchTxList *transactionlist, normalTxList *transactionlist, state trie.Mutable, alreadyValidated bool) *transition {
	var step int
	if alreadyValidated {
		step = stepValidated
	} else {
		step = stepInited
	}
	return &transition{
		parent:             parent,
		patchTransactions:  patchTxList,
		normalTransactions: normalTxList,
		db:                 parent.db,
		transitionState:    newInitialTransitionState(state),
		step:               step,
	}
}

// all parameters should be valid.
func newInitTransition(db db.Database, result []byte, validatorList module.ValidatorList) *transition {
	return &transition{
		db:              db,
		result:          result,
		transitionState: newFinalTransitionState(validatorList),
		step:            stepComplete,
	}
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
	// TODO copy validator list after defining validator struct
	return t.nextValidatorList
}

// LogBloom returns log bloom filter for this transition.
// It may return nil before cb.OnExecute is called back by Execute.
func (t *transition) LogBloom() []byte {
	if t.step != stepComplete {
		return nil
	}
	b := make([]byte, len(t.logBloom))
	copy(b, t.logBloom)
	return t.logBloom
}

func (t *transition) executeSync(alreadyValidated bool) {
	// TODO check better way for nil result in the parent transition
	var stateHash []byte
	if t.parent.result != nil {
		stateHash = t.parent.result.stateHash()
	} else {
		stateHash = nil
	}

	if !alreadyValidated {
		txdb, err := t.db.GetBucket(db.TransactionLocatorByHash)
		if err != nil {
			panic("FAIL to get bucket TransactionLocatorByHash")
		}

		t.state = trie_manager.NewMutable(t.db, stateHash)
		if !t.validateTxs(t.patchTransactions, txdb) || !t.validateTxs(t.normalTransactions, txdb) {
			return
		}
		if t.cb != nil {
			t.cb.OnValidate(t, nil)
		}

		t.mutex.Lock()
		t.step = stepExecuting
		t.mutex.Unlock()
	} else {
		if t.cb != nil {
			t.cb.OnValidate(t, nil)
		}

	}

	t.state = trie_manager.NewMutable(t.db, stateHash)
	t.patchReceipts = &receiptList{}
	t.normalReceipts = &receiptList{}
	t.logBloom = make([]byte, 0)
	if !t.executeTxs(t.patchTransactions) || !t.executeTxs(t.normalTransactions) {
		return
	}
	t.result = newResultBytesFromData(t.state, t.patchReceipts, t.normalReceipts)
	// TODO update validators; it just copied the previous one.
	t.nextValidatorList = t.parent.nextValidatorList

	t.mutex.Lock()
	t.step = stepComplete
	t.mutex.Unlock()
	if t.cb != nil {
		t.cb.OnExecute(t, nil)
	}
}

func (t *transition) validateTxs(txList *transactionlist, txDB db.Bucket) bool {
	canceled := false
	for _, tx := range txList.txs {
		if t.step == stepCanceled {
			canceled = true
			break
		}

		if err := tx.validate(t.state, txDB); err != nil {
			t.mutex.Lock()
			t.step = stepError
			t.mutex.Unlock()
			t.cb.OnValidate(t, err)
			return false
		}
	}
	return !canceled
}

func (t *transition) executeTxs(txList *transactionlist) bool {
	canceled := false
	for _, tx := range txList.txs {
		if t.step == stepCanceled {
			canceled = true
			break
		}

		// TODO: 아래 db 추가에 대한 추후 검토 필요. KN.KIM. execute내에서 account를 생성하기 위해 db가 필요.
		if err := tx.execute(t.transitionState, t.db); err != nil {
			t.mutex.Lock()
			t.step = stepError
			t.mutex.Unlock()
			t.cb.OnExecute(t, err)
			return false
		}
	}
	return !canceled
}

func (t *transition) finalizeNormalTransaction() {
	t.normalTransactions.flush()
}

func (t *transition) finalizePatchTransaction() {
	t.patchTransactions.flush()
}

func (t *transition) finalizeResult() {
	t.state.GetSnapshot().Flush()
	// Disconnect the useless parent transition
	t.parent = nil
}

func (t *transition) hasValidResult() bool {
	if t.result != nil && t.nextValidatorList != nil {
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

func newResultBytesFromData(state trie.Mutable, patchRcList *receiptList, normalRcList *receiptList) resultBytes {
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
