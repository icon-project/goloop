package module

import "math/big"

type TraceRange int

const (
	TraceRangeBlock TraceRange = iota
	TraceRangeTransaction
	TraceRangeBlockTransaction
)

type TraceInfo struct {
	TraceMode  TraceMode
	TraceBlock TraceBlock

	Range TraceRange
	// Group and Index are valid only if Range is TraceRangeTransaction
	Group TransactionGroup
	Index int

	Callback TraceCallback
}

type TraceBlock interface {
	ID() []byte
	GetReceipt(txIndex int) Receipt
}

type TraceLevel int

const (
	TDebugLevel TraceLevel = iota
	TTraceLevel
	TSystemLevel
)

type TraceMode int

const (
	TraceModeNone TraceMode = iota
	TraceModeInvoke
	TraceModeBalanceChange
)

type OpType int

const (
	Genesis OpType = iota
	Transfer
	Fee
	Issue
	Burn
	Lost
	FSDeposit
	FSWithdraw
	FSFee
	Stake
	Unstake
	Claim
	Ghost
	Reward
	RegPRep
)

type ExecutionPhase int

const (
	EPhaseTransaction ExecutionPhase = iota
	EPhaseExecutionEnd
)

type TraceCallback interface {
	OnLog(level TraceLevel, msg string)
	OnEnd(e error)

	OnTransactionStart(txIndex int, txHash []byte, isBlockTx bool) error
	OnTransactionReset() error
	OnTransactionEnd(txIndex int, txHash []byte) error
	OnFrameEnter() error
	OnFrameExit(success bool) error
	OnBalanceChange(opType OpType, from, to Address, amount *big.Int) error
}
