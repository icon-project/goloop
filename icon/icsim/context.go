/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package icsim

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/trace"
)

var (
	treasury = common.MustNewAddressFromString("hx1000000000000000000000000000000000000000")
)

type WorldContext interface {
	icmodule.WorldContext
	GetExtensionState() state.ExtensionState
	BlockTimeStamp() int64
}

type worldContext struct {
	state.WorldState
	blockHeight    int64
	blockTimestamp int64
	csi            module.ConsensusInfo
	origin         module.Address
	revision       module.Revision
	stepPrice      *big.Int
	txId           []byte
}

func (ctx *worldContext) addBalance(address module.Address, amount *big.Int) error {
	as := ctx.GetAccountState(address.ID())
	ob := as.GetBalance()
	return setBalance(address, as, new(big.Int).Add(ob, amount))
}

func (ctx *worldContext) Revision() module.Revision {
	return ctx.revision
}

func (ctx *worldContext) BlockHeight() int64 {
	return ctx.blockHeight
}

func (ctx *worldContext) Origin() module.Address {
	return ctx.origin
}

func (ctx *worldContext) Treasury() module.Address {
	return treasury
}

func (ctx *worldContext) TransactionID() []byte {
	return ctx.txId
}

func (ctx *worldContext) ConsensusInfo() module.ConsensusInfo {
	return ctx.csi
}

func (ctx *worldContext) GetBalance(address module.Address) *big.Int {
	account := ctx.GetAccountState(address.ID())
	return account.GetBalance()
}

func (ctx *worldContext) Deposit(address module.Address, amount *big.Int) error {
	if err := validateAmount(amount); err != nil {
		return err
	}
	if amount.Sign() == 0 {
		return nil
	}
	return ctx.addBalance(address, amount)
}

func (ctx *worldContext) Withdraw(address module.Address, amount *big.Int) error {
	if err := validateAmount(amount); err != nil {
		return err
	}
	if amount.Sign() == 0 {
		return nil
	}
	return ctx.addBalance(address, new(big.Int).Neg(amount))
}

func (ctx *worldContext) Transfer(from module.Address, to module.Address, amount *big.Int) (err error) {
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

func (ctx *worldContext) GetTotalSupply() *big.Int {
	as := ctx.GetAccountState(state.SystemID)
	tsVar := scoredb.NewVarDB(as, state.VarTotalSupply)
	if ts := tsVar.BigInt(); ts != nil {
		return ts
	}
	return icmodule.BigIntZero
}

func (ctx *worldContext) AddTotalSupply(amount *big.Int) (*big.Int, error) {
	as := ctx.GetAccountState(state.SystemID)
	varDB := scoredb.NewVarDB(as, state.VarTotalSupply)
	ts := new(big.Int).Add(varDB.BigInt(), amount)
	if ts.Sign() < 0 {
		return nil, errors.Errorf("TotalSupply < 0")
	}
	return ts, varDB.Set(ts)
}

func (ctx *worldContext) SetValidators(validators []module.Validator) error {
	return ctx.GetValidatorState().Set(validators)
}

func (ctx *worldContext) SumOfStepUsed() *big.Int {
	return icmodule.BigIntZero
}

func (ctx *worldContext) StepPrice() *big.Int {
	return ctx.stepPrice
}

func (ctx *worldContext) BlockTimeStamp() int64 {
	return ctx.blockTimestamp
}

func (ctx *worldContext) GetScoreOwner(score module.Address) (module.Address, error) {
	if score == nil || !score.IsContract() {
		return nil, scoreresult.InvalidParameterError.Errorf("Invalid score address")
	}
	as := ctx.GetAccountState(score.ID())
	if as == nil || !as.IsContract() {
		return nil, scoreresult.InvalidParameterError.Errorf("Invalid score account")
	}
	return as.ContractOwner(), nil
}

func (ctx *worldContext) SetScoreOwner(from module.Address, score module.Address, owner module.Address) error {
	// Parameter sanity check
	if !score.IsContract() {
		return scoreresult.InvalidParameterError.Errorf("Invalid score address")
	}
	if from == nil || from.Equal(owner) {
		return scoreresult.InvalidParameterError.Errorf("Invalid owner")
	}

	as := ctx.GetAccountState(score.ID())
	if !as.IsContract() {
		return scoreresult.InvalidParameterError.Errorf("Invalid score account")
	}

	// Check if s.from is the owner of a given contract
	oldOwner := as.ContractOwner()
	if oldOwner == nil || !oldOwner.Equal(from) {
		return scoreresult.InvalidParameterError.Errorf("Invalid owner: %s != %s", oldOwner, owner)
	}

	// Check if the score is active
	if as.IsBlocked() {
		return scoreresult.AccessDeniedError.Errorf("Not allowed: blocked score")
	}
	if as.IsDisabled() {
		return scoreresult.AccessDeniedError.Errorf("Not allowed: disabled score")
	}
	return as.SetContractOwner(owner)
}

func NewWorldContext(
	ws state.WorldState, blockHeight int64, revision module.Revision,
	csi module.ConsensusInfo, stepPrice *big.Int,
) WorldContext {
	return &worldContext{
		WorldState:     ws,
		blockHeight:    blockHeight,
		blockTimestamp: blockHeight * 2_000_000,
		csi:            csi,
		revision:       revision,
		stepPrice:      stepPrice,
	}
}

type callContext struct {
	WorldContext
	from module.Address
}

func (ctx *callContext) From() module.Address {
	return ctx.from
}

func (ctx *callContext) HandleBurn(from module.Address, amount *big.Int) error {
	sign := amount.Sign()
	if sign < 0 {
		return errors.Errorf("Invalid amount: %v", amount)
	}
	if sign > 0 {
		ts, err := ctx.AddTotalSupply(new(big.Int).Neg(amount))
		if err != nil {
			return err
		}
		ctx.onICXBurnedEvent(from, amount, ts)
	}
	return nil
}

func (ctx *callContext) onICXBurnedEvent(from module.Address, amount, ts *big.Int) {
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
			[][]byte{[]byte("ICXBurnedV2(Address,int,int)"), from.Bytes()},
			[][]byte{intconv.BigIntToBytes(amount), intconv.BigIntToBytes(ts)},
		)
	}
}

func (ctx *callContext) SumOfStepUsed() *big.Int {
	return icmodule.BigIntZero
}

func (ctx *callContext) OnEvent(addr module.Address, indexed, data [][]byte) {
}

func (ctx *callContext) CallOnTimer(to module.Address, params []byte) error {
	return nil
}

func (ctx *callContext) Governance() module.Address {
	return ctx.Governance()
}

func (ctx *callContext) FrameLogger() *trace.Logger {
	return trace.LoggerOf(log.GlobalLogger())
}

func (ctx *callContext) TransactionInfo() *state.TransactionInfo {
	panic("implement me")
}

func NewCallContext(wc WorldContext, from module.Address) icmodule.CallContext {
	return &callContext{
		WorldContext: wc,
		from:         from,
	}
}
