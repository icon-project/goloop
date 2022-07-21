package module

import "math/big"

type TraceRange int

const (
	TraceRangeBlock TraceRange = iota
	TraceRangeTransaction
	TraceRangeBlockTransaction
)

type TraceInfo struct {
	Range TraceRange
	// Group and Index are valid only if Range is TraceRangeTransaction
	Group    TransactionGroup
	Index    int
	Callback TraceCallback
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

type BalanceTracer interface {
	OnTransactionStart(txIndex int32, txHash []byte) error
	OnTransactionRerun(txIndex int32, txHash []byte) error
	OnTransactionEnd(txIndex int32, txHash []byte) error
	OnEnter() error
	OnLeave(success bool) error
	OnBalanceChange(opType OpType, from, to Address, amount *big.Int) error
}

type TraceCallback interface {
	TraceMode() TraceMode
	OnLog(level TraceLevel, msg string)
	OnEnd(e error)

	GetReceipt(txIndex int) Receipt
	BalanceTracer
}
