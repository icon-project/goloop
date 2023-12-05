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
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
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

	var err error
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
	events := receipts[1].Events()
	for i, pt := range []icmodule.PenaltyType{
		icmodule.PenaltyPRepDisqualification,
		icmodule.PenaltyAccumulatedValidationFailure,
		icmodule.PenaltyMissedNetworkProposalVote,
		icmodule.PenaltyDoubleSign,
	}{
		assert.True(t, CheckSlashingRateSetEvent(events[i], pt, expRates[pt.String()]))
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
	var csi module.ConsensusInfo
	initRevision := icmodule.ValueToRevision(icmodule.RevisionIISS4R0)
	env, err := NewEnv(cfg, initRevision)
	assert.NoError(t, err)
	assert.NotNil(t, env)
	sim := env.Simulator()
	assert.Equal(t, initRevision, sim.Revision())
	gov := env.Governance()

	// T(0) --------------------------------------------------
	assert.NoError(t, sim.Go(nil, 2))

	csi = NewConsensusInfoBySim(sim)

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
	tx = sim.SetSlashingRates(gov, expRates)
	receipts, err := sim.GoByTransaction(csi, tx)
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
	events := receipts[1].Events()
	assert.Equal(t, 4, len(events))
	for _, e := range events {
		signature, indexed, data, err := e.DecodeParams()
		assert.NoError(t, err)

		assert.True(t, e.Address.Equal(state.SystemAddress))
		assert.Equal(t, signature, iiss.EventSlashingRateSet)
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
	receipts, err = sim.GoBySetMinimumBond(csi, gov, minBond)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(receipts[1]))

	events = receipts[1].Events()
	assert.Equal(t, 1, len(events))
	expData := []any{minBond}
	assert.NoError(t, events[0].Assert(state.SystemAddress, iiss.EventMinimumBondSet, nil, expData))

	newMinBond := sim.GetMinimumBond()
	assert.Zero(t, minBond.Cmp(newMinBond))

	// SetRewardFundAllocation2
	values := map[icstate.RFundKey]icmodule.Rate{
		icstate.KeyIcps:   icmodule.ToRate(10),
		icstate.KeyIprep:  icmodule.ToRate(85),
		icstate.KeyIrelay: icmodule.ToRate(0),
		icstate.KeyIwage:  icmodule.ToRate(5),
	}
	receipts, err = sim.GoBySetRewardFundAllocation2(csi, gov, values)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(receipts[1]))

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
	}

	// PRepCountConfig
	pcc := sim.PRepCountConfig()
	assert.NoError(t, err)
	assert.Equal(t, 22, pcc.MainPReps())
	assert.Equal(t, 78, pcc.SubPReps())
	assert.Equal(t, 3, pcc.ExtraMainPReps())

	// SetRevision: IISS4R0 -> IISS4R1
	revision := icmodule.ValueToRevision(icmodule.RevisionIISS4R1)
	receipts, err = sim.GoBySetRevision(csi, gov, revision)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(receipts[1]))
	assert.Equal(t, revision, sim.Revision())

	assert.NoError(t, sim.GoToTermEnd(csi))

	// T(1) ----------------------------------------------------------
	assert.NoError(t, sim.Go(csi, 2))

	idx := 10
	voted := make([]bool, pcc.MainPReps()+pcc.ExtraMainPReps())
	for i := 0; i < len(voted); i++ {
		voted[i] = true
	}
	voted[idx] = false

	vl := sim.ValidatorList()
	csi = NewConsensusInfo(sim.Database(), vl, voted)
	assert.NoError(t, sim.Go(csi, validationPenaltyCondition-1))

	owner := vl[idx].Address()
	prep := sim.GetPRepByOwner(owner)
	assert.True(t, CheckElectablePRep(prep, icstate.GradeMain))
	assert.Equal(t, validationPenaltyCondition-1, prep.GetVFailCont(sim.BlockHeight()))
	oldBonded := prep.Bonded()

	// We expect this prep to get penalized for validationFailurePenalty
	assert.NoError(t, sim.Go(csi, 1))
	prep = sim.GetPRepByOwner(owner)
	assert.True(t, CheckPenalizedPRep(prep))
	// Bond of prep is not slashed
	assert.Zero(t, prep.Bonded().Cmp(oldBonded))
	assert.True(t, ValidatorIndexOf(sim.ValidatorList(), owner) < 0)

	assert.NoError(t, sim.Go(csi, 2))
	csi = NewConsensusInfoBySim(sim)
	assert.NoError(t, sim.GoToTermEnd(csi))

	// T(2) --------------------------------------------------------
	assert.NoError(t, sim.Go(csi, 2))
	csi = NewConsensusInfoBySim(sim)

	prep = sim.GetPRepByOwner(owner)
	assert.True(t, CheckPenalizedPRep(prep))

	receipts, err = sim.GoByRequestUnjail(csi, owner)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(receipts[1]))
	prep = sim.GetPRepByOwner(owner)
	assert.True(t, CheckUnjailingPRep(prep))

	assert.NoError(t, sim.GoToTermEnd(csi))

	// T(3) ----------------------------------------
	assert.NoError(t, sim.Go(csi, 2))

	prep = sim.GetPRepByOwner(owner)
	assert.True(t, CheckElectablePRep(prep, icstate.GradeMain))

	voted[idx] = true
	vl = sim.ValidatorList()
	idx = ValidatorIndexOf(vl, owner)
	assert.True(t, idx >= 0)
	voted[idx] = false

	csi = NewConsensusInfo(sim.Database(), vl, voted)
	assert.NoError(t, sim.Go(csi, validationPenaltyCondition-1))
	prep = sim.GetPRepByOwner(owner)
	assert.True(t, CheckElectablePRep(prep, icstate.GradeMain))

	csi = NewConsensusInfoBySim(sim)
	assert.NoError(t, sim.Go(csi, 1))
	prep = sim.GetPRepByOwner(owner)
	assert.True(t, CheckElectablePRep(prep, icstate.GradeMain))

	assert.NoError(t, sim.Go(csi, 1))
	prep = sim.GetPRepByOwner(owner)
	assert.True(t, CheckElectablePRep(prep, icstate.GradeMain))
	oldBonded = prep.Bonded()

	// The second validationFailurePenalty
	csi = NewConsensusInfoBySim(sim, sim.ValidatorIndexOf(owner))
	assert.NoError(t, sim.Go(csi, validationPenaltyCondition))
	prep = sim.GetPRepByOwner(owner)
	assert.True(t, CheckPenalizedPRep(prep))
	assert.True(t, ValidatorIndexOf(sim.ValidatorList(), owner) < 0)
	assert.Equal(t, 2, prep.GetVPenaltyCount())
	assert.Zero(t, prep.Bonded().Cmp(oldBonded))

	assert.NoError(t, sim.Go(csi, 2))

	// RequestUnjail
	csi = NewConsensusInfoBySim(sim)
	receipts, err = sim.GoByRequestUnjail(csi, owner)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(receipts[1]))

	prep = sim.GetPRepByOwner(owner)
	assert.True(t, CheckUnjailingPRep(prep))

	assert.NoError(t, sim.GoToTermEnd(csi))

	// T(4) -------------------------------------------
	assert.NoError(t, sim.Go(csi, 2))

	prep = sim.GetPRepByOwner(owner)
	assert.True(t, CheckElectablePRep(prep, icstate.GradeMain))

	csi = NewConsensusInfoBySim(sim, sim.ValidatorIndexOf(owner))

	assert.NoError(t, sim.Go(csi, validationPenaltyCondition-1))
	prep = sim.GetPRepByOwner(owner)
	assert.True(t, CheckElectablePRep(prep, icstate.GradeMain))

	receipts, err = sim.GoByBlock(csi, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(receipts))

	prep = sim.GetPRepByOwner(owner)
	assert.True(t, CheckPenalizedPRep(prep))
	assert.True(t, ValidatorIndexOf(sim.ValidatorList(), owner) < 0)
	assert.Equal(t, 3, prep.GetVPenaltyCount())

	pt := icmodule.PenaltyAccumulatedValidationFailure
	rate, err := sim.GetSlashingRate(pt)
	assert.Equal(t, expRates[pt.String()], rate)
	assert.NoError(t, err)
	slashed := rate.MulBigInt(oldBonded)
	bonded := prep.Bonded()
	assert.Zero(t, bonded.Cmp(new(big.Int).Sub(oldBonded, slashed)))

	// Check the receipt of baseTx on imposing penalty
	// PenaltyImposed, PenaltyImposed, Slashed, ICXBurnedV2
	events = receipts[0].Events()
	assert.True(t, CheckPenaltyImposedEvent(
		events[0], owner, icstate.Active, icmodule.PenaltyValidationFailure))
	assert.True(t, CheckPenaltyImposedEvent(
		events[1], owner, icstate.Active, icmodule.PenaltyAccumulatedValidationFailure))
	bonders := sim.GetBonderList(owner)
	assert.True(t, CheckSlashedEvent(events[2], owner, bonders[0], slashed))
	assert.True(t, CheckICXBurnedV2Event(events[3], state.SystemAddress, slashed, sim.TotalSupply()))

	// SetPRepCountConfig
	receipts, err = sim.GoBySetPRepCountConfig(csi, gov, map[string]int64{"main": 19, "sub": 81, "extra": 9})
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(receipts...))
	assert.Equal(t, 25, len(sim.ValidatorList()))

	events = receipts[1].Events()
	assert.True(t, CheckPRepCountConfigSetEvent(events[0], 19, 81, 9))

	assert.NoError(t, sim.GoToTermEnd(csi))

	// T(5) ------------------------------------------------
	assert.NoError(t, sim.Go(csi, 2))

	var grade icstate.Grade
	term := sim.TermSnapshot()
	pssList := term.PRepSnapshots()
	for i, pss := range pssList {
		prep = sim.GetPRepByOwner(pss.Owner())
		if i < 28 {
			grade = icstate.GradeMain
		} else {
			grade = icstate.GradeSub
		}
		CheckElectablePRep(prep, grade)
	}

	vl = sim.ValidatorList()
	assert.Equal(t, 28, len(vl))
	for i, v := range vl {
		prep = sim.GetPRepByNode(v.Address())
		assert.True(t, pssList[i].Owner().Equal(prep.Owner()))
	}

	bh := int64(500)
	prep = sim.GetPRepByOwner(owner)
	node := prep.NodeAddress()
	receipts, err = sim.GoByHandleDoubleSignReport(
		csi, state.SystemAddress, module.DSTVote, bh, node)
	assert.NoError(t, err)
	events = receipts[1].Events()
	assert.True(t, CheckDoubleSignReportedEvent(events[0], node, bh, module.DSTVote))

	prep = sim.GetPRepByOwner(owner)
	assert.True(t, CheckPenalizedPRep(prep))
	assert.False(t, prep.IsDoubleSignReportable(nil, sim.BlockHeight()))
}
