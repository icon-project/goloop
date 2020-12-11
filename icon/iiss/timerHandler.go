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
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"math/big"
)

func HandleTimerJob(wc state.WorldContext) (err error) {
	es := wc.GetExtensionState().(*ExtensionStateImpl)
	if bt, err := es.GetUnbondingTimerState(wc.BlockHeight()); err != nil {
		return err
	} else {
		err = handleUnBoningTimer(es, bt.Addresses, bt.Height)
	}
	if st, err := es.GetUnstakingTimerState(wc.BlockHeight()); err != nil {
		return err
	} else {
		err = handleUnStakingTimer(wc, es, st.Addresses, st.Height)
	}
	return
}

func handleUnStakingTimer(wc state.WorldContext, es *ExtensionStateImpl, al []module.Address, h int64) error {
	for _, a := range al {
		ea, err := es.GetAccountState(a)
		if err != nil {
			return err
		}

		ra, err := ea.RemoveUnStaking(h)
		if err != nil {
			return err
		}

		wa := wc.GetAccountState(ea.Address().ID())
		b := wa.GetBalance()
		wa.SetBalance(new(big.Int).Add(b, ra))
	}
	return nil
}

func handleUnBoningTimer(es *ExtensionStateImpl, al []module.Address, h int64) error {
	for _, a := range al {
		as, err := es.GetAccountState(a)
		if err != nil {
			return err
		}
		if err = as.RemoveUnBonding(h); err != nil {
			return err
		}
	}
	return nil
}
