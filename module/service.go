package module

import "io"

// TransitionCallback provides transition change notifications. All functions
// are called back with the same Transition instance for the convenience.
type TransitionCallback interface {
	// Called if error is occured.
	OnError(Transition, error)

	// Called if validation is done.
	OnValidate(Transition)

	// Called if execution is done.
	OnExecute(Transition)
}

type Transaction interface {
	ID() []byte
	Version() int
	Bytes() ([]byte, error)
	Verify() error
}

type TransactionList interface {
	Get(int) (Transaction, error)
	Size() int
	Hash() []byte
}
type Receipt interface {
	Bytes() ([]byte, error)
}

type ReceiptList interface {
	Get(int) (Receipt, error)
	Size() int
	Hash() []byte
}

type Transition interface {
	PatchTransactions() TransactionList
	NormalTransactions() TransactionList

	// Execute executes this transition.
	// The result is asynchronously notified by cb. canceler can be used
	// to cancel it after calling Execute. After canceler returns true,
	// all succeeding cb functions may not be called back.
	Execute(cb TransitionCallback) (canceler func() bool, err error)

	// Result returns service manager defined result bytes.
	Result() []byte

	// NextValidators returns the addresses of validators as a result of
	// transaction processing.
	// It may return nil before cb.OnExecute is called back by Execute.
	NextValidators() []Validator

	// PatchReceipts returns patch receipts.
	// It may return nil before cb.OnExecute is called back by Execute.
	PatchReceipts() ReceiptList
	// NormalReceipts returns receipts.
	// It may return nil before cb.OnExecute is called back by Execute.
	NormalReceipts() ReceiptList

	// LogBloom returns log bloom filter for this transition.
	// It may return nil before cb.OnExecute is called back by Execute.
	LogBloom() []byte
}

// Options for finalize
const (
	FinalizeNormalTransaction = 1 << iota
	FinalizePatchTransaction
	FinalizeResult
)

// ServiceManager provides Service APIs.
// For a block proposal, it is usually called as follows:
// 		1. GetPatches
//		2. if any changes of patches exist from GetPatches
//			2.1 PatchTransaction
//			2.2 Transition.Execute
// 		3. ProposeTransition
//		4. Transition.Execute
// For a block validation,
//		1. if any changes of patches are detected from a new block
//			1.1 PatchTransition
//			1.2 Transition.Execute
//		2. create Transaction instances by TransactionFromReader
//		3. CreateTransition with TransactionList
//		4. Transition.Execute
type ServiceManager interface {
	// ProposeTransition proposes a Transition following the parent Transition.
	// Returned Transition always passes validation.
	ProposeTransition(parent Transition) (Transition, error)
	// CreateInitialTransition creates an initial Transition
	CreateInitialTransition(result []byte, nextValidators []Validator) (Transition, error)
	// CreateTransition creates a Transition following parent Transition.
	CreateTransition(parent Transition, txs TransactionList) (Transition, error)
	// GetPatches returns all patch transactions based on the parent transition.
	GetPatches(parent Transition) TransactionList
	// PatchTransition creates a Transition by overwriting patches on the transition.
	PatchTransition(transition Transition, patches TransactionList) Transition

	// Finalize finalizes data related to the transition. It usually stores
	// data to a persistent storage. dataBitMask indicates which data are
	// finalized.
	// It should be called for every transition.
	Finalize(transition Transition, opt int)

	// TransactionFromReader returns a Transaction instance from bytes
	// read by Reader.
	TransactionFromReader(r io.Reader) Transaction
}
