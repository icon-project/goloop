package icmodule

import (
	"math/big"

	"github.com/icon-project/goloop/module"
)

type WorldContext interface {
	Revision() module.Revision
	BlockHeight() int64
	Origin() module.Address
	Treasury() module.Address
	TransactionID() []byte
	ConsensusInfo() module.ConsensusInfo
	GetBalance(address module.Address) *big.Int
	Deposit(address module.Address, amount *big.Int) error
	Withdraw(address module.Address, amount *big.Int) error
	Transfer(from module.Address, to module.Address, amount *big.Int) error
	GetTotalSupply() *big.Int
	AddTotalSupply(amount *big.Int) (*big.Int, error)
	SetValidators(validators []module.Validator) error
	StepPrice() *big.Int
}

type CallContext interface {
	WorldContext
	From() module.Address
	Burn(address module.Address, amount *big.Int) error
	SumOfStepUsed() *big.Int
	OnEvent(addr module.Address, indexed, data [][]byte)
}
