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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate/migrate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/trace"
)

type governanceHandler struct {
	ch    contract.ContractHandler
	ctype int
	call  *contract.DataCallJSON
	fid   int
	log   *trace.Logger
}

func (g *governanceHandler) Prepare(ctx contract.Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{state.WorldIDStr, state.AccountWriteLock},
	}
	return ctx.GetFuture(lq), nil
}

func (g *governanceHandler) Init(fid int, logger log.Logger) {
	g.fid = fid
	g.log = trace.LoggerOf(logger)
}

func applyGovernanceVariablesToSystem(cc contract.CallContext, govAs, sysAs containerdb.BytesStoreState) error {
	price := scoredb.NewVarDB(govAs, state.VarStepPrice).Int64()
	if price == 0 {
		// INV migration happened
		return nil
	}
	_ = scoredb.NewVarDB(sysAs, state.VarStepPrice).Set(price)
	// stepCosts
	stepTypes := scoredb.NewArrayDB(sysAs, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(sysAs, state.VarStepCosts, 1)
	stepCostGov := scoredb.NewDictDB(govAs, state.VarStepCosts, 1)
	tcount := stepTypes.Size()
	for i := 0; i < tcount; i++ {
		tname := stepTypes.Get(i).String()
		if cost := stepCostGov.Get(tname); cost != nil {
			_ = stepCostDB.Set(tname, cost.Int64())
		}
	}
	// maxStepLimits
	stepLimitTypes := scoredb.NewArrayDB(sysAs, state.VarStepLimitTypes)
	stepLimitDB := scoredb.NewDictDB(sysAs, state.VarStepLimit, 1)
	stepLimitGov := scoredb.NewDictDB(govAs, "max_step_limits", 1)
	tcount = stepLimitTypes.Size()
	for i := 0; i < tcount; i++ {
		tname := stepLimitTypes.Get(i).String()
		if value := stepLimitGov.Get(tname); value != nil {
			_ = stepLimitDB.Set(tname, value.Int64())
		}
	}

	if revision := scoredb.NewVarDB(govAs, "revision_code"); revision != nil {
		sysRev := scoredb.NewVarDB(sysAs, state.VarRevision)
		if sysRev.Int64() < revision.Int64() {
			chainSCORE, _ := newChainScore(cc, govAddress, new(big.Int))
			if err := chainSCORE.(*chainScore).Ex_setRevision(common.NewHexInt(revision.Int64())); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *governanceHandler) handleRevisionChange(cc contract.CallContext, r1, r2 int) {
	if r1 >= r2 {
		return
	}
	if r1 < icmodule.RevisionFixInvalidUnstake && r2 >= icmodule.RevisionFixInvalidUnstake {
		migrate.WriteInvalidUnstakeFixedEventLogs(cc)
	}
}

func (g *governanceHandler) ExecuteSync(cc contract.CallContext) (error, *codec.TypedObj, module.Address) {
	g.log.TSystemf("FRAME[%d] GOV start", g.fid)
	defer g.log.TSystemf("FRAME[%d] GOV end", g.fid)

	rev := cc.Revision().Value()

	gss := cc.GetAccountSnapshot(govAddress.ID())
	status, steps, result, score := cc.Call(g.ch, cc.StepAvailable())
	cc.DeductSteps(steps)
	if status == nil && rev <= icmodule.Revision8 {
		if gss2 := cc.GetAccountSnapshot(govAddress.ID()); gss2.StorageChangedAfter(gss) {
			sysAs := cc.GetAccountState(state.SystemID)
			govAs := scoredb.NewStateStoreWith(gss2)
			if err := applyGovernanceVariablesToSystem(cc, govAs, sysAs); err != nil {
				return err, nil, nil
			}
		}
	}
	if status == nil {
		sysAs := cc.GetAccountState(state.SystemID)
		rev2 := int(scoredb.NewVarDB(sysAs, state.VarRevision).Int64())
		if rev != rev2 {
			g.log.TSystemf("FRAME[%d] GOV handleRevisionChange rev1=%d rev2=%d", g.fid, rev, rev2)
			g.handleRevisionChange(cc, rev, rev2)
		}
	}
	return status, result, score
}

func newGovernanceHandler(ch contract.ContractHandler) *governanceHandler {
	return &governanceHandler{ch: ch}
}
