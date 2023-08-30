/*
 * Copyright 2023 ICON Foundation
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

	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

func recordSlashingRateChangedV2Event(cc icmodule.CallContext, penaltyType icmodule.PenaltyType, rate icmodule.Rate) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("SlashingRateChangedV2(int,int)")},
		[][]byte{
			intconv.Int64ToBytes(int64(penaltyType)),
			intconv.Int64ToBytes(rate.NumInt64()),
		},
	)
}

func recordCommissionRateInitializedEvent(
	cc icmodule.CallContext, owner module.Address, rate, maxRate, maxChangeRate icmodule.Rate) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("CommissionRateInitialized(Address,int,int,int)"), owner.Bytes()},
		[][]byte{
			intconv.Int64ToBytes(rate.NumInt64()),
			intconv.Int64ToBytes(maxRate.NumInt64()),
			intconv.Int64ToBytes(maxChangeRate.NumInt64()),
		},
	)
}

func recordCommissionRateChangedEvent(
	cc icmodule.CallContext, owner module.Address, rate icmodule.Rate) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("CommissionRateChanged(Address,int)"), owner.Bytes()},
		[][]byte{
			intconv.Int64ToBytes(rate.NumInt64()),
		},
	)
}

func recordPenaltyImposedEvent(cc icmodule.CallContext, ps *icstate.PRepStatusState, pt icmodule.PenaltyType) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PenaltyImposed(Address,int,int)"), ps.Owner().Bytes()},
		[][]byte{
			intconv.Int64ToBytes(int64(ps.Status())),
			intconv.Int64ToBytes(int64(pt)),
		},
	)
}

func recordSlashedEvent(cc icmodule.CallContext, owner, bonder module.Address, amount *big.Int) {
	cc.OnEvent(
		state.SystemAddress,
		[][]byte{[]byte("Slashed(Address,Address,int)"), owner.Bytes()},
		[][]byte{bonder.Bytes(), intconv.BigIntToBytes(amount)},
	)
}

func RecordIScoreClaimEvent(cc icmodule.CallContext, address module.Address, claim, icx *big.Int) {
	revision := cc.Revision().Value()
	if revision < icmodule.Revision9 {
		cc.OnEvent(state.SystemAddress,
			[][]byte{
				[]byte("IScoreClaimed(int,int)"),
			},
			[][]byte{
				intconv.BigIntToBytes(claim),
				intconv.BigIntToBytes(icx),
			},
		)
	} else {
		cc.OnEvent(state.SystemAddress,
			[][]byte{
				[]byte("IScoreClaimedV2(Address,int,int)"),
				address.Bytes(),
			},
			[][]byte{
				intconv.BigIntToBytes(claim),
				intconv.BigIntToBytes(icx),
			},
		)
	}
}

func recordPRepIssuedEvent(cc icmodule.CallContext, prep *IssuePRepJSON) {
	if prep != nil {
		cc.OnEvent(state.SystemAddress,
			[][]byte{[]byte("PRepIssued(int,int,int,int)")},
			[][]byte{
				intconv.BigIntToBytes(prep.GetIRep()),
				intconv.BigIntToBytes(prep.GetRRep()),
				intconv.BigIntToBytes(prep.GetTotalDelegation()),
				intconv.BigIntToBytes(prep.GetValue()),
			},
		)
	}
}

func recordICXIssuedEvent(cc icmodule.CallContext, result *IssueResultJSON, issue *icstate.Issue) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("ICXIssued(int,int,int,int)")},
		[][]byte{
			intconv.BigIntToBytes(result.GetByFee()),
			intconv.BigIntToBytes(result.GetByOverIssuedICX()),
			intconv.BigIntToBytes(result.GetIssue()),
			intconv.BigIntToBytes(issue.GetOverIssuedICX()),
		},
	)
}

func recordTermStartedEvent(cc icmodule.CallContext, term *icstate.TermSnapshot) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("TermStarted(int,int,int)")},
		[][]byte{
			intconv.Int64ToBytes(int64(term.Sequence())),
			intconv.Int64ToBytes(term.StartHeight()),
			intconv.Int64ToBytes(term.GetEndHeight()),
		},
	)
}

func recordPRepRegisteredEvent(cc icmodule.CallContext, from module.Address) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepRegistered(Address)")},
		[][]byte{from.Bytes()},
	)
}

func recordPRepSetEvent(cc icmodule.CallContext, from module.Address) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepSet(Address)")},
		[][]byte{from.Bytes()},
	)
}

func recordRewardFundTransferredEvent(cc icmodule.CallContext, key string, from, to module.Address, amount *big.Int) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("RewardFundTransferred(str,Address,Address,int)")},
		[][]byte{
			[]byte(key),
			from.Bytes(),
			to.Bytes(),
			intconv.BigIntToBytes(amount),
		},
	)
}

func recordRewardFundBurnedEvent(cc icmodule.CallContext, key string, from module.Address, amount *big.Int) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("RewardFundBurned(str,Address,int)")},
		[][]byte{
			[]byte(key),
			from.Bytes(),
			intconv.BigIntToBytes(amount),
		},
	)
}

func recordPRepUnregisteredEvent(cc icmodule.CallContext, owner module.Address) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepUnregistered(Address)")},
		[][]byte{owner.Bytes()},
	)
}