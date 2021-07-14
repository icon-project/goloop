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
package iiss

import (
	"math/big"

	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/service/state"
)

func (es *ExtensionStateImpl) HandleTimerJob(wc state.WorldContext) (err error) {
	bh := wc.BlockHeight()
	es.logger.Tracef("HandleTimerJob() start BH-%d", bh)
	bt := es.State.GetUnbondingTimerSnapshot(bh)
	if bt != nil {
		err = es.handleUnbondingTimer(bt, bh)
		if err != nil {
			return
		}
	}

	st := es.State.GetUnstakingTimerSnapshot(wc.BlockHeight())
	if st != nil {
		err = es.handleUnstakingTimer(wc, st, bh)
	}
	es.logger.Tracef("HandleTimerJob() end BH-%d", bh)
	return
}

func (es *ExtensionStateImpl) handleUnstakingTimer(wc state.WorldContext, ts *icstate.TimerSnapshot, h int64) error {
	es.logger.Tracef("handleUnstakingTimer() start: bh=%d", h)
	for itr := ts.Iterator() ; itr.Has() ; itr.Next() {
		a, _ := itr.Get()
		ea := es.State.GetAccountState(a)
		es.logger.Tracef("account : %s", ea)
		ra, err := ea.RemoveUnstake(h)
		if err != nil {
			return err
		}

		wa := wc.GetAccountState(a.ID())
		b := wa.GetBalance()
		wa.SetBalance(new(big.Int).Add(b, ra))
		blockHeight := wc.BlockHeight()
		es.logger.Tracef(
			"after remove unstake, stake information of %s : %s",
			a, ea.GetStakeInJSON(blockHeight),
		)
	}
	es.logger.Tracef("handleUnstakingTimer() end")
	return nil
}

func (es *ExtensionStateImpl) handleUnbondingTimer(ts *icstate.TimerSnapshot, h int64) error {
	es.logger.Tracef("handleUnbondingTimer() start: bh=%d", h)
	for itr := ts.Iterator() ; itr.Has() ; itr.Next() {
		a, _ := itr.Get()
		es.logger.Tracef("account : %s", a)
		as := es.State.GetAccountState(a)
		if err := as.RemoveUnbond(h); err != nil {
			return err
		}
		es.logger.Tracef("after remove unbonds, unbond information of %s : %s", a, as.GetUnbondsInJSON())
	}
	es.logger.Tracef("handleUnbondingTimer() end")
	return nil
}
