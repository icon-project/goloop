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
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/iccache"
	"github.com/icon-project/goloop/icon/iiss/icutils"
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
	return newContractManager(p, dbase, dir, logger)
}

func (p *platform) NewExtensionSnapshot(dbase db.Database, raw []byte) state.ExtensionSnapshot {
	// TODO return valid ExtensionSnapshot(not nil) which can return valid ExtensionState.
	//  with that state, we may change state of extension.
	//  For initial state, the snapshot returns nil for Bytes() method.
	if len(raw) == 0 {
		return nil
	}
	dbase = iccache.AttachStateNodeCache(dbase)
	return iiss.NewExtensionSnapshot(dbase, raw)
}

func (p *platform) NewExtensionWithBuilder(builder merkle.Builder, raw []byte) state.ExtensionSnapshot {
	return iiss.NewExtensionSnapshotWithBuilder(builder, raw)
}

func (p *platform) ToRevision(value int) module.Revision {
	return icmodule.ValueToRevision(value)
}

func (p *platform) NewBaseTransaction(wc state.WorldContext) (module.Transaction, error) {
	// calculate issued i-score and amount balance. No changes on world context.
	es := p.getExtensionState(wc, nil)
	if es == nil || !es.IsDecentralized() {
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

func (p *platform) OnExtensionSnapshotFinalization(ess state.ExtensionSnapshot, logger log.Logger) {
	// Start background calculator if it's not started.
	if ess != nil && p.calculator.CheckToRun(ess) {
		go func(snapshot state.ExtensionSnapshot) {
			err := p.calculator.Run(snapshot, logger)
			if err != nil {
				logger.Errorf("Failed to calculate reward. %+v", err)
			}
		}(ess)
	}
}

func (p *platform) OnExecutionBegin(wc state.WorldContext, logger log.Logger) error {
	revision := wc.Revision().Value()
	if revision < icmodule.RevisionIISS {
		return nil
	}
	es := p.getExtensionState(wc, logger)
	if es == nil {
		return nil
	}
	return es.OnExecutionBegin(wc)

}

func (p *platform) OnExecutionEnd(wc state.WorldContext, er service.ExecutionResult, logger log.Logger) error {
	revision := wc.Revision().Value()
	if revision < icmodule.RevisionIISS {
		return nil
	}
	es := p.getExtensionState(wc, logger)
	if es == nil {
		return nil
	}

	term := es.State.GetTerm()
	if term.IsDecentralized() || wc.BlockHeight() == 10362082 {
		if err := es.UpdateIssueInfoFee(er.TotalFee()); err != nil {
			return err
		}
	}
	if err := es.HandleTimerJob(wc); err != nil {
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

func (p *platform) getExtensionState(wc state.WorldContext, logger log.Logger) *iiss.ExtensionStateImpl {
	es := wc.GetExtensionState()
	if es == nil {
		return nil
	}
	esi, ok := es.(*iiss.ExtensionStateImpl)
	if !ok {
		return nil
	}
	if logger != nil {
		esi.SetLogger(icutils.NewIconLogger(logger))
	}
	return esi
}
