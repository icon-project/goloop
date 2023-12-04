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

package icsim

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

func validateAmount(amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return errors.Errorf("Invalid amount: %v", amount)
	}
	return nil
}

func setBalance(address module.Address, as state.AccountState, balance *big.Int) error {
	if balance.Sign() < 0 {
		return errors.Errorf(
			"Invalid balance: address=%v balance=%v",
			address, balance,
		)
	}
	as.SetBalance(balance)
	return nil
}

func checkReceipts(receipts []Receipt) bool {
	for _, r := range receipts {
		if r.Status() != Success {
			return false
		}
	}
	return true
}

func CheckReceiptSuccess(receipts ...Receipt) bool {
	for _, rcpt := range receipts {
		if rcpt.Status() != 1 {
			return false
		}
	}
	return true
}

func NewConsensusInfo(dbase db.Database, vl []module.Validator, voted []bool) module.ConsensusInfo {
	vss, err := state.ValidatorSnapshotFromSlice(dbase, vl)
	if err != nil {
		return nil
	}
	if len(vl) != len(voted) {
		return nil
	}

	v, _ := vss.Get(vss.Len() - 1)
	copiedVoted := make([]bool, vss.Len())
	copy(copiedVoted, voted)
	return common.NewConsensusInfo(v.Address(), vss, copiedVoted)
}

func NewConsensusInfoBySim(sim Simulator, nilVoteIndices ...int) module.ConsensusInfo {
	vl := sim.ValidatorList()
	voted := make([]bool, len(vl))
	for i := 0; i < len(voted); i++ {
		voted[i] = true
	}
	for _, i := range nilVoteIndices {
		voted[i] = false
	}
	return NewConsensusInfo(sim.Database(), vl, voted)
}

func ValidatorIndexOf(vl []module.Validator, address module.Address) int {
	for i, v := range vl {
		if v.Address().Equal(address) {
			return i
		}
	}
	return -1
}

func CheckElectablePRep(prep *icstate.PRep, expGrade icstate.Grade) bool {
	return prep.Grade() == expGrade &&
		prep.IsActive() &&
		prep.IsJailInfoElectable() &&
		prep.IsInJail() == false &&
		prep.IsAlreadyPenalized() == false &&
		prep.IsUnjailing() == false &&
		prep.IsUnjailable() == false
}

func CheckPenalizedPRep(prep *icstate.PRep) bool {
	return prep.Grade() == icstate.GradeCandidate &&
		prep.IsActive() &&
		prep.IsInJail() &&
		prep.IsAlreadyPenalized() &&
		prep.IsUnjailable() &&
		prep.IsJailInfoElectable() == false &&
		prep.IsUnjailing() == false
}

func CheckUnjailingPRep(prep *icstate.PRep) bool {
	return prep.Grade() == icstate.GradeCandidate &&
		prep.IsActive() &&
		prep.IsJailInfoElectable() &&
		prep.IsInJail() &&
		prep.IsAlreadyPenalized() &&
		prep.IsUnjailing() &&
		prep.IsUnjailable() == false
}
