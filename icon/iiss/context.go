package iiss

import (
	"math/big"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

func validateAmount(amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return errors.Errorf("Invalid amount: %v", amount)
	}
	return nil
}

func setBalance(address module.Address, as state.AccountState, balance *big.Int) error {
	if balance.Sign() < 0 {
		return errors.Errorf(
			"Invalid balance: address=%v balance=%v",
			address, balance,
		)
	}
	as.SetBalance(balance)
	return nil
}

type callContextImpl struct {
	from module.Address
	cc contract.CallContext
}

func (ctx *callContextImpl) Revision() module.Revision {
	return ctx.cc.Revision()
}

func (ctx *callContextImpl) BlockHeight() int64 {
	return ctx.cc.BlockHeight()
}

func (ctx *callContextImpl) From() module.Address {
	return ctx.from
}

func (ctx *callContextImpl) Origin() module.Address {
	return ctx.cc.TransactionInfo().From
}

func (ctx *callContextImpl) Treasury() module.Address {
	return ctx.cc.Treasury()
}

func (ctx *callContextImpl) TransactionID() []byte {
	return ctx.cc.TransactionID()
}

func (ctx *callContextImpl) ConsensusInfo() module.ConsensusInfo {
	return ctx.cc.ConsensusInfo()
}

func (ctx *callContextImpl) GetBalance(address module.Address) *big.Int {
	account := ctx.cc.GetAccountState(address.ID())
	return account.GetBalance()
}

func (ctx *callContextImpl) Deposit(address module.Address, amount *big.Int) error {
	if err := validateAmount(amount); err != nil {
		return err
	}
	if amount.Sign() == 0 {
		return nil
	}
	return ctx.addBalance(address, amount)
}

func (ctx *callContextImpl) Withdraw(address module.Address, amount *big.Int) error {
	if err := validateAmount(amount); err != nil {
		return err
	}
	if amount.Sign() == 0 {
		return nil
	}
	return ctx.addBalance(address, new(big.Int).Neg(amount))
}

func (ctx *callContextImpl) Transfer(from module.Address, to module.Address, amount *big.Int) (err error) {
	if err = validateAmount(amount); err != nil {
		return
	}
	if amount.Sign() == 0 || from.Equal(to) {
		return nil
	}
	// Subtract amount from the balance of "from" address
	if err = ctx.addBalance(from, new(big.Int).Neg(amount)); err != nil {
		return
	}
	// Add amount to "to" address
	if err = ctx.addBalance(to, amount); err != nil {
		return
	}
	return
}

func (ctx *callContextImpl) addBalance(address module.Address, amount *big.Int) error {
	as := ctx.cc.GetAccountState(address.ID())
	ob := as.GetBalance()
	return setBalance(address, as, new(big.Int).Add(ob, amount))
}

func (ctx *callContextImpl) Burn(address module.Address, amount *big.Int) error {
	sign := amount.Sign()
	if sign < 0 {
		return errors.Errorf("Invalid amount: %v", amount)
	}
	if sign > 0 {
		ts, err := ctx.AddTotalSupply(new(big.Int).Neg(amount))
		if err != nil {
			return err
		}
		ctx.OnBurn(address, amount, ts)
	}
	return nil
}

func (ctx *callContextImpl) OnBurn(address module.Address, amount, ts *big.Int) {
	rev := ctx.Revision().Value()
	if rev < icmodule.RevisionBurnV2 {
		var burnSig string
		if rev < icmodule.RevisionFixBurnEventSignature {
			burnSig = "ICXBurned"
		} else {
			burnSig = "ICXBurned(int)"
		}
		ctx.OnEvent(state.SystemAddress,
			[][]byte{[]byte(burnSig)},
			[][]byte{intconv.BigIntToBytes(amount)},
		)
	} else {
		ctx.OnEvent(state.SystemAddress,
			[][]byte{[]byte("ICXBurnedV2(Address,int,int)"), address.Bytes()},
			[][]byte{intconv.BigIntToBytes(amount), intconv.BigIntToBytes(ts)},
		)
	}
}

func (ctx *callContextImpl) OnEvent(address module.Address, indexed, data [][]byte) {
	ctx.cc.OnEvent(address, indexed, data)
}

func (ctx *callContextImpl) GetTotalSupply() *big.Int {
	as := ctx.cc.GetAccountState(state.SystemID)
	tsVar := scoredb.NewVarDB(as, state.VarTotalSupply)
	if ts := tsVar.BigInt(); ts != nil {
		return ts
	}
	return icmodule.BigIntZero
}

func (ctx *callContextImpl) AddTotalSupply(amount *big.Int) (*big.Int, error) {
	as := ctx.cc.GetAccountState(state.SystemID)
	varDB := scoredb.NewVarDB(as, state.VarTotalSupply)
	ts := new(big.Int).Add(varDB.BigInt(), amount)
	if ts.Sign() < 0 {
		return nil, errors.Errorf("TotalSupply < 0")
	}
	return ts, varDB.Set(ts)
}

func (ctx *callContextImpl) SetValidators(validators []module.Validator) error {
	return ctx.cc.GetValidatorState().Set(validators)
}

func (ctx *callContextImpl) SumOfStepUsed() *big.Int {
	return ctx.cc.SumOfStepUsed()
}

func (ctx *callContextImpl) StepPrice() *big.Int {
	return ctx.cc.StepPrice()
}

func NewCallContext(cc contract.CallContext, from module.Address) icmodule.CallContext {
	return &callContextImpl{
		from: from,
		cc: cc,
	}
}
