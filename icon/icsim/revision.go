package icsim

import (
	"math/big"

	"github.com/icon-project/goloop/icon"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

type RevHandler func(ws state.WorldState) error

func (sim *simulatorImpl) initRevHandler() {
	sim.revHandlers = map[int]RevHandler{
		icmodule.Revision5:  sim.handleRev5,
		icmodule.Revision6:  sim.handleRev6,
		icmodule.Revision9:  sim.handleRev9,
		icmodule.Revision14: sim.handleRev14,
		icmodule.Revision15: sim.handleRev15,
		icmodule.Revision17: sim.handleRev17, // Enable ExtraMainPReps
	}
}

func (sim *simulatorImpl) handleRevisionChange(ws state.WorldState, oldRev, newRev int) error {
	if oldRev >= newRev {
		return nil
	}
	for rev := oldRev + 1; rev <= newRev; rev++ {
		if handler, ok := sim.revHandlers[rev]; ok {
			if err := handler(ws); err != nil {
				return err
			}
		}
	}
	return nil
}

func (sim *simulatorImpl) handleRev5(ws state.WorldState) error {
	revision := icmodule.Revision5
	as := ws.GetAccountState(state.SystemID)

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

	cfg := sim.config
	ws.GetExtensionState().Reset(iiss.NewExtensionSnapshot(ws.Database(), nil))
	es := getExtensionState(ws)

	if err := es.State.SetIISSVersion(icstate.IISSVersion2); err != nil {
		return err
	}
	if err := es.State.SetTermPeriod(cfg.TermPeriod); err != nil {
		return err
	}
	if err := es.State.SetIRep(big.NewInt(cfg.Irep)); err != nil {
		return err
	}
	if err := es.State.SetRRep(big.NewInt(cfg.Rrep)); err != nil {
		return err
	}
	if err := es.State.SetMainPRepCount(cfg.MainPRepCount); err != nil {
		return err
	}
	if err := es.State.SetSubPRepCount(cfg.SubPRepCount); err != nil {
		return err
	}
	if err := es.State.SetBondRequirement(cfg.BondRequirement); err != nil {
		return err
	}
	if err := es.State.SetLockVariables(big.NewInt(cfg.LockMinMultiplier), big.NewInt(cfg.LockMaxMultiplier)); err != nil {
		return err
	}
	if err := es.State.SetUnbondingPeriodMultiplier(cfg.UnbondingPeriodMultiplier); err != nil {
		return err
	}
	if err := es.State.SetDelegationSlotMax(cfg.DelegationSlotMax); err != nil {
		return err
	}
	if err := applyRewardFund(cfg, es.State); err != nil {
		return err
	}
	if err := es.State.SetUnstakeSlotMax(cfg.UnstakeSlotMax); err != nil {
		return err
	}
	if err := es.State.SetUnbondingMax(cfg.UnbondingMax); err != nil {
		return err
	}
	if err := es.State.SetValidationPenaltyCondition(cfg.ValidationPenaltyCondition); err != nil {
		return err
	}
	if err := es.State.SetConsistentValidationPenaltyCondition(
		cfg.ConsistentValidationPenaltyCondition); err != nil {
		return err
	}
	if err := es.State.SetConsistentValidationPenaltyMask(
		cfg.ConsistentValidationPenaltyMask); err != nil {
		return err
	}
	if err := es.State.SetSlashingRate(
		revision,
		icmodule.PenaltyAccumulatedValidationFailure,
		cfg.ConsistentValidationPenaltySlashRate); err != nil {
		return err
	}

	return es.GenesisTerm(sim.blockHeight, revision)
}

func applyRewardFund(config *SimConfig, s *icstate.State) error {
	rf := icstate.NewRewardFund(icstate.RFVersion1)
	if err := rf.SetIGlobal(big.NewInt(config.RewardFund.Iglobal)); err != nil {
		return err
	}
	if err := rf.SetAllocation(
		map[icstate.RFundKey]icmodule.Rate{
			icstate.KeyIprep:  icmodule.ToRate(config.RewardFund.Iprep),
			icstate.KeyIcps:   icmodule.ToRate(config.RewardFund.Icps),
			icstate.KeyIrelay: icmodule.ToRate(config.RewardFund.Irelay),
			icstate.KeyIvoter: icmodule.ToRate(config.RewardFund.Ivoter),
		},
	); err != nil {
		return err
	}
	if err := s.SetRewardFund(rf); err != nil {
		return err
	}
	return nil
}

// handleRev6: icmodule.RevisionDecentralize
func (sim *simulatorImpl) handleRev6(ws state.WorldState) error {
	es := getExtensionState(ws)
	if termPeriod := es.State.GetTermPeriod(); termPeriod == icmodule.InitialTermPeriod {
		if err := es.State.SetTermPeriod(icmodule.DecentralizedTermPeriod); err != nil {
			return err
		}
	}
	return nil
}

func (sim *simulatorImpl) handleRev9(ws state.WorldState) error {
	es := getExtensionState(ws)
	if unstakeSlotMax := es.State.GetUnstakeSlotMax(); unstakeSlotMax == icmodule.InitialUnstakeSlotMax {
		if err := es.State.SetUnstakeSlotMax(icmodule.DefaultUnstakeSlotMax); err != nil {
			return err
		}
	}
	if dSlotMax := es.State.GetDelegationSlotMax(); dSlotMax == icmodule.InitialDelegationSlotMax {
		if err := es.State.SetDelegationSlotMax(icmodule.DefaultDelegationSlotMax); err != nil {
			return err
		}
	}
	if irep := es.State.GetIRep(); irep.Sign() == 0 {
		if term := es.State.GetTermSnapshot(); term != nil {
			if err := es.State.SetIRep(term.Irep()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (sim *simulatorImpl) handleRev14(ws state.WorldState) error {
	es := getExtensionState(ws)
	as := ws.GetAccountState(state.SystemID)

	if err := es.State.SetIISSVersion(icstate.IISSVersion3); err != nil {
		return err
	}
	// disable Virtual step
	if err := scoredb.NewVarDB(as, state.VarDepositTerm).Set(icmodule.DisableDepositTerm); err != nil {
		return err
	}
	// using v2 block for ICON2
	if err := scoredb.NewVarDB(as, state.VarNextBlockVersion).Set(module.BlockVersion2); err != nil {
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

func (sim *simulatorImpl) handleRev15(ws state.WorldState) error {
	as := ws.GetAccountState(state.SystemID)

	if err := scoredb.NewVarDB(as, state.VarEnabledEETypes).Set(icon.EETypesJavaAndPython); err != nil {
		return err
	}
	return nil
}

func (sim *simulatorImpl) handleRev17(ws state.WorldState) error {
	revision := icmodule.Revision17
	cfg := sim.config
	es := getExtensionState(ws)

	// Set slashingRate for PenaltyAccumulatedValidationFailure
	if err := es.State.SetSlashingRate(
		revision,
		icmodule.PenaltyAccumulatedValidationFailure,
		cfg.ConsistentValidationPenaltySlashRate); err != nil {
		return err
	}
	// Set slashingRate for PenaltyMissedNetworkProposalVote
	if err := es.State.SetSlashingRate(
		revision,
		icmodule.PenaltyMissedNetworkProposalVote,
		cfg.NonVotePenaltySlashRate); err != nil {
		return err
	}
	// Enable ExtraMainPReps
	if err := es.State.SetExtraMainPRepCount(cfg.ExtraMainPRepCount); err != nil {
		return err
	}
	return nil
}
