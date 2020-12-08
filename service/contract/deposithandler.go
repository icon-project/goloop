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
	"github.com/icon-project/goloop/service/trace"
)

const (
	DepositActionAdd      = "add"
	DepositActionWithdraw = "withdraw"
)

var depositMinimumValue, _ = new(big.Int).SetString("5000000000000000000000", 10)

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
			{string(h.from.ID()), state.AccountWriteLock},
			{string(h.to.ID()), state.AccountWriteLock},
		}
	}
	return ctx.GetFuture(lq), nil
}

func (h *DepositHandler) ExecuteSync(cc CallContext) (err error, ro *codec.TypedObj, addr module.Address) {
	h.log = trace.LoggerOf(cc.Logger())

	h.log.TSystemf("DEPOSIT start to=%s action=%s", h.to, h.data.Action)
	defer func() {
		if err != nil {
			h.log.TSystemf("DEPOSIT done status=%s msg=%v", err.Error(), err)
		}
	}()

	if !h.ApplyStepsForInterCall(cc) {
		return scoreresult.OutOfStepError.New("OutOfStepForInterCall"), nil, nil
	}

	if cc.QueryMode() {
		return scoreresult.AccessDeniedError.New("DepositControlIsNotAllowed"), nil, nil
	}

	as1 := cc.GetAccountState(h.from.ID())
	if as1.IsContract() != h.from.IsContract() {
		return scoreresult.InvalidParameterError.Errorf(
			"InvalidAddress(%s)", h.from.String()), nil, nil
	}

	as2 := cc.GetAccountState(h.to.ID())
	if as2.IsContract() != h.to.IsContract() || !as2.IsContract() {
		return scoreresult.InvalidParameterError.Errorf(
			"InvalidAddress(%s)", h.to.String()), nil, nil
	}

	if owner := as2.ContractOwner(); !owner.Equal(h.from) {
		return scoreresult.AccessDeniedError.New("NotOwner"), nil, nil
	}

	switch h.data.Action {
	case DepositActionAdd:
		if !cc.FeeSharingEnabled() {
			return scoreresult.MethodNotFoundError.New("NotSupported"), nil, nil
		}

		if h.data.ID != nil {
			return scoreresult.InvalidParameterError.New("UnknownField(id)"), nil, nil
		}
		if h.value == nil || h.value.Sign() == -1 {
			return scoreresult.InvalidParameterError.New("InvalidValue"), nil, nil
		}

		id := cc.TransactionID()
		term := cc.DepositTerm()
		if term > 0 {
			if h.value.Cmp(depositMinimumValue) < 0 {
				return scoreresult.InvalidParameterError.New("InvalidValue"), nil, nil
			}
		} else {
			id = []byte{}
		}

		bal1 := as1.GetBalance()
		if bal1.Cmp(h.value) < 0 {
			return scoreresult.ErrOutOfBalance, nil, nil
		}

		as1.SetBalance(new(big.Int).Sub(bal1, h.value))
		if err := as2.AddDeposit(cc, h.value); err != nil {
			return err, nil, nil
		}
		cc.OnEvent(h.to, [][]byte{
			[]byte("DepositAdded(bytes,Address,int,int)"),
			id,
			h.from.Bytes(),
		}, [][]byte{
			intconv.BigIntToBytes(h.value),
			intconv.Int64ToBytes(term),
		})
		return nil, nil, nil
	case DepositActionWithdraw:
		if h.value != nil && h.value.Sign() != 0 {
			return scoreresult.MethodNotPayableError.Errorf(
				"NotPayable(value=%d)", h.value), nil, nil
		}

		id := h.data.ID.Bytes()
		value := h.data.Amount.Value()
		if id == nil {
			id = []byte{}
		}

		if amount, fee, err := as2.WithdrawDeposit(cc, id, value); err != nil {
			return err, nil, nil
		} else {
			as1.SetBalance(new(big.Int).Add(as1.GetBalance(), amount))

			treasury := cc.GetAccountState(cc.Treasury().ID())
			treasury.SetBalance(new(big.Int).Add(treasury.GetBalance(), fee))

			cc.OnEvent(h.to, [][]byte{
				[]byte("DepositWithdrawn(bytes,Address,int,int)"),
				id,
				h.from.Bytes(),
			}, [][]byte{
				intconv.BigIntToBytes(amount),
				intconv.BigIntToBytes(fee),
			})
			return nil, nil, nil
		}
	default:
		return scoreresult.InvalidParameterError.Errorf(
			"InvalidAction(action=%s)", h.data.Action), nil, nil
	}
}

func newDepositHandler(ch *CommonHandler, data []byte) (ContractHandler, error) {
	dd, err := ParseDepositData(data)
	if err != nil {
		return nil, scoreresult.InvalidParameterError.Wrap(err, "InvalidData")
	}
	return &DepositHandler{
		CommonHandler: ch,
		data:          dd,
	}, nil
}
