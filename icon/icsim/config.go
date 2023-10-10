package icsim

import (
	"strings"

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
	BondRequirement                      icmodule.Rate
	UnbondingPeriodMultiplier            int64
	UnstakeSlotMax                       int64
	LockMinMultiplier                    int64
	LockMaxMultiplier                    int64
	UnbondingMax                         int64
	ValidationPenaltyCondition           int64
	ConsistentValidationPenaltyCondition int64
	ConsistentValidationPenaltyMask      int64
	ConsistentValidationPenaltySlashRate icmodule.Rate
	NonVotePenaltySlashRate              icmodule.Rate
	DelegationSlotMax                    int64
	RewardFund
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

func NewSimConfigWithParams(params map[string]interface{}) *SimConfig {
	cfg := NewSimConfig()
	for k, v := range params {
		k = strings.ToLower(k)
		switch k {
		case "br", strings.ToLower("BondRequirement"):
			cfg.BondRequirement = v.(icmodule.Rate)
		case "tp", "tperiod", strings.ToLower("TermPeriod"):
			cfg.TermPeriod = v.(int64)
		case "vpc", strings.ToLower("ValidationPenaltyCondition"):
			cfg.ValidationPenaltyCondition = v.(int64)
		case "cvpc", strings.ToLower("ConsistentValidationPenaltyCondition"):
			cfg.ConsistentValidationPenaltyCondition = v.(int64)
		case strings.ToLower("ConsistentValidationPenaltySlashRate"):
			cfg.ConsistentValidationPenaltySlashRate = v.(icmodule.Rate)
		case strings.ToLower("NonVotePenaltySlashRate"):
			cfg.NonVotePenaltySlashRate = v.(icmodule.Rate)
		case strings.ToLower("MainPReps"), strings.ToLower("MainPRepCount"):
			cfg.MainPRepCount = v.(int64)
		case strings.ToLower("SubPReps"), strings.ToLower("SubPRepCount"):
			cfg.SubPRepCount = v.(int64)
		case strings.ToLower("ExtraMainPReps"), strings.ToLower("ExtraMainPRepCount"):
			cfg.ExtraMainPRepCount = v.(int64)
		}
	}

	return cfg
}
