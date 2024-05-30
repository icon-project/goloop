/*
 * Copyright 2024 ICON Foundation
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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

func assertBondRequirement(t *testing.T, sim Simulator, br, nextBr icmodule.Rate) {
	term := sim.GetPRepTermInJSON()
	networkInfo := sim.GetNetworkInfoInJSON()

	if sim.Revision().Value() < icmodule.RevisionSetBondRequirementRate {
		assert.Equal(t, br.Percent(), term["bondRequirement"])
		assert.Equal(t, nextBr.Percent(), networkInfo["bondRequirement"])
	} else {
		assert.Equal(t, br.NumInt64(), term["bondRequirementRate"])
		assert.Equal(t, nextBr.NumInt64(), networkInfo["bondRequirementRate"])
	}
}

func assertEventSetBondRequirementRate(t *testing.T, ev *txresult.TestEventLog, rate icmodule.Rate) {
	err := ev.Assert(state.SystemAddress, iiss.EventBondRequirementRateSet, nil, []any{rate.NumInt64()})
	assert.NoError(t, err)
}

func TestSimulatorImpl_SetBondRequirementRate(t *testing.T) {
	const (
		termPeriod = int64(10)
	)
	var err error
	var csi module.ConsensusInfo
	var receipts []Receipt
	var nextBr icmodule.Rate
	br := icmodule.ToRate(5)
	rev := icmodule.ValueToRevision(icmodule.RevisionSetBondRequirementRate - 1)

	cfg := NewSimConfigWithParams(map[SimConfigOption]interface{}{
		SCOTermPeriod:      termPeriod,
		SCOBondRequirement: br,
	})
	env, err := NewEnv(cfg, rev)
	sim := env.Simulator()
	assert.NoError(t, err)
	assert.NotNil(t, sim)
	assert.Equal(t, sim.Revision().Value(), rev.Value())

	// T(0)
	assertBondRequirement(t, sim, br, br)
	assert.NoError(t, sim.Go(csi, 1))
	assertBondRequirement(t, sim, br, br)
	assert.NoError(t, sim.GoToTermEnd(csi))

	// T(1)
	// SetBondRequirementRate is forbidden before icmodule.RevisionSetBondRequirementRate
	receipts, err = sim.GoBySetBondRequirementRate(csi, env.Governance(), icmodule.ToRate(3))
	assert.NoError(t, err)
	assert.Equal(t, Failure, receipts[1].Status())
	assertBondRequirement(t, sim, br, br)

	// Revision update to RevisionSetBondRequirementRate
	rev = icmodule.ValueToRevision(icmodule.RevisionSetBondRequirementRate)
	receipts, err = sim.GoBySetRevision(csi, env.Governance(), rev)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(receipts))
	assert.Equal(t, Success, receipts[1].Status())
	assert.Equal(t, sim.Revision().Value(), rev.Value())

	// GetBondRequirementRate() works after RevisionSetBondRequirementRate
	assertBondRequirement(t, sim, br, br)

	// Ensure that calling setBondRequirementRate() more than once during the same term works well
	for i := 0; i < 2; i++ {
		nextBr = icmodule.ToRate(int64(i))
		receipts, err = sim.GoBySetBondRequirementRate(csi, env.Governance(), nextBr)
		rcpt := receipts[1]
		assert.NoError(t, err)
		assert.Equal(t, 2, len(receipts))
		assert.Equal(t, Success, rcpt.Status())
		assert.Equal(t, 1, len(rcpt.Events()))
		assertEventSetBondRequirementRate(t, rcpt.Events()[0], nextBr)
		assertBondRequirement(t, sim, br, nextBr)
	}

	// When calling setBondRequirementRate() with the new rate that is the same as the existing one,
	// the transaction succeeds without any event logs
	receipts, err = sim.GoBySetBondRequirementRate(csi, env.Governance(), nextBr)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(receipts))
	rcpt := receipts[1]
	assert.Equal(t, Success, rcpt.Status())
	assert.Zero(t, len(rcpt.Events()))
	assertBondRequirement(t, sim, br, nextBr)

	assert.NoError(t, sim.GoToTermEnd(csi))

	// T(2)
	assertBondRequirement(t, sim, nextBr, nextBr)
	assert.NoError(t, sim.Go(csi, 1))
	assertBondRequirement(t, sim, nextBr, nextBr)
	assert.NoError(t, sim.GoToTermEnd(csi))

	// T(3)
	assertBondRequirement(t, sim, nextBr, nextBr)
	assert.NoError(t, sim.Go(csi, 1))
	assertBondRequirement(t, sim, nextBr, nextBr)

	// Ensure that only valid BondRequirementRates are allowed
	for _, rate := range []icmodule.Rate{-1, icmodule.ToRate(101)} {
		receipts, err = sim.GoBySetBondRequirementRate(csi, env.Governance(), rate)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(receipts))
		assert.Equal(t, Failure, receipts[1].Status())
		assert.Zero(t, len(receipts[1].Events()))
		assertBondRequirement(t, sim, nextBr, nextBr)
	}
}
