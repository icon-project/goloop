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

package icon

import (
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

type TransferHandler struct {
	*contract.CommonHandler
}

func newTransferHandler(from, to module.Address, value *big.Int, call bool, logger log.Logger) *TransferHandler {
	return &TransferHandler{
		contract.NewCommonHandler(from, to, value, call, logger),
	}
}

func (h *TransferHandler) ExecuteSync(cc contract.CallContext) (err error, ro *codec.TypedObj, addr module.Address) {
	h.Log.TSystemf("TRANSFER start from=%s to=%s value=%s", h.From, h.To, h.Value)
	defer func() {
		if err != nil {
			h.Log.TSystemf("TRANSFER done status=%s msg=%v", err.Error(), err)
		}
	}()

	if err2 := h.ApplyStepsForInterCall(cc); err2 != nil {
		return err2, nil, nil
	}
	return h.DoExecuteSync(cc)
}

func (h *TransferHandler) DoExecuteSync(cc contract.CallContext) (err error, ro *codec.TypedObj, addr module.Address) {
	if cc.ReadOnlyMode() {
		return scoreresult.AccessDeniedError.New("TransferIsNotAllowed"), nil, nil
	}
	as1 := cc.GetAccountState(h.From.ID())
	if as1.IsContract() != h.From.IsContract() {
		return scoreresult.InvalidParameterError.Errorf(
			"InvalidAddress(%s)", h.From.String()), nil, nil
	}
	if h.Value.Sign() == -1 {
		return scoreresult.InvalidParameterError.Errorf(
			"InvalidValue(value=%s)", h.Value.String()), nil, nil
	}
	bal1 := as1.GetBalance()
	if bal1.Cmp(h.Value) < 0 {
		return scoreresult.ErrOutOfBalance, nil, nil
	}
	as1.SetBalance(new(big.Int).Sub(bal1, h.Value))

	as2 := cc.GetAccountState(h.To.ID())
	opType := module.Transfer
	recipient := h.To
	if as2.IsContract() {
		cc.Logger().Debugf("LOST transfer address=%s", h.To.String())
		as2 = cc.GetAccountState(state.LostID)
		opType = module.Lost
		recipient = state.LostAddress
	}
	bal2 := as2.GetBalance()
	as2.SetBalance(new(big.Int).Add(bal2, h.Value))

	if h.From.IsContract() && h.Value.Sign() > 0 {
		indexed := make([][]byte, 4, 4)
		indexed[0] = []byte(txresult.EventLogICXTransfer)
		indexed[1] = h.From.Bytes()
		indexed[2] = h.To.Bytes()
		indexed[3] = intconv.BigIntToBytes(h.Value)
		cc.OnEvent(h.From, indexed, make([][]byte, 0))
	}

	h.Log.OnBalanceChange(opType, h.From, recipient, h.Value)
	return nil, nil, nil
}
