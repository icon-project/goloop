package module

import (
	"math/big"
)

// TransitionCallback provides transition change notifications. All functions
// are called back with the same Transition instance for the convenience.
type TransitionCallback interface {
	// Called if validation is done.
	OnValidate(Transition, error)

	// Called if execution is done.
	OnExecute(Transition, error)
}

type Transaction interface {
	Group() TransactionGroup
	ID() []byte
	Version() int
	Bytes() ([]byte, error)
	Verify() error
	From() Address
	To() Address
	Value() *big.Int
	StepLimit() *big.Int
	Timestamp() int64
	NID() int
	Nonce() int64
	Hash() []byte
	Signature() []byte
}

type TransactionIterator interface {
	Has() bool
	Next() error
	Get() (Transaction, int, error)
}

type TransactionList interface {
	Get(int) (Transaction, error)
	Iterator() TransactionIterator
	Hash() []byte
	Equal(TransactionList) bool
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
	// REMARK: It is assumed to be called once. Any additional call returns
	// error.
	Execute(cb TransitionCallback) (canceler func() bool, err error)

	// Result returns service manager defined result bytes.
	// For example, it can be "[world_state_hash][patch_tx_hash][normal_tx_hash]".
	Result() []byte

	// NextValidators returns the addresses of validators as a result of
	// transaction processing.
	// It may return nil before cb.OnExecute is called back by Execute.
	NextValidators() ValidatorList

	// LogBloom returns log bloom filter for this transition.
	// It may return nil before cb.OnExecute is called back by Execute.
	LogBloom() []byte
}

// Options for finalize
const (
	FinalizeNormalTransaction = 1 << iota
	FinalizePatchTransaction
	FinalizeResult

	FinalizeWriteReceiptIndex
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
	CreateInitialTransition(result []byte, nextValidators ValidatorList) (Transition, error)
	// CreateTransition creates a Transition following parent Transition.
	CreateTransition(parent Transition, txs TransactionList) (Transition, error)
	// GetPatches returns all patch transactions based on the parent transition.
	GetPatches(parent Transition) TransactionList
	// PatchTransition creates a Transition by overwriting patches on the transition.
	PatchTransition(transition Transition, patches TransactionList) Transition

	// Finalize finalizes data related to the transition. It usually stores
	// data to a persistent storage. opt indicates which data are finalized.
	// It should be called for every transition.
	Finalize(transition Transition, opt int)

	// TransactionFromBytes returns a Transaction instance from bytes.
	TransactionFromBytes(b []byte) Transaction

	// TransactionListFromHash returns a TransactionList instance from
	// the hash of transactions
	TransactionListFromHash(hash []byte) TransactionList

	// ReceiptFromTransactionID returns receipt from legacy receipt bucket.
	ReceiptFromTransactionID(id []byte) Receipt

	// ReceiptListFromResult returns list of receipts from result.
	ReceiptListFromResult(result []byte, g TransactionGroup) ReceiptList

	// TransactionListFromSlice returns list of transactions.
	TransactionListFromSlice(txs []Transaction, version int) TransactionList

	// SendTransaction adds transaction to a transaction pool.
	SendTransaction(tx Transaction) error
}
