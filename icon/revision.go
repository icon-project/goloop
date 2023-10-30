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

type handleRevFunc func(s *chainScore, rev, toRev int) error
type revHandlerItem struct {
	rev int
	fn  handleRevFunc
}

var revHandlerTable = []revHandlerItem{
	{icmodule.RevisionIISS, onRevIISS},
	{icmodule.RevisionDecentralize, onRevDecentralize},
	{icmodule.RevisionIISS2, onRevIISS2},
	{icmodule.RevisionICON2R0, onRevICON2R0},
	{icmodule.RevisionICON2R1, onRevICON2R1},
	{icmodule.RevisionICON2R2, onRevEnableJavaEE},
	{icmodule.RevisionICON2R3, onRevICON2R3},
	{icmodule.RevisionBlockAccounts2, onRevBlockAccounts2},
	{icmodule.RevisionIISS4R0, onRevIISS4R0},
	{icmodule.RevisionIISS4R1, onRevIISS4R1},
}

// DO NOT update revHandlerMap manually
var revHandlerMap = make(map[int][]revHandlerItem)

func init() {
	for _, item := range revHandlerTable {
		rev := item.rev
		items, _ := revHandlerMap[rev]
		revHandlerMap[rev] = append(items, item)
	}
	revHandlerTable = nil
}

func (s *chainScore) handleRevisionChange(r1, r2 int) error {
	s.log.Infof("handleRevisionChange %d->%d", r1, r2)
	if r1 >= r2 {
		return nil
	}

	for rev := r1 + 1; rev <= r2; rev++ {
		if items, ok := revHandlerMap[rev]; ok {
			for _, item := range items {
				if err := item.fn(s, rev, r2); err != nil {
					s.log.Infof("call handleRevFunc for %d", rev)
					return err
				}
			}
		}
	}
	return nil
}

func onRevIISS(s *chainScore, _, toRev int) error {
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

	if err := es.GenesisTerm(s.cc.BlockHeight(), toRev); err != nil {
		return err
	}

	return nil
}

func onRevDecentralize(s *chainScore, _, _ int) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	if termPeriod := es.State.GetTermPeriod(); termPeriod == icmodule.InitialTermPeriod {
		if err := es.State.SetTermPeriod(icmodule.DecentralizedTermPeriod); err != nil {
			return err
		}
	}
	return nil
}

func onRevIISS2(s *chainScore, _, _ int) error {
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

func onRevICON2R0(s *chainScore, _, _ int) error {
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

func onRevICON2R1(s *chainScore, _, _ int) error {
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

func onRevEnableJavaEE(s *chainScore, _, _ int) error {
	as := s.cc.GetAccountState(state.SystemID)

	// Enable JavaEE
	if err := scoredb.NewVarDB(as, state.VarEnabledEETypes).Set(EETypesJavaAndPython); err != nil {
		return err
	}

	return nil
}

func onRevICON2R3(s *chainScore, _, _ int) error {
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

func onRevBlockAccounts2(s *chainScore, _, _ int) error {
	if s.cc.ChainID() == CIDForMainNet {
		s.blockAccounts2()
	}

	return nil
}

// onRevision23 handles states in PreIISS4 phase
func onRevIISS4R0(s *chainScore, rev, _ int) error {
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

	// slashingRates migration for AccumulatedValidationFailure and MissedNetworkProposalVote
	for _, pt := range []icmodule.PenaltyType{
		icmodule.PenaltyAccumulatedValidationFailure,
		icmodule.PenaltyMissedNetworkProposalVote,
	} {
		rate, err := es.State.GetSlashingRate(rev-1, pt)
		if err != nil {
			return err
		}
		if rate > 0 {
			if err = es.State.SetSlashingRate(rev, pt, rate); err != nil {
				return err
			}
		}
	}

	return nil
}

// onRevision24 handles states in IISS4 phase
func onRevIISS4R1(s *chainScore, _, _ int) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)

	// IISS 4.0
	if err := es.State.SetIISSVersion(icstate.IISSVersion4); err != nil {
		return err
	}

	return nil
}
