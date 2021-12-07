package icsim

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/icmodule"
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
	var voted []bool
	//var blockHeight int64
	var csi module.ConsensusInfo
	var vl []module.Validator
	//var prep *icstate.PRepSet

	c := NewConfig()
	c.MainPRepCount = mainPRepCount
	c.TermPeriod = termPeriod
	c.ValidationPenaltyCondition = validationPenaltyCondition
	c.ConsistentValidationPenaltyCondition = consistentValidationPenaltyCondition

	// size: 22, cap: 25
	voted = make([]bool, mainPRepCount, newMainPRepCount)
	for i := 0; i < mainPRepCount; i++ {
		voted[i] = true
	}

	// Decentralization is activated
	env := initEnv(t, c, icmodule.Revision13)
	sim := env.sim

	// Set revision to 16 to activate extra main preps
	tx := sim.SetRevision(icmodule.RevisionExtraMainPReps)
	_, err = sim.GoByTransaction(tx, csi)
	assert.NoError(t, err)
	err = sim.GoToTermEnd(csi)
	assert.NoError(t, err)

	vl = sim.ValidatorList()
	assert.Len(t, vl, newMainPRepCount)
}
