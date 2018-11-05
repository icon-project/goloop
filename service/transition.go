package service

import (
	"errors"
	"sync"

	"github.com/icon-project/goloop/common"
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

// TODO Need to define Validator struct
type transitionState struct {
	// state always stores the initial state at the beginning and changes
	// according to transaction executions of this transition.
	// It will be initiated from parent transition at the top of Execute()
	// to set the base of transaction execution.
	// It can't be modified out of this package, so use the pointer directly
	// without copying.
	state trie.Mutable

	nextValidators module.ValidatorList
	normalReceipts *receiptList
	patchReceipts  *receiptList
}

type transition struct {
	trieManager trie.Manager
	parent      *transition

	patchTransactions  *txList
	normalTransactions *txList

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
func newTransition(parent *transition, patchTxs *txList, normalTxs *txList, state trie.Mutable, alreadyValidated bool) *transition {
	var step int
	if alreadyValidated {
		step = stepValidated
	} else {
		step = stepInited
	}
	return &transition{
		parent:             parent,
		patchTransactions:  patchTxs,
		normalTransactions: normalTxs,
		transitionState: &transitionState{
			state: state,
		},
		step: step,
	}
}

// all parameters should be valid.
func newInitTransition(tm trie.Manager, result []byte, validators module.ValidatorList) *transition {
	return &transition{
		trieManager: tm,
		result:      result,
		transitionState: &transitionState{
			nextValidators: validators,
		},
		step: stepComplete,
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
		t.trieManager = t.parent.trieManager
		t.state = t.trieManager.NewMutable(t.result.stateHash())
		t.step = stepValidating
	case stepValidated:
		// when this transition created by this node
		t.step = stepExecuting
	default:
		return nil, errors.New("Invalid transition state: " + t.stepString())
	}
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
	// TODO fix it based on ValidatorList
	return nil
	/*
		vals := make([]module.Validator, len(t.nextValidators))
		copy(vals, t.nextValidators)
		return vals
	*/
}

// PatchReceipts returns patch receipts.
// It may return nil before cb.OnExecute is called back by Execute.
func (t *transition) PatchReceipts() module.ReceiptList {
	if t.patchReceipts == nil {
		return nil
	}
	return t.patchReceipts
}

// NormalReceipts returns receipts.
// It may return nil before cb.OnExecute is called back by Execute.
func (t *transition) NormalReceipts() module.ReceiptList { return t.normalReceipts }

// LogBloom returns log bloom filter for this transition.
// It may return nil before cb.OnExecute is called back by Execute.
func (t *transition) LogBloom() []byte {
	b := make([]byte, len(t.logBloom))
	copy(b, t.logBloom)
	return t.logBloom
}

func (t *transition) executeSync(alreadyValidated bool) {
	if !alreadyValidated {
		if !t.validateTxs(t.patchTransactions) || !t.validateTxs(t.normalTransactions) {
			return
		}
		t.cb.OnValidate(t, nil)

		t.mutex.Lock()
		t.step = stepExecuting
		t.mutex.Unlock()
	} else {
		t.cb.OnValidate(t, nil)
	}

	if !t.executeTxs(t.patchTransactions) || !t.executeTxs(t.normalTransactions) {
		return
	}
	t.result = newResultBytesFromData(t.state, t.patchReceipts, t.normalReceipts)

	t.mutex.Lock()
	t.step = stepComplete
	t.mutex.Unlock()
	t.cb.OnExecute(t, nil)
}

func (t *transition) validateTxs(txList *txList) bool {
	canceled := false
	for _, tx := range txList.txs {
		if t.step == stepCanceled {
			canceled = true
			break
		}

		if err := tx.validate(t.state); err != nil {
			t.mutex.Lock()
			t.step = stepError
			t.mutex.Unlock()
			t.cb.OnValidate(t, err)
			return false
		}
	}
	return !canceled
}

func (t *transition) executeTxs(txList *txList) bool {
	canceled := false
	for _, tx := range txList.txs {
		if t.step == stepCanceled {
			canceled = true
			break
		}

		if err := tx.execute(t.transitionState); err != nil {
			t.mutex.Lock()
			t.step = stepError
			t.mutex.Unlock()
			t.cb.OnExecute(t, err)
			return false
		}
	}
	return !canceled
}

func (t *transition) finalize(opt int) {
	if opt&module.FinalizeNormalTransaction == module.FinalizeNormalTransaction {
		// TODO store DB
	}
	if opt&module.FinalizePatchTransaction == module.FinalizePatchTransaction {
		// TODO store DB
	}
	if opt&module.FinalizeResult == module.FinalizeResult {
		t.state.GetSnapshot().Flush()
		// TODO store index DB
		// Disconnect the useless parent transition
		t.parent = nil
	}
}

func (t *transition) hasValidResult() bool {
	if t.result != nil && t.nextValidators != nil {
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

// TODO confirm result byte format depending on the existence of patch
type resultBytes []byte

func newResultBytes(result []byte) (resultBytes, error) {
	if len(result) != 96 && len(result) != 64 {
		return nil, common.ErrIllegalArgument
	}
	bytes := make([]byte, len(result))
	copy(bytes, result)
	return resultBytes(bytes), nil
}

func newResultBytesFromData(state trie.Mutable, patch *receiptList, normal *receiptList) resultBytes {
	hasPatch := len(patch.receipts) > 0
	bytes := make([]byte, 0, 96)
	bytes = append(bytes, state.GetSnapshot().RootHash()...)
	if hasPatch {
		bytes = append(bytes, patch.Hash()...)
	}
	bytes = append(bytes, normal.Hash()...)
	return resultBytes(bytes)
}

func (r resultBytes) stateHash() []byte {
	// assumes bytes are already valid
	return r[0:32]
}

// It returns nil for no patch receipt
func (r resultBytes) patchReceiptHash() []byte {
	if len(r) == 64 {
		return nil
	}
	// assumes bytes are already valid
	return r[64:96]
}

func (r resultBytes) normalReceiptHash() []byte {
	// assumes bytes are already valid
	return r[32:64]
}
