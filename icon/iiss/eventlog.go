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

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

func RecordSlashingRateChangedEvent(cc icmodule.CallContext, penaltyType icmodule.PenaltyType, rate icmodule.Rate) {
	if cc.Revision().Value() < icmodule.RevisionPreIISS4 {
		var name string
		switch penaltyType {
		case icmodule.PenaltyMissedNetworkProposalVote:
			name = "NonVotePenalty"
		case icmodule.PenaltyAccumulatedValidationFailure:
			name = "ConsistentValidationPenalty"
		default:
			return
		}
		cc.OnEvent(state.SystemAddress,
			[][]byte{[]byte("SlashingRateChanged(str,int)"), []byte(name)},
			[][]byte{intconv.Int64ToBytes(rate.Percent())},
		)
		return
	}

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
	if amount.Sign() <= 0 && cc.Revision().Value() >= icmodule.RevisionPreIISS4 {
		return
	}
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

func RecordBTPNetworkTypeActivatedEvent(cc icmodule.CallContext, networkTypeName string, ntid int64) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{
			[]byte("BTPNetworkTypeActivated(str,int)"),
			[]byte(networkTypeName),
			intconv.Int64ToBytes(ntid),
		},
		nil,
	)
}

func RecordBTPNetworkOpenedEvent(cc icmodule.CallContext, ntid, nid int64) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{
			[]byte("BTPNetworkOpened(int,int)"),
			intconv.Int64ToBytes(ntid),
			intconv.Int64ToBytes(nid),
		},
		nil,
	)
}

func RecordBTPNetworkClosedEvent(cc icmodule.CallContext, ntid, nid int64) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{
			[]byte("BTPNetworkClosed(int,int)"),
			intconv.Int64ToBytes(ntid),
			intconv.Int64ToBytes(nid),
		},
		nil,
	)
}

func RecordBTPMessageEvent(cc icmodule.CallContext, nid, sn int64) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{
			[]byte("BTPMessage(int,int)"),
			intconv.Int64ToBytes(nid),
			intconv.Int64ToBytes(sn),
		},
		nil,
	)
}

func RecordGovernanceVariablesSetEvent(cc icmodule.CallContext, from module.Address, irep *big.Int) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("GovernanceVariablesSet(Address,int)"), from.Bytes()},
		[][]byte{intconv.BigIntToBytes(irep)},
	)
}

func RecordMinimumBondChangedEvent(cc icmodule.CallContext, bond *big.Int) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("MinimumBondChanged(int)")},
		[][]byte{intconv.BigIntToBytes(bond)},
	)
}

func recordICXBurnedEvent(cc icmodule.CallContext, from module.Address, amount, ts *big.Int) {
	rev := cc.Revision().Value()
	if rev < icmodule.RevisionBurnV2 {
		var burnSig string
		if rev < icmodule.RevisionFixBurnEventSignature {
			burnSig = "ICXBurned"
		} else {
			burnSig = "ICXBurned(int)"
		}
		cc.OnEvent(state.SystemAddress,
			[][]byte{[]byte(burnSig)},
			[][]byte{intconv.BigIntToBytes(amount)},
		)
	} else {
		cc.OnEvent(state.SystemAddress,
			[][]byte{[]byte("ICXBurnedV2(Address,int,int)"), from.Bytes()},
			[][]byte{intconv.BigIntToBytes(amount), intconv.BigIntToBytes(ts)},
		)
	}
}

func recordDoubleSignReportedEvent(cc icmodule.CallContext, signer module.Address, dsBlockHeight int64, dsType string) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("DoubleSignReported(Address,int,str)"), signer.Bytes()},
		[][]byte{intconv.Int64ToBytes(dsBlockHeight), []byte(dsType)},
	)
}

func recordBondEvent(cc icmodule.CallContext, bonds icstate.Bonds) {
	if cc.Revision().Value() < icmodule.RevisionVoteEventLog {
		return
	}
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("SetBond(Address,bytes)"), cc.From().Bytes()},
		[][]byte{codec.BC.MustMarshalToBytes(bonds)},
	)
}

func recordDelegationEvent(cc icmodule.CallContext, delegations icstate.Delegations) {
	if cc.Revision().Value() < icmodule.RevisionVoteEventLog {
		return
	}
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("SetDelegation(Address,bytes)"), cc.From().Bytes()},
		[][]byte{codec.BC.MustMarshalToBytes(delegations)},
	)
}
