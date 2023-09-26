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
	DefaultDoubleSignSlashingRate                 = Rate(1000)        // 10%
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

var BlockedAccount = map[string]bool{
	"hx76dcc464a27d74ca7798dd789d2e1da8193219b4": true,
	"hxac5c6e6f7a6e8ae1baba5f0cb512f7596b95f1fe": true,
	"hx966f5f9e2ab5b80a0f2125378e85d17a661352f4": true,
	"hxad2bc6446ee3ae23228889d21f1871ed182ca2ca": true,
	"hxc39a4c8438abbcb6b49de4691f07ee9b24968a1b": true,
	"hx96505aac67c4f9033e4bac47397d760f121bcc44": true,
	"hxf5bbebeb7a7d37d2aee5d93a8459e182cbeb725d": true,
	"hx4602589eb91cf99b27296e5bd712387a23dd8ce5": true,
	"hxa67e30ec59e73b9e15c7f2c4ddc42a13b44b2097": true,
	"hx52c32d0b82f46596f697d8ba2afb39105f3a6360": true,
	"hx985cf67b563fb908543385da806f297482f517b4": true,
	"hxc0567bbcba511b84012103a2360825fddcd058ab": true,
	"hx20be21b8afbbc0ba46f0671508cfe797c7bb91be": true,
	"hx19e551eae80f9b9dcfed1554192c91c96a9c71d1": true,
	"hx0607341382dee5e039a87562dcb966e71881f336": true,
	"hxdea6fe8d6811ec28db095b97762fdd78b48c291f": true,
	"hxaf3a561e3888a2b497941e464f82fd4456db3ebf": true,
	"hx061b01c59bd9fc1282e7494ff03d75d0e7187f47": true,
	"hx10d12d5726f50e4cf92c5fad090637b403516a41": true,
	"hx10e8a7289c3989eac07828a840905344d8ed559b": true,
}
