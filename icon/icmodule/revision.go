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

import "github.com/icon-project/goloop/module"

const (
	Revision1 = iota + 1
	Revision2
	Revision3
	Revision4
	Revision5
	Revision6
	Revision7
	Revision8
	Revision9
	Revision10
	Revision11
	Revision12
	Revision13
	Revision14
	Revision15
	Revision16
	Revision17
	Revision18
	Revision19
	Revision20
	Revision21
	Revision22
	RevisionReserved
)

const (
	DefaultRevision = Revision1
	MaxRevision     = RevisionReserved - 1
	LatestRevision  = Revision22
)

const (
	RevisionIISS = Revision5

	RevisionDecentralize = Revision6

	RevisionFixTotalDelegated = Revision7

	RevisionFixBugDisabledPRep = Revision8

	RevisionFixBurnEventSignature     = Revision9
	RevisionMultipleUnstakes          = Revision9
	RevisionFixEmailValidation        = Revision9
	RevisionDelegationSlotMaxTo100    = Revision9
	RevisionSystemSCORE               = Revision9
	RevisionSetIRepViaNetworkProposal = Revision9
	RevisionPreventDuplicatedEndpoint = Revision9

	// RevisionLockAddress = Revision10

	RevisionFixInvalidUnstake = Revision11

	RevisionBurnV2 = Revision12

	RevisionICON2R0              = Revision13
	RevisionFixClaimIScore       = Revision13
	RevisionFixSetDelegation     = Revision13
	RevisionFixRLPBug            = Revision13
	RevisionResetPenaltyMask     = Revision13
	RevisionEnableBondAPIs       = Revision13
	RevisionFixIllegalDelegation = Revision13
	RevisionStopICON1Support     = Revision13

	RevisionICON2R1       = Revision14
	RevisionEnableIISS3   = Revision14
	RevisionEnableFee3    = Revision14
	RevisionBlockAccounts = Revision14

	RevisionICON2R2      = Revision15
	RevisionEnableJavaEE = Revision15

	RevisionFixIGlobal = Revision16

	RevisionICON2R3             = Revision17
	RevisionEnableSetScoreOwner = Revision17
	RevisionExtraMainPReps      = Revision17
	RevisionFixVotingReward     = Revision17

	RevisionFixTransferRewardFund = Revision18

	// Unused
	// RevisionJavaPurgeEnumCache = Revision19

	// Unused
	// RevisionJavaFixMapValues = Revision20

	RevisionBTP2 = Revision21

	RevisionUpdatePRepStats = Revision22
	RevisionBlockAccountAPI = Revision22
)

var revisionFlags = []module.Revision{
	// Revision0
	module.UseChainID | module.UseMPTOnEvents | module.UseCompactAPIInfo | module.LegacyFeeCharge | module.LegacyFallbackCheck | module.LegacyContentCount | module.LegacyBalanceCheck | module.LegacyNoTimeout,
	// Revision1
	0,
	// Revision2
	module.AutoAcceptGovernance,
	// Revision3
	module.LegacyInputJSON | module.LegacyFallbackCheck | module.LegacyContentCount | module.LegacyBalanceCheck,
	// Revision4
	0,
	// Revision5
	0,
	// Revision6
	0,
	// Revision7
	0,
	// Revision8
	0,
	// Revision9
	0,
	// Revision10
	0,
	// Revision11
	0,
	// Revision12
	0,
	// Revision13
	module.LegacyFeeCharge | module.LegacyNoTimeout,
	// Revision14
	module.LegacyInputJSON | module.InputCostingWithJSON,
	// Revision15
	0,
	// Revision16
	0,
	// Revision17
	0,
	// Revision18
	module.FixLostFeeByDeposit,
	// Revision19
	module.PurgeEnumCache,
	// Revision20
	module.FixMapValues,
	// Revision21
	module.MultipleFeePayers,
	// Revision22
	module.FixJCLSteps,
}

func init() {
	var revSum module.Revision
	for idx, rev := range revisionFlags {
		revSum ^= rev
		revisionFlags[idx] = revSum
	}
}

func ValueToRevision(v int) module.Revision {
	if v < Revision1 {
		return revisionFlags[0]
	}
	if v >= len(revisionFlags) {
		return module.Revision(v) + revisionFlags[len(revisionFlags)-1]
	} else {
		return module.Revision(v) + revisionFlags[v]
	}
}
