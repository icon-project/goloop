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

package test

import (
	"encoding/json"
	"math/big"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

const VarTest = "test"

type transactionJSON struct {
	TimeStamp        common.HexInt64   `json:"timestamp"`
	Type             string            `json:"type"`
	Validators       []*common.Address `json:"validators,omitempty"`
	NextBlockVersion *common.HexInt32  `json:"nextBlockVersion,omitempty"`
	VarTest          *string           `json:"varTest,omitempty"`
}

type Transaction struct {
	json transactionJSON

	id []byte
}

func (t *Transaction) Prepare(ctx contract.Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{state.WorldIDStr, state.AccountWriteLock},
	}
	return ctx.GetFuture(lq), nil
}

func (t *Transaction) Execute(ctx contract.Context, estimate bool) (txresult.Receipt, error) {
	if t.json.Validators != nil {
		var vl []module.Validator
		for _, addr := range t.json.Validators {
			v, err := state.ValidatorFromAddress(addr)
			if err != nil {
				return nil, err
			}
			vl = append(vl, v)
		}
		vs, err := state.ValidatorSnapshotFromSlice(ctx.Database(), vl)
		if err != nil {
			return nil, err
		}
		ctx.GetValidatorState().Reset(vs)
	}
	if t.json.NextBlockVersion != nil {
		as := ctx.GetAccountState(state.SystemID)
		prop := scoredb.NewVarDB(as, state.VarNextBlockVersion)
		prop.Set(t.json.NextBlockVersion.Value)
	}
	if t.json.VarTest != nil {
		as := ctx.GetAccountState(state.SystemID)
		prop := scoredb.NewVarDB(as, VarTest)
		prop.Set(*t.json.VarTest)
	}
	r := txresult.NewReceipt(ctx.Database(), ctx.Revision(), t.To())
	r.SetResult(module.StatusSuccess, big.NewInt(0), big.NewInt(0), nil)
	return r, nil
}

func (t *Transaction) Dispose() {
}

func (t *Transaction) Group() module.TransactionGroup {
	return module.TransactionGroupNormal
}

func (t *Transaction) ID() []byte {
	if t.id == nil {
		t.id = crypto.SHA3Sum256(t.Bytes())
	}
	return t.id
}

func (t *Transaction) From() module.Address {
	return state.SystemAddress
}

func (t *Transaction) Bytes() []byte {
	jsn, _ := json.Marshal(t.json)
	return jsn
}

func (t *Transaction) Hash() []byte {
	return t.ID()
}

func (t *Transaction) Verify() error {
	return nil
}

func (t *Transaction) Version() int {
	return module.TransactionVersion3
}

func (t *Transaction) ToJSON(version module.JSONVersion) (interface{}, error) {
	res := map[string]interface{}{
		"timestamp": &t.json.TimeStamp,
		"type": "test",
	}
	if t.json.Validators != nil {
		res["validators"] = t.json.Validators
	}
	if t.json.NextBlockVersion != nil {
		res["nextBlockVersion"] = t.json.NextBlockVersion
	}
	if t.json.VarTest != nil {
		res["varTest"] = t.json.VarTest
	}
	return res, nil
}

func (t *Transaction) ValidateNetwork(nid int) bool {
	return true
}

func (t *Transaction) PreValidate(wc state.WorldContext, update bool) error {
	return nil
}

func (t *Transaction) GetHandler(cm contract.ContractManager) (transaction.Handler, error) {
	return t, nil
}

func (t *Transaction) Timestamp() int64 {
	return t.json.TimeStamp.Value
}

func (t *Transaction) Nonce() *big.Int {
	return nil
}

func (t *Transaction) To() module.Address {
	return state.SystemAddress
}

func (t *Transaction) IsSkippable() bool {
	return false
}

func checkJSONTX(tx map[string]interface{}) bool {
	val, ok := tx["type"]
	return ok && val=="test"
}

func parseJSONTX(js []byte, raw bool) (transaction.Transaction, error) {
	t := &Transaction{}
	if err := json.Unmarshal(js, &t.json); err != nil {
		return nil, err
	}
	return t, nil
}

var once sync.Once

func RegisterTransactionFactory() {
	once.Do(func() {
		transaction.RegisterFactory(&transaction.Factory{
			Priority: 5,
			CheckJSON: checkJSONTX,
			ParseJSON: parseJSONTX,
		})
	})
}
