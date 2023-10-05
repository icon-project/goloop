package icsim

import "github.com/icon-project/goloop/icon/icmodule"

type RewardFund struct {
	Iglobal int64
	Iprep   int64
	Icps    int64
	Irelay  int64
	Ivoter  int64
}

type config struct {
	TermPeriod                           int64
	MainPRepCount                        int64
	SubPRepCount                         int64
	ExtraMainPRepCount                   int64
	Irep                                 int64
	Rrep                                 int64
	BondRequirement                      int64
	UnbondingPeriodMultiplier            int64
	UnstakeSlotMax                       int64
	LockMinMultiplier                    int64
	LockMaxMultiplier                    int64
	UnbondingMax                         int64
	ValidationPenaltyCondition           int
	ConsistentValidationPenaltyCondition int64
	ConsistentValidationPenaltyMask      int64
	ConsistentValidationPenaltySlashRate icmodule.Rate
	NonVotePenaltySlashRate              icmodule.Rate
	DelegationSlotMax                    int64
	RewardFund
	BondedPRepCount int
}

func NewConfig() *config {
	return &config{
		TermPeriod:                           icmodule.DecentralizedTermPeriod,
		MainPRepCount:                        icmodule.DefaultMainPRepCount,
		SubPRepCount:                         icmodule.DefaultSubPRepCount,
		ExtraMainPRepCount:                   icmodule.DefaultExtraMainPRepCount,
		Irep:                                 icmodule.InitialIRep,
		Rrep:                                 0,
		BondRequirement:                      0,
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
