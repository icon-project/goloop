package module

import (
	"fmt"
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

// Block information used by service manager.
type BlockInfo interface {
	Height() int64
	Timestamp() int64
}

type Transaction interface {
	Group() TransactionGroup
	ID() []byte
	From() Address
	Bytes() []byte
	Hash() []byte
	Verify() error
	Version() int
	ToJSON(version int) (interface{}, error)
	ValidateNetwork(nid int) bool

	// Version() int
	// To() Address
	// Value() *big.Int
	// StepLimit() *big.Int
	// Timestamp() int64
	// NID() int
	// Nonce() int64
	// Signature() []byte
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
	Flush() error
}

type Status int

const (
	StatusSuccess Status = iota
	StatusSystemError
	StatusContractNotFound
	StatusMethodNotFound
	StatusMethodNotPayable
	StatusIllegalFormat
	StatusInvalidParameter
	StatusInvalidInstance
	StatusInvalidContainerAccess
	StatusAccessDenied
	StatusOutOfStep
	StatusOutOfBalance
	StatusTimeout
	StatusStackOverflow
	StatusUser = 32
)

func (s Status) String() string {
	switch s {
	case StatusSuccess:
		return "Success"
	case StatusSystemError:
		return "SystemError"
	case StatusContractNotFound:
		return "ContractNotFound"
	case StatusMethodNotFound:
		return "MethodNotFound"
	case StatusMethodNotPayable:
		return "MethodNotPayable"
	case StatusIllegalFormat:
		return "IllegalFormat"
	case StatusInvalidParameter:
		return "InvalidParameter"
	case StatusInvalidInstance:
		return "InvalidInstance"
	case StatusInvalidContainerAccess:
		return "InvalidContainerAccess"
	case StatusAccessDenied:
		return "AccessDenied"
	case StatusOutOfStep:
		return "OutOfStep"
	case StatusOutOfBalance:
		return "OutOfBalance"
	case StatusStackOverflow:
		return "StackOverflow"
	default:
		if int(s) >= StatusUser {
			return fmt.Sprintf("User(%d)", s-StatusUser)
		} else {
			return fmt.Sprintf("Unknown(code=%d)", s)
		}
	}
}

type EventLog interface {
	Address() Address
	Indexed() [][]byte
	Data() [][]byte
}

type EventLogIterator interface {
	Has() bool
	Next() error
	Get() (EventLog, error)
}

type Receipt interface {
	Bytes() []byte
	To() Address
	CumulativeStepUsed() *big.Int
	StepPrice() *big.Int
	StepUsed() *big.Int
	Status() Status
	SCOREAddress() Address
	Check(r Receipt) error
	ToJSON(int) (interface{}, error)
	LogBloom() LogBloom
	EventLogIterator() EventLogIterator
}

type ReceiptIterator interface {
	Has() bool
	Next() error
	Get() (Receipt, error)
}

type ReceiptList interface {
	Get(int) (Receipt, error)
	GetProof(n int) ([][]byte, error)
	Iterator() ReceiptIterator
	Hash() []byte
	Flush() error
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
	LogBloom() LogBloom
}

type APIInfo interface {
	ToJSON(int) (interface{}, error)
}

// Options for finalize
const (
	FinalizeNormalTransaction = 1 << iota
	FinalizePatchTransaction
	FinalizeResult

	// TODO It's only necessary if storing receipt index is determined by
	// block manager. The current service manager determines by itself according
	// to version, so it doesn't use it.
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
//		2. create Transaction instances by TransactionFromBytes
//		3. CreateTransition with TransactionList
//		4. Transition.Execute
type ServiceManager interface {
	// ProposeTransition proposes a Transition following the parent Transition.
	// Returned Transition always passes validation.
	ProposeTransition(parent Transition, bi BlockInfo) (Transition, error)
	// CreateInitialTransition creates an initial Transition.
	CreateInitialTransition(result []byte, nextValidators ValidatorList) (Transition, error)
	// CreateTransition creates a Transition following parent Transition.
	CreateTransition(parent Transition, txs TransactionList, bi BlockInfo) (Transition, error)
	// GetPatches returns all patch transactions based on the parent transition.
	GetPatches(parent Transition) TransactionList
	// PatchTransition creates a Transition by overwriting patches on the transition.
	PatchTransition(transition Transition, patches TransactionList) Transition

	// Start starts service module.
	Start()
	// Term terminates serviceManager instance.
	Term()

	// Finalize finalizes data related to the transition. It usually stores
	// data to a persistent storage. opt indicates which data are finalized.
	// It should be called for every transition.
	Finalize(transition Transition, opt int)

	// TransactionFromBytes returns a Transaction instance from bytes.
	TransactionFromBytes(b []byte, blockVersion int) (Transaction, error)

	// TransactionListFromHash returns a TransactionList instance from
	// the hash of transactions or nil when no transactions exist.
	// It assumes it's called only by new version block, so it doesn't receive
	// version value.
	TransactionListFromHash(hash []byte) TransactionList

	// TransactionListFromSlice returns list of transactions.
	TransactionListFromSlice(txs []Transaction, version int) TransactionList

	// ReceiptListFromResult returns list of receipts from result.
	ReceiptListFromResult(result []byte, g TransactionGroup) ReceiptList

	// SendTransaction adds transaction to a transaction pool.
	SendTransaction(tx interface{}) ([]byte, error)

	// Call handles read-only contract API call.
	Call(result []byte, vl ValidatorList, js []byte, bi BlockInfo) (interface{}, error)

	// ValidatorListFromHash returns ValidatorList from hash.
	ValidatorListFromHash(hash []byte) ValidatorList

	// GetBalance get balance of the account
	GetBalance(result []byte, addr Address) *big.Int

	// GetTotalSupply get total supplied coin
	GetTotalSupply(result []byte) *big.Int

	// GetNetworkID get network ID of of the state
	GetNetworkID(result []byte) (int64, error)

	// GetAPIInfo get API info of the contract
	GetAPIInfo(result []byte, addr Address) (APIInfo, error)
}
