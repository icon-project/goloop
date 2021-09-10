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
	RevisionReserved
)

const (
	DefaultRevision = Revision1
	MaxRevision     = RevisionReserved - 1
	LatestRevision  = Revision14
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

	RevisionLockAddress = Revision10

	RevisionFixInvalidUnstake = Revision11

	RevisionBurnV2 = Revision12

	RevisionICON2R0              = Revision13
	RevisionFixClaimIScore       = RevisionICON2R0
	RevisionFixSetDelegation     = RevisionICON2R0
	RevisionFixRLPBug            = RevisionICON2R0
	RevisionResetPenaltyMask     = RevisionICON2R0
	RevisionEnableBondAPIs       = RevisionICON2R0
	RevisionDisableIRep          = RevisionICON2R0
	RevisionDisableRRep          = RevisionICON2R0
	RevisionFixIllegalDelegation = RevisionICON2R0
	RevisionStopICON1Support     = RevisionICON2R0

	RevisionICON2R1     = Revision14
	RevisionEnableIISS3 = RevisionICON2R1
	RevisionEnableFee3  = RevisionICON2R1

	RevisionICON2R2      = Revision15
	RevisionEnableJavaEE = RevisionICON2R2

	// TODO: Fix a revision for enabling extra main preps
	RevisionExtraMainPReps = 100
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
