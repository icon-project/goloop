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

type handleRevFunc func(sim *simulatorImpl, wc WorldContext, rev, targetRev int) error
type revHandlerItem struct {
	rev int
	fn  handleRevFunc
}

var revHandlerTable = []revHandlerItem{
	{icmodule.RevisionIISS, onRevIISS},
	{icmodule.RevisionDecentralize, onRevDecentralize},
	{icmodule.RevisionIISS2, onRevIISS2},
	//{icmodule.RevisionICON2R0, onRevICON2R0},
	{icmodule.RevisionICON2R1, onRevICON2R1},
	{icmodule.RevisionICON2R2, onRevEnableJavaEE},
	{icmodule.RevisionICON2R3, onRevICON2R3},
	//{icmodule.RevisionBlockAccounts2, onRevBlockAccounts2},
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

func (sim *simulatorImpl) handleRevisionChange(wc WorldContext, oldRev, newRev int) error {
	if oldRev >= newRev {
		return nil
	}
	for rev := oldRev + 1; rev <= newRev; rev++ {
		if items, ok := revHandlerMap[rev]; ok {
			for _, item := range items {
				if err := item.fn(sim, wc, rev, newRev); err != nil {
					return err
				}
			}
		}
	}
	sim.revision = icmodule.ValueToRevision(newRev)
	return nil
}

func onRevIISS(sim *simulatorImpl, wc WorldContext, rev, _ int) error {
	as := wc.GetAccountState(state.SystemID)

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
	wc.GetExtensionState().Reset(iiss.NewExtensionSnapshot(wc.Database(), nil))
	es := getExtensionState(wc)

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
	if err := es.State.SetBondRequirement(rev, cfg.BondRequirement); err != nil {
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
	if err := es.State.SetValidationPenaltyCondition(int(cfg.ValidationPenaltyCondition)); err != nil {
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
		rev,
		icmodule.PenaltyAccumulatedValidationFailure,
		cfg.ConsistentValidationPenaltySlashRate); err != nil {
		return err
	}

	return es.GenesisTerm(sim.blockHeight, rev)
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

func onRevDecentralize(_ *simulatorImpl, wc WorldContext, _, _ int) error {
	es := getExtensionState(wc)
	if termPeriod := es.State.GetTermPeriod(); termPeriod == icmodule.InitialTermPeriod {
		if err := es.State.SetTermPeriod(icmodule.DecentralizedTermPeriod); err != nil {
			return err
		}
	}
	return nil
}

func onRevIISS2(_ *simulatorImpl, wc WorldContext, _, _ int) error {
	es := getExtensionState(wc)
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

func onRevICON2R1(_ *simulatorImpl, wc WorldContext, rev, _ int) error {
	es := getExtensionState(wc)
	as := wc.GetAccountState(state.SystemID)

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
	if es.State.GetBondRequirement(rev) == icmodule.ToRate(icmodule.IISS2BondRequirement) {
		if err := es.State.SetBondRequirement(rev, icmodule.ToRate(icmodule.DefaultBondRequirement)); err != nil {
			return err
		}
	}
	if err := es.ClearPRepIllegalDelegated(); err != nil {
		return err
	}
	return nil
}

func onRevEnableJavaEE(_ *simulatorImpl, wc WorldContext, _, _ int) error {
	as := wc.GetAccountState(state.SystemID)

	if err := scoredb.NewVarDB(as, state.VarEnabledEETypes).Set(icon.EETypesJavaAndPython); err != nil {
		return err
	}
	return nil
}

func onRevICON2R3(sim *simulatorImpl, wc WorldContext, _, _ int) error {
	revision := icmodule.Revision17
	cfg := sim.config
	es := getExtensionState(wc)

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

func onRevIISS4R0(_ *simulatorImpl, wc WorldContext, _, _ int) error {
	es := getExtensionState(wc)

	// RewardFundAllocation2
	r := es.State.GetRewardFundV1()
	return es.State.SetRewardFund(r.ToRewardFundV2())
}

func onRevIISS4R1(_ *simulatorImpl, wc WorldContext, _, _ int) error {
	es := getExtensionState(wc)
	return es.State.SetIISSVersion(icstate.IISSVersion4)
}
