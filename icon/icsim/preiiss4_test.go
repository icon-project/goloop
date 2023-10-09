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

/*
func TestSimulatorImpl_SetSlashingRate(t *testing.T) {
	const (
		termPeriod                           = int64(100)
		mainPRepCount                        = int64(22)
		validationPenaltyCondition           = int64(5)
		consistentValidationPenaltyCondition = int64(3)
	)

	cfg := NewSimConfigWithParams(map[string]interface{}{
		"mainPReps":                            mainPRepCount,
		"termPeriod":                           termPeriod,
		"validationPenaltyCondition":           validationPenaltyCondition,
		"consistentValidationPenaltyCondition": consistentValidationPenaltyCondition,
	})

	var tx Transaction
	initRevision := icmodule.ValueToRevision(icmodule.RevisionPreIISS4)
	env, err := NewEnv(cfg, initRevision)
	assert.NoError(t, err)
	assert.NotNil(t, env)
	sim := env.Simulator()
	assert.Equal(t, initRevision, sim.Revision())

	_, err = sim.GetSlashingRates(nil)
	assert.NoError(t, err)

	expRates := map[string]icmodule.Rate{
		icmodule.PenaltyPRepDisqualification.String(): icmodule.ToRate(100),
		icmodule.PenaltyValidationFailure.String(): icmodule.Rate(0),
		icmodule.PenaltyAccumulatedValidationFailure.String(): icmodule.Rate(1),
		icmodule.PenaltyMissedNetworkProposalVote.String(): icmodule.ToRate(1),
		icmodule.PenaltyDoubleVote.String(): icmodule.ToRate(10),
	}
	tx = sim.SetSlashingRates(env.Governance(), expRates)
	receipts, err := sim.GoByTransaction(nil, tx)
	assert.NoError(t, err)
	assert.True(t, CheckReceiptSuccess(receipts...))

	rates, err := sim.GetSlashingRates(nil)
	assert.Equal(t, expRates, rates)
}
 */
