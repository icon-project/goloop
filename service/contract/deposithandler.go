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
	"fmt"
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
var depositIssueRate = big.NewInt(8)

type DepositJSON struct {
	Action string           `json:"action"`
	ID     *common.HexBytes `json:"id,omitempty"`
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

type depositContext struct {
	CallContext
}

func (*depositContext) DepositIssueRate() *big.Int {
	return depositIssueRate
}

func NewDepositContext(cc CallContext) state.DepositContext {
	return &depositContext{cc}
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
		if h.data.ID != nil {
			return scoreresult.InvalidParameterError.New("UnknownField(id)"), nil, nil
		}
		if h.value == nil || h.value.Cmp(depositMinimumValue) < 0 {
			return scoreresult.InvalidParameterError.New("InvalidValue"), nil, nil
		}

		if cc.DepositTerm() == 0 {
			return scoreresult.MethodNotFoundError.New("NotSupported"), nil, nil
		}

		bal1 := as1.GetBalance()
		if bal1.Cmp(h.value) < 0 {
			return scoreresult.ErrOutOfBalance, nil, nil
		}

		as1.SetBalance(new(big.Int).Sub(bal1, h.value))
		dc := NewDepositContext(cc)
		period := int64(1)
		if err := as2.AddDeposit(dc, h.value, period); err != nil {
			return err, nil, nil
		}
		cc.OnEvent(h.to, [][]byte{
			[]byte("DepositAdd(bytes,Address,Address,int,int)"),
			cc.TransactionID(),
			h.to.Bytes(),
			h.from.Bytes(),
		}, [][]byte{
			intconv.BigIntToBytes(h.value),
			intconv.Int64ToBytes(dc.DepositTerm() * period),
		})
		return nil, nil, nil
	case DepositActionWithdraw:
		if h.data.ID == nil {
			return scoreresult.InvalidParameterError.New("IDNotFoundForWithdraw"), nil, nil
		}
		fmt.Printf("ID:%#x\n", h.data.ID.Bytes())
		dc := NewDepositContext(cc)
		if amount, fee, err := as2.WithdrawDeposit(dc, h.data.ID.Bytes()); err != nil {
			return err, nil, nil
		} else {
			treasury := cc.GetAccountState(cc.Treasury().ID())
			treasury.SetBalance(new(big.Int).Add(treasury.GetBalance(), fee))

			cc.OnEvent(h.to, [][]byte{
				[]byte("DepositWithdraw(bytes,Address,Address,int,int)"),
				h.data.ID.Bytes(),
				h.to.Bytes(),
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
