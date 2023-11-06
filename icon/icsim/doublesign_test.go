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
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

func TestDoubleSign_RequestUnjailNormalCase(t *testing.T) {
	const (
		termPeriod = int64(10)
	)
	var err error
	var dsBlockHeight int64
	var csi module.ConsensusInfo
	var rcpt Receipt

	cfg := NewSimConfigWithParams(map[string]interface{}{
		"TermPeriod": termPeriod,
	})
	env, err := NewEnv(cfg, icmodule.RevisionIISS4R1)
	sim := env.Simulator()
	assert.NoError(t, err)
	assert.NotNil(t, sim)
	assert.Equal(t, sim.Revision(), icmodule.ValueToRevision(icmodule.RevisionIISS4R1))

	// T(0)
	assert.NoError(t, sim.GoToTermEnd(nil))
	term := sim.TermSnapshot()
	assert.Equal(t, icstate.IISSVersion4, term.GetIISSVersion())

	// Next Term

	prep0 := env.preps[0]
	prep0Sub := env.preps[cfg.TotalMainPRepCount()]
	prep := sim.GetPRep(prep0)
	assert.Equal(t, icstate.GradeMain, prep.Grade())
	assert.Zero(t, prep.JailFlags())

	// T(1) : SuccessCase(HandleDoubleSignReport)
	dsType := module.DSTProposal
	dsBlockHeight = sim.BlockHeight() - 10
	rcpt, err = sim.GoByHandleDoubleSignReport(csi, state.SystemAddress, dsType, dsBlockHeight, prep0)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(rcpt))
	// Check the status of prep0
	prep = sim.GetPRep(prep0)
	assert.Equal(t, icstate.GradeCandidate, prep.Grade())
	assert.True(t, icutils.MatchAll(prep.JailFlags(), icstate.JFlagInJail|icstate.JFlagDoubleSign))
	assert.Zero(t, prep.MinDoubleSignHeight())
	// Check the status of prep0Sub(prep25)
	prep = sim.GetPRep(prep0Sub)
	assert.Equal(t, icstate.GradeMain, prep.Grade())

	// Move to the block which is 5 blocks earlier
	term = sim.TermSnapshot()
	assert.NoError(t, sim.GoTo(csi, term.GetEndHeight()-5))

	// T(100 - 5) : DoubleSignReport for the PRep with JailFlagDoubleSign is ignored silently (Success)
	rcpt, err = sim.GoByHandleDoubleSignReport(csi, state.SystemAddress, module.DSTVote, dsBlockHeight+1, prep0)
	assert.NoError(t, err)
	assert.True(t, rcpt.Status() == 1)

	// T(100 - 5 + 1)
	assert.NoError(t, sim.GoToTermEnd(csi))

	// Next Term

	// T(0) : RequestUnjail transaction
	prep = sim.GetPRep(prep0)
	assert.Equal(t, icstate.GradeCandidate, prep.Grade())
	rcpt, err = sim.GoByRequestUnjail(csi, prep0)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(rcpt))
	prep = sim.GetPRep(prep0)
	assert.True(t, icutils.MatchAll(
		prep.JailFlags(), icstate.JFlagUnjailing|icstate.JFlagDoubleSign|icstate.JFlagInJail))
	assert.Equal(t, icstate.GradeCandidate, prep.Grade())
	assert.True(t, prep.IsJailInfoElectable())

	// T(2) : Go to term end
	assert.NoError(t, sim.GoToTermEnd(nil))
	prep = sim.GetPRep(prep0)
	assert.Equal(t, icstate.GradeMain, prep.Grade())
	assert.Zero(t, prep.JailFlags())
	assert.Equal(t, sim.BlockHeight(), prep.MinDoubleSignHeight())
	vl := sim.ValidatorList()
	assert.True(t, vl[0].Address().Equal(prep0))
}

func TestHandleDoubleSignReport_Slashing(t *testing.T) {
	const (
		termPeriod = int64(10)
	)
	var err error
	var dsBlockHeight int64
	var csi module.ConsensusInfo
	var rcpt Receipt
	var revision module.Revision
	slashingRate := icmodule.ToRate(10)

	cfg := NewSimConfigWithParams(map[string]interface{}{
		"TermPeriod": termPeriod,
	})
	revision = icmodule.ValueToRevision(icmodule.RevisionIISS4R0)
	env, err := NewEnv(cfg, revision)
	sim := env.Simulator()
	assert.NoError(t, err)
	assert.NotNil(t, sim)
	assert.Equal(t, revision, sim.Revision())

	// Term
	// T(0)
	term := sim.TermSnapshot()
	assert.Equal(t, icstate.IISSVersion3, term.GetIISSVersion())
	revision = icmodule.ValueToRevision(icmodule.RevisionIISS4R1)
	rcpt, err = sim.GoBySetRevision(csi, env.Governance(), revision)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(rcpt))

	// T(1)
	const penaltyType = icmodule.PenaltyDoubleSign
	rcpt, err = sim.GoBySetSlashingRates(csi, env.Governance(), map[string]icmodule.Rate{
		penaltyType.String(): slashingRate,
	})
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(rcpt))
	jso, err := sim.GetSlashingRates()
	assert.NoError(t, err)
	assert.Equal(t, slashingRate.NumInt64(), jso[penaltyType.String()])

	assert.NoError(t, sim.GoToTermEnd(nil))

	// Next Term
	// T(0)
	term = sim.TermSnapshot()
	assert.Equal(t, icstate.IISSVersion4, term.GetIISSVersion())

	prep0 := env.preps[0]
	prep0Sub := env.preps[cfg.TotalMainPRepCount()]
	prep := sim.GetPRep(prep0)
	assert.Equal(t, icstate.GradeMain, prep.Grade())
	assert.Zero(t, prep.JailFlags())

	// T(1) : SuccessCase(HandleDoubleSignReport)
	prep = sim.GetPRep(prep0)
	oldBonded := prep.Bonded()
	oldTotalSupply := sim.TotalSupply()
	oldTotalStake := sim.TotalStake()

	dsType := module.DSTProposal
	dsBlockHeight = sim.BlockHeight() - 10
	rcpt, err = sim.GoByHandleDoubleSignReport(csi, state.SystemAddress, dsType, dsBlockHeight, prep0)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(rcpt))
	// Check the status of prep0
	prep = sim.GetPRep(prep0)
	assert.Equal(t, icstate.GradeCandidate, prep.Grade())
	assert.True(t, icutils.MatchAll(prep.JailFlags(), icstate.JFlagInJail|icstate.JFlagDoubleSign))
	assert.Zero(t, prep.MinDoubleSignHeight())
	// Check the status of prep0Sub(prep25)
	prep = sim.GetPRep(prep0Sub)
	assert.Equal(t, icstate.GradeMain, prep.Grade())
	// Slashing for DoubleSignPenalty
	prep = sim.GetPRep(prep0)
	newBonded := prep.Bonded()
	slashed := estimateSlashed(slashingRate, oldBonded)
	assert.Zero(t, newBonded.Cmp(new(big.Int).Sub(oldBonded, slashed)))
	// Check if the slashed amount is burned
	assert.Zero(t, sim.TotalSupply().Cmp(new(big.Int).Sub(oldTotalSupply, slashed)))
	assert.Zero(t, sim.TotalStake().Cmp(new(big.Int).Sub(oldTotalStake, slashed)))
	bonderAccount := sim.GetAccountSnapshot(env.bonders[0])
	assert.Zero(t, newBonded.Cmp(bonderAccount.Bond()))
}

func TestDoubleSign_HandleDoubleSignReportErrorCases(t *testing.T) {
	const (
		termPeriod = int64(10)
	)
	var err error
	var csi module.ConsensusInfo
	var rcpt Receipt

	cfg := NewSimConfigWithParams(map[string]interface{}{
		"TermPeriod": termPeriod,
	})
	env, err := NewEnv(cfg, icmodule.RevisionIISS4R1)
	sim := env.Simulator()
	assert.NoError(t, err)
	assert.NotNil(t, sim)
	assert.Equal(t, sim.Revision(), icmodule.ValueToRevision(icmodule.RevisionIISS4R1))

	// T(0)
	assert.NoError(t, sim.GoToTermEnd(nil))
	term := sim.TermSnapshot()
	assert.Equal(t, icstate.IISSVersion4, term.GetIISSVersion())

	// Next Term
	prep0 := env.preps[0]
	dsBlockHeight := sim.BlockHeight() - 10

	// T(0) : ErrorCases of HandleDoubleSignReport
	args := []struct {
		dsType        string
		dsBlockHeight int64
		signer        module.Address
	}{
		{"InvalidType", dsBlockHeight, prep0},
		{module.DSTProposal, int64(-1), prep0},
		{module.DSTVote, int64(0), prep0},
		{module.DSTProposal, int64(-2), prep0},
		{module.DSTVote, int64(-3), prep0},
		{module.DSTProposal, sim.BlockHeight() - 10, env.users[99]},
		{module.DSTVote, sim.BlockHeight() - 10, env.users[99]},
	}
	for i, arg := range args {
		name := fmt.Sprintf("case-%02d", i)
		t.Run(name, func(t *testing.T) {
			dsBlockHeight = arg.dsBlockHeight
			switch dsBlockHeight {
			case -2:
				dsBlockHeight = sim.BlockHeight() + 1
			case -3:
				dsBlockHeight = sim.BlockHeight() + 2
			}

			// ErrorCase(InvalidDoubleSignBlockHeight)
			rcpt, err = sim.GoByHandleDoubleSignReport(
				csi, state.SystemAddress, arg.dsType, dsBlockHeight, arg.signer)
			assert.NoError(t, err)
			assert.True(t, rcpt.Status() == 0)

			// No impact on PRep's JailFlags
			prep := sim.GetPRep(prep0)
			assert.Zero(t, prep.JailFlags())
			assert.Equal(t, icstate.GradeMain, prep.Grade())
			assert.Zero(t, prep.MinDoubleSignHeight())
		})
	}
}

func TestDoubleSign_RequestUnjailForNormalPRep(t *testing.T) {
	const (
		termPeriod = int64(10)
	)
	var err error
	var csi module.ConsensusInfo
	var rcpt Receipt

	cfg := NewSimConfigWithParams(map[string]interface{}{
		"TermPeriod": termPeriod,
	})
	env, err := NewEnv(cfg, icmodule.RevisionIISS4R1)
	sim := env.Simulator()
	assert.NoError(t, err)
	assert.NotNil(t, sim)
	assert.Equal(t, sim.Revision(), icmodule.ValueToRevision(icmodule.RevisionIISS4R1))

	// T(0)
	assert.NoError(t, sim.GoToTermEnd(nil))
	term := sim.TermSnapshot()
	assert.Equal(t, icstate.IISSVersion4, term.GetIISSVersion())

	// Next Term

	// T(0)
	prep0 := env.preps[0]
	prep := sim.GetPRep(prep0)
	assert.Equal(t, icstate.GradeMain, prep.Grade())
	assert.Zero(t, prep.JailFlags())

	// UnjailRequest for a normal PRep will cause transaction failure
	rcpt, err = sim.GoByRequestUnjail(csi, prep0)
	assert.NoError(t, err)
	assert.Zero(t, rcpt.Status())
	assert.Zero(t, sim.GetPRep(prep0).JailFlags())
}