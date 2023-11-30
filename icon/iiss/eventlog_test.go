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
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

func getEventLog(cc *mockCallContext) *txresult.TestEventLog {
	calls := cc.GetCalls("OnEvent")
	if calls == nil {
		return nil
	}
	call := calls[0]
	params := call.Params()
	return &txresult.TestEventLog{
		Address: params[0].(module.Address),
		Indexed: params[1].([][]byte),
		Data:    params[2].([][]byte),
	}
}

func TestEmitSlashingRateChanged(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})

	type input struct {
		rev  int
		pt   icmodule.PenaltyType
		rate icmodule.Rate
	}
	type output struct {
		penaltyName string
	}
	args := []struct {
		in  input
		out output
	}{
		{
			in: input{
				rev:  icmodule.RevisionIISS4R0 - 1,
				pt:   icmodule.PenaltyAccumulatedValidationFailure,
				rate: icmodule.ToRate(10),
			},
			out: output{
				penaltyName: "ConsistentValidationPenalty",
			},
		},
		{
			in: input{
				rev:  icmodule.RevisionIISS4R0 - 1,
				pt:   icmodule.PenaltyMissedNetworkProposalVote,
				rate: icmodule.ToRate(1),
			},
			out: output{
				penaltyName: "NonVotePenalty",
			},
		},
		{
			in: input{
				rev:  icmodule.RevisionIISS4R0 - 1,
				pt:   icmodule.PenaltyPRepDisqualification,
				rate: icmodule.ToRate(100),
			},
			out: output{
				penaltyName: "",
			},
		},
		{
			in: input{
				rev:  icmodule.RevisionIISS4R0 - 1,
				pt:   icmodule.PenaltyValidationFailure,
				rate: icmodule.ToRate(1),
			},
			out: output{
				penaltyName: "",
			},
		},
		{
			in: input{
				rev:  icmodule.RevisionIISS4R0 - 1,
				pt:   icmodule.PenaltyDoubleSign,
				rate: icmodule.ToRate(10),
			},
			out: output{
				penaltyName: "",
			},
		},
	}

	for i, arg := range args {
		name := fmt.Sprintf("case-%02d", i)
		rev := arg.in.rev
		cc.Clear()
		cc.SetRevision(icmodule.ValueToRevision(rev))

		t.Run(name, func(t *testing.T) {
			EmitSlashingRateSetEvent(cc, arg.in.pt, arg.in.rate)

			e := getEventLog(cc)
			if arg.out.penaltyName == "" {
				assert.Nil(t, e)
				return
			}

			expIndexed := []any{
				arg.out.penaltyName,
			}
			expData := []any{
				arg.in.rate.Percent(),
			}
			assert.NoError(t, e.Assert(state.SystemAddress, EventSlashingRateChanged, expIndexed, expData))
		})
	}
}

func TestEmitSlashingRateSetEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})

	args := []struct {
		rev  int
		pt   icmodule.PenaltyType
		rate icmodule.Rate
	}{
		{
			rev:  icmodule.RevisionIISS4R0,
			pt:   icmodule.PenaltyPRepDisqualification,
			rate: icmodule.ToRate(100),
		},
		{
			rev:  icmodule.RevisionIISS4R0,
			pt:   icmodule.PenaltyAccumulatedValidationFailure,
			rate: icmodule.ToRate(10),
		},
		{
			rev:  icmodule.RevisionIISS4R0,
			pt:   icmodule.PenaltyValidationFailure,
			rate: icmodule.Rate(1),
		},
		{
			rev:  icmodule.RevisionIISS4R0,
			pt:   icmodule.PenaltyMissedNetworkProposalVote,
			rate: icmodule.ToRate(1),
		},
		{
			rev:  icmodule.RevisionIISS4R0,
			pt:   icmodule.PenaltyDoubleSign,
			rate: icmodule.ToRate(20),
		},
		{
			rev:  icmodule.RevisionIISS4R1,
			pt:   icmodule.PenaltyDoubleSign,
			rate: icmodule.ToRate(12),
		},
	}

	for i, arg := range args {
		name := fmt.Sprintf("case-%02d", i)
		cc.Clear()
		cc.SetRevision(icmodule.ValueToRevision(arg.rev))

		t.Run(name, func(t *testing.T) {
			EmitSlashingRateSetEvent(cc, arg.pt, arg.rate)

			e := getEventLog(cc)
			expData := []any{
				arg.pt.String(),
				arg.rate.NumInt64(),
			}
			assert.NoError(t, e.Assert(state.SystemAddress, EventSlashingRateSet, nil, expData))
		})
	}
}

func TestEmitCommissionRateInitializedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        newDummyAddress(1),
	})
	rate := icmodule.ToRate(0)
	maxRate := icmodule.ToRate(100)
	changeRate := icmodule.ToRate(1)

	EmitCommissionRateInitializedEvent(cc, rate, maxRate, changeRate)
	e := getEventLog(cc)
	expIndexed := []any{
		cc.From(),
	}
	expData := []any{
		rate.NumInt64(),
		maxRate.NumInt64(),
		changeRate.NumInt64(),
	}
	assert.NoError(t, e.Assert(state.SystemAddress, EventCommissionRateInitialized, expIndexed, expData))
}

func TestEmitCommissionRateSetEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	rate := icmodule.ToRate(1)

	EmitCommissionRateSetEvent(cc, rate)

	e := getEventLog(cc)
	expIndexed := []any{
		cc.From(),
	}
	expData := []any{
		rate.NumInt64(),
	}
	assert.NoError(t, e.Assert(state.SystemAddress, EventCommissionRateSet, expIndexed, expData))
}

func TestEmitPenaltyImposedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})
	args := []struct {
		status icstate.Status
		pt     icmodule.PenaltyType
	}{
		{
			status: icstate.Disqualified,
			pt:     icmodule.PenaltyPRepDisqualification,
		},
		{
			status: icstate.Active,
			pt:     icmodule.PenaltyAccumulatedValidationFailure,
		},
		{
			status: icstate.Active,
			pt:     icmodule.PenaltyValidationFailure,
		},
		{
			status: icstate.Active,
			pt:     icmodule.PenaltyMissedNetworkProposalVote,
		},
		{
			status: icstate.Active,
			pt:     icmodule.PenaltyDoubleSign,
		},
	}

	for i, arg := range args {
		name := fmt.Sprintf("case-%02d", i)
		owner := newDummyAddress(i + 1)
		ps := icstate.NewPRepStatus(owner)
		ps.SetStatus(arg.status)
		pt := arg.pt

		t.Run(name, func(t *testing.T) {
			EmitPenaltyImposedEvent(cc, ps, pt)

			e := getEventLog(cc)
			expIndexed := []any{ps.Owner()}
			expData := []any{int64(ps.Status()), int(pt)}
			assert.NoError(t, e.Assert(state.SystemAddress, EventPenaltyImposed, expIndexed, expData))
		})

		cc.Clear()
	}
}

func TestEmitSlashedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})
	owner := newDummyAddress(2)
	bonder := newDummyAddress(3)
	amount := icutils.ToLoop(1)

	EmitSlashedEvent(cc, owner, bonder, amount)

	e := getEventLog(cc)
	expIndexed := []any{
		owner,
	}
	expData := []any{
		bonder,
		amount,
	}
	assert.NoError(t, e.Assert(state.SystemAddress, EventSlashed, expIndexed, expData))
}

func TestEmitIScoreClaimEventV2(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	icx := icutils.ToLoop(1)
	claim := icutils.ICXToIScore(icx)

	for _, rev := range []int{icmodule.RevisionIISS2, icmodule.RevisionIISS4R0} {
		revision := icmodule.ValueToRevision(rev)
		cc.SetRevision(revision)

		EmitIScoreClaimEvent(cc, claim, icx)

		e := getEventLog(cc)
		expIndexed := []any{from}
		expData := []any{claim, icx}
		assert.NoError(t, e.Assert(state.SystemAddress, EventIScoreClaimedV2, expIndexed, expData))

		cc.Clear()
	}
}

func TestEmitPRepIssuedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})

	prep := &IssuePRepJSON{
		IRep:            common.NewHexInt(0),
		RRep:            common.NewHexInt(0),
		TotalDelegation: common.NewHexInt(1000),
		Value:           common.NewHexInt(100),
	}

	EmitPRepIssuedEvent(cc, prep)

	e := getEventLog(cc)
	expData := []any{
		prep.IRep.Value(),
		prep.RRep.Value(),
		prep.TotalDelegation.Value(),
		prep.Value.Value(),
	}
	assert.NoError(t, e.Assert(state.SystemAddress, EventPRepIssued, nil, expData))
}

func TestEmitICXIssuedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})

	result := &IssueResultJSON{
		ByFee:           common.NewHexInt(10),
		ByOverIssuedICX: common.NewHexInt(100),
		Issue:           common.NewHexInt(1000),
	}
	issue := icstate.NewIssue()
	icx := big.NewInt(1234)
	issue.SetOverIssuedIScore(icutils.ICXToIScore(icx))

	EmitICXIssuedEvent(cc, result, issue)

	e := getEventLog(cc)
	expData := []any{
		result.ByFee.Value(),
		result.ByOverIssuedICX.Value(),
		result.Issue.Value(),
		icx,
	}

	assert.NoError(t, e.Assert(state.SystemAddress, EventICXIssued, nil, expData))
}

func TestEmitTermStartedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})
	sequence := 10
	startHeight := int64(1000)
	endHeight := startHeight + icmodule.DefaultTermPeriod - 1

	EmitTermStartedEvent(cc, sequence, startHeight, endHeight)

	e := getEventLog(cc)
	expData := []any{sequence, startHeight, endHeight}
	assert.NoError(t, e.Assert(state.SystemAddress, EventTermStarted, nil, expData))
}

func TestEmitPRepRegisteredEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})

	EmitPRepRegisteredEvent(cc)

	e := getEventLog(cc)
	expData := []any{cc.From()}
	assert.NoError(t, e.Assert(state.SystemAddress, EventPRepRegistered, nil, expData))
}

func TestEmitPRepSetEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})

	EmitPRepSetEvent(cc)

	e := getEventLog(cc)
	expData := []any{cc.From()}
	assert.NoError(t, e.Assert(state.SystemAddress, EventPRepSet, nil, expData))
}

func TestEmitRewardFundTransferredEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	to := newDummyAddress(2)
	amount := icutils.ToLoop(1)

	EmitRewardFundTransferredEvent(cc, icstate.CPSKey, from, to, amount)

	e := getEventLog(cc)
	expData := []any{icstate.CPSKey, from, to, amount}
	assert.NoError(t, e.Assert(state.SystemAddress, EventRewardFundTransferred, nil, expData))
}

func TestEmitRewardFundBurnedEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	key := icstate.RelayKey
	amount := icutils.ToLoop(1)

	EmitRewardFundBurnedEvent(cc, key, from, amount)

	e := getEventLog(cc)
	expData := []any{key, from, amount}
	assert.NoError(t, e.Assert(state.SystemAddress, EventRewardFundBurned, nil, expData))
}

func TestEmitPRepUnregisteredEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})

	EmitPRepUnregisteredEvent(cc)

	e := getEventLog(cc)
	expData := []any{cc.From()}
	assert.NoError(t, e.Assert(state.SystemAddress, EventPRepUnregistered, nil, expData))
}

func TestEmitBTPNetworkTypeActivatedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})
	networkTypeName := "icon"
	ntid := int64(7)

	EmitBTPNetworkTypeActivatedEvent(cc, networkTypeName, ntid)

	e := getEventLog(cc)
	expIndexed := []any{networkTypeName, ntid}
	assert.NoError(t, e.Assert(state.SystemAddress, EventBTPNetworkTypeActivated, expIndexed, nil))
}

func TestEmitBTPNetworkOpenedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})
	ntid := int64(7)
	nid := int64(1)

	EmitBTPNetworkOpenedEvent(cc, ntid, nid)

	e := getEventLog(cc)
	expIndexed := []any{ntid, nid}
	assert.NoError(t, e.Assert(state.SystemAddress, EventBTPNetworkOpened, expIndexed, nil))
}

func TestEmitBTPNetworkClosedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})
	ntid := int64(7)
	nid := int64(1)

	EmitBTPNetworkClosedEvent(cc, ntid, nid)

	e := getEventLog(cc)
	expIndexed := []any{ntid, nid}
	assert.NoError(t, e.Assert(state.SystemAddress, EventBTPNetworkClosed, expIndexed, nil))
}

func TestEmitBTPMessageEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})
	nid := int64(7)
	sn := int64(100)

	EmitBTPMessageEvent(cc, nid, sn)

	e := getEventLog(cc)
	expIndexed := []any{nid, sn}
	assert.NoError(t, e.Assert(state.SystemAddress, EventBTPMessage, expIndexed, nil))
}

func TestEmitGovernanceVariablesSetEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	irep := icutils.ToLoop(100)

	EmitGovernanceVariablesSetEvent(cc, irep)

	e := getEventLog(cc)
	expIndexed := []any{cc.From()}
	expData := []any{irep}
	assert.NoError(t, e.Assert(state.SystemAddress, EventGovernanceVariablesSet, expIndexed, expData))
}

func TestEmitMinimumBondSetEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	bond := icutils.ToLoop(10_000)

	EmitMinimumBondSetEvent(cc, bond)

	e := getEventLog(cc)
	expData := []any{bond}
	assert.NoError(t, e.Assert(state.SystemAddress, EventMinimumBondSet, nil, expData))
}

func TestEmitICXBurnedEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	amount := icutils.ToLoop(1)
	ts := icutils.ToLoop(100)

	for _, rev := range []int{icmodule.RevisionBurnV2, icmodule.RevisionIISS4R0} {
		revision := icmodule.ValueToRevision(rev)
		cc.SetRevision(revision)

		EmitICXBurnedEvent(cc, from, amount, ts)

		e := getEventLog(cc)
		expIndexed := []any{from}
		expData := []any{amount, ts}
		assert.NoError(t, e.Assert(state.SystemAddress, EventICXBurnedV2, expIndexed, expData))

		cc.Clear()
	}
}

func TestEmitDoubleSignReportedEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	signer := newDummyAddress(100)
	dsBlockHeight := int64(700)
	dsType := module.DSTProposal

	EmitDoubleSignReportedEvent(cc, signer, dsBlockHeight, dsType)

	e := getEventLog(cc)
	expIndexed := []any{signer}
	expData := []any{dsBlockHeight, dsType}
	assert.NoError(t, e.Assert(state.SystemAddress, EventDoubleSignReported, expIndexed, expData))
}

func TestEmitBondSetEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionChainScoreEventLog),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	var bonds icstate.Bonds
	for i := 0; i < 3; i++ {
		addr := newDummyAddress(i + 10)
		bond := icstate.NewBond(common.AddressToPtr(addr), icutils.ToLoop(i+1))
		bonds = append(bonds, bond)
	}

	EmitBondSetEvent(cc, bonds)

	e := getEventLog(cc)
	expIndexed := []any{cc.From()}
	expData := []any{codec.BC.MustMarshalToBytes(bonds)}
	assert.NoError(t, e.Assert(state.SystemAddress, EventBondSet, expIndexed, expData))
}

func TestEmitDelegationSetEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionChainScoreEventLog),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	var ds icstate.Delegations
	for i := 0; i < 3; i++ {
		addr := newDummyAddress(i + 10)
		d := icstate.NewDelegation(common.AddressToPtr(addr), icutils.ToLoop(i+1))
		ds = append(ds, d)
	}

	EmitDelegationSetEvent(cc, ds)

	e := getEventLog(cc)
	expIndexed := []any{cc.From()}
	expData := []any{codec.BC.MustMarshalToBytes(ds)}
	assert.NoError(t, e.Assert(state.SystemAddress, EventDelegationSet, expIndexed, expData))
}

func TestEmitPRepCountConfigSetEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	main := int64(22)
	sub := int64(78)
	extra := int64(3)

	EmitPRepCountConfigSetEvent(cc, main, sub, extra)

	e := getEventLog(cc)
	expData := []any{main, sub, extra}
	assert.NoError(t, e.Assert(state.SystemAddress, EventPRepCountConfigSet, nil, expData))
}

func TestEmitRewardFundSetEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision: icmodule.ValueToRevision(icmodule.RevisionIISS4R1),
	})
	value := big.NewInt(1000)

	EmitRewardFundSetEvent(cc, value)

	e := getEventLog(cc)
	expData := []any{value}
	assert.NoError(t, e.Assert(state.SystemAddress, EventRewardFundSet, nil, expData))
}

func TestEmitRewardFundAllocationSetEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision: icmodule.ValueToRevision(icmodule.RevisionIISS4R1),
	})
	key := icstate.KeyIvoter
	rate := icmodule.Rate(1000)

	EmitRewardFundAllocationSetEvent(cc, key, rate)

	e := getEventLog(cc)
	expData := []any{string(key), rate.NumInt64()}
	assert.NoError(t, e.Assert(state.SystemAddress, EventRewardFundAllocationSet, nil, expData))
}

func TestEmitNetworkScoreSetEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision: icmodule.ValueToRevision(icmodule.RevisionIISS4R1),
	})
	key := icstate.CPSKey
	addr := common.MustNewAddressFromString("cx123")

	EmitNetworkScoreSetEvent(cc, icstate.CPSKey, addr)

	e := getEventLog(cc)
	expData := []any{key, addr}
	assert.NoError(t, e.Assert(state.SystemAddress, EventNetworkScoreSet, nil, expData))
}
