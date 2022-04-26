package iiss

import (
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/trace"
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

type worldContextImpl struct {
	state.WorldContext
}

func (ctx *worldContextImpl) Origin() module.Address {
	return ctx.TransactionInfo().From
}

func (ctx *worldContextImpl) GetBalance(address module.Address) *big.Int {
	account := ctx.GetAccountState(address.ID())
	return account.GetBalance()
}

func (ctx *worldContextImpl) Deposit(address module.Address, amount *big.Int) error {
	if err := validateAmount(amount); err != nil {
		return err
	}
	if amount.Sign() == 0 {
		return nil
	}
	return ctx.addBalance(address, amount)
}

func (ctx *worldContextImpl) Withdraw(address module.Address, amount *big.Int) error {
	if err := validateAmount(amount); err != nil {
		return err
	}
	if amount.Sign() == 0 {
		return nil
	}
	return ctx.addBalance(address, new(big.Int).Neg(amount))
}

func (ctx *worldContextImpl) Transfer(from module.Address, to module.Address, amount *big.Int) (err error) {
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

func (ctx *worldContextImpl) addBalance(address module.Address, amount *big.Int) error {
	as := ctx.GetAccountState(address.ID())
	ob := as.GetBalance()
	return setBalance(address, as, new(big.Int).Add(ob, amount))
}

func (ctx *worldContextImpl) GetTotalSupply() *big.Int {
	as := ctx.GetAccountState(state.SystemID)
	tsVar := scoredb.NewVarDB(as, state.VarTotalSupply)
	if ts := tsVar.BigInt(); ts != nil {
		return ts
	}
	return icmodule.BigIntZero
}

func (ctx *worldContextImpl) AddTotalSupply(amount *big.Int) (*big.Int, error) {
	as := ctx.GetAccountState(state.SystemID)
	varDB := scoredb.NewVarDB(as, state.VarTotalSupply)
	oldTs := varDB.BigInt()
	if oldTs == nil {
		oldTs = new(big.Int)
	}
	ts := new(big.Int).Add(oldTs, amount)
	if ts.Sign() < 0 {
		return nil, errors.Errorf("TotalSupply < 0")
	}
	return ts, varDB.Set(ts)
}

func (ctx *worldContextImpl) SetValidators(validators []module.Validator) error {
	return ctx.GetValidatorState().Set(validators)
}

func (ctx *worldContextImpl) GetScoreOwner(score module.Address) (module.Address, error) {
	if score == nil || !score.IsContract() {
		return nil, scoreresult.InvalidParameterError.Errorf("Invalid score address")
	}
	as := ctx.GetAccountState(score.ID())
	if icutils.IsNil(as) || !as.IsContract() {
		return nil, scoreresult.InvalidParameterError.Errorf("Score not found")
	}
	return as.ContractOwner(), nil
}

func (ctx *worldContextImpl) SetScoreOwner(from module.Address, score module.Address, newOwner module.Address) error {
	// Parameter sanity check
	if from == nil {
		return scoreresult.InvalidParameterError.Errorf("Invalid sender")
	}
	if score == nil {
		return scoreresult.InvalidParameterError.Errorf("Invalid score address")
	}
	if !score.IsContract() {
		return icmodule.IllegalArgumentError.Errorf("Invalid score address")
	}
	if newOwner == nil {
		return scoreresult.InvalidParameterError.Errorf("Invalid owner")
	}

	as := ctx.GetAccountState(score.ID())
	if icutils.IsNil(as) || !as.IsContract() {
		return icmodule.IllegalArgumentError.Errorf("Score not found")
	}

	// Check if s.from is the owner of a given contract
	owner := as.ContractOwner()
	if owner == nil || !owner.Equal(from) {
		return scoreresult.AccessDeniedError.Errorf("Invalid owner")
	}

	// Check if the score is active
	if as.IsBlocked() {
		return scoreresult.AccessDeniedError.Errorf("Blocked score")
	}
	if as.IsDisabled() {
		return scoreresult.AccessDeniedError.Errorf("Disabled score")
	}
	return as.SetContractOwner(newOwner)
}

func NewWorldContext(ctx state.WorldContext) icmodule.WorldContext {
	return &worldContextImpl{
		WorldContext: ctx,
	}
}

type callContextImpl struct {
	icmodule.WorldContext
	cc   contract.CallContext
	from module.Address
}

func (ctx *callContextImpl) From() module.Address {
	return ctx.from
}

func (ctx *callContextImpl) HandleBurn(requestor module.Address, amount *big.Int) error {
	sign := amount.Sign()
	if sign < 0 {
		return errors.Errorf("Invalid amount: %v", amount)
	}
	if sign > 0 {
		ts, err := ctx.AddTotalSupply(new(big.Int).Neg(amount))
		if err != nil {
			return err
		}
		ctx.onICXBurnedEvent(requestor, amount, ts)
	}
	return nil
}

func (ctx *callContextImpl) onICXBurnedEvent(requestor module.Address, amount, ts *big.Int) {
	rev := ctx.Revision().Value()
	if rev < icmodule.RevisionBurnV2 {
		var burnSig string
		if rev < icmodule.RevisionFixBurnEventSignature {
			burnSig = "ICXBurned"
		} else {
			burnSig = "ICXBurned(int)"
		}
		ctx.cc.OnEvent(state.SystemAddress,
			[][]byte{[]byte(burnSig)},
			[][]byte{intconv.BigIntToBytes(amount)},
		)
	} else {
		ctx.cc.OnEvent(state.SystemAddress,
			[][]byte{[]byte("ICXBurnedV2(Address,int,int)"), requestor.Bytes()},
			[][]byte{intconv.BigIntToBytes(amount), intconv.BigIntToBytes(ts)},
		)
	}
}

func (ctx *callContextImpl) SumOfStepUsed() *big.Int {
	return ctx.cc.SumOfStepUsed()
}

func (ctx *callContextImpl) OnEvent(addr module.Address, indexed, data [][]byte) {
	ctx.cc.OnEvent(addr, indexed, data)
}

func (ctx *callContextImpl) CallOnTimer(to module.Address, params []byte) error {
	cc := ctx.cc
	cm := cc.ContractManager()
	jso := &contract.DataCallJSON{Method: "onTimer", Params: params}
	callData, _ := json.Marshal(jso)
	sl := cc.GetStepLimit(state.StepLimitTypeInvoke)
	ch, err := cm.GetHandler(
		state.SystemAddress,
		to,
		new(big.Int),
		contract.CTypeCall,
		callData,
	)
	if err != nil {
		return err
	}
	if err, _, _, _ = cc.Call(ch, sl); err != nil {
		return err
	}
	return nil
}

func (ctx *callContextImpl) Governance() module.Address {
	return ctx.cc.Governance()
}

func (ctx *callContextImpl) FrameLogger() *trace.Logger {
	return ctx.cc.FrameLogger()
}

func NewCallContext(cc contract.CallContext, from module.Address) icmodule.CallContext {
	return &callContextImpl{
		WorldContext: NewWorldContext(cc),
		cc:           cc,
		from:         from,
	}
}
