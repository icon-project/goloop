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
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

func newConsensusInfo(dbase db.Database, vl []module.Validator, voted []bool) module.ConsensusInfo {
	vss, err := state.ValidatorSnapshotFromSlice(dbase, vl)
	if err != nil {
		return nil
	}
	v, _ := vss.Get(vss.Len() - 1)
	copiedVoted := make([]bool, vss.Len())
	copy(copiedVoted, voted)
	return common.NewConsensusInfo(v.Address(), vss, copiedVoted)
}

func initEnv(t *testing.T, c *config, revision module.Revision) *Env {
	var env *Env
	var err error
	mainPRepCount := int(c.MainPRepCount)

	// Decentralization is activated
	env, err = NewEnv(c, revision)
	assert.NoError(t, err)
	sim := env.sim

	// Check if decentralization is done
	vl := sim.ValidatorList()
	for i := 0; i < mainPRepCount; i++ {
		assert.True(t, env.preps[i].Equal(vl[i].Address()))
	}
	jso := sim.GetMainPReps()
	assert.Equal(t, mainPRepCount, len(jso["preps"].([]interface{})))

	blockHeight := sim.BlockHeight()
	for i := 0; i < len(env.preps); i++ {
		prep := sim.GetPRep(env.preps[i])
		if i < mainPRepCount {
			assert.Equal(t, icstate.GradeMain, prep.Grade())
		} else {
			assert.Equal(t, icstate.GradeSub, prep.Grade())
		}
		assert.Zero(t, prep.GetVTotal(blockHeight))
		assert.Zero(t, prep.GetVFail(blockHeight))
		assert.Zero(t, prep.GetVFailCont(blockHeight))
	}
	return env
}

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

/*
// Initial prep balance: 3000 ICX
// Initial user balance: 1000 ICX
func TestSimulator_init(t *testing.T) {
	//var err error
	var jso map[string]interface{}
	var block Block
	var tx Transaction
	var receipts []Receipt
	totalSupply := new(big.Int)
	const (
		initRevision = icmodule.Revision5
		validatorLen = 22
		userLen      = 3
		prepLen      = 30
	)

	userAddrs := make([]module.Address, userLen)
	for i := 0; i < userLen; i++ {
		userAddrs[i] = newDummyAddress(1000 + i)
	}

	initPRepBalance := icutils.ToLoop(2000)
	initUserBalance := icutils.ToLoop(1000)

	prepAddrs := make([]module.Address, prepLen)
	for i := 0; i < prepLen; i++ {
		prepAddrs[i] = newDummyAddress(i + 100)
	}

	validators := make([]module.Validator, validatorLen)
	for i := 0; i < validatorLen; i++ {
		validator, _ := state.ValidatorFromAddress(prepAddrs[i])
		validators[i] = validator
	}

	// Assign initial balances to preps and users
	balances := make(map[string]*big.Int)
	for _, address := range prepAddrs {
		balances[icutils.ToKey(address)] = initPRepBalance
		totalSupply.Add(totalSupply, initPRepBalance)
	}
	for _, address := range userAddrs {
		balances[icutils.ToKey(address)] = initUserBalance
		totalSupply.Add(totalSupply, initUserBalance)
	}

	// Create a Simulator
	c := NewConfig()
	c.TermPeriod = 100
	c.ValidationPenaltyCondition = 5
	c.ConsistentValidationPenaltyCondition = 3
	sim := NewSimulator(initRevision, validators, balances, c)
	assert.Equal(t, initRevision, sim.Revision().Value())
	assert.Zero(t, sim.BlockHeight())

	// Check initial balances and totalSupply
	for key, value := range balances {
		address := common.MustNewAddress([]byte(key))
		balance := sim.GetBalance(address)
		assert.Zero(t, value.Cmp(balance))
	}
	assert.Zero(t, totalSupply.Cmp(sim.TotalSupply()))

	// Check if no prep is registered
	jso = sim.GetPReps()
	assert.Zero(t, len(jso["preps"].([]interface{})))
	jso = sim.GetMainPReps()
	assert.Zero(t, len(jso["preps"].([]interface{})))
	jso = sim.GetSubPReps()
	assert.Zero(t, len(jso["preps"].([]interface{})))

	// Generate blocks until block height reaches to 100
	blockHeight := int64(100)
	err := sim.GoTo(blockHeight)
	assert.NoError(t, err)
	assert.Equal(t, blockHeight, sim.BlockHeight())

	// Add RegisterPRep transactions to a block
	block = NewBlock()
	for i, from := range prepAddrs {
		info := newDummyPRepInfo(i)
		tx := sim.RegisterPRep(from, info)
		block.AddTransaction(tx)
	}
	receipts, err = sim.GoByBlock(block)
	assert.NoError(t, err)
	assert.True(t, checkReceipts(receipts))
	blockHeight++
	assert.Equal(t, blockHeight, sim.BlockHeight())

	// Users set 1000 icx stakes
	block = NewBlock()
	stake := icutils.ToLoop(1000)
	for _, from := range userAddrs {
		tx := sim.SetStake(from, stake)
		block.AddTransaction(tx)
	}
	blockHeight = sim.BlockHeight()
	receipts, err = sim.GoByBlock(block)
	checkBlockResult(t, receipts, err)
	assert.Equal(t, blockHeight + 1, sim.BlockHeight())

	// Check if RegisterPRep transactions are executed
	sim.GetPReps()
	jso = sim.GetPReps()
	assert.Equal(t, len(prepAddrs), len(jso["preps"].([]interface{})))

	// Check if RegPRepFee(2000 ICX) is charged from prep balances
	for _, addr := range prepAddrs {
		balance := sim.GetBalance(addr)
		expected := new(big.Int).Sub(initPRepBalance, icmodule.BigIntRegPRepFee)
		assert.Zero(t, expected.Cmp(balance))
	}

	for _, addr := range userAddrs {
		jso = sim.GetStake(addr)
		stake2 := jso["stake"].(*big.Int)
		assert.Zero(t, stake.Cmp(stake2))
	}

	// Delegate 100 ICX to each PRep
	block = NewBlock()
	for j := 0; j < 3; j++ {
		user := userAddrs[j]
		ds := make([]*icstate.Delegation, 10)
		for i := 0; i < 10; i++ {
			prep := prepAddrs[j * 10 + i]
			amount := icutils.ToLoop(100)
			amount.Sub(amount, big.NewInt(int64(j * 10 + i)))
			ds[i] = icstate.NewDelegation(common.AddressToPtr(prep), amount)
		}
		block.AddTransaction(sim.SetDelegation(user, ds))
	}
	receipts, err = sim.GoByBlock(block)
	assert.NoError(t, err)
	assert.True(t, checkReceipts(receipts))
	blockHeight++
	assert.Equal(t, blockHeight, sim.BlockHeight())

	votingPower := int64(45)
	for _, from := range userAddrs {
		jso = sim.GetDelegation(from)
		assert.Equal(t, votingPower, jso["votingPower"].(*big.Int).Int64())
		votingPower += int64(100)
	}

	// Activate Decentralization
	tx = sim.SetRevision(icmodule.RevisionDecentralize)
	receipts, err = sim.GoByTransaction(tx)
	checkBlockResult(t, receipts, err)

	assert.NoError(t, sim.GoToTermEnd())
	jso = sim.GetPRepTerm()
	startBH := jso["startBlockHeight"].(int64)
	assert.Equal(t, startBH - 1, sim.BlockHeight())

	jso = sim.GetMainPReps()
	preps := jso["preps"].([]interface{})
	var prevTbd *big.Int
	assert.Equal(t, validatorLen, len(preps))
	for i, _ := range preps {
		prep := preps[i].(map[string]interface{})
		tbd := prep["delegated"].(*big.Int)
		if prevTbd == nil {
			prevTbd = tbd
		} else {
			assert.True(t, prevTbd.Cmp(tbd) > 0)
		}
	}

	// Set Revision to 13
	blockHeight = sim.BlockHeight()
	term := sim.TermSnapshot()
	tx = sim.SetRevision(icmodule.Revision13)
	receipts, err = sim.GoByTransaction(tx)
	checkBlockResult(t, receipts, err)
	assert.NoError(t, sim.GoToTermEnd())
	assert.Equal(t, icmodule.Revision13, sim.Revision().Value())
	assert.Equal(t, blockHeight + term.Period(), sim.BlockHeight())

	blockHeight = sim.BlockHeight()
	for _, prepAddr := range prepAddrs {
		prep := sim.GetPRep(prepAddr)
		assert.Zero(t, prep.GetVPenaltyCount())
		assert.Zero(t, prep.GetVFail(blockHeight))
		assert.Zero(t, prep.GetVTotal(blockHeight))
	}

	//assert.NoError(t, sim.Go(6))
	//jso = sim.GetPRep(prepAddrs[0])
	//assert.Equal(t, int(icstate.GradeCandidate), jso["grade"].(int))
	//assert.Equal(t, int64(icmodule.PenaltyBlockValidation), jso["penalty"].(int64))
	//assert.Equal(t, int(icstate.Active), jso["status"].(int))
	//assert.Equal(t, int64(0), jso["validatedBlocks"].(int64))
	//assert.Equal(t, int64(5), jso["totalBlocks"].(int64))
	//assert.NoError(t, sim.GoToTermEnd())
	//
	//for i := 0; i < 2; i++ {
	//	assert.NoError(t, sim.GoToTermEnd())
	//}
	//jso = sim.GetPRepStats(prepAddrs[0])
	//assert.Equal(t, int(icstate.GradeMain), jso["grade"].(int))
	//assert.Equal(t, 3, jso["penalties"].(int))
	//assert.Equal(t, int(icstate.Active), jso["status"].(int))

	//assert.Equal(t, int64(0), jso["validatedBlocks"].(int64))
	//assert.Equal(t, int64(15), jso["totalBlocks"].(int64))

	//for i := 0; i < 3; i++ {
	//	jso = sim.GetPRep(prep)
	//	assert.Equal(t, int(icstate.GradeMain), jso["grade"].(int))
	//	assert.Equal(t, int64(icmodule.PenaltyNone), jso["penalty"].(int64))
	//	assert.Equal(t, int(icstate.Active), jso["status"].(int))
	//	assert.Equal(t, jso["validatedBlocks"].(int64), jso["totalBlocks"].(int64))
	//}
}
 */

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
	var prep *icstate.PRep

	c := NewConfig()
	c.MainPRepCount = mainPRepCount
	c.TermPeriod = termPeriod
	c.ValidationPenaltyCondition = validationPenaltyCondition
	c.ConsistentValidationPenaltyCondition = consistentValidationPenaltyCondition

	voted = make([]bool, mainPRepCount)
	for i := 0; i < mainPRepCount; i++ {
		voted[i] = true
	}

	// Decentralization is activated
	env := initEnv(t, c, icmodule.Revision13)
	sim := env.sim

	// Term

	// prep0 gets penalized and prep22 will become a new main prep instead of prep0
	vl = sim.ValidatorList()
	voted[0] = false
	csi = newConsensusInfo(sim.Database(), vl, voted)
	err = sim.Go(5, csi)
	assert.NoError(t, err)

	blockHeight = sim.BlockHeight()
	for i := 1; i < mainPRepCount; i++ {
		prep = sim.GetPRep(vl[i].Address())
		assert.Equal(t, int64(5), prep.GetVTotal(blockHeight))
		assert.Zero(t, prep.GetVFail(blockHeight))
		assert.Zero(t, prep.GetVFailCont(blockHeight))
	}
	prep = sim.GetPRep(env.preps[0])
	assert.Equal(t, int64(5), prep.GetVTotal(blockHeight))
	assert.Equal(t, int64(5), prep.GetVFail(blockHeight))
	assert.Equal(t, int64(0), prep.GetVFailCont(blockHeight))

	// Main PRep change: env.preps[0] -> env.preps[22]
	vl = sim.ValidatorList()
	assert.True(t, vl[0].Address().Equal(env.preps[22]))

	voted[0] = true
	csi = newConsensusInfo(sim.Database(), vl, voted)
	err = sim.Go(c.TermPeriod - 5 - 3, csi)
	assert.NoError(t, err)

	voted[0] = false
	csi = newConsensusInfo(sim.Database(), vl, voted)
	err = sim.GoToTermEnd(csi)
	assert.NoError(t, err)

	blockHeight = sim.BlockHeight()
	prep = sim.GetPRep(vl[0].Address())
	assert.Equal(t, int64(3), prep.GetVFail(blockHeight))
	assert.Equal(t, int64(3), prep.GetVFailCont(blockHeight))

	// Term start

	// ValidatorList is reverted to the initial list
	voted[0] = false
	csi = newConsensusInfo(sim.Database(), vl, voted)
	err = sim.Go(2, csi)

	blockHeight = sim.BlockHeight()
	prep = sim.GetPRep(env.preps[22])
	assert.Equal(t, 1, prep.GetVPenaltyCount())
	assert.Equal(t, icstate.GradeCandidate, prep.Grade())
	assert.Equal(t, int64(95 + 2), prep.GetVTotal(blockHeight))
	assert.Equal(t, int64(5), prep.GetVFail(blockHeight))
	assert.Equal(t, int64(0), prep.GetVFailCont(blockHeight))
	assert.Zero(t, prep.GetVFailCont(blockHeight))

	vl = sim.ValidatorList()
	for i := 0; i < mainPRepCount; i++ {
		assert.True(t, vl[i].Address().Equal(env.preps[i]))
	}

	voted[0] = false
	csi = newConsensusInfo(sim.Database(), vl, voted)
	err = sim.Go(5, csi)
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
	var prep *icstate.PRep
	var receipts []Receipt
	var oldBonded, bonded, slashed *big.Int

	c := NewConfig()
	c.MainPRepCount = mainPRepCount
	c.TermPeriod = termPeriod
	c.ValidationPenaltyCondition = validationPenaltyCondition
	c.ConsistentValidationPenaltyCondition = consistentValidationPenaltyCondition

	voted = make([]bool, mainPRepCount)
	for i := 0; i < len(voted); i++ {
		voted[i] = true
	}

	// Decentralization is activated
	env = initEnv(t, c, icmodule.Revision13)
	sim := env.sim

	// Term

	for i := 0; i < consistentValidationPenaltyCondition; i++ {
		// PenaltyCount is reset to 0 at the beginning of every term on rev 13
		prep = sim.GetPRep(env.preps[0])
		assert.Equal(t, icstate.GradeMain, prep.Grade())
		assert.Zero(t, prep.GetVPenaltyCount())

		vl = sim.ValidatorList()
		voted[0] = false
		csi = newConsensusInfo(sim.Database(), vl, voted)
		err = sim.Go(validationPenaltyCondition, csi)
		assert.NoError(t, err)

		// Check if 1st validator got penalized after 5 blocks
		prep = sim.GetPRep(vl[0].Address())
		assert.Equal(t, icstate.GradeCandidate, prep.Grade())
		assert.Equal(t, 1, prep.GetVPenaltyCount())

		// Check if prep22 acts as a validator instead of prep0
		// prep22 was a sub prep before prep0 got penalized
		vl = sim.ValidatorList()
		assert.True(t, env.preps[mainPRepCount].Equal(vl[0].Address()))
		voted[0] = true
		csi = newConsensusInfo(sim.Database(), vl, voted)
		err = sim.GoToTermEnd(csi)
		assert.NoError(t, err)
	}

	// Set 14 to revision
	vl = sim.ValidatorList()
	assert.True(t, env.preps[0].Equal(vl[0].Address()))
	voted[0] = true
	csi = newConsensusInfo(sim.Database(), vl, voted)
	tx := sim.SetRevision(icmodule.RevisionICON2R1)
	receipts, err = sim.GoByTransaction(tx, csi)
	assert.True(t, checkReceipts(receipts))
	assert.NoError(t, err)

	prep = sim.GetPRep(env.preps[0])
	oldBonded = prep.Bonded()

	for i := 0; i < consistentValidationPenaltyCondition; i++ {
		// PenaltyCount is not reset after revision is 14
		prep = sim.GetPRep(env.preps[0])
		assert.Equal(t, icstate.GradeMain, prep.Grade())
		assert.Equal(t, i, prep.GetVPenaltyCount())

		// Make the case when prep0 fails to vote for blocks to validate
		vl = sim.ValidatorList()
		voted[0] = false
		csi = newConsensusInfo(sim.Database(), vl, voted)
		err = sim.Go(validationPenaltyCondition, csi)
		assert.NoError(t, err)

		// Check if prep0 got penalized after 5 blocks
		prep = sim.GetPRep(vl[0].Address())
		assert.Equal(t, icstate.GradeCandidate, prep.Grade())
		assert.Equal(t, i + 1, prep.GetVPenaltyCount())

		// Check if prep22 acts as a validator instead of prep0
		// prep22 was a sub prep before prep0 got penalized
		vl = sim.ValidatorList()
		assert.True(t, env.preps[mainPRepCount].Equal(vl[0].Address()))
		voted[0] = true
		csi = newConsensusInfo(sim.Database(), vl, voted)
		err = sim.GoToTermEnd(csi)
		assert.NoError(t, err)
	}

	// Check if the bond of prep0 is slashed by 10%
	prep = sim.GetPRep(env.preps[0])
	bonded = prep.Bonded()
	slashed = new(big.Int).Div(oldBonded, big.NewInt(10))
	assert.Zero(t, bonded.Cmp(new(big.Int).Sub(oldBonded, slashed)))

	vl = sim.ValidatorList()
	assert.True(t, vl[0].Address().Equal(env.preps[0]))

	// Case: prep0 has already been penalized 3 times.
	// From now on, the bond of prep0 will be slashed every penalty
	for i := 0; i < 3; i++ {
		prep = sim.GetPRep(env.preps[0])
		oldBonded = prep.Bonded()
		penaltyCount := consistentValidationPenaltyCondition + i

		prep = sim.GetPRep(env.preps[0])
		assert.Equal(t, icstate.GradeMain, prep.Grade())
		assert.Equal(t, penaltyCount, prep.GetVPenaltyCount())

		// Make the case when prep0 fails to vote for blocks to validate
		vl = sim.ValidatorList()
		voted[0] = false
		csi = newConsensusInfo(sim.Database(), vl, voted)
		err = sim.Go(validationPenaltyCondition, csi)
		assert.NoError(t, err)

		// Check if prep0 was slashed after 5 blocks
		penaltyCount++
		prep = sim.GetPRep(vl[0].Address())
		assert.Equal(t, icstate.GradeCandidate, prep.Grade())
		assert.Equal(t, penaltyCount, prep.GetVPenaltyCount())
		slashed = new(big.Int).Div(oldBonded, big.NewInt(10))
		assert.Zero(t, prep.Bonded().Cmp(new(big.Int).Sub(oldBonded, slashed)))

		// Check if prep22 acts as a validator instead of prep0
		// prep22 was a sub prep before prep0 got penalized
		vl = sim.ValidatorList()
		assert.True(t, env.preps[mainPRepCount].Equal(vl[0].Address()))
		voted[0] = true
		csi = newConsensusInfo(sim.Database(), vl, voted)
		err = sim.GoToTermEnd(csi)
		assert.NoError(t, err)
	}

	// Case: Accumulated penaltyCount will be reset to 0 after 30 terms when prep0 acts as a main prep
	vl = sim.ValidatorList()
	assert.True(t, env.preps[0].Equal(vl[0].Address()))

	for i := 0; i < 23; i++ {
		prep = sim.GetPRep(env.preps[0])
		assert.Equal(t, 6, prep.GetVPenaltyCount())

		csi = newConsensusInfo(sim.Database(), vl, voted)
		err = sim.GoToTermEnd(csi)
		assert.NoError(t, err)
	}

	for i := 0; i < 6; i++ {
		prep = sim.GetPRep(env.preps[0])
		assert.Equal(t, 6 - i, prep.GetVPenaltyCount())

		csi = newConsensusInfo(sim.Database(), vl, voted)
		err = sim.GoToTermEnd(csi)
		assert.NoError(t, err)
	}

	prep = sim.GetPRep(env.preps[0])
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

	c := NewConfig()
	c.MainPRepCount = mainPRepCount
	c.TermPeriod = termPeriod
	c.ValidationPenaltyCondition = validationPenaltyCondition
	c.ConsistentValidationPenaltyCondition = consistentValidationPenaltyCondition

	voted = make([]bool, mainPRepCount)
	for i := 0; i < len(voted); i++ {
		voted[i] = true
	}

	// Decentralization is activated
	env = initEnv(t, c, icmodule.Revision13)
	sim := env.sim

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
	csi = newConsensusInfo(sim.Database(), vl0, voted)
	err = sim.Go(validationPenaltyCondition, csi)
	assert.NoError(t, err)

	// Check if 1st validator got penalized after 5 blocks
	prep = sim.GetPRep(env.preps[0])
	assert.Equal(t, icstate.GradeCandidate, prep.Grade())
	assert.Equal(t, 1, prep.GetVPenaltyCount())

	// Check if prep22 is newly included in validatorList instead of prep00
	// Go ahead until term end
	vl = sim.ValidatorList()
	assert.True(t, checkValidatorList(vl, vl1))
	voted[0] = true
	csi = newConsensusInfo(sim.Database(), vl, voted)
	err = sim.GoToTermEnd(csi)
	assert.NoError(t, err)

	// term 2
	// prep0 -> main prep, prep21 -> sub prep
	// The first 2 consensus info follows the prev term validator list
	// prep22 fails to vote for the first 2 blocks of this term
	voted[0] = false
	csi = newConsensusInfo(sim.Database(), vl1, voted)
	err = sim.Go(2, csi)
	assert.NoError(t, err)

	// go ahead until term end without any false votes
	vl = sim.ValidatorList()
	assert.True(t, checkValidatorList(vl0, vl))
	voted[0] = true
	csi = newConsensusInfo(sim.Database(), vl, voted)
	err = sim.GoToTermEnd(csi)
	assert.NoError(t, err)

	// prep00 fails to vote for 7 consecutive blocks and gets penalized
	voted[0] = false
	csi = newConsensusInfo(sim.Database(), vl0, voted)
	err = sim.Go(validationPenaltyCondition, csi)
	assert.NoError(t, err)
	// prep00: mainPRep -> candidate, prep22: subPRep -> mainPRep
	vl = sim.ValidatorList()
	assert.True(t, checkValidatorList(vl1, vl))
	prep = sim.GetPRep(env.preps[0])
	assert.Equal(t, icstate.GradeCandidate, prep.Grade())

	// prep00 fails to vote for 2 blocks, getting penalized
	csi = newConsensusInfo(sim.Database(), vl0, voted)
	err = sim.Go(2, csi)
	assert.NoError(t, err)

	// Check if prep22 becomes a main prep instead of prep00
	prep = sim.GetPRep(env.preps[22])
	assert.Equal(t, int64(2), prep.GetVFailCont(sim.BlockHeight()))
	assert.Equal(t, icstate.GradeMain, prep.Grade())
	assert.Zero(t, prep.GetVPenaltyCount())
	vl = sim.ValidatorList()
	assert.True(t, checkValidatorList(vl, vl1))

	voted[0] = false
	csi = newConsensusInfo(sim.Database(), vl1, voted)
	err = sim.Go(3, csi)
	assert.NoError(t, err)

	// prep21 got penalized and its penaltyCount is set to 1
	prep = sim.GetPRep(env.preps[22])
	assert.Equal(t, 1, prep.GetVPenaltyCount())
	// prep00: candidate, prep21: mainPRep -> candidate, prep22: subPRep -> mainPRep
	vl = sim.ValidatorList()
	assert.True(t, checkValidatorList(vl, vl2))

	// Create 2 blocks
	voted[0] = false
	csi = newConsensusInfo(sim.Database(), vl1, voted)
	err = sim.Go(2, csi)
	assert.NoError(t, err)

	// prep0 and prep21 have got penalized and their grade is set to candidate
	// prep22 will be the new main prep
	prep = sim.GetPRep(env.preps[23])
	assert.Zero(t, prep.GetVPenaltyCount())
	assert.Zero(t, prep.GetVTotal(sim.BlockHeight()))
}
