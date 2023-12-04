package icsim

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

func Test_ExtraMainPReps(t *testing.T) {
	const (
		termPeriod                           = 100
		mainPRepCount                        = 22
		validationPenaltyCondition           = 5
		consistentValidationPenaltyCondition = 3
		extraMainPRepCount                   = 3
		newMainPRepCount                     = mainPRepCount + extraMainPRepCount
	)

	var err error
	var csi module.ConsensusInfo
	var vl []module.Validator
	br := icmodule.ToRate(5) // 5%

	c := NewSimConfigWithParams(map[SimConfigOption]interface{}{
		SCOMainPReps:                         int64(mainPRepCount),
		SCOExtraMainPReps:                    int64(extraMainPRepCount),
		SCOTermPeriod:                        int64(termPeriod),
		SCOValidationFailurePenaltyCondition: int64(validationPenaltyCondition),
		SCOAccumulatedValidationFailurePenaltyCondition: int64(consistentValidationPenaltyCondition),
		SCOBondRequirement: icmodule.ToRate(5),
	})

	// Decentralization is activated
	env, err := NewEnv(c, icmodule.Revision13)
	assert.NoError(t, err)
	sim := env.sim
	csi = sim.NewDefaultConsensusInfo()

	// Set revision to 17 to activate extra main preps
	rcpt, err := sim.GoBySetRevision(csi, env.Governance(), icmodule.RevisionExtraMainPReps)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(rcpt))
	err = sim.GoToTermEnd(csi)
	assert.NoError(t, err)

	vl = sim.ValidatorList()
	assert.Len(t, vl, newMainPRepCount)
	csi = sim.NewDefaultConsensusInfo()

	bonders := make(map[string]bool)

	// Make 75 out of 100 PReps have no bond
	emptyBonds := make([]*icstate.Bond, 0)
	size := len(env.bonders) - newMainPRepCount
	if size < 0 {
		size = 0
	}
	for i := 0; i < size; i++ {
		bonder := env.bonders[i]
		rcpt, err = sim.GoBySetBond(csi, bonder, emptyBonds)
		assert.NoError(t, err)
		assert.True(t, CheckReceiptSuccess(rcpt))

		assert.False(t, bonders[icutils.ToKey(bonder)])
		bonders[icutils.ToKey(bonder)] = true
	}
	assert.NoError(t, sim.GoToTermEnd(csi))

	for i := 0; i < 3; i++ {
		mainPReps := sim.GetPReps(icstate.GradeMain)
		assert.Equal(t, newMainPRepCount-i, len(mainPReps))
		owner := mainPReps[0].Owner()
		bonderList := sim.GetBonderList(owner)

		assert.False(t, bonders[icutils.ToKey(bonderList[0])])
		rcpt, err = sim.GoBySetBond(csi, bonderList[0], emptyBonds)
		assert.NoError(t, err)
		assert.True(t, CheckReceiptSuccess(rcpt))

		// Move to the next term
		assert.NoError(t, sim.GoToTermEnd(csi))

		// The bonds of all validators (= main preps) should be larger than 0
		vl = sim.ValidatorList()
		assert.Len(t, vl, newMainPRepCount-i-1)
		for _, v := range vl {
			prep := sim.GetPRep(v.Address())
			assert.True(t, prep.Bonded().Sign() > 0)
			assert.True(t, prep.GetPower(br).Sign() > 0)
		}

		bondedPReps := 0
		for _, address := range env.preps {
			prep := sim.GetPRep(address)
			if prep.Bonded().Sign() > 0 {
				bondedPReps++
			}
		}
		assert.Equal(t, newMainPRepCount-i-1, bondedPReps)
	}
}