package icsim

import (
	"github.com/icon-project/goloop/icon/icmodule"
)

type RewardFund struct {
	Iglobal int64
	Iprep   int64
	Icps    int64
	Irelay  int64
	Ivoter  int64
}

type SimConfig struct {
	TermPeriod                           int64
	MainPRepCount                        int64
	SubPRepCount                         int64
	ExtraMainPRepCount                   int64
	Irep                                 int64
	Rrep                                 int64
	UnbondingPeriodMultiplier            int64
	UnstakeSlotMax                       int64
	LockMinMultiplier                    int64
	LockMaxMultiplier                    int64
	UnbondingMax                         int64
	ValidationPenaltyCondition           int64
	DelegationSlotMax                    int64
	ConsistentValidationPenaltyCondition int64
	ConsistentValidationPenaltyMask      int64
	ConsistentValidationPenaltySlashRate icmodule.Rate
	NonVotePenaltySlashRate              icmodule.Rate
	BondRequirement                      icmodule.Rate
	RewardFund
}

func (cfg *SimConfig) TotalMainPRepCount() int64 {
	return cfg.MainPRepCount + cfg.ExtraMainPRepCount
}

func NewSimConfig() *SimConfig {
	return &SimConfig{
		TermPeriod:                           icmodule.DecentralizedTermPeriod,
		MainPRepCount:                        icmodule.DefaultMainPRepCount,
		SubPRepCount:                         icmodule.DefaultSubPRepCount,
		ExtraMainPRepCount:                   icmodule.DefaultExtraMainPRepCount,
		Irep:                                 icmodule.InitialIRep,
		Rrep:                                 0,
		BondRequirement:                      icmodule.ToRate(icmodule.DefaultBondRequirement),
		UnbondingPeriodMultiplier:            icmodule.DefaultUnbondingPeriodMultiplier,
		UnstakeSlotMax:                       icmodule.InitialUnstakeSlotMax,
		LockMinMultiplier:                    icmodule.DefaultLockMinMultiplier,
		LockMaxMultiplier:                    icmodule.DefaultLockMaxMultiplier,
		UnbondingMax:                         icmodule.DefaultUnbondingMax,
		ValidationPenaltyCondition:           icmodule.DefaultValidationPenaltyCondition,
		ConsistentValidationPenaltyCondition: icmodule.DefaultConsistentValidationPenaltyCondition,
		ConsistentValidationPenaltyMask:      icmodule.DefaultConsistentValidationPenaltyMask,
		ConsistentValidationPenaltySlashRate: icmodule.ToRate(icmodule.DefaultConsistentValidationPenaltySlashRate),
		NonVotePenaltySlashRate:              icmodule.ToRate(icmodule.DefaultNonVotePenaltySlashRate),
		DelegationSlotMax:                    icmodule.DefaultDelegationSlotMax,
		RewardFund: RewardFund{
			Iglobal: icmodule.DefaultIglobal,
			Iprep:   icmodule.DefaultIprep,
			Ivoter:  icmodule.DefaultIvoter,
			Icps:    icmodule.DefaultIcps,
			Irelay:  icmodule.DefaultIrelay,
		},
	}
}

type SimConfigOption int

const (
	SCOBondRequirement SimConfigOption = iota
	SCOTermPeriod
	SCOValidationFailurePenaltyCondition
	SCOAccumulatedValidationFailurePenaltyCondition
	SCOAccumulatedValidationFailurePenaltySlashingRate
	SCOMissedNetworkProposalVotePenaltySlashingRate
	SCOMainPReps
	SCOSubPReps
	SCOExtraMainPReps
)

func NewSimConfigWithParams(params map[SimConfigOption]interface{}) *SimConfig {
	cfg := NewSimConfig()
	for k, v := range params {
		switch k {
		case SCOBondRequirement:
			cfg.BondRequirement = v.(icmodule.Rate)
		case SCOTermPeriod:
			cfg.TermPeriod = v.(int64)
		case SCOValidationFailurePenaltyCondition:
			cfg.ValidationPenaltyCondition = v.(int64)
		case SCOAccumulatedValidationFailurePenaltyCondition:
			cfg.ConsistentValidationPenaltyCondition = v.(int64)
		case SCOAccumulatedValidationFailurePenaltySlashingRate:
			cfg.ConsistentValidationPenaltySlashRate = v.(icmodule.Rate)
		case SCOMissedNetworkProposalVotePenaltySlashingRate:
			cfg.NonVotePenaltySlashRate = v.(icmodule.Rate)
		case SCOMainPReps:
			cfg.MainPRepCount = v.(int64)
		case SCOSubPReps:
			cfg.SubPRepCount = v.(int64)
		case SCOExtraMainPReps:
			cfg.ExtraMainPRepCount = v.(int64)
		}
	}

	return cfg
}
