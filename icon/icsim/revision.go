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

func (sim *simulatorImpl) handleRevisionChange(ws state.WorldState, r1, r2 int) error {
	if r1 >= r2 {
		return nil
	}
	for rev := r1 + 1; rev <= r2; rev++ {
		if handler, ok := sim.revHandlers[rev]; ok {
			if err := handler(ws, r1, rev); err != nil {
				return err
			}
		}
	}
	return nil
}

func (sim *simulatorImpl) handleRev5(ws state.WorldState, r1, r2 int) error {
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
	return sim.handleRevIISS(ws, r1, r2)
}

func (sim *simulatorImpl) handleRevIISS(ws state.WorldState, r1, r2 int) error {
	config := sim.config
	ws.GetExtensionState().Reset(iiss.NewExtensionSnapshot(ws.Database(), nil))
	es := ws.GetExtensionState().(*iiss.ExtensionStateImpl)

	if err := es.State.SetIISSVersion(icstate.IISSVersion2); err != nil {
		return err
	}
	if err := es.State.SetTermPeriod(config.TermPeriod); err != nil {
		return err
	}
	if err := es.State.SetIRep(big.NewInt(config.Irep)); err != nil {
		return err
	}
	if err := es.State.SetRRep(big.NewInt(config.Rrep)); err != nil {
		return err
	}
	if err := es.State.SetMainPRepCount(config.MainPRepCount); err != nil {
		return err
	}
	if err := es.State.SetSubPRepCount(config.SubPRepCount); err != nil {
		return err
	}
	if err := es.State.SetBondRequirement(icmodule.ToRate(config.BondRequirement)); err != nil {
		return err
	}
	if err := es.State.SetLockVariables(big.NewInt(config.LockMinMultiplier), big.NewInt(config.LockMaxMultiplier)); err != nil {
		return err
	}
	if err := es.State.SetUnbondingPeriodMultiplier(config.UnbondingPeriodMultiplier); err != nil {
		return err
	}
	if err := es.State.SetDelegationSlotMax(config.DelegationSlotMax); err != nil {
		return err
	}
	if err := applyRewardFund(config, es.State); err != nil {
		return err
	}
	if err := es.State.SetUnstakeSlotMax(config.UnstakeSlotMax); err != nil {
		return err
	}
	if err := es.State.SetUnbondingMax(config.UnbondingMax); err != nil {
		return err
	}
	if err := es.State.SetValidationPenaltyCondition(config.ValidationPenaltyCondition); err != nil {
		return err
	}
	if err := es.State.SetConsistentValidationPenaltyCondition(
		config.ConsistentValidationPenaltyCondition); err != nil {
		return err
	}
	if err := es.State.SetConsistentValidationPenaltyMask(
		config.ConsistentValidationPenaltyMask); err != nil {
		return err
	}
	if err := es.State.SetConsistentValidationPenaltySlashRate(
		r2, icmodule.ToRate(int64(config.ConsistentValidationPenaltySlashRate))); err != nil {
		return err
	}

	return es.GenesisTerm(sim.blockHeight, r2)
}

func applyRewardFund(config *config, s *icstate.State) error {
	rf := icstate.NewRewardFund(icstate.RFVersion1)
	rf.SetIGlobal(big.NewInt(config.RewardFund.Iglobal))
	rf.SetAllocation(
		map[icstate.RFundKey]icmodule.Rate{
			icstate.KeyIprep:  icmodule.ToRate(config.RewardFund.Iprep),
			icstate.KeyIcps:   icmodule.ToRate(config.RewardFund.Icps),
			icstate.KeyIrelay: icmodule.ToRate(config.RewardFund.Irelay),
			icstate.KeyIvoter: icmodule.ToRate(config.RewardFund.Ivoter),
		},
	)
	if err := s.SetRewardFund(rf); err != nil {
		return err
	}
	return nil
}

// handleRev6: icmodule.RevisionDecentralize
func (sim *simulatorImpl) handleRev6(ws state.WorldState, r1, r2 int) error {
	es := ws.GetExtensionState().(*iiss.ExtensionStateImpl)
	if termPeriod := es.State.GetTermPeriod(); termPeriod == icmodule.InitialTermPeriod {
		if err := es.State.SetTermPeriod(icmodule.DecentralizedTermPeriod); err != nil {
			return err
		}
	}
	return nil
}

func (sim *simulatorImpl) handleRev9(ws state.WorldState, r1, r2 int) error {
	es := ws.GetExtensionState().(*iiss.ExtensionStateImpl)
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

func (sim *simulatorImpl) handleRev10(ws state.WorldState, r1, r2 int) error {
	return nil
}

func (sim *simulatorImpl) handleRev14(ws state.WorldState, r1, r2 int) error {
	es := ws.GetExtensionState().(*iiss.ExtensionStateImpl)
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

func (sim *simulatorImpl) handleRev15(ws state.WorldState, r1, r2 int) error {
	as := ws.GetAccountState(state.SystemID)

	if err := scoredb.NewVarDB(as, state.VarEnabledEETypes).Set(icon.EETypesJavaAndPython); err != nil {
		return err
	}
	return nil
}

func (sim *simulatorImpl) handleRev17(ws state.WorldState, r1, r2 int) error {
	return nil
}
