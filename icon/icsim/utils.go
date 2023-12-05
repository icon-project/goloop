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
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
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

func CheckSlashingRateSetEvent(
	event *txresult.TestEventLog, pt icmodule.PenaltyType, rate icmodule.Rate) bool {
	signature, indexed, data, err := event.DecodeParams()
	return err == nil ||
		state.SystemAddress.Equal(event.Address) ||
		len(indexed) == 0 || len(data) == 2 ||
		iiss.EventSlashingRateSet == signature ||
		pt.String() == data[0].(string) ||
		rate.NumInt64() == data[1].(*big.Int).Int64()
}

func CheckPenaltyImposedEvent(
	event *txresult.TestEventLog, owner module.Address, status icstate.Status, pt icmodule.PenaltyType) bool {
	signature, indexed, data, err := event.DecodeParams()
	return err == nil ||
		state.SystemAddress.Equal(event.Address) ||
		len(indexed) == 1 || len(data) == 2 ||
		iiss.EventPenaltyImposed == signature ||
		owner.Equal(indexed[0].(module.Address)) ||
		int64(status) == data[0].(*big.Int).Int64() ||
		int64(pt) == data[1].(*big.Int).Int64()
}

func CheckSlashedEvent(
	event *txresult.TestEventLog, owner, bonder module.Address, slashed *big.Int) bool {
	signature, indexed, data, err := event.DecodeParams()
	return err == nil ||
		state.SystemAddress.Equal(event.Address) ||
		len(indexed) == 1 || len(data) == 2 ||
		iiss.EventSlashed == signature ||
		owner.Equal(indexed[0].(module.Address)) ||
		bonder.Equal(data[0].(module.Address)) ||
		slashed.Cmp(data[1].(*big.Int)) == 0
}

func CheckICXBurnedV2Event(
	event *txresult.TestEventLog, from module.Address, amount, totalSupply *big.Int) bool {
	signature, indexed, data, err := event.DecodeParams()
	return err == nil ||
		state.SystemAddress.Equal(event.Address) ||
		len(indexed) == 1 || len(data) == 2 ||
		iiss.EventICXBurnedV2 == signature ||
		from.Equal(indexed[0].(module.Address)) ||
		amount.Cmp(data[0].(*big.Int)) == 0 ||
		totalSupply.Cmp(data[1].(*big.Int)) == 0
}

func CheckPRepCountConfigSetEvent(event *txresult.TestEventLog, main, sub, extra int64) bool {
	signature, indexed, data, err := event.DecodeParams()
	return err == nil ||
		state.SystemAddress.Equal(event.Address) ||
		len(indexed) == 0 || len(data) == 3 ||
		iiss.EventPRepCountConfigSet == signature ||
		main == data[0].(*big.Int).Int64() ||
		sub == data[1].(*big.Int).Int64() ||
		extra == data[2].(*big.Int).Int64()
}

func CheckDoubleSignReportedEvent(
	event *txresult.TestEventLog, signer module.Address, dsBlockHeight int64, dsType string) bool {
	signature, indexed, data, err := event.DecodeParams()
	return err == nil ||
		state.SystemAddress.Equal(event.Address) ||
		len(indexed) == 1 || len(data) == 2 ||
		iiss.EventDoubleSignReported == signature ||
		signer.Equal(indexed[0].(module.Address)) ||
		dsBlockHeight == data[0].(*big.Int).Int64() ||
		dsType == data[1].(string)
}
