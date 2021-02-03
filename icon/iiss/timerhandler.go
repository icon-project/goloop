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
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/service/state"
	"math/big"
)

func HandleTimerJob(wc state.WorldContext) (err error) {
	es := wc.GetExtensionState().(*ExtensionStateImpl)
	if bt, err := es.GetUnbondingTimerState(wc.BlockHeight(), false); err != nil {
		return err
	} else if bt != nil {
		err = handleUnbondingTimer(es, bt.Addresses, bt.Height)
	}
	if st, err := es.GetUnstakingTimerState(wc.BlockHeight(), false); err != nil {
		return err
	} else if st != nil {
		err = handleUnstakingTimer(wc, es, st.Addresses, st.Height)
	}
	return
}

func handleUnstakingTimer(wc state.WorldContext, es *ExtensionStateImpl, al []*common.Address, h int64) error {
	for _, a := range al {
		ea, err := es.GetAccount(a)
		if err != nil {
			return err
		}

		ra, err := ea.RemoveUnstaking(h)
		if err != nil {
			return err
		}

		wa := wc.GetAccountState(ea.Address().ID())
		b := wa.GetBalance()
		wa.SetBalance(new(big.Int).Add(b, ra))
	}
	return nil
}

func handleUnbondingTimer(es *ExtensionStateImpl, al []*common.Address, h int64) error {
	for _, a := range al {
		as, err := es.GetAccount(a)
		if err != nil {
			return err
		}
		if err = as.RemoveUnbonding(h); err != nil {
			return err
		}
	}
	return nil
}
