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

package icon

import (
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

type handleRevFunc func(s *chainScore, targetRev int) error

var handleRevFuncs = map[int]handleRevFunc{
	icmodule.Revision5:  onRevision5,
	icmodule.Revision6:  onRevision6,
	icmodule.Revision9:  onRevision9,
	icmodule.Revision13: onRevision13,
	icmodule.Revision14: onRevision14,
	icmodule.Revision15: onRevision15,
	icmodule.Revision17: onRevision17,
	icmodule.Revision21: onRevision21,
	icmodule.Revision23: onRevision23,
	icmodule.Revision24: onRevision24,
}

func onRevision5(s *chainScore, targetRev int) error {
	// goloop engine

	as := s.cc.GetAccountState(state.SystemID)

	// enable Fee sharing 2.0
	systemConfig := scoredb.NewVarDB(as, state.VarServiceConfig).Int64()
	systemConfig |= state.SysConfigFeeSharing
	if err := scoredb.NewVarDB(as, state.VarServiceConfig).Set(systemConfig); err != nil {
		return err
	}
	// enable Virtual step
	depositTerm := scoredb.NewVarDB(as, state.VarDepositTerm).Int64()
	if depositTerm == icmodule.DisableDepositTerm {
		if err := scoredb.NewVarDB(as, state.VarDepositTerm).Set(icmodule.InitialDepositTerm); err != nil {
			return err
		}
	}

	// RevisionIISS
	iconConfig := s.loadIconConfig()
	s.cc.Logger().Infof("IconConfig: %s", iconConfig)

	s.cc.GetExtensionState().Reset(iiss.NewExtensionSnapshot(s.cc.Database(), nil))
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	if err := es.State.SetIISSVersion(icstate.IISSVersion2); err != nil {
		return err
	}
	if err := es.State.SetTermPeriod(iconConfig.TermPeriod.Int64()); err != nil {
		return err
	}
	if err := es.State.SetIRep(iconConfig.Irep.Value()); err != nil {
		return err
	}
	if err := es.State.SetRRep(iconConfig.Rrep.Value()); err != nil {
		return err
	}
	if err := es.State.SetMainPRepCount(iconConfig.MainPRepCount.Int64()); err != nil {
		return err
	}
	if err := es.State.SetSubPRepCount(iconConfig.SubPRepCount.Int64()); err != nil {
		return err
	}
	if err := es.State.SetBondRequirement(icmodule.ToRate(iconConfig.BondRequirement.Int64())); err != nil {
		return err
	}
	if err := es.State.SetLockVariables(iconConfig.LockMinMultiplier.Value(), iconConfig.LockMaxMultiplier.Value()); err != nil {
		return err
	}
	if err := es.State.SetUnbondingPeriodMultiplier(iconConfig.UnbondingPeriodMultiplier.Int64()); err != nil {
		return err
	}
	if err := es.State.SetDelegationSlotMax(iconConfig.DelegationSlotMax.Int64()); err != nil {
		return err
	}
	if err := applyRewardFund(iconConfig, es.State); err != nil {
		return err
	}
	if err := es.State.SetUnstakeSlotMax(iconConfig.UnstakeSlotMax.Int64()); err != nil {
		return err
	}
	if err := es.State.SetUnbondingMax(iconConfig.UnbondingMax.Int64()); err != nil {
		return err
	}
	if err := es.State.SetValidationPenaltyCondition(int(iconConfig.ValidationPenaltyCondition.Int64())); err != nil {
		return err
	}
	if err := es.State.SetConsistentValidationPenaltyCondition(
		iconConfig.ConsistentValidationPenaltyCondition.Int64()); err != nil {
		return err
	}
	if err := es.State.SetConsistentValidationPenaltyMask(
		iconConfig.ConsistentValidationPenaltyMask.Int64()); err != nil {
		return err
	}
	// 10% slashRate is hardcoded for backward compatibility
	if err := es.State.SetSlashingRate(
		icmodule.RevisionIISS,
		icmodule.PenaltyAccumulatedValidationFailure,
		icmodule.ToRate(10)); err != nil {
		return err
	}

	if err := es.State.SetIISSVersion(icstate.IISSVersion2); err != nil {
		return err
	}
	if unstakeSlotMax := es.State.GetUnstakeSlotMax(); unstakeSlotMax == icmodule.DefaultUnstakeSlotMax {
		if err := es.State.SetUnstakeSlotMax(icmodule.InitialUnstakeSlotMax); err != nil {
			return err
		}
	}
	if delegationSlotMax := es.State.GetDelegationSlotMax(); delegationSlotMax == icmodule.DefaultDelegationSlotMax {
		if err := es.State.SetDelegationSlotMax(icmodule.InitialDelegationSlotMax); err != nil {
			return err
		}
	}
	if es.State.GetBondRequirement() == icmodule.ToRate(icmodule.DefaultBondRequirement) {
		if err := es.State.SetBondRequirement(icmodule.ToRate(icmodule.IISS2BondRequirement)); err != nil {
			return err
		}
	}

	if err := es.GenesisTerm(s.cc.BlockHeight(), targetRev); err != nil {
		return err
	}

	return nil
}

func onRevision6(s *chainScore, _ int) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	if termPeriod := es.State.GetTermPeriod(); termPeriod == icmodule.InitialTermPeriod {
		if err := es.State.SetTermPeriod(icmodule.DecentralizedTermPeriod); err != nil {
			return err
		}
	}
	return nil
}

func onRevision9(s *chainScore, _ int) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)

	// RevisionMultipleUnstakes
	if unstakeSlotMax := es.State.GetUnstakeSlotMax(); unstakeSlotMax == icmodule.InitialUnstakeSlotMax {
		if err := es.State.SetUnstakeSlotMax(icmodule.DefaultUnstakeSlotMax); err != nil {
			return err
		}
	}

	// RevisionDelegationSlotMaxTo100
	if dSlotMax := es.State.GetDelegationSlotMax(); dSlotMax == icmodule.InitialDelegationSlotMax {
		if err := es.State.SetDelegationSlotMax(icmodule.DefaultDelegationSlotMax); err != nil {
			return err
		}
	}

	// RevisionSetIRepViaNetworkProposal
	if irep := es.State.GetIRep(); irep.Sign() == 0 {
		if term := es.State.GetTermSnapshot(); term != nil {
			if err := es.State.SetIRep(term.Irep()); err != nil {
				return err
			}
		}
	}

	return nil
}

func onRevision13(s *chainScore, _ int) error {
	as := s.cc.GetAccountState(state.SystemID)

	// using v2 block for ICON2
	if err := scoredb.NewVarDB(as, state.VarNextBlockVersion).Set(module.BlockVersion2); err != nil {
		return err
	}
	if s.cc.ChainID() == CIDForMainNet {
		if err := scoredb.NewVarDB(as, state.VarRoundLimitFactor).Set(9); err != nil {
			return err
		}
	}
	return nil
}

func onRevision14(s *chainScore, _ int) error {
	if s.cc.ChainID() == CIDForMainNet {
		// The time when predefined accounts will be blocked is changed from rev10 to rev14
		s.blockAccounts()
	}

	as := s.cc.GetAccountState(state.SystemID)

	// disable Virtual step
	if err := scoredb.NewVarDB(as, state.VarDepositTerm).Set(icmodule.DisableDepositTerm); err != nil {
		return err
	}

	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)

	// Rev13: IISS-2.x works on goloop engine, enabling some IISS-3.x related APIs.
	//     (getBond, setBond, getBonderList, setBonderList)
	// Rev14: IISS-3.x works fully on goloop engine.
	if err := es.State.SetIISSVersion(icstate.IISSVersion3); err != nil {
		return err
	}

	if es.State.GetBondRequirement() == icmodule.ToRate(icmodule.IISS2BondRequirement) {
		if err := es.State.SetBondRequirement(icmodule.ToRate(icmodule.DefaultBondRequirement)); err != nil {
			return err
		}
	}
	if err := es.ClearPRepIllegalDelegated(); err != nil {
		return err
	}
	return nil
}

func onRevision15(s *chainScore, _ int) error {
	as := s.cc.GetAccountState(state.SystemID)

	// Enable JavaEE
	if err := scoredb.NewVarDB(as, state.VarEnabledEETypes).Set(EETypesJavaAndPython); err != nil {
		return err
	}

	return nil
}

func onRevision17(s *chainScore, _ int) error {
	revision := icmodule.Revision17
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	iconConfig := s.loadIconConfig()

	// Set slash rate of Non Vote Penalty
	if err := es.State.SetSlashingRate(
		revision,
		icmodule.PenaltyAccumulatedValidationFailure,
		icmodule.ToRate(iconConfig.ConsistentValidationPenaltySlashRate.Int64())); err != nil {
		return err
	}
	if err := es.State.SetSlashingRate(
		revision,
		icmodule.PenaltyMissedNetworkProposalVote,
		icmodule.ToRate(iconConfig.NonVotePenaltySlashRate.Int64())); err != nil {
		return err
	}

	// Enable ExtraMainPReps
	extraMainPRepCount := iconConfig.ExtraMainPRepCount.Int64()
	if err := es.State.SetExtraMainPRepCount(extraMainPRepCount); err != nil {
		return err
	}
	return nil
}

func onRevision21(s *chainScore, _ int) error {
	if s.cc.ChainID() == CIDForMainNet {
		s.blockAccounts2()
	}

	return nil
}

func onRevision23(s *chainScore, _ int) error {
	revision := icmodule.RevisionPreIISS4
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)

	// RewardFundAllocation2
	r := es.State.GetRewardFundV1()
	if err := es.State.SetRewardFund(r.ToRewardFundV2()); err != nil {
		return err
	}

	// minBond for minimum wage
	if err := es.State.SetMinimumBond(icmodule.DefaultMinBond); err != nil {
		return err
	}

	// slashing rate
	if s.cc.ChainID() == CIDForMainNet {
		items := []struct {
			pt   icmodule.PenaltyType
			rate icmodule.Rate
		}{
			{icmodule.PenaltyPRepDisqualification, icmodule.DefaultPRepDisqualificationSlashingRate},
			{icmodule.PenaltyAccumulatedValidationFailure, icmodule.DefaultContinuousBlockValidationSlashingRate},
			{icmodule.PenaltyValidationFailure, icmodule.DefaultBlockValidationSlashingRate},
			{icmodule.PenaltyMissedNetworkProposalVote, icmodule.DefaultMissingNetworkProposalVoteSlashingRate},
			{icmodule.PenaltyDoubleVote, icmodule.DefaultDoubleVoteSlashingRate},
		}
		for _, item := range items {
			if err := es.State.SetSlashingRate(revision, item.pt, item.rate); err != nil {
				return err
			}
		}
	}
	return nil
}

func onRevision24(s *chainScore, _ int) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)

	// IISS 4.0
	if err := es.State.SetIISSVersion(icstate.IISSVersion4); err != nil {
		return err
	}

	return nil
}
