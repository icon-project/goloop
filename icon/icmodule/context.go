package icmodule

import (
	"math/big"

	"github.com/icon-project/goloop/module"
)

type WorldContext interface {
	Revision() module.Revision
	BlockHeight() int64
	From() module.Address
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
	Burn(address module.Address, amount *big.Int) error
	SetValidators(validators []module.Validator) error
	SumOfStepUsed() *big.Int
	StepPrice() *big.Int
}

type CallContext interface {
	WorldContext
	OnEvent(addr module.Address, indexed, data [][]byte)
}
