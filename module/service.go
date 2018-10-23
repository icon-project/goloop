package module

import "io"

type TransitionCallback interface {
	//	Called if error is occured.
	OnError(tr Transition, error)

	//	Called if validation is done.
	OnValidate(Transition)

	//	Called if execution is done.
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
	Parent() Transition
	//	NextValidators returns the addresses of next validators.
	//	The function returns nil if the transition is not created by
	//	ServiceManager.ProposeTransition and is not validated yet.
	NextValidators() []Address
	PatchTransactions() TransactionList
	NormalTransactions() TransactionList

	//	Execute executes this transition.
	//	The result is asynchronously notified by cb. canceler cancels the
	//	operation. canceler returns true and cb is not called if the
	//	cancellation was successful.
	Execute(cb TransitionCallback) (canceler func() bool, err error)

	State() State

	PatchReceipts() ReceiptList
	NormalReceipts() ReceiptList

	//	LogBloom returns log bloom filter for this transition.
	//	The function returns nil if the transition execution is not completed.
	LogBloom() []byte
}

type State interface {
	Verify([]byte) bool
	Bytes() ([]byte, error)
}

type ServiceManager interface {
	//	ProposeTransition proposes a Transition following the parent Transition.
	//	Returned Transition always passes validation.
	ProposeTransition(parent Transition) (Transition, error)
	//	CreateTransition creates a Transition following parent Transition.
	CreateTransition(parent Transition, txs TransactionList) (Transition, error)
	GetPatches(parent Transition) TransactionList
	//	PatchTransition creates a Transition by adding patch on a transition.
	PatchTransition(transition Transition, patches TransactionList) Transition

	Commit(Transition)
	Finalize(Transition)
	FinalizeTransactions(txs TransactionList)
	TransactionFromReader(r io.Reader) Transaction
}
