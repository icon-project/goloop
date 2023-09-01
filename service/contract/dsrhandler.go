/*
 * Copyright 2023 ICON Foundation
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
 *
 */

package contract

import (
	"bytes"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

type DSRHandler struct {
	*CommonHandler
	dsr *DoubleSignReport
}

// HandleDoubleSignReport has name of method of chain SCORE to handle double sign report.
// Its signature looks like "handleDoubleSignReport(type:str,height:int,signer:Address)".
const HandleDoubleSignReport = "handleDoubleSignReport"

func (h *DSRHandler) ExecuteSync(cc CallContext) (err error, ro *codec.TypedObj, _ module.Address) {
	h.Log.TSystemf("DSR start from=%s", h.From)
	defer h.Log.TSystemf("DSR done status=%v", err)

	return h.DoExecuteSync(cc)
}

func (h *DSRHandler) verifyHashOfHeight(cc CallContext, height int64, hash []byte) error {
	history, err := NewDSContextHistoryDB(cc.GetAccountState(state.SystemID))
	if err != nil {
		return err
	}
	vh := history.Get(height-2)
	if !bytes.Equal(vh, hash) {
		return scoreresult.InvalidParameterError.New("FailToVerifyContextData")
	}
	return nil
}

func (h *DSRHandler) DoExecuteSync(cc CallContext) (error, *codec.TypedObj, module.Address) {
	dsds, dsc, err := h.dsr.Decode(cc, true)
	if err != nil {
		return scoreresult.InvalidParameterError.Wrap(err, "InvalidFormat"), nil, nil
	}
	dsr1, dsr2 := dsds[0], dsds[1]
	if !dsr1.IsConflictWith(dsr2) {
		return scoreresult.InvalidParameterError.Wrap(err, "DoubleSignDataDoesntConflict"), nil, nil
	}

	if dsr1.Height() > cc.BlockHeight() {
		return scoreresult.InvalidParameterError.New("FutureDoubleSignReport"), nil, nil
	}

	signer := dsc.AddressOf(dsr1.Signer())
	if signer == nil {
		return scoreresult.InvalidParameterError.New("NotValidSigner"), nil, nil
	}

	if err := h.verifyHashOfHeight(cc, dsr1.Height(), dsc.Hash()); err != nil {
		return err, nil, nil
	}

	params := common.MustEncodeAny([]interface{} { dsr1.Type(), dsr1.Height(), signer })
	ch := NewCommonHandler(state.SystemAddress, state.SystemAddress, nil, true, h.Log)
	ah := newCallHandlerWithParams(ch, HandleDoubleSignReport, params, false)

	err, _, _, _ = cc.Call(ah, cc.GetStepLimit(state.StepLimitTypeInvoke))
	return err, nil, nil
}

type DoubleSignReport struct {
	Type    string            `json:"type"`
	Data    []common.HexBytes `json:"data"`
	Context common.HexBytes   `json:"context"`

	dsd []module.DoubleSignData
	dsc module.DoubleSignContext
}

func NewDoubleSignReport(data []module.DoubleSignData, context module.DoubleSignContext) *DoubleSignReport {
	data1 := data[0].Bytes()
	data2 := data[1].Bytes()
	if bytes.Compare(data1, data2) > 0 {
		data1, data2 = data2, data1
		data = []module.DoubleSignData{ data[1], data[0] }
	}
	return &DoubleSignReport{
		Type:    data[0].Type(),
		Data:    []common.HexBytes{data1, data2},
		Context: context.Bytes(),
		dsd:     data,
		dsc:	 context,
	}
}

func (dsr *DoubleSignReport) Decode(wc state.WorldContext, force bool) ([]module.DoubleSignData, module.DoubleSignContext, error) {
	if dsr.dsd == nil || dsr.dsc == nil || force {
		if len(dsr.Data) != 2 {
			return nil, nil, errors.New("InvalidDataLength")
		}
		data1 := dsr.Data[0].Bytes()
		data2 := dsr.Data[1].Bytes()
		if bytes.Compare(data1, data2) > 0 {
			return nil, nil, errors.New("InvalidDataOrder")
		}
		d1, err := wc.DecodeDoubleSignData(dsr.Type, data1)
		if err != nil {
			return nil, nil, errors.Wrap(err, "InvalidData1")
		}
		d2, err := wc.DecodeDoubleSignData(dsr.Type, data2)
		if err != nil {
			return nil, nil, errors.Wrap(err, "InvalidData2")
		}
		dsc, err := wc.DecodeDoubleSignContext(dsr.Type, dsr.Context.Bytes())
		if err != nil {
			return nil, nil, errors.Wrap(err, "InvalidContext")
		}
		dsr.dsd = []module.DoubleSignData{ d1, d2 }
		dsr.dsc = dsc
	}
	return dsr.dsd, dsr.dsc, nil
}

func NewDSRHandler(from module.Address, dsr *DoubleSignReport, logger log.Logger) *DSRHandler {
	ch := NewCommonHandler(from, state.SystemAddress, new(big.Int), false, logger)
	return &DSRHandler{
		CommonHandler: ch,
		dsr:           dsr,
	}
}

