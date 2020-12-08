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

// -build base

package iiss

import (
	"bytes"
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

type baseV3Data struct {
	Version   common.HexUint16 `json:"version"`
	From      *common.Address  `json:"from,omitempty"` // it should be nil
	TimeStamp common.HexInt64  `json:"timestamp"`
	DataType  string           `json:"dataType,omitempty"`
	Data      json.RawMessage  `json:"data,omitempty"`
}

func (tx *baseV3Data) calcHash() ([]byte, error) {
	sha := bytes.NewBuffer(nil)
	sha.Write([]byte("icx_sendTransaction"))

	// data
	if tx.Data != nil {
		sha.Write([]byte(".data."))
		if len(tx.Data) > 0 {
			var obj interface{}
			if err := json.Unmarshal(tx.Data, &obj); err != nil {
				return nil, err
			}
			if bs, err := transaction.SerializeValue(obj); err != nil {
				return nil, err
			} else {
				sha.Write(bs)
			}
		}
	}

	// dataType
	sha.Write([]byte(".dataType."))
	sha.Write([]byte(tx.DataType))

	// timestamp
	sha.Write([]byte(".timestamp."))
	sha.Write([]byte(tx.TimeStamp.String()))

	// version
	sha.Write([]byte(".version."))
	sha.Write([]byte(tx.Version.String()))

	return crypto.SHA3Sum256(sha.Bytes()), nil
}

type baseV3 struct {
	baseV3Data

	id    []byte
	hash  []byte
	bytes []byte
}

func (tx *baseV3) Version() int {
	return module.TransactionVersion3
}

func (tx *baseV3) Prepare(ctx contract.Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{state.WorldIDStr, state.AccountWriteLock},
	}
	wc := ctx.GetFuture(lq)
	wc.WorldVirtualState().Ensure()

	return wc, nil
}

func (tx *baseV3) Execute(ctx contract.Context, estimate bool) (txresult.Receipt, error) {
	info := ctx.TransactionInfo()
	if info == nil {
		return nil, errors.InvalidStateError.New("TransactionInfoUnavailable")
	}
	if info.Index != 0 {
		return nil, errors.CriticalFormatError.New("BaseMustBeTheFirst")
	}
	// TODO process something for base tx
	return nil, nil
}

func (tx *baseV3) Dispose() {
	panic("implement me")
}

func (tx *baseV3) Group() module.TransactionGroup {
	return module.TransactionGroupNormal
}

func (tx *baseV3) ID() []byte {
	if tx.id == nil {
		if bs, err := tx.baseV3Data.calcHash(); err != nil {
			panic(err)
		} else {
			tx.id = bs
		}
	}
	return tx.id
}

func (tx *baseV3) From() module.Address {
	return state.SystemAddress
}

func (tx *baseV3) Bytes() []byte {
	if tx.bytes == nil {
		if bs, err := codec.BC.MarshalToBytes(&tx.baseV3Data); err != nil {
			panic(err)
		} else {
			tx.bytes = bs
		}
	}
	return tx.bytes
}

func (tx *baseV3) Hash() []byte {
	if tx.hash == nil {
		tx.hash = crypto.SHA3Sum256(tx.Bytes())
	}
	return tx.hash
}

func (tx *baseV3) Verify() error {
	return nil
}

func (tx *baseV3) ToJSON(version module.JSONVersion) (interface{}, error) {
	jso := map[string]interface{}{
		"version":   &tx.baseV3Data.Version,
		"timestamp": &tx.baseV3Data.TimeStamp,
		"dataType":  tx.baseV3Data.DataType,
		"data":      tx.baseV3Data.Data,
	}
	jso["txHash"] = common.HexBytes(tx.ID())
	return jso, nil
}

func (tx *baseV3) ValidateNetwork(nid int) bool {
	return true
}

func (tx *baseV3) PreValidate(wc state.WorldContext, update bool) error {
	return nil
}

func (tx *baseV3) GetHandler(cm contract.ContractManager) (transaction.Handler, error) {
	return tx, nil
}

func (tx *baseV3) Timestamp() int64 {
	return tx.baseV3Data.TimeStamp.Value
}

func (tx *baseV3) Nonce() *big.Int {
	return nil
}

func (tx *baseV3) To() module.Address {
	return state.SystemAddress
}

func checkBaseV3JSON(jso map[string]interface{}) bool {
	if d, ok := jso["dataType"]; !ok || d != "base" {
		return false
	}
	if v, ok := jso["version"]; !ok || v != "0x3" {
		return false
	}
	return true
}

func parseBaseV3JSON(bs []byte, raw bool) (transaction.Transaction, error) {
	tx := new(baseV3)
	if err := json.Unmarshal(bs, &tx.baseV3Data); err != nil {
		return nil, transaction.InvalidFormat.Wrap(err, "InvalidJSON")
	}
	if tx.baseV3Data.From != nil {
		return nil, transaction.InvalidFormat.New("InvalidFromValue(NonNil)")
	}
	return tx, nil
}

type baseV3Header struct {
	Version common.HexUint16 `json:"version"`
	From    *common.Address  `json:"from"` // it should be nil
}

func checkBaseV3Bytes(bs []byte) bool {
	var vh baseV3Header
	if _, err := codec.BC.UnmarshalFromBytes(bs, &vh); err != nil {
		return false
	}
	return vh.From == nil
}

func parseBaseV3Bytes(bs []byte) (transaction.Transaction, error) {
	tx := new(baseV3)
	if _, err := codec.BC.UnmarshalFromBytes(bs, &tx.baseV3Data); err != nil {
		return nil, err
	}
	return tx, nil
}

func RegisterBaseTx() {
	transaction.RegisterFactory(&transaction.Factory{
		Priority:    15,
		CheckJSON:   checkBaseV3JSON,
		ParseJSON:   parseBaseV3JSON,
		CheckBinary: checkBaseV3Bytes,
		ParseBinary: parseBaseV3Bytes,
	})
}
