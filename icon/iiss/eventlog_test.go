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
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

type EventLogSignature struct {
	name   string
	params []string
}

func (s *EventLogSignature) Name() string {
	return s.name
}

func (s *EventLogSignature) Param(i int) string {
	if i < 0 || i >= len(s.params) {
		return ""
	}
	return s.params[i]
}

func (s *EventLogSignature) ParamLen() int {
	return len(s.params)
}

func (s *EventLogSignature) String() string {
	sb := strings.Builder{}
	sb.WriteString(s.name)
	sb.WriteByte('(')
	for i, param := range s.params {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(param)
	}
	sb.WriteByte(')')
	return sb.String()
}

func NewEventLogSignature(sig string) (*EventLogSignature, error) {
	if len(sig) < 3 {
		return nil, errors.New("NoName")
	}
	idx := strings.Index(sig, "(")
	if idx <= 0 {
		return nil, errors.New("'(' NotFound")
	}
	endIdx := len(sig) - 1
	if sig[endIdx] != ')' {
		return nil, errors.New("')' NotFound")
	}

	name := sig[:idx]
	if !(name[0] >= 'A' && name[0] <= 'Z') {
		return nil, errors.New("FirstCharacterIsNotUpperCase")
	}

	params := strings.Split(sig[idx+1:endIdx], ",")
	for _, param := range params {
		switch param {
		case "Address":
		case "bytes":
		case "bool":
		case "int":
		case "str":
		default:
			return nil, errors.Errorf("UnknownParam(%s)", param)
		}
	}

	return &EventLogSignature{
		name:   name,
		params: params,
	}, nil
}

func getParams(cc *mockCallContext) (module.Address, [][]byte, [][]byte) {
	call := cc.GetCalls("OnEvent")[0]
	params := call.Params()
	return params[0].(module.Address), params[1].([][]byte), params[2].([][]byte)
}

func checkEventLogSignature(t *testing.T, scoreAddress module.Address, indexed, data [][]byte) {
	assert.True(t, state.SystemAddress.Equal(scoreAddress))
	sig := string(indexed[0])
	elSig, err := NewEventLogSignature(sig)
	assert.NoError(t, err)
	assert.Equal(t, sig, elSig.String())
	assert.Equal(t, len(indexed)+len(data), elSig.ParamLen()+1)
}

func TestEmitSlashingRateSetEvent(t *testing.T) {
	from := newDummyAddress(1)
	rate := icmodule.Rate(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})

	for _, rev := range []int{icmodule.RevisionIISS4R0 - 1, icmodule.RevisionIISS4R0} {
		revision := icmodule.ValueToRevision(rev)
		cc.SetRevision(revision)
		cc.Clear()

		EmitSlashingRateSetEvent(cc, icmodule.PenaltyAccumulatedValidationFailure, rate)
		scoreAddress, indexed, data := getParams(cc)
		checkEventLogSignature(t, scoreAddress, indexed, data)
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
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
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
	scoreAddress, indexed, data := getParams(cc)
	// Indexed
	checkEventLogSignature(t, scoreAddress, indexed, data)
	assert.Zero(t, bytes.Compare(from.Bytes(), indexed[1]))
	// Data
	assert.Equal(t, 1, len(data))
}

func TestEmitPenaltyImposedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})
	owner := newDummyAddress(2)
	ps := icstate.NewPRepStatus(owner)

	EmitPenaltyImposedEvent(cc, ps, icmodule.PenaltyDoubleSign)

	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
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

	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitIScoreClaimEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	icx := icutils.ToLoop(1)
	claim := icutils.ICXToIScore(icx)

	for _, rev := range []int{icmodule.RevisionIISS2-1, icmodule.RevisionIISS4R0} {
		revision := icmodule.ValueToRevision(rev)
		cc.SetRevision(revision)
		EmitIScoreClaimEvent(cc, claim, icx)
		scoreAddress, indexed, data := getParams(cc)
		checkEventLogSignature(t, scoreAddress, indexed, data)
		cc.Clear()
	}
}

func TestEmitPRepIssuedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})

	prep := &IssuePRepJSON{
		IRep: common.NewHexInt(0),
		RRep: common.NewHexInt(0),
		TotalDelegation: common.NewHexInt(1000),
		Value: common.NewHexInt(100),
	}

	EmitPRepIssuedEvent(cc, prep)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitICXIssuedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})

	result := &IssueResultJSON{
		ByFee: common.NewHexInt(10),
		ByOverIssuedICX: common.NewHexInt(100),
		Issue: common.NewHexInt(1000),
	}
	issue := icstate.NewIssue()

	EmitICXIssuedEvent(cc, result, issue)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitTermStartedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})
	term := &icstate.TermSnapshot{}

	EmitTermStartedEvent(cc, term)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitPRepRegisteredEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})

	EmitPRepRegisteredEvent(cc)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitPRepSetEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})

	EmitPRepSetEvent(cc)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
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
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitRewardFundBurnedEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	amount := icutils.ToLoop(1)

	EmitRewardFundBurnedEvent(cc, icstate.RelayKey, from, amount)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitPRepUnregisteredEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})

	EmitPRepUnregisteredEvent(cc)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitBTPNetworkTypeActivatedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})

	EmitBTPNetworkTypeActivatedEvent(cc, "icon", 7)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitBTPNetworkOpenedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})

	EmitBTPNetworkOpenedEvent(cc, 7, 1)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitBTPNetworkClosedEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})

	EmitBTPNetworkClosedEvent(cc, 7, 1)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitBTPMessageEvent(t *testing.T) {
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
	})

	EmitBTPMessageEvent(cc, 7, 100)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
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
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
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
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
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

	for _, rev := range []int{icmodule.RevisionFixBurnEventSignature, icmodule.RevisionIISS4R0} {
		revision := icmodule.ValueToRevision(rev)
		cc.SetRevision(revision)
		EmitICXBurnedEvent(cc, from, amount, ts)
		scoreAddress, indexed, data := getParams(cc)
		checkEventLogSignature(t, scoreAddress, indexed, data)
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

	EmitDoubleSignReportedEvent(cc, signer, int64(700), module.DSTProposal)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitBondSetEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionVoteEventLog),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	var bonds icstate.Bonds
	for i := 0; i < 3; i++ {
		addr := newDummyAddress(i+10)
		bond := icstate.NewBond(common.AddressToPtr(addr), icutils.ToLoop(i+1))
		bonds = append(bonds, bond)
	}

	EmitBondSetEvent(cc, bonds)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitDelegationSetEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionVoteEventLog),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})
	var ds icstate.Delegations
	for i := 0; i < 3; i++ {
		addr := newDummyAddress(i+10)
		d := icstate.NewDelegation(common.AddressToPtr(addr), icutils.ToLoop(i+1))
		ds = append(ds, d)
	}

	EmitDelegationSetEvent(cc, ds)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}

func TestEmitPRepCountConfigSetEvent(t *testing.T) {
	from := newDummyAddress(1)
	cc := newMockCallContext(map[CallCtxOption]interface{}{
		CallCtxOptionRevision:    icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
		CallCtxOptionBlockHeight: int64(1000),
		CallCtxOptionFrom:        from,
	})

	EmitPRepCountConfigSetEvent(cc, 22, 78, 3)
	scoreAddress, indexed, data := getParams(cc)
	checkEventLogSignature(t, scoreAddress, indexed, data)
}
