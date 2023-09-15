package module

import (
	"container/list"
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common/db"
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

type ConsensusInfo interface {
	Proposer() Address
	Voters() ValidatorList
	Voted() []bool
}

type Transaction interface {
	Group() TransactionGroup
	ID() []byte
	From() Address
	Bytes() []byte
	Hash() []byte
	Verify() error
	Version() int
	ToJSON(version JSONVersion) (interface{}, error)
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

	// length if Hash() is 0 iff empty
	Hash() []byte

	Equal(TransactionList) bool
	Flush() error
}

type Status int

const (
	StatusSuccess Status = iota
	StatusUnknownFailure
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
	StatusSkipTransaction
	StatusInvalidPackage
	StatusReverted Status = 32

	StatusLimitRev5 Status = 99
	StatusLimit     Status = 999
)

func (s Status) String() string {
	switch s {
	case StatusSuccess:
		return "Success"
	case StatusUnknownFailure:
		return "UnknownFailure"
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
	case StatusTimeout:
		return "Timeout"
	case StatusStackOverflow:
		return "StackOverflow"
	case StatusSkipTransaction:
		return "SkipTransaction"
	case StatusInvalidPackage:
		return "InvalidPackage"
	default:
		if s >= StatusReverted {
			return fmt.Sprintf("Reverted(%d)", s-StatusReverted)
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

type FeePayment interface {
	Payer() Address
	Amount() *big.Int
}

type FeePaymentIterator interface {
	Has() bool
	Next() error
	Get() (FeePayment, error)
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
	ToJSON(version JSONVersion) (interface{}, error)
	LogsBloom() LogsBloom
	BTPMessages() *list.List
	EventLogIterator() EventLogIterator
	FeePaymentIterator() FeePaymentIterator
	LogsBloomDisabled() bool
	GetProofOfEvent(int) ([][]byte, error)
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

	PatchReceipts() ReceiptList
	NormalReceipts() ReceiptList
	// Execute executes this transition.
	// The result is asynchronously notified by cb. canceler can be used
	// to cancel it after calling Execute. After canceler returns true,
	// all succeeding cb functions may not be called back.
	// REMARK: It is assumed to be called once. Any additional call returns
	// error.
	Execute(cb TransitionCallback) (canceler func() bool, err error)

	// ExecuteForTrace executes this transition until it executes the transaction
	// at offset `n` of normal transactions. If it fails, then OnValidate or
	// OnExecute will be called with an error.
	ExecuteForTrace(ti TraceInfo) (canceler func() bool, err error)

	// Result returns service manager defined result bytes.
	// For example, it can be "[world_state_hash][patch_tx_hash][normal_tx_hash]".
	Result() []byte

	// NextValidators returns the addresses of validators as a result of
	// transaction processing.
	// It may return nil before cb.OnExecute is called back by Execute.
	NextValidators() ValidatorList

	// LogsBloom returns log bloom filter for this transition.
	// It may return nil before cb.OnExecute is called back by Execute.
	LogsBloom() LogsBloom

	// BlockInfo returns block information for the normal transaction.
	BlockInfo() BlockInfo

	// Equal check equality of inputs of transition.
	Equal(Transition) bool

	// BTPSection returns the BTPSection as a result of transaction processing.
	// It may return empty one before cb.OnExecute is called back by Execute.
	BTPSection() BTPSection
}

type APIInfo interface {
	ToJSON(JSONVersion) (interface{}, error)
}

type SCOREStatus interface {
	ToJSON(height int64, version JSONVersion) (interface{}, error)
}

// Options for finalize
const (
	FinalizeNormalTransaction = 1 << iota
	FinalizePatchTransaction
	FinalizeResult
	KeepingParent

	// TODO It's only necessary if storing receipt index is determined by
	// block manager. The current service manager determines by itself according
	// to version, so it doesn't use it.
	FinalizeWriteReceiptIndex
)

// TransitionManager provides Transition APIs.
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
type TransitionManager interface {
	// ProposeTransition proposes a Transition following the parent Transition.
	// Returned Transition always passes validation.
	ProposeTransition(parent Transition, bi BlockInfo, csi ConsensusInfo) (Transition, error)
	// CreateInitialTransition creates an initial Transition.
	CreateInitialTransition(result []byte, nextValidators ValidatorList) (Transition, error)
	// CreateTransition creates a Transition following parent Transition.
	CreateTransition(parent Transition, txs TransactionList, bi BlockInfo, csi ConsensusInfo, validated bool) (Transition, error)
	// GetPatches returns all patch transactions based on the parent transition.
	// bi is the block info of the block that will contain the patches
	GetPatches(parent Transition, bi BlockInfo) TransactionList
	// PatchTransition creates a Transition by overwriting patches on the transition.
	// bi is the block info of the block that contains the patches,
	// or nil if the patches are already prevalidated.
	PatchTransition(transition Transition, patches TransactionList, bi BlockInfo) Transition
	CreateSyncTransition(transition Transition, result []byte, vlHash []byte, noBuffer bool) Transition
	// Finalize finalizes data related to the transition. It usually stores
	// data to a persistent storage. opt indicates which data are finalized.
	// It should be called for every transition.
	Finalize(transition Transition, opt int) error
	// WaitTransaction waits for a transaction with timestamp between
	// bi.Timestamp() - TimestampThreshold and current time +
	// TimestampThreshold. If such a transaction is available now, the function
	// returns false and callback cb is not called.
	WaitForTransaction(parent Transition, bi BlockInfo, cb func()) bool
}

type ServiceManager interface {
	TransitionManager

	// Start starts service module.
	Start()
	// Term terminates serviceManager instance.
	Term()

	// TransactionFromBytes returns a Transaction instance from bytes.
	TransactionFromBytes(b []byte, blockVersion int) (Transaction, error)

	// GenesisTransactionFromBytes returns a Genesis Transaction instance from bytes.
	GenesisTransactionFromBytes(b []byte, blockVersion int) (Transaction, error)

	// TransactionListFromHash returns a TransactionList instance from
	// the hash of transactions or nil when no transactions exist.
	// It assumes it's called only by new version block, so it doesn't receive
	// version value.
	TransactionListFromHash(hash []byte) TransactionList

	// TransactionListFromSlice returns list of transactions.
	TransactionListFromSlice(txs []Transaction, version int) TransactionList

	// ReceiptListFromResult returns list of receipts from result.
	ReceiptListFromResult(result []byte, g TransactionGroup) (ReceiptList, error)

	// SendTransaction adds transaction to a transaction pool.
	SendTransaction(result []byte, height int64, tx interface{}) ([]byte, error)

	// SendPatch sends a patch
	SendPatch(patch Patch) error

	// Call handles read-only contract API call.
	Call(result []byte, vl ValidatorList, js []byte, bi BlockInfo) (interface{}, error)

	// ValidatorListFromHash returns ValidatorList from hash.
	ValidatorListFromHash(hash []byte) ValidatorList

	// GetBalance returns balance of the account
	GetBalance(result []byte, addr Address) (*big.Int, error)

	// GetTotalSupply returns total supplied coin
	GetTotalSupply(result []byte) (*big.Int, error)

	// GetNetworkID returns network ID of the state
	GetNetworkID(result []byte) (int64, error)

	// GetChainID returns chain ID of the state
	GetChainID(result []byte) (int64, error)

	// GetAPIInfo returns API info of the contract
	GetAPIInfo(result []byte, addr Address) (APIInfo, error)

	// GetSCOREStatus returns status of the contract
	GetSCOREStatus(result []byte, addr Address) (SCOREStatus, error)

	// GetMembers returns network member list
	GetMembers(result []byte) (MemberList, error)

	// GetRoundLimit returns round limit
	GetRoundLimit(result []byte, vl int) int64

	// GetStepPrice returns the step price of the state
	GetStepPrice(result []byte) (*big.Int, error)

	// GetMinimizeBlockGen returns minimize empty block generation flag
	GetMinimizeBlockGen(result []byte) bool

	// GetNextBlockVersion returns version of next block
	GetNextBlockVersion(result []byte) int

	// BTPSectionFromResult returns BTPSection for the result
	BTPSectionFromResult(result []byte) (BTPSection, error)

	// BTPDigestFromResult returns BTPDigest for the result
	BTPDigestFromResult(result []byte) (BTPDigest, error)

	// NextProofContextMapFromResult returns BTPProofContextMap for the result
	NextProofContextMapFromResult(result []byte) (BTPProofContextMap, error)

	BTPNetworkTypeFromResult(result []byte, ntid int64) (BTPNetworkType, error)

	BTPNetworkFromResult(result []byte, nid int64) (BTPNetwork, error)

	BTPNetworkTypeIDsFromResult(result []byte) ([]int64, error)

	// HasTransaction returns whether it has specified transaction in the pool
	HasTransaction(id []byte) bool

	// SendTransactionAndWait send transaction and return channel for result
	SendTransactionAndWait(result []byte, height int64, tx interface{}) ([]byte, <-chan interface{}, error)

	// WaitTransactionResult return channel for result.
	WaitTransactionResult(id []byte) (<-chan interface{}, error)

	// ExportResult exports all related entries related with the result
	// should be exported to the database
	ExportResult(result []byte, vh []byte, dst db.Database) error

	// ImportResult imports all related entries related with the result
	// should be imported from the database
	ImportResult(result []byte, vh []byte, src db.Database) error

	// ExecuteTransaction executes the transaction on the specified state.
	// Then it returns the expected result of the transaction.
	// It ignores supplied step limit.
	ExecuteTransaction(result []byte, vh []byte, js []byte, bi BlockInfo) (Receipt, error)

	// AddSyncRequest add sync request for specified data.
	AddSyncRequest(id db.BucketID, key []byte) error

	// SendDoubleSignReport sends double sign reports. result and vh represents base state of the data.
	// If the data has votes for the height H, then result is base state deciding whether it has double signs.
	// So, result and vh should come from previous block at the height H-1.
	SendDoubleSignReport(result []byte, vh []byte, data []DoubleSignData) error
}
