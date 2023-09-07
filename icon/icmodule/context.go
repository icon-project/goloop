package icmodule

import (
	"math/big"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/trace"
)

type WorldContext interface {
	Revision() module.Revision
	BlockHeight() int64
	Origin() module.Address
	Treasury() module.Address
	TransactionID() []byte
	ConsensusInfo() module.ConsensusInfo
	GetBalance(address module.Address) *big.Int
	Deposit(address module.Address, amount *big.Int, opType module.OpType) error
	Withdraw(address module.Address, amount *big.Int, opType module.OpType) error
	Transfer(from module.Address, to module.Address, amount *big.Int, opType module.OpType) error
	GetTotalSupply() *big.Int
	AddTotalSupply(amount *big.Int) (*big.Int, error)
	SetValidators(validators []module.Validator) error
	StepPrice() *big.Int
	GetScoreOwner(score module.Address) (module.Address, error)
	SetScoreOwner(from module.Address, score module.Address, owner module.Address) error
	GetBTPContext() state.BTPContext
	GetActiveDSAMask() int64
}

type CallContext interface {
	WorldContext
	From() module.Address
	HandleBurn(address module.Address, amount *big.Int) error
	SumOfStepUsed() *big.Int
	OnEvent(addr module.Address, indexed, data [][]byte)
	CallOnTimer(to module.Address, params []byte) error
	Governance() module.Address
	FrameLogger() *trace.Logger
	TransactionInfo() *state.TransactionInfo
}

type StateContext interface {
	BlockHeight() int64
	Revision() int
	TermRevision() int
	IsIISS4Activated() bool
	GetActiveDSAMask() int64
	GetBondRequirement() Rate
	AddEventEnable(from module.Address, status EnableStatus) error
}
