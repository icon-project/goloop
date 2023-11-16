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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
)

func TestSimulatorImpl_SetPRepCountConfig(t *testing.T) {
	const (
		termPeriod            = int64(10)
		rev                   = icmodule.RevisionIISS4R0
		newMainPRepCount      = int64(19)
		newSubPRepCount       = int64(81)
		newExtraMainPRepCount = int64(9)
	)
	var err error
	var csi module.ConsensusInfo
	var rcpt Receipt

	cfg := NewSimConfigWithParams(map[SimConfigOption]interface{}{
		SCOTermPeriod: termPeriod,
	})
	env, err := NewEnv(cfg, rev)
	sim := env.Simulator()
	assert.NoError(t, err)
	assert.NotNil(t, sim)
	assert.Equal(t, sim.Revision(), icmodule.ValueToRevision(rev))

	// T(0)
	jso, err := sim.GetPRepCountConfig()
	assert.NoError(t, err)
	assert.Equal(t, cfg.MainPRepCount, jso["main"].(int64))
	assert.Equal(t, cfg.SubPRepCount, jso["sub"].(int64))
	assert.Equal(t, cfg.ExtraMainPRepCount, jso["extra"].(int64))
	assert.Equal(t, len(sim.ValidatorList()), int(cfg.TotalMainPRepCount()))

	counts := map[string]int64{
		"main":  newMainPRepCount,
		"sub":   newSubPRepCount,
		"extra": newExtraMainPRepCount,
	}
	rcpt, err = sim.GoBySetPRepCountConfig(csi, env.Governance(), counts)
	assert.NoError(t, err)
	CheckReceiptSuccess(rcpt)

	assert.NoError(t, sim.GoToTermEnd(csi))

	// T(1)
	jso, err = sim.GetPRepCountConfig()
	assert.NoError(t, err)
	assert.Equal(t, newMainPRepCount, jso["main"].(int64))
	assert.Equal(t, newSubPRepCount, jso["sub"].(int64))
	assert.Equal(t, newExtraMainPRepCount, jso["extra"].(int64))
	assert.Equal(t, len(sim.ValidatorList()), int(newMainPRepCount+newExtraMainPRepCount))

	mainPReps := sim.GetPReps(icstate.GradeMain)
	assert.Equal(t, int(newMainPRepCount+newExtraMainPRepCount), len(mainPReps))
	subPReps := sim.GetPReps(icstate.GradeSub)
	assert.Equal(t, int(newSubPRepCount-newExtraMainPRepCount), len(subPReps))

	term := sim.TermSnapshot()
	for i := 0; i < term.GetPRepSnapshotCount(); i++ {
		prepSnapshot := term.GetPRepSnapshotByIndex(i)
		prep := sim.GetPRep(prepSnapshot.Owner())
		if i < int(newMainPRepCount+newExtraMainPRepCount) {
			assert.Equal(t, icstate.GradeMain, prep.Grade())
		} else {
			assert.Equal(t, icstate.GradeSub, prep.Grade())
		}
	}
}