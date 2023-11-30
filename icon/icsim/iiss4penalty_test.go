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

package icsim

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/service/state"
)

func TestSimulatorImpl_SetSlashingRate(t *testing.T) {
	const (
		termPeriod                           = int64(100)
		mainPRepCount                        = int64(22)
		validationPenaltyCondition           = int64(5)
		consistentValidationPenaltyCondition = int64(3)
	)

	cfg := NewSimConfigWithParams(map[SimConfigOption]interface{}{
		SCOMainPReps:                         mainPRepCount,
		SCOTermPeriod:                        termPeriod,
		SCOValidationFailurePenaltyCondition: validationPenaltyCondition,
		SCOAccumulatedValidationFailurePenaltyCondition: consistentValidationPenaltyCondition,
	})

	var tx Transaction
	initRevision := icmodule.ValueToRevision(icmodule.RevisionIISS4R0)
	env, err := NewEnv(cfg, initRevision)
	assert.NoError(t, err)
	assert.NotNil(t, env)
	sim := env.Simulator()
	assert.Equal(t, initRevision, sim.Revision())

	_, err = sim.GetSlashingRates()
	assert.NoError(t, err)

	// Set new slashingRates
	expRates := map[string]icmodule.Rate{
		icmodule.PenaltyPRepDisqualification.String():         icmodule.ToRate(50),
		icmodule.PenaltyValidationFailure.String():            icmodule.Rate(0),
		icmodule.PenaltyAccumulatedValidationFailure.String(): icmodule.Rate(52),
		icmodule.PenaltyMissedNetworkProposalVote.String():    icmodule.ToRate(53),
		icmodule.PenaltyDoubleSign.String():                   icmodule.ToRate(54),
	}
	tx = sim.SetSlashingRates(env.Governance(), expRates)
	receipts, err := sim.GoByTransaction(nil, tx)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(receipts...))

	// Check if slashingRates are set properly
	rates, err := sim.GetSlashingRates()
	assert.Equal(t, len(expRates), len(rates))
	for key, value := range expRates {
		assert.Equal(t, value.NumInt64(), rates[key].(int64))
	}

	// Check eventLogs for slashingRate
	// There is no eventLog for ValidationFailurePenalty, as its rate is not changed
	m := make(map[string]bool)
	events := receipts[0].Events()
	assert.Equal(t, 4, len(events))
	for _, e := range events {
		signature, indexed, data, err := e.DecodeParams()
		assert.NoError(t, err)

		assert.True(t, e.Address.Equal(state.SystemAddress))
		assert.Equal(t, signature, "SlashingRateSet(str,int)")
		assert.Zero(t, len(indexed))
		assert.Equal(t, 2, len(data))

		// duplicate eventLog check
		penaltyName := data[0].(string)
		assert.False(t, m[penaltyName])
		m[penaltyName] = true

		rate := icmodule.Rate(data[1].(*big.Int).Int64())
		assert.Equal(t, expRates[penaltyName], rate)
	}
}

func TestSimulatorImpl_IISS4PenaltySystem(t *testing.T) {
	const (
		termPeriod                           = int64(100)
		mainPRepCount                        = int64(22)
		validationPenaltyCondition           = int64(5)
		consistentValidationPenaltyCondition = int64(3)
	)

	cfg := NewSimConfigWithParams(map[SimConfigOption]interface{}{
		SCOMainPReps:                         mainPRepCount,
		SCOTermPeriod:                        termPeriod,
		SCOValidationFailurePenaltyCondition: validationPenaltyCondition,
		SCOAccumulatedValidationFailurePenaltyCondition: consistentValidationPenaltyCondition,
	})

	// Initialize simulation environment based on a specific revision
	var tx Transaction
	var rcpt Receipt
	initRevision := icmodule.ValueToRevision(icmodule.RevisionIISS4R0)
	env, err := NewEnv(cfg, initRevision)
	assert.NoError(t, err)
	assert.NotNil(t, env)
	sim := env.Simulator()
	assert.Equal(t, initRevision, sim.Revision())

	// T(0) --------------------------------------------------
	// SetSlashingRAtes
	_, err = sim.GetSlashingRates()
	assert.NoError(t, err)

	// Set new slashingRates
	expRates := map[string]icmodule.Rate{
		icmodule.PenaltyPRepDisqualification.String():         icmodule.ToRate(100), // 100%
		icmodule.PenaltyValidationFailure.String():            icmodule.Rate(0),     // 0%
		icmodule.PenaltyAccumulatedValidationFailure.String(): icmodule.Rate(1),     // 0.01%
		icmodule.PenaltyMissedNetworkProposalVote.String():    icmodule.Rate(1),     // 0.01%
		icmodule.PenaltyDoubleSign.String():                   icmodule.ToRate(10),  // 10%
	}
	tx = sim.SetSlashingRates(env.Governance(), expRates)
	receipts, err := sim.GoByTransaction(nil, tx)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(receipts...))

	// Check if slashingRates are set properly
	rates, err := sim.GetSlashingRates()
	assert.Equal(t, len(expRates), len(rates))
	for key, value := range expRates {
		assert.Equal(t, value.NumInt64(), rates[key].(int64))
	}

	// Check eventLogs for slashingRate
	// There is no eventLog for ValidationFailurePenalty which rate is not changed
	events := receipts[0].Events()
	assert.Equal(t, 4, len(events))
	for _, e := range events {
		signature, indexed, data, err := e.DecodeParams()
		assert.NoError(t, err)

		assert.True(t, e.Address.Equal(state.SystemAddress))
		assert.Equal(t, signature, "SlashingRateSet(str,int)")
		assert.Zero(t, len(indexed))
		assert.Equal(t, 2, len(data))

		penaltyName := data[0].(string)
		rate := icmodule.Rate(data[1].(*big.Int).Int64())
		assert.Equal(t, expRates[penaltyName], rate)
	}

	// SetMinimumBond
	minBond := sim.GetMinimumBond()
	assert.Zero(t, minBond.Sign())

	minBond = icutils.ToLoop(10_000)
	rcpt, err = sim.GoBySetMinimumBond(nil, env.Governance(), minBond)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(rcpt))

	events = rcpt.Events()
	assert.Equal(t, 1, len(events))
	expData := []any{minBond}
	assert.NoError(t, events[0].Assert(state.SystemAddress, "MinimumBondSet(int)", nil, expData))

	newMinBond := sim.GetMinimumBond()
	assert.Zero(t, minBond.Cmp(newMinBond))

	// SetRewardFundAllocation2
	values := map[icstate.RFundKey]icmodule.Rate{
		icstate.KeyIcps:   icmodule.ToRate(0),
		icstate.KeyIprep:  icmodule.ToRate(90),
		icstate.KeyIrelay: icmodule.ToRate(0),
		icstate.KeyIwage:  icmodule.ToRate(10),
	}
	rcpt, err = sim.GoBySetRewardFundAllocation2(nil, env.Governance(), values)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(rcpt))

	rf := sim.GetRewardFundAllocation2()
	for k, v := range values {
		switch k {
		case icstate.KeyIcps:
			assert.Equal(t, v, rf.ICps())
		case icstate.KeyIprep:
			assert.Equal(t, v, rf.IPrep())
		case icstate.KeyIrelay:
			assert.Equal(t, v, rf.IRelay())
		case icstate.KeyIwage:
			assert.Equal(t, v, rf.Iwage())
		default:
			assert.True(t, false, "InvalidRFundKey(%s)", k)
		}

		// T(1) --------------------------------------------------
	}
}
