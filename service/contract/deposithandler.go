/*
 * Copyright 2020 ICON Foundation
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

package contract

import (
	"bytes"
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

const (
	DepositActionAdd      = "add"
	DepositActionWithdraw = "withdraw"
)

var depositMinimumValue, _ = new(big.Int).SetString("5000_000000000000000000", 0)
var depositMaximumValue, _ = new(big.Int).SetString("100_000_000000000000000000", 0)

type DepositJSON struct {
	Action string           `json:"action"`
	ID     *common.HexBytes `json:"id,omitempty"`
	Amount *common.HexInt   `json:"amount,omitempty"`
}

func ParseDepositData(data []byte) (*DepositJSON, error) {
	jso := new(DepositJSON)
	jd := json.NewDecoder(bytes.NewBuffer(data))
	jd.DisallowUnknownFields()
	if err := jd.Decode(jso); err != nil {
		return nil, err
	}
	return jso, nil
}

type DepositHandler struct {
	*CommonHandler
	data *DepositJSON
}

func (h *DepositHandler) Prepare(ctx Context) (state.WorldContext, error) {
	var lq []state.LockRequest
	if h.data != nil && h.data.Action == DepositActionWithdraw {
		lq = []state.LockRequest{
			{state.WorldIDStr, state.AccountWriteLock},
		}
	} else {
		lq = []state.LockRequest{
			{string(h.From.ID()), state.AccountWriteLock},
			{string(h.To.ID()), state.AccountWriteLock},
		}
	}
	return ctx.GetFuture(lq), nil
}

func (h *DepositHandler) ExecuteSync(cc CallContext) (err error, ro *codec.TypedObj, addr module.Address) {
	var action string
	if h.data != nil {
		action = h.data.Action
	} else {
		action = "None"
	}
	h.Log.TSystemf("DEPOSIT start to=%s action=%s", h.To, action)
	defer func() {
		if err != nil {
			h.Log.TSystemf("DEPOSIT done status=%s msg=%v", err.Error(), err)
		}
	}()

	if err2 := h.ApplyStepsForInterCall(cc); err2 != nil {
		return err2, nil, nil
	}

	if cc.QueryMode() {
		return scoreresult.AccessDeniedError.New("DepositControlIsNotAllowed"), nil, nil
	}

	if h.data == nil {
		return scoreresult.InvalidParameterError.New("InvalidDepositData"), nil, nil
	}

	as1 := cc.GetAccountState(h.From.ID())
	if as1.IsContract() != h.From.IsContract() {
		return scoreresult.InvalidRequestError.Errorf(
			"InvalidAddress(%s)", h.From.String()), nil, nil
	}

	as2 := cc.GetAccountState(h.To.ID())
	if as2.IsContract() != h.To.IsContract() || !as2.IsContract() {
		return scoreresult.InvalidRequestError.Errorf(
			"InvalidAddress(%s)", h.To.String()), nil, nil
	}

	if owner := as2.ContractOwner(); !owner.Equal(h.From) {
		return scoreresult.InvalidRequestError.New("NotOwner"), nil, nil
	}

	switch h.data.Action {
	case DepositActionAdd:
		if !cc.FeeSharingEnabled() {
			return scoreresult.MethodNotFoundError.New("NotSupported"), nil, nil
		}

		// if h.data.ID != nil {
		// 	return scoreresult.IllegalFormatError.New("UnknownField(id)"), nil, nil
		// }
		if h.Value == nil || h.Value.Sign() == -1 {
			return scoreresult.InvalidRequestError.New("InvalidValue"), nil, nil
		}

		id := cc.TransactionID()
		term := cc.DepositTerm()
		if term > 0 {
			if h.Value.Cmp(depositMinimumValue) < 0 || h.Value.Cmp(depositMaximumValue) > 0 {
				return scoreresult.InvalidRequestError.New("InvalidValue"), nil, nil
			}
		} else {
			id = []byte{}
		}

		bal1 := as1.GetBalance()
		if bal1.Cmp(h.Value) < 0 {
			return scoreresult.ErrOutOfBalance, nil, nil
		}

		as1.SetBalance(new(big.Int).Sub(bal1, h.Value))
		if err := as2.AddDeposit(cc, h.Value); err != nil {
			return err, nil, nil
		}
		cc.OnEvent(h.To, [][]byte{
			[]byte("DepositAdded(bytes,Address,int,int)"),
			id,
			h.From.Bytes(),
		}, [][]byte{
			intconv.BigIntToBytes(h.Value),
			intconv.Int64ToBytes(term),
		})
		h.Log.OnBalanceChange(module.FSDeposit, h.From, nil, h.Value)
		return nil, nil, nil
	case DepositActionWithdraw:
		if h.Value != nil && h.Value.Sign() != 0 {
			return scoreresult.MethodNotPayableError.Errorf(
				"NotPayable(value=%d)", h.Value), nil, nil
		}

		var id []byte
		if h.data.ID != nil {
			id = h.data.ID.Bytes()
		} else {
			id = []byte{}
		}
		value := h.data.Amount.Value()

		if amount, fee, err := as2.WithdrawDeposit(cc, id, value); err != nil {
			return err, nil, nil
		} else {
			as1.SetBalance(new(big.Int).Add(as1.GetBalance(), amount))

			treasury := cc.GetAccountState(cc.Treasury().ID())
			treasury.SetBalance(new(big.Int).Add(treasury.GetBalance(), fee))

			cc.OnEvent(h.To, [][]byte{
				[]byte("DepositWithdrawn(bytes,Address,int,int)"),
				id,
				h.From.Bytes(),
			}, [][]byte{
				intconv.BigIntToBytes(amount),
				intconv.BigIntToBytes(fee),
			})

			h.Log.OnBalanceChange(module.FSWithdraw, nil, h.From, amount)
			h.Log.OnBalanceChange(module.FSFee, nil, cc.Treasury(), fee)
			return nil, nil, nil
		}
	default:
		return scoreresult.InvalidRequestError.Errorf(
			"InvalidAction(action=%s)", h.data.Action), nil, nil
	}
}

func newDepositHandler(ch *CommonHandler, data []byte) (ContractHandler, error) {
	dd, _ := ParseDepositData(data)
	return &DepositHandler{
		CommonHandler: ch,
		data:          dd,
	}, nil
}
