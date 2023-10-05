package icsim

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
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
		bondedPRepCount                      = newMainPRepCount
	)

	var err error
	var voted []bool
	//var blockHeight int64
	var csi module.ConsensusInfo
	var vl []module.Validator
	br := icmodule.ToRate(5) // 5%
	//var prep *icstate.PRepSet

	c := NewConfig()
	c.MainPRepCount = mainPRepCount
	c.ExtraMainPRepCount = extraMainPRepCount
	c.BondedPRepCount = bondedPRepCount
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

	// Set revision to 17 to activate extra main preps
	tx := sim.SetRevision(icmodule.RevisionExtraMainPReps)
	_, err = sim.GoByTransaction(csi, tx)
	assert.NoError(t, err)
	err = sim.GoToTermEnd(csi)
	assert.NoError(t, err)

	vl = sim.ValidatorList()
	assert.Len(t, vl, newMainPRepCount)

	emptyBonds := make([]*icstate.Bond, 0, 0)
	for i := 0; i < 3; i++ {
		tx = sim.SetBond(env.bonders[i], emptyBonds)
		_, err = sim.GoByTransaction(csi, tx)
		assert.NoError(t, err)

		err = sim.GoToTermEnd(csi)
		assert.NoError(t, err)

		// All validators (= main preps) should have 1 or more bonded
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
		assert.Equal(t, bondedPReps, newMainPRepCount-i-1)
	}
}

func Test_PreventZeroPowerExtraMainPReps(t *testing.T) {
	const (
		termPeriod                           = 100
		mainPRepCount                        = 22
		validationPenaltyCondition           = 5
		consistentValidationPenaltyCondition = 3
		extraMainPRepCount                   = 3
		newMainPRepCount                     = mainPRepCount + extraMainPRepCount
		bondedPRepCount                      = mainPRepCount + 2
	)

	var err error
	var voted []bool
	//var blockHeight int64
	var csi module.ConsensusInfo
	var vl []module.Validator
	//var prep *icstate.PRepSet

	c := NewConfig()
	c.MainPRepCount = mainPRepCount
	c.ExtraMainPRepCount = extraMainPRepCount
	c.TermPeriod = termPeriod
	c.ValidationPenaltyCondition = validationPenaltyCondition
	c.ConsistentValidationPenaltyCondition = consistentValidationPenaltyCondition
	c.BondedPRepCount = bondedPRepCount

	// size: 22, cap: 25
	voted = make([]bool, mainPRepCount, newMainPRepCount)
	for i := 0; i < mainPRepCount; i++ {
		voted[i] = true
	}

	// Decentralization is activated
	env := initEnv(t, c, icmodule.Revision13)
	sim := env.sim

	// Set revision to 17 to activate extra main preps
	tx := sim.SetRevision(icmodule.RevisionExtraMainPReps)
	_, err = sim.GoByTransaction(csi, tx)
	assert.NoError(t, err)
	err = sim.GoToTermEnd(csi)
	assert.NoError(t, err)

	vl = sim.ValidatorList()
	assert.Len(t, vl, bondedPRepCount)
}
