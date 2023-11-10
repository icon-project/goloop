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

package transaction

import (
	"bytes"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/trace"
	"github.com/icon-project/goloop/service/txresult"
)

func serializeDoubleSignReport(d *contract.DoubleSignReport, buf *bytes.Buffer) {
	buf.Write([]byte("context."))
	buf.Write([]byte(d.Context.String()))

	buf.Write([]byte(".data.["))
	for idx, item := range d.Data {
		value := item.String()
		if idx>0 {
			buf.Write([]byte("."))
		}
		buf.Write([]byte(value))
	}

	buf.Write([]byte("]"))

	buf.Write([]byte(".type."+string(serializeString(d.Type))))
}

type doubleSignReportTxData struct {
	Version   common.HexUint16  `json:"version"`
	From      *common.Address   `json:"from,omitempty"`
	Timestamp common.HexInt64   `json:"timestamp"`
	DataType  string                     `json:"dataType"`
	Data      *contract.DoubleSignReport `json:"data"`
	NID       common.HexInt64 			 `json:"nid"`
	Signature *common.Signature          `json:"signature,omitempty"`
}

func (d *doubleSignReportTxData) serialize(buf *bytes.Buffer) {
	buf.Write([]byte("icx_sendTransaction"))

	buf.Write([]byte(".data.{"))
	serializeDoubleSignReport(d.Data, buf)
	buf.Write([]byte("}"))

	buf.Write([]byte(".dataType."))
	buf.Write(serializeString(d.DataType))

	if d.From != nil {
		buf.Write([]byte(".from."))
		buf.Write([]byte(d.From.String()))
	}
	buf.Write([]byte(".nid."))
	buf.Write([]byte(d.NID.String()))

	buf.Write([]byte(".timestamp."))
	buf.Write([]byte(d.Timestamp.String()))

	buf.Write([]byte(".version."))
	buf.Write([]byte(d.Version.String()))
}

type doubleSignReportTx struct {
	wrapper[doubleSignReportTxData,*doubleSignReportTxData]
}

func (tx *doubleSignReportTx) Group() module.TransactionGroup {
	return module.TransactionGroupNormal
}

func (tx *doubleSignReportTx) From() module.Address {
	if tx.data.From == nil {
		return state.SystemAddress
	} else {
		return tx.data.From
	}
}

func (tx *doubleSignReportTx) Verify() error {
	if tx.data.Data == nil {
		return InvalidFormat.New("NoDSR")
	}
	if size := len(tx.data.Data.Data) ; size != 2 {
		return InvalidFormat.Errorf("InsufficientDoubleSignData(len=%d)", size)
	}
	if tx.data.From != nil && tx.data.Signature != nil {
		pk, err := tx.data.Signature.RecoverPublicKey(tx.ID())
		if err != nil {
			return InvalidSignatureError.New("FailToRecover")
		}
		signer := common.NewAccountAddressFromPublicKey(pk)
		if !common.AddressEqual(tx.data.From, signer) {
			return InvalidSignatureError.Errorf("InvalidSignature(from=%s,signer=%s)",
				tx.data.From.String(), signer.String())
		}
	} else if tx.data.From != nil || tx.data.Signature != nil {
		return InvalidFormat.New("InvalidSignature")
	}
	return nil
}

func (tx *doubleSignReportTx) Version() int {
	return int(tx.data.Version.Value)
}

func (tx *doubleSignReportTx) ToJSON(version module.JSONVersion) (interface{}, error) {
	jso := map[string]interface{}{
		"dataType": tx.data.DataType,
		"data": tx.data.Data,
		"nid": tx.data.NID,
		"txHash": common.HexBytes(tx.ID()),
		"timestamp": tx.data.Timestamp,
		"version": tx.data.Version,
	}
	if tx.data.From != nil {
		jso["from"] = tx.data.From
		jso["signature"] = tx.data.Signature
	}
	return jso, nil
}

func (tx *doubleSignReportTx) ValidateNetwork(nid int) bool {
	return tx.data.NID.Value == int64(nid)
}

func (tx *doubleSignReportTx) decodeDoubleSignReport(wc state.WorldContext) ([]module.DoubleSignData, module.DoubleSignContext, error) {
	data, context, err := tx.data.Data.Decode(wc, false)
	if err != nil {
		return nil, nil, InvalidFormat.Wrap(err, "InvalidTransactionData")
	} else {
		return data, context, nil
	}
}

func (tx *doubleSignReportTx) PreValidate(wc state.WorldContext, update bool) error {
	if !wc.Revision().Has(module.ReportDoubleSign) {
		return errors.InvalidStateError.New("ReportDoubleSignIsDisabled")
	}
	if tx.data.From != nil {
		return InvalidFormat.New("UnsupportedDoubleSignReport")
	}
	if data, dsc, err := tx.decodeDoubleSignReport(wc) ; err != nil {
		return err
	} else {
		if !data[0].ValidateNetwork(int(tx.data.NID.Value)) || !data[1].ValidateNetwork(int(tx.data.NID.Value)) {
			return InvalidFormat.New("NotBelongToCurrentNetwork")
		}
		if !data[0].IsConflictWith(data[1]) || dsc.AddressOf(data[0].Signer())==nil {
			return InvalidFormat.New("InvalidDoubleSignReport")
		}
	}
	return nil
}

func (tx *doubleSignReportTx) GetHandler(cm contract.ContractManager) (Handler, error) {
	if tx.data.From != nil {
		return nil, errors.UnsupportedError.New("UnsupportedToHandleNormalTX")
	}
	handler := contract.NewDSRHandler(
		tx.From(), tx.data.Data, cm.Logger())
	return &dsrTxHandler{handler: handler}, nil
}

func (tx *doubleSignReportTx) Timestamp() int64 {
	return tx.data.Timestamp.Value
}

func (tx *doubleSignReportTx) Nonce() *big.Int {
	return nil
}

func (tx *doubleSignReportTx) To() module.Address {
	return state.SystemAddress
}

func (tx *doubleSignReportTx) IsSkippable() bool {
	return false
}

func NewDoubleSignReportTx(data []module.DoubleSignData, context module.DoubleSignContext, nid int, ts int64) Transaction {
	tx := new(doubleSignReportTx)
	tx.data.Version.Value = Version3
	tx.data.NID.Value = int64(nid)
	tx.data.Timestamp.Value = ts
	tx.data.DataType = contract.DataTypeDSR
	tx.data.Data = contract.NewDoubleSignReport(data, context)
	return Wrap(tx)
}

type dsrTxHandler struct {
	handler *contract.DSRHandler
	cc      contract.CallContext
	log     *trace.Logger
}

func (th *dsrTxHandler) Prepare(ctx contract.Context) (state.WorldContext, error) {
	return th.handler.Prepare(ctx)
}

func (th *dsrTxHandler) Execute(ctx contract.Context, wcs state.WorldSnapshot, estimate bool) (txresult.Receipt, error) {
	invokeLimit := ctx.GetStepLimit(state.StepLimitTypeInvoke)
	cc := contract.NewCallContext(ctx, invokeLimit, false)
	th.cc = cc
	th.log = cc.FrameLogger()

	status, _, _ := th.handler.ExecuteSync(cc)

	if status != nil {
		th.log.Warnf("Fail to handle DSR err=%+v", status)
		return nil, errors.IllegalArgumentError.Wrap(status, "InvalidDSR")
	}

	receipt := txresult.NewReceipt(ctx.Database(), ctx.Revision(), state.SystemAddress)
	cc.GetEventLogs(receipt)
	cc.GetBTPMessages(receipt)
	receipt.SetResult(module.StatusSuccess, new(big.Int), new(big.Int), nil)
	return receipt, nil
}

func (th *dsrTxHandler) Dispose() {
	if th.cc != nil {
		th.cc.Dispose()
		th.cc = nil
	}
}

type doubleSignReportTxHeader struct {
	Version   common.HexUint16  `json:"version"`
	From      *common.Address   `json:"from,omitempty"`
	Timestamp common.HexInt64   `json:"timestamp"`
	DataType  string            `json:"dataType,omitempty"`
}

func checkDSRTxBytes(bs []byte) bool {
	var th doubleSignReportTxHeader
	if _, err := codec.BC.UnmarshalFromBytes(bs, &th); err != nil {
		return false
	}
	return th.From == nil && th.DataType == contract.DataTypeDSR
}

func parseDSRTxBytes(bs []byte) (Transaction, error) {
	tx := new(doubleSignReportTx)
	if _, err := codec.BC.UnmarshalFromBytes(bs, &tx.data); err != nil {
		return nil, err
	} else {
		return tx, nil
	}
}

func init() {
	RegisterFactory(&Factory{
		Priority: 10,
		CheckBinary: checkDSRTxBytes,
		ParseBinary: parseDSRTxBytes,
	})
}

func TryGetDoubleSignReportInfo(wc state.WorldContext, tx module.Transaction) (int64, module.Address, bool) {
	dsr, ok := Unwrap(tx).(*doubleSignReportTx)
	if !ok {
		return 0, nil, false
	}
	data, context, err := dsr.decodeDoubleSignReport(wc)
	if err != nil {
		return 0, nil, false
	}
	signer := context.AddressOf(data[0].Signer())
	if signer == nil {
		return 0, nil, false
	}
	return data[0].Height(), signer, true
}