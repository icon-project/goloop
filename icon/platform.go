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

package icon

import (
	"encoding/json"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

type platform struct {
	calculator *iiss.Calculator
}

func (p *platform) NewContractManager(dbase db.Database, dir string, logger log.Logger) (contract.ContractManager, error) {
	// TODO find right position
	if err := p.calculator.Init(dbase); err != nil {
		return nil, err
	}
	return newContractManager(p, dbase, dir, logger)
}

func (p *platform) NewExtensionSnapshot(dbase db.Database, raw []byte) state.ExtensionSnapshot {
	// TODO return valid ExtensionSnapshot(not nil) which can return valid ExtensionState.
	//  with that state, we may change state of extension.
	//  For initial state, the snapshot returns nil for Bytes() method.
	return iiss.NewExtensionSnapshot(dbase, raw)
}

func (p *platform) NewExtensionWithBuilder(builder merkle.Builder, raw []byte) state.ExtensionSnapshot {
	// TODO return ExtensionSnapshot instance after requesting required data to
	//  the builder.
	return nil
}

func (p *platform) ToRevision(value int) module.Revision {
	return valueToRevision(value)
}

func (p *platform) NewBaseTransaction(wc state.WorldContext) (module.Transaction, error) {
	// TODO calculate issued i-score and amount balance. No changes on world context.
	es := wc.GetExtensionState().(*iiss.ExtensionStateImpl)
	if !es.State.GetTerm().IsDecentralized() {
		return nil, nil
	}

	t := common.HexInt64{Value: wc.BlockTimeStamp()}
	v := common.HexUint16{Value: module.TransactionVersion3}
	prep, issue := iiss.GetIssueData(es)
	data := make(map[string]interface{})
	if prep != nil {
		data["prep"] = prep
	}
	if issue != nil {
		data["result"] = issue
	}
	mtx := map[string]interface{}{
		"timestamp": t,
		"version":   v,
		"dataType":  "base",
		"data":      data,
	}
	bs, err := json.Marshal(mtx)
	if err != nil {
		return nil, err
	}
	tx, err := transaction.NewTransactionFromJSON(bs)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (p *platform) OnExtensionSnapshotFinalization(ess state.ExtensionSnapshot) {
	// TODO start background calculator if it's not started.
	go p.calculator.Run(ess.(*iiss.ExtensionSnapshotImpl))
}

func (p *platform) OnExecutionEnd(wc state.WorldContext, er service.ExecutionResult) error {
	if wc.Revision().Value() < RevisionIISS {
		return nil
	}
	ext := wc.GetExtensionState()
	es := ext.(*iiss.ExtensionStateImpl)

	if err := es.UpdateIssueInfoFee(er.TotalFee()); err != nil {
		return err
	}
	return es.OnExecutionEnd(wc, p.calculator)
}

func (p *platform) Term() {
	// Terminate
}

func NewPlatform(base string, cid int) (service.Platform, error) {
	return &platform{
		calculator: iiss.NewCalculator(),
	}, nil
}

func init() {
	iiss.RegisterBaseTx()
}
