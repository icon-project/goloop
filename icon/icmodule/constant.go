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

package icmodule

import (
	"math/big"
)

const (
	DayBlock     = 24 * 60 * 60 / 2
	DayPerMonth  = 30
	MonthBlock   = DayBlock * DayPerMonth
	MonthPerYear = 12
	YearBlock    = MonthBlock * MonthPerYear

	MinRrep        = 200
	RrepMultiplier = 3      // rrep = rrep + eep + dbp = 3 * rrep
	RrepDivider    = 10_000 // rrep(10_000) = 100.00%, rrep(200) = 2.00%
	MinDelegation  = YearBlock / IScoreICXRatio * (RrepDivider / MinRrep)
)

var (
	BigIntMinDelegation = big.NewInt(int64(MinDelegation))
)

const (
	ConfigFile               = "./icon_config.json"
	IScoreICXRatio           = 1_000
	VotedRewardMultiplier    = 100
	InitialTermPeriod        = DayBlock
	DecentralizedTermPeriod  = 43120
	InitialDepositTerm       = 1_296_000
	DisableDepositTerm       = 0
	InitialUnstakeSlotMax    = 1
	InitialDelegationSlotMax = 10
	IISS2BondRequirement     = 0
	InitialIRep              = 50_000 // in icx, not loop
	MinIRep                  = 10_000
	RewardPoint              = 0.7
	ICX                      = 1_000_000_000_000_000_000

	DefaultTermPeriod                           = InitialTermPeriod
	DefaultUnbondingPeriodMultiplier            = 7
	DefaultUnstakeSlotMax                       = 1000
	DefaultMainPRepCount                        = 22
	DefaultSubPRepCount                         = 78
	DefaultIRep                                 = 0
	DefaultRRep                                 = 1200
	DefaultBondRequirement                      = 5 // 5%
	DefaultLockMinMultiplier                    = 5
	DefaultLockMaxMultiplier                    = 20
	DefaultIglobal                              = YearBlock * IScoreICXRatio
	DefaultIprep                                = 50 // 50%
	DefaultIcps                                 = 0  // 0%
	DefaultIrelay                               = 0  // 0%
	DefaultIvoter                               = 50 // 50%
	DefaultUnbondingMax                         = 100
	DefaultValidationPenaltyCondition           = 660
	DefaultConsistentValidationPenaltyCondition = 5
	DefaultConsistentValidationPenaltyMask      = 30
	DefaultConsistentValidationPenaltySlashRate = 0 // 0%
	DefaultDelegationSlotMax                    = 100
	DefaultExtraMainPRepCount                   = 3
	DefaultNonVotePenaltySlashRate              = 0 // 0%

	// IISS-4.0
	DefaultPRepDisqualificationSlashingRate       = Rate(DenomInRate) // 100%
	DefaultContinuousBlockValidationSlashingRate  = Rate(1)           // 0.01%
	DefaultBlockValidationSlashingRate            = Rate(0)           // 0%
	DefaultMissingNetworkProposalVoteSlashingRate = Rate(1)           // 0.01%
	DefaultDoubleVoteSlashingRate                 = Rate(1000)        // 10%
)

// The following variables are read-only
var (
	BigIntZero           = new(big.Int)
	BigIntICX            = big.NewInt(ICX)
	BigIntInitialIRep    = new(big.Int).Mul(big.NewInt(InitialIRep), BigIntICX)
	BigIntMinIRep        = new(big.Int).Mul(big.NewInt(MinIRep), BigIntICX)
	BigIntIScoreICXRatio = big.NewInt(IScoreICXRatio)
	BigIntRegPRepFee     = new(big.Int).Mul(big.NewInt(2000), BigIntICX)
	BigIntDayBlocks      = big.NewInt(DayBlock)

	DefaultMinBond = new(big.Int).Mul(big.NewInt(10_000), BigIntICX)
)
