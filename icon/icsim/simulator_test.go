/*
 * Copyright 2021 ICON Foundation
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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

func checkValidatorList(vl0, vl1 []module.Validator) bool {
	if len(vl0) != len(vl1) {
		return false
	}
	for i := 0; i < len(vl0); i++ {
		if !vl0[i].Address().Equal(vl1[i].Address()) {
			return false
		}
	}
	return true
}

func estimateSlashed(slashRate icmodule.Rate, oldBonded *big.Int) *big.Int {
	return slashRate.MulBigInt(oldBonded)
}

func assertPower(t *testing.T, p map[string]interface{}) bool {
	var ok bool
	var power *big.Int

	_, ok = p["bondedDelegation"]
	assert.False(t, ok)
	power, ok = p["power"].(*big.Int)
	assert.True(t, ok)
	assert.True(t, power.Sign() >= 0)
	return true
}

func TestSimulator_CandidateIsPenalized(t *testing.T) {
	const (
		termPeriod                           = 100
		mainPRepCount                        = 22
		validationPenaltyCondition           = 5
		consistentValidationPenaltyCondition = 3
	)

	var err error
	var voted []bool
	var blockHeight int64
	var csi module.ConsensusInfo
	var vl []module.Validator

	c := NewSimConfig()
	c.MainPRepCount = mainPRepCount
	c.TermPeriod = termPeriod
	c.ValidationPenaltyCondition = validationPenaltyCondition
	c.ConsistentValidationPenaltyCondition = consistentValidationPenaltyCondition

	voted = make([]bool, mainPRepCount)
	for i := 0; i < mainPRepCount; i++ {
		voted[i] = true
	}

	// Decentralization is activated
	env, err := NewEnv(c, icmodule.Revision13)
	assert.NoError(t, err)
	sim := env.Simulator()

	// Term

	// T(0)
	// Skip 2 blocks after decentralization
	assert.NoError(t, sim.Go(nil, 2))

	// T(2)
	// prep0 gets penalized and prep22 will become a new main prep instead of prep0
	vl = sim.ValidatorList()
	voted[0] = false
	csi = NewConsensusInfo(sim.Database(), vl, voted)
	blocks := int64(validationPenaltyCondition)
	assert.NoError(t, sim.Go(csi, blocks))

	// T(7)
	blockHeight = sim.BlockHeight()
	for i := 1; i < mainPRepCount; i++ {
		prep := sim.GetPRep(vl[i].Address())
		assert.Equal(t, blocks, prep.GetVTotal(blockHeight))
		assert.Zero(t, prep.GetVFail(blockHeight))
		assert.Zero(t, prep.GetVFailCont(blockHeight))
	}
	prep := sim.GetPRep(env.preps[0])
	assert.Equal(t, blocks, prep.GetVTotal(blockHeight))
	assert.Equal(t, blocks, prep.GetVFail(blockHeight))
	assert.Equal(t, int64(0), prep.GetVFailCont(blockHeight))

	// Main PRep change: env.preps[0] -> env.preps[22]
	vl = sim.ValidatorList()
	assert.True(t, vl[0].Address().Equal(env.preps[22]))

	voted[0] = true
	csi = NewConsensusInfo(sim.Database(), vl, voted)
	term := sim.TermSnapshot()
	assert.NoError(t, sim.GoTo(csi, term.GetEndHeight()-3))

	// T(7) -> T(99)
	voted[0] = false
	csi = NewConsensusInfo(sim.Database(), vl, voted)
	assert.NoError(t, sim.GoToTermEnd(csi))

	blockHeight = sim.BlockHeight()
	prep = sim.GetPRep(vl[0].Address())
	assert.Equal(t, int64(3), prep.GetVFail(blockHeight))
	assert.Equal(t, int64(3), prep.GetVFailCont(blockHeight))

	// Term start

	// T(0) -> T(1)
	// ValidatorList is reverted to the initial list
	voted[0] = false
	csi = NewConsensusInfo(sim.Database(), vl, voted)
	err = sim.Go(csi, 2)

	// T(2)
	blockHeight = sim.BlockHeight()
	prep = sim.GetPRep(env.preps[22])
	assert.Equal(t, 1, prep.GetVPenaltyCount())
	assert.Equal(t, icstate.GradeCandidate, prep.Grade())
	assert.Equal(t, int64(93+2), prep.GetVTotal(blockHeight))
	assert.Equal(t, int64(5), prep.GetVFail(blockHeight))
	assert.Equal(t, int64(0), prep.GetVFailCont(blockHeight))
	assert.Zero(t, prep.GetVFailCont(blockHeight))

	vl = sim.ValidatorList()
	for i := 0; i < mainPRepCount; i++ {
		assert.True(t, vl[i].Address().Equal(env.preps[i]))
	}

	voted[0] = false
	csi = NewConsensusInfo(sim.Database(), vl, voted)
	err = sim.Go(csi, 5)
	assert.NoError(t, err)
}

func TestSimulator_SlashIsDisabledOnRev13AndEnabledOnRev14(t *testing.T) {
	const (
		termPeriod                           = 100
		mainPRepCount                        = 22
		validationPenaltyCondition           = 5
		consistentValidationPenaltyCondition = 3
	)

	var err error
	var voted []bool
	var csi module.ConsensusInfo
	var vl []module.Validator
	var env *Env
	var receipts []Receipt
	var oldBonded, bonded, slashed *big.Int
	var slashRate = icmodule.ToRate(5) // 5%
	var tx Transaction

	c := NewSimConfig()
	c.MainPRepCount = mainPRepCount
	c.TermPeriod = termPeriod
	c.ValidationPenaltyCondition = validationPenaltyCondition
	c.ConsistentValidationPenaltyCondition = consistentValidationPenaltyCondition
	c.ConsistentValidationPenaltySlashRate = slashRate

	voted = make([]bool, mainPRepCount)
	for i := 0; i < len(voted); i++ {
		voted[i] = true
	}

	// Decentralization is activated
	env, err = NewEnv(c, icmodule.Revision13)
	assert.NoError(t, err)
	sim := env.Simulator()

	// Term
	prep0 := env.preps[0]
	prep22 := env.preps[mainPRepCount]

	// env.users[0] transfers 2000 ICX to env.preps[0]
	amount := icutils.ToLoop(2000)
	rcpt, err := sim.GoByTransfer(nil, env.users[0], prep0, amount)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(rcpt))
	assert.Zero(t, amount.Cmp(sim.GetBalance(prep0)))

	// prep0 stakes 2000 ICX to bond to itself
	// Increase the bonds of env.preps[0] by 2000 ICX (others have 2000 ICX)
	prep := sim.GetPRep(prep0)
	oldBonded = prep.Bonded()

	receipts, err = sim.GoByTransaction(
		nil,
		sim.SetStake(prep0, amount),
		sim.SetBond(prep0, icstate.Bonds{icstate.NewBond(common.AddressToPtr(prep0), amount)}),
	)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(receipts...))

	newBonded := sim.GetPRep(prep0).Bonded()
	assert.Zero(t, newBonded.Cmp(new(big.Int).Add(oldBonded, amount)))

	// Skip a term to ignore exceptions at the 1st term after decentralization
	assert.NoError(t, sim.GoToTermEnd(nil))

	// Term

	for i := 0; i < consistentValidationPenaltyCondition; i++ {
		// PenaltyCount is reset to 0 at the beginning of every term on rev 13
		prep = sim.GetPRep(prep0)
		assert.Equal(t, icstate.GradeMain, prep.Grade())
		assert.Zero(t, prep.GetVPenaltyCount())

		// 1st validator does not vote for 5 consecutive blocks
		vl = sim.ValidatorList()
		voted[0] = false
		csi = NewConsensusInfo(sim.Database(), vl, voted)
		assert.NoError(t, sim.Go(csi, validationPenaltyCondition))

		// Check if 1st validator got penalized after 5 blocks
		prep = sim.GetPRep(vl[0].Address())
		assert.Equal(t, icstate.GradeCandidate, prep.Grade())
		assert.Equal(t, 1, prep.GetVPenaltyCount())

		// Check if prep22 acts as a validator instead of prep0
		// prep22 was a sub prep before prep0 got penalized
		vl = sim.ValidatorList()
		assert.True(t, prep22.Equal(vl[0].Address()))
		voted[0] = true
		csi = NewConsensusInfo(sim.Database(), vl, voted)
		assert.NoError(t, sim.GoToTermEnd(csi))
	}

	// Next Term

	// T(0)
	vl = sim.ValidatorList()
	assert.True(t, prep0.Equal(vl[0].Address()))
	prep = sim.GetPRep(vl[0].Address())
	assert.Equal(t, icstate.GradeMain, prep.Grade())

	// Set 14 to revision
	voted[0] = true
	csi = NewConsensusInfo(sim.Database(), vl, voted)
	tx = sim.SetRevision(env.Governance(), icmodule.Revision14)
	receipts, err = sim.GoByTransaction(csi, tx)
	assert.True(t, checkReceipts(receipts))
	assert.NoError(t, err)

	// go to Term.revision == 14
	assert.NoError(t, sim.GoToTermEnd(nil))

	// Next Term on Rev14

	prep = sim.GetPRep(prep0)
	oldBonded = prep.Bonded()
	oldTotalBond := sim.TotalBond()
	oldTotalStake := sim.TotalStake()
	oldTotalSupply := sim.TotalSupply()

	for i := 0; i < consistentValidationPenaltyCondition; i++ {
		// PenaltyCount is not reset after revision is 14
		prep = sim.GetPRep(prep0)
		assert.Equal(t, icstate.GradeMain, prep.Grade())
		assert.Equal(t, i, prep.GetVPenaltyCount())

		// Create a scenario when prep0 fails to vote for 5 blocks to validate
		vl = sim.ValidatorList()
		voted[0] = false
		csi = NewConsensusInfo(sim.Database(), vl, voted)
		assert.NoError(t, sim.Go(csi, validationPenaltyCondition))

		// Check if prep0 got penalized after 5 blocks
		prep = sim.GetPRep(vl[0].Address())
		assert.True(t, prep.Owner().Equal(vl[0].Address()))
		assert.Equal(t, icstate.GradeCandidate, prep.Grade())
		assert.Equal(t, i+1, prep.GetVPenaltyCount())

		// Check if prep22 acts as a validator instead of prep0
		// prep22 was a sub prep before prep0 got penalized
		vl = sim.ValidatorList()
		assert.True(t, prep22.Equal(vl[0].Address()))
		prep = sim.GetPRep(vl[0].Address())
		assert.Equal(t, icstate.GradeMain, prep.Grade())
		voted[0] = true
		csi = NewConsensusInfo(sim.Database(), vl, voted)
		assert.NoError(t, sim.GoToTermEnd(csi))
	}

	// Check if the bond of prep0 is slashed by default rate
	// Slashed bond will be burned
	prep = sim.GetPRep(prep0)
	bonded = prep.Bonded()
	slashed = estimateSlashed(slashRate, oldBonded)
	assert.Zero(t, bonded.Cmp(new(big.Int).Sub(oldBonded, slashed)))

	// Check if totalBond is reduced by slashed amount
	totalBond := sim.TotalBond()
	assert.Zero(t, totalBond.Cmp(new(big.Int).Sub(oldTotalBond, slashed)))

	// Check if totalStake is reduced by slashed amount
	totalStake := sim.TotalStake()
	assert.Zero(t, totalStake.Cmp(new(big.Int).Sub(oldTotalStake, slashed)))

	// Check if totalSupply is reduced by slashed amount
	totalSupply := sim.TotalSupply()
	assert.Zero(t, totalSupply.Cmp(new(big.Int).Sub(oldTotalSupply, slashed)))

	vl = sim.ValidatorList()
	assert.True(t, vl[0].Address().Equal(prep0))

	// Case: prep0 has already been penalized 3 times.
	// From now on, the bond of prep0 will be slashed every penalty
	for i := 0; i < 3; i++ {
		prep = sim.GetPRep(prep0)
		oldBonded = prep.Bonded()
		penaltyCount := consistentValidationPenaltyCondition + i
		assert.Equal(t, icstate.GradeMain, prep.Grade())
		assert.Equal(t, penaltyCount, prep.GetVPenaltyCount())

		// Make the case when prep0 fails to vote for blocks to validate
		voted[0] = false
		csi, err = sim.NewConsensusInfo(voted)
		assert.NoError(t, err)
		assert.NoError(t, sim.Go(csi, validationPenaltyCondition))

		// Check if prep0 was slashed after 5 blocks
		penaltyCount++
		prep = sim.GetPRep(env.preps[0])
		assert.Equal(t, icstate.GradeCandidate, prep.Grade())
		assert.Equal(t, penaltyCount, prep.GetVPenaltyCount())
		slashed = estimateSlashed(slashRate, oldBonded)
		assert.Zero(t, prep.Bonded().Cmp(new(big.Int).Sub(oldBonded, slashed)))

		// Check if prep22 acts as a validator instead of prep0
		// prep22 was a sub prep before prep0 got penalized
		vl = sim.ValidatorList()
		assert.True(t, prep22.Equal(vl[0].Address()))
		voted[0] = true
		csi, err = sim.NewConsensusInfo(voted)
		assert.NoError(t, err)
		assert.NoError(t, sim.GoToTermEnd(csi))
	}

	// Case: Accumulated penaltyCount will be reset to 0 after 30 terms when prep0 acts as a main prep
	vl = sim.ValidatorList()
	assert.True(t, prep0.Equal(vl[0].Address()))

	for i := 0; i < 23; i++ {
		prep = sim.GetPRep(prep0)
		assert.Equal(t, 6, prep.GetVPenaltyCount())

		csi = sim.NewDefaultConsensusInfo()
		assert.NoError(t, sim.GoToTermEnd(csi))
	}

	for i := 0; i < 6; i++ {
		prep = sim.GetPRep(prep0)
		assert.Equal(t, 6-i, prep.GetVPenaltyCount())

		csi = sim.NewDefaultConsensusInfo()
		assert.NoError(t, sim.GoToTermEnd(csi))
	}

	prep = sim.GetPRep(prep0)
	assert.Zero(t, prep.GetVPenaltyCount())
}

func TestSimulator_CheckIfVFailContWorks(t *testing.T) {
	const (
		termPeriod                           = 100
		mainPRepCount                        = 22
		validationPenaltyCondition           = 5
		consistentValidationPenaltyCondition = 3
	)

	var err error
	var voted []bool
	var csi module.ConsensusInfo
	var vl []module.Validator
	var env *Env
	var prep *icstate.PRep
	//var receipts []Receipt
	//var oldBonded, bonded, slashed *big.Int

	c := NewSimConfig()
	c.MainPRepCount = mainPRepCount
	c.ExtraMainPRepCount = int64(0)
	c.TermPeriod = termPeriod
	c.ValidationPenaltyCondition = validationPenaltyCondition
	c.ConsistentValidationPenaltyCondition = consistentValidationPenaltyCondition

	voted = make([]bool, mainPRepCount)
	for i := 0; i < len(voted); i++ {
		voted[i] = true
	}

	// Decentralization is activated
	env, err = NewEnv(c, icmodule.Revision13)
	assert.NoError(t, err)
	sim := env.Simulator()

	vl0 := make([]module.Validator, mainPRepCount)
	vl1 := make([]module.Validator, mainPRepCount)
	vl2 := make([]module.Validator, mainPRepCount)
	for i := 0; i < mainPRepCount; i++ {
		vl0[i], _ = state.ValidatorFromAddress(env.preps[i])
		vl1[i], _ = state.ValidatorFromAddress(env.preps[i])
		vl2[i], _ = state.ValidatorFromAddress(env.preps[i])
	}
	vl1[0], _ = state.ValidatorFromAddress(env.preps[22])
	vl2[0], _ = state.ValidatorFromAddress(env.preps[23])

	// Skip the first term after decentralization
	err = sim.GoToTermEnd(nil)
	assert.NoError(t, err)

	// term 1
	voted[0] = false
	csi = NewConsensusInfo(sim.Database(), vl0, voted)
	err = sim.Go(csi, validationPenaltyCondition)
	assert.NoError(t, err)

	// Check if 1st validator got penalized after 5 blocks
	prep = sim.GetPRep(env.preps[0])
	assert.Equal(t, icstate.GradeCandidate, prep.Grade())
	assert.Equal(t, 1, prep.GetVPenaltyCount())

	// Check if prep22 is newly included in validatorList instead of prep0
	// Go ahead until term end
	vl = sim.ValidatorList()
	assert.True(t, checkValidatorList(vl, vl1))
	voted[0] = true
	csi = NewConsensusInfo(sim.Database(), vl, voted)
	err = sim.GoToTermEnd(csi)
	assert.NoError(t, err)

	// term 2
	// prep0 -> main prep, prep21 -> sub prep
	// The first 2 consensus info follows the prev term validator list
	// prep22 fails to vote for the first 2 blocks of this term
	voted[0] = false
	csi = NewConsensusInfo(sim.Database(), vl1, voted)
	err = sim.Go(csi, 2)
	assert.NoError(t, err)

	// go ahead until term end without any false votes
	vl = sim.ValidatorList()
	assert.True(t, checkValidatorList(vl0, vl))
	voted[0] = true
	csi = NewConsensusInfo(sim.Database(), vl, voted)
	err = sim.GoToTermEnd(csi)
	assert.NoError(t, err)

	// prep0 fails to vote for 7 consecutive blocks and gets penalized
	voted[0] = false
	csi = NewConsensusInfo(sim.Database(), vl0, voted)
	err = sim.Go(csi, validationPenaltyCondition)
	assert.NoError(t, err)
	// prep0: mainPRep -> candidate, prep22: subPRep -> mainPRep
	vl = sim.ValidatorList()
	assert.True(t, checkValidatorList(vl1, vl))
	prep = sim.GetPRep(env.preps[0])
	assert.Equal(t, icstate.GradeCandidate, prep.Grade())

	// prep0 fails to vote for 2 blocks, getting penalized
	csi = NewConsensusInfo(sim.Database(), vl0, voted)
	err = sim.Go(csi, 2)
	assert.NoError(t, err)

	// Check if prep22 becomes a main prep instead of prep0
	prep = sim.GetPRep(env.preps[22])
	assert.Equal(t, int64(2), prep.GetVFailCont(sim.BlockHeight()))
	assert.Equal(t, icstate.GradeMain, prep.Grade())
	assert.Zero(t, prep.GetVPenaltyCount())
	vl = sim.ValidatorList()
	assert.True(t, checkValidatorList(vl, vl1))

	voted[0] = false
	csi = NewConsensusInfo(sim.Database(), vl1, voted)
	err = sim.Go(csi, 3)
	assert.NoError(t, err)

	// prep21 got penalized and its penaltyCount is set to 1
	prep = sim.GetPRep(env.preps[22])
	assert.Equal(t, 1, prep.GetVPenaltyCount())
	// prep0: candidate, prep21: mainPRep -> candidate, prep22: subPRep -> mainPRep
	vl = sim.ValidatorList()
	assert.True(t, checkValidatorList(vl, vl2))

	// Create 2 blocks
	voted[0] = false
	csi = NewConsensusInfo(sim.Database(), vl1, voted)
	err = sim.Go(csi, 2)
	assert.NoError(t, err)

	// prep0 and prep21 have got penalized and their grade is set to candidate
	// prep22 will be the new main prep
	prep = sim.GetPRep(env.preps[23])
	assert.Zero(t, prep.GetVPenaltyCount())
	assert.Zero(t, prep.GetVTotal(sim.BlockHeight()))
}

func TestSimulator_PenalizeMultiplePReps(t *testing.T) {
	const (
		termPeriod                           = 100
		mainPRepCount                        = 22
		validationPenaltyCondition           = 5
		consistentValidationPenaltyCondition = 3
	)

	var err error
	var voted []bool
	var csi module.ConsensusInfo
	var vl []module.Validator
	var env *Env
	var prep *icstate.PRep
	var blockHeight int64

	c := NewSimConfig()
	c.MainPRepCount = mainPRepCount
	c.TermPeriod = termPeriod
	c.ValidationPenaltyCondition = validationPenaltyCondition
	c.ConsistentValidationPenaltyCondition = consistentValidationPenaltyCondition

	voted = make([]bool, mainPRepCount)
	for i := 0; i < len(voted); i++ {
		voted[i] = true
	}

	// Decentralization is activated
	env, err = NewEnv(c, icmodule.Revision13)
	assert.NoError(t, err)
	sim := env.Simulator()

	vl0 := make([]module.Validator, mainPRepCount)
	vl1 := make([]module.Validator, mainPRepCount)
	for i := 0; i < mainPRepCount; i++ {
		vl0[i], _ = state.ValidatorFromAddress(env.preps[i])
		vl1[i], _ = state.ValidatorFromAddress(env.preps[i])
	}
	vl1[1], _ = state.ValidatorFromAddress(env.preps[22])
	vl1[2], _ = state.ValidatorFromAddress(env.preps[23])

	// Skip the first term after decentralization
	err = sim.GoToTermEnd(nil)
	assert.NoError(t, err)

	// term 1
	voted[1] = false // prep1
	voted[2] = false // prep2
	csi = NewConsensusInfo(sim.Database(), vl0, voted)
	err = sim.Go(csi, validationPenaltyCondition)
	assert.NoError(t, err)

	// Check if prep1 and prep2 got penalized after 5 blocks
	for i := 1; i <= 2; i++ {
		prep = sim.GetPRep(env.preps[i])
		assert.Equal(t, icstate.GradeCandidate, prep.Grade())
		assert.Equal(t, 1, prep.GetVPenaltyCount())
	}
	// Check if prep22 and prep23 become main preps
	for i := 22; i <= 23; i++ {
		prep = sim.GetPRep(env.preps[i])
		assert.Equal(t, icstate.GradeMain, prep.Grade())
		assert.Equal(t, 0, prep.GetVPenaltyCount())
	}

	// Go ahead 2 blocks to simulate on the next mechanism
	err = sim.Go(csi, 2)
	assert.NoError(t, err)

	blockHeight = sim.BlockHeight()
	// Check the states of prep1 and prep2 after additional 2 blocks
	for i := 1; i <= 2; i++ {
		prep = sim.GetPRep(env.preps[i])
		assert.Equal(t, icstate.GradeCandidate, prep.Grade())
		assert.Equal(t, 1, prep.GetVPenaltyCount())
		assert.Equal(t, int64(7), prep.GetVTotal(blockHeight))
		assert.Equal(t, int64(7), prep.GetVFail(blockHeight))
		// VFailCont is reset to 0 when the validator gets penalized
		assert.Equal(t, int64(2), prep.GetVFailCont(blockHeight))
	}
	// Check if prep22 and prep23 become main preps
	for i := 22; i <= 23; i++ {
		prep = sim.GetPRep(env.preps[i])
		assert.Equal(t, icstate.GradeMain, prep.Grade())
		assert.Zero(t, prep.GetVPenaltyCount())
		assert.Zero(t, prep.GetVTotal(blockHeight))
		assert.Zero(t, prep.GetVFail(blockHeight))
		assert.Zero(t, prep.GetVFailCont(blockHeight))
	}

	// Check if prep22 and prep23 are newly included in validatorList instead of prep0 and prep01
	// Go ahead until term end
	vl = sim.ValidatorList()
	assert.True(t, checkValidatorList(vl, vl1))
	voted[1] = true
	voted[2] = true
	csi = NewConsensusInfo(sim.Database(), vl, voted)
	err = sim.GoToTermEnd(csi)
	assert.NoError(t, err)

	// term 2
	vl = sim.ValidatorList()
	assert.True(t, checkValidatorList(vl, vl0))

	blockHeight = sim.BlockHeight()
	// Check if prep1 and prep2 return to main preps
	for i := 1; i <= 2; i++ {
		prep = sim.GetPRep(env.preps[i])
		assert.Equal(t, icstate.GradeMain, prep.Grade())
		// PenaltyCount is reset to 0 when a prep becomes a main prep on rev13
		assert.Equal(t, 0, prep.GetVPenaltyCount())
		assert.Equal(t, int64(7), prep.GetVTotal(blockHeight))
		assert.Equal(t, int64(7), prep.GetVFail(blockHeight))
	}
	// Check if prep22 and prep23 return to sub preps
	for i := 22; i <= 23; i++ {
		prep = sim.GetPRep(env.preps[i])
		assert.Equal(t, icstate.GradeSub, prep.Grade())
		assert.Equal(t, 0, prep.GetVPenaltyCount())
		assert.Equal(t, int64(termPeriod-7), prep.GetVTotal(blockHeight))
		assert.Zero(t, prep.GetVFail(blockHeight))
		assert.Zero(t, prep.GetVFailCont(blockHeight))
	}
}

func TestSimulator_ReplaceBondedDelegationWithPower(t *testing.T) {
	const (
		termPeriod                           = 100
		mainPRepCount                        = 22
		validationPenaltyCondition           = 5
		consistentValidationPenaltyCondition = 3
	)

	var prep *icstate.PRep
	var jso map[string]interface{}
	var ok bool

	c := NewSimConfig()
	c.MainPRepCount = mainPRepCount
	c.TermPeriod = termPeriod
	c.ValidationPenaltyCondition = validationPenaltyCondition
	c.ConsistentValidationPenaltyCondition = consistentValidationPenaltyCondition

	// Decentralization is activated
	env, err := NewEnv(c, icmodule.Revision13)
	assert.NoError(t, err)
	sim := env.Simulator()

	address := env.preps[0]

	// Check getPRep
	sc := sim.GetStateContext()
	prep = sim.GetPRep(address)
	jso = prep.ToJSON(sc)
	assertPower(t, jso)

	// Check getPReps
	jso = sim.GetPRepsInJSON()
	_, ok = jso["totalBondedDelegated"]
	assert.False(t, ok)
	preps := jso["preps"].([]interface{})
	for i := range preps {
		assertPower(t, preps[i].(map[string]interface{}))
	}

	// Check getMainPReps
	jso = sim.GetMainPRepsInJSON()
	_, ok = jso["totalPower"].(*big.Int)
	assert.True(t, ok)
	preps = jso["preps"].([]interface{})
	for i := range preps {
		assertPower(t, preps[i].(map[string]interface{}))
	}

	// Check getSubPReps
	jso = sim.GetSubPRepsInJSON()
	_, ok = jso["totalPower"].(*big.Int)
	assert.True(t, ok)
	preps = jso["preps"].([]interface{})
	for i := range preps {
		assertPower(t, preps[i].(map[string]interface{}))
	}

	// Check getPRepTerm
	jso = sim.GetPRepTermInJSON()
	_, ok = jso["totalPower"].(*big.Int)
	assert.True(t, ok)
	preps = jso["preps"].([]interface{})
	for i := range preps {
		assertPower(t, preps[i].(map[string]interface{}))
	}

	// Check getNetworkInfo
	jso = sim.GetNetworkInfoInJSON()
	_, ok = jso["totalPower"].(*big.Int)
	assert.True(t, ok)
}
