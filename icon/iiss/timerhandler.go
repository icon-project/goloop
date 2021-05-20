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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/service/state"
)

func (s *ExtensionStateImpl) HandleTimerJob(wc state.WorldContext) (err error) {
	bh := wc.BlockHeight()
	s.logger.Tracef("HandleTimerJob() start BH-%d", bh)
	bt := s.GetUnbondingTimerState(bh, false)
	if bt != nil {
		err = s.handleUnbondingTimer(bt.Addresses, bh)
		if err != nil {
			return
		}
	}

	st := s.GetUnstakingTimerState(wc.BlockHeight(), false)
	if st != nil {
		err = s.handleUnstakingTimer(wc, st.Addresses, bh)
	}
	s.logger.Tracef("HandleTimerJob() end BH-%d", bh)
	return
}

func (s *ExtensionStateImpl) handleUnstakingTimer(wc state.WorldContext, al []*common.Address, h int64) error {
	s.logger.Tracef("handleUnstakingTimer() start: bh=%d", h)
	for _, a := range al {
		ea := s.State.GetAccountState(a)
		s.logger.Tracef("account : %s", ea)
		ra, err := ea.RemoveUnstake(h)
		if err != nil {
			return err
		}

		wa := wc.GetAccountState(a.ID())
		b := wa.GetBalance()
		wa.SetBalance(new(big.Int).Add(b, ra))
		blockHeight := wc.BlockHeight()
		s.logger.Tracef(
			"after remove unstake, stake information of %s : %s",
			a, ea.GetStakeInJSON(blockHeight),
		)
	}
	s.logger.Tracef("handleUnstakingTimer() end")
	return nil
}

func (s *ExtensionStateImpl) handleUnbondingTimer(al []*common.Address, h int64) error {
	s.logger.Tracef("handleUnbondingTimer() start: bh=%d", h)
	for _, a := range al {
		s.logger.Tracef("account : %s", a)
		as := s.State.GetAccountState(a)
		if err := as.RemoveUnbond(h); err != nil {
			return err
		}
		s.logger.Tracef("after remove unbonds, unbond information of %s : %s", a, as.GetUnbondsInJSON())
	}
	s.logger.Tracef("handleUnbondingTimer() end")
	return nil
}
