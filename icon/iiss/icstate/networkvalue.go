/*
 * Copyright 2020 ICON Foundation
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

package icstate

import (
	"math/big"

	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
)

const (
	VarIRep                                 = "irep"
	VarRRep                                 = "rrep"
	VarMainPRepCount                        = "main_prep_count"
	VarSubPRepCount                         = "sub_prep_count"
	VarExtraMainPRepCount                   = "extra_main_prep_count"
	VarTotalStake                           = "total_stake"
	VarIISSVersion                          = "iiss_version"
	VarTermPeriod                           = "term_period"
	VarBondRequirement                      = "bond_requirement"
	VarBondRequirement2                     = "bond_requirement2"
	VarUnbondingPeriodMultiplier            = "unbonding_period_multiplier"
	VarLockMinMultiplier                    = "lockMinMultiplier"
	VarLockMaxMultiplier                    = "lockMaxMultiplier"
	VarRewardFund                           = "reward_fund"
	VarRewardFund2                          = "reward_fund2"
	VarUnbondingMax                         = "unbonding_max"
	VarValidationPenaltyCondition           = "validation_penalty_condition"
	VarConsistentValidationPenaltyCondition = "consistent_validation_penalty_condition"
	VarConsistentValidationPenaltyMask      = "consistent_validation_penalty_mask"
	VarConsistentValidationPenaltySlashRate = "consistent_validation_penalty_slashRatio"
	VarDelegationSlotMax                    = "delegation_slot_max"
	DictNetworkScores                       = "network_scores"
	VarNonVotePenaltySlashRate              = "nonvote_penalty_slashRatio"
	DictSlashingRate                        = "slashing_rate"
	VarMinBond                              = "minimum_bond"
)

const (
	IISSVersion2 int = iota + 2
	IISSVersion3
	IISSVersion4
)

const (
	CPSKey        = "cps"
	RelayKey      = "relay"
	GovernanceKey = "governance"
)

var AdditionalNetworkScoreKeys = []string{CPSKey, RelayKey}

func getValue(store containerdb.ObjectStoreState, key string) containerdb.Value {
	return containerdb.NewVarDB(
		store,
		containerdb.ToKey(containerdb.HashBuilder, scoredb.VarDBPrefix, key),
	)
}

func setValue(store containerdb.ObjectStoreState, key string, value interface{}) error {
	db := containerdb.NewVarDB(
		store,
		containerdb.ToKey(containerdb.HashBuilder, scoredb.VarDBPrefix, key),
	)
	if err := db.Set(value); err != nil {
		return err
	}
	return nil
}

func (s *State) getDictDB(key string) *containerdb.DictDB {
	return containerdb.NewDictDB(
		s.store,
		1,
		containerdb.ToKey(containerdb.HashBuilder, scoredb.DictDBPrefix, key))
}

func (s *State) SetNetworkScore(role string, address module.Address) error {
	if role == GovernanceKey {
		return scoreresult.AccessDeniedError.New("Permission denied")
	}
	for _, k := range AdditionalNetworkScoreKeys {
		if role == k {
			db := containerdb.NewDictDB(
				s.store,
				1,
				containerdb.ToKey(containerdb.HashBuilder, scoredb.DictDBPrefix, DictNetworkScores))
			if address == nil {
				return db.Delete(role)
			}
			return db.Set(role, address)
		}
	}
	return icmodule.IllegalArgumentError.New("Invalid Network SCORE role")
}

func (s *State) GetNetworkScores(cc icmodule.CallContext) map[string]module.Address {
	networkScores := map[string]module.Address{
		GovernanceKey: cc.Governance(),
	}
	db := containerdb.NewDictDB(
		s.store,
		1,
		containerdb.ToKey(containerdb.HashBuilder, scoredb.DictDBPrefix, DictNetworkScores))
	for _, k := range AdditionalNetworkScoreKeys {
		v := db.Get(k)
		if v == nil {
			continue
		}
		networkScores[k] = v.Address()
	}
	return networkScores
}

func (s *State) GetIISSVersion() int {
	return int(getValue(s.store, VarIISSVersion).Int64())
}

func (s *State) SetIISSVersion(value int) error {
	return setValue(s.store, VarIISSVersion, value)
}

func (s *State) GetTermPeriod() int64 {
	return getValue(s.store, VarTermPeriod).Int64()
}

func (s *State) SetTermPeriod(value int64) error {
	return setValue(s.store, VarTermPeriod, value)
}

func (s *State) GetIRep() *big.Int {
	ret := getValue(s.store, VarIRep).BigInt()
	if ret == nil {
		ret = icmodule.BigIntZero
	}
	return ret
}

func (s *State) SetIRep(value *big.Int) error {
	return setValue(s.store, VarIRep, value)
}

func (s *State) GetRRep() *big.Int {
	ret := getValue(s.store, VarRRep).BigInt()
	if ret == nil {
		ret = icmodule.BigIntZero
	}
	return ret
}

func (s *State) SetRRep(value *big.Int) error {
	return setValue(s.store, VarRRep, value)
}

// GetMainPRepCount returns the number of main preps excluding extra main preps
// This value is the number of main preps as configuration
// If you want to get the actual number of main preps in this term, use termData.MainPRepCount() instead.
func (s *State) GetMainPRepCount() int64 {
	return getValue(s.store, VarMainPRepCount).Int64()
}

func (s *State) SetMainPRepCount(value int64) error {
	if value < 0 {
		return errors.ErrIllegalArgument
	}
	return setValue(s.store, VarMainPRepCount, value)
}

// GetExtraMainPRepCount returns # of extra main preps
// Extra MainPRep means the PRep which plays a validator least recently
func (s *State) GetExtraMainPRepCount() int64 {
	varDB := containerdb.NewVarDB(
		s.store,
		containerdb.ToKey(containerdb.HashBuilder, scoredb.VarDBPrefix, VarExtraMainPRepCount),
	)
	if varDB.Bytes() == nil {
		return icmodule.DefaultExtraMainPRepCount
	}
	return varDB.Int64()
}

func (s *State) SetExtraMainPRepCount(value int64) error {
	if value < 0 {
		return errors.ErrIllegalArgument
	}
	return setValue(s.store, VarExtraMainPRepCount, value)
}

// GetSubPRepCount returns the number of sub preps including extra main preps
func (s *State) GetSubPRepCount() int64 {
	return getValue(s.store, VarSubPRepCount).Int64()
}

func (s *State) SetSubPRepCount(value int64) error {
	if value < 0 {
		return errors.ErrIllegalArgument
	}
	return setValue(s.store, VarSubPRepCount, value)
}

func (s *State) GetPRepCountConfig(revision int) PRepCountConfig {
	mainPReps := s.GetMainPRepCount()
	subPReps := s.GetSubPRepCount()
	extraMainPReps := int64(0)
	if revision >= icmodule.RevisionExtraMainPReps {
		extraMainPReps = s.GetExtraMainPRepCount()
	}
	return prepCountConfig{
		mainPReps:      int(mainPReps),
		subPReps:       int(subPReps),
		extraMainPReps: int(extraMainPReps),
	}
}

func (s *State) GetTotalStake() *big.Int {
	value := getValue(s.store, VarTotalStake).BigInt()
	if value == nil {
		value = new(big.Int)
	}
	return value
}

func (s *State) SetTotalStake(value *big.Int) error {
	return setValue(s.store, VarTotalStake, value)
}

func (s *State) GetBondRequirement(revision int) icmodule.Rate {
	if revision < icmodule.RevisionSetBondRequirementRate {
		return s.getBondRequirementV1()
	} else {
		return s.getBondRequirementV2()
	}
}

func (s *State) getBondRequirementV1() icmodule.Rate {
	v := getValue(s.store, VarBondRequirement).Int64()
	return icmodule.ToRate(v)
}

func (s *State) getBondRequirementV2() icmodule.Rate {
	bs := getValue(s.store, VarBondRequirement2).Bytes()
	if bs == nil {
		return s.getBondRequirementV1()
	}
	brInfo, err := NewBondRequirementInfoFromByte(bs)
	if brInfo == nil {
		// TODO: What to do if brInfo is nil
		s.logger.Panicf("FailedToGetBondRequirementInfo(err=%s)", err.Error())
	}
	return brInfo.Rate()
}

func (s *State) GetBondRequirementInfo(revision int) *BondRequirementInfo {
	if revision < icmodule.RevisionSetBondRequirementRate {
		// BondRequirementInfo is enabled after RevisionSetBondRequirementRate
		return nil
	}

	bs := getValue(s.store, VarBondRequirement2).Bytes()
	if bs == nil {
		rate := s.getBondRequirementV1()
		return NewBondRequirementInfo(rate, rate)
	}
	if info, err := NewBondRequirementInfoFromByte(bs); err == nil {
		return info
	}
	return nil
}

func (s *State) setBondRequirementInfo(revision int, brInfo *BondRequirementInfo) error {
	if revision < icmodule.RevisionSetBondRequirementRate {
		return errors.InvalidStateError.Errorf("SetBondRequirementInfoNotAllowed(rev=%d)", revision)
	}
	if brInfo == nil {
		return scoreresult.InvalidParameterError.New("BondRequirementInfoIsNil")
	}
	if !brInfo.Rate().IsValid() || !brInfo.NextRate().IsValid() {
		return scoreresult.InvalidParameterError.Errorf(
			"InvalidBondRequirementRate(rate=%d,nextRate=%d)",
			brInfo.Rate(), brInfo.NextRate())
	}

	return setValue(s.store, VarBondRequirement2, brInfo.Bytes())
}

func (s *State) SetBondRequirement(revision int, br icmodule.Rate) error {
	if !br.IsValid() {
		return errors.IllegalArgumentError.New("Bond Requirement should range from 0% to 100%")
	}

	if revision < icmodule.RevisionSetBondRequirementRate {
		return setValue(s.store, VarBondRequirement, br.Percent())
	} else {
		brInfo := s.GetBondRequirementInfo(revision)
		if brInfo == nil {
			return errors.InvalidStateError.Errorf("GetBondRequirementInfoFailure(rev=%d)", revision)
		}
		brInfo.SetNextRate(br)
		return s.setBondRequirementInfo(revision, brInfo)
	}
}

func (s *State) MigrateBondRequirement(revision int) error {
	if revision != icmodule.RevisionSetBondRequirementRate {
		return errors.InvalidStateError.Errorf("BondRequirementRateMigrationNotAllowed(rev=%d)", revision)
	}
	rate := s.getBondRequirementV1()
	brInfo := NewBondRequirementInfo(rate, rate)
	return s.setBondRequirementInfo(revision, brInfo)
}

func (s *State) ShiftBondRequirement(revision int) error {
	if revision >= icmodule.RevisionSetBondRequirementRate {
		brInfo := s.GetBondRequirementInfo(revision)
		if brInfo == nil {
			return errors.CriticalUnknownError.Errorf("BondRequirementInfoIsNil(rev=%d)", revision)
		}
		if brInfo.Rate() != brInfo.NextRate() {
			brInfo.SetRate(brInfo.NextRate())
			return s.setBondRequirementInfo(revision, brInfo)
		}
	}
	return nil
}

func (s *State) SetUnbondingPeriodMultiplier(value int64) error {
	if value <= 0 {
		return errors.IllegalArgumentError.New("unbondingPeriodMultiplier must be positive number")
	}
	return setValue(s.store, VarUnbondingPeriodMultiplier, value)
}

func (s *State) GetUnbondingPeriodMultiplier() int64 {
	return getValue(s.store, VarUnbondingPeriodMultiplier).Int64()
}

func (s *State) GetLockMinMultiplier() *big.Int {
	value := getValue(s.store, VarLockMinMultiplier).BigInt()
	return value
}

func (s *State) setLockMinMultiplier(value *big.Int) error {
	if value.Sign() != 1 {
		return errors.IllegalArgumentError.New("LockMinMultiplier must have positive value")
	}
	return setValue(s.store, VarLockMinMultiplier, value)
}

func (s *State) GetLockMaxMultiplier() *big.Int {
	value := getValue(s.store, VarLockMaxMultiplier).BigInt()
	return value
}

func (s *State) setLockMaxMultiplier(value *big.Int) error {
	if value.Sign() != 1 {
		return errors.IllegalArgumentError.New("LockMaxMultiplier must have positive value")
	}
	return setValue(s.store, VarLockMaxMultiplier, value)
}

func (s *State) SetLockVariables(lockMin *big.Int, lockMax *big.Int) error {
	if lockMax.Cmp(lockMin) == -1 {
		return errors.IllegalArgumentError.New("LockMaxMultiplier < LockMinMultiplier")
	}
	if err := s.setLockMinMultiplier(lockMin); err != nil {
		return err
	}
	if err := s.setLockMaxMultiplier(lockMax); err != nil {
		return err
	}
	return nil
}

func (s *State) GetRewardFundV1() *RewardFund {
	bs := getValue(s.store, VarRewardFund).Bytes()
	rc, _ := NewRewardFundFromByte(bs)
	return rc
}

func (s *State) GetRewardFundV2() *RewardFund {
	bs := getValue(s.store, VarRewardFund2).Bytes()
	rc, _ := NewRewardFundFromByte(bs)
	return rc
}

func (s *State) SetRewardFund(r *RewardFund) error {
	switch r.version {
	case RFVersion1:
		return setValue(s.store, VarRewardFund, r.Bytes())
	case RFVersion2:
		return setValue(s.store, VarRewardFund2, r.Bytes())
	default:
		return icmodule.IllegalArgumentError.Errorf("invalid reward fund version %d", r.version)
	}
}

func (s *State) GetRewardFund(revision int) *RewardFund {
	if revision <= icmodule.RevisionIISS4R0 {
		return s.GetRewardFundV1()
	} else {
		return s.GetRewardFundV2()
	}
}

func (s *State) GetUnbondingMax() int64 {
	return getValue(s.store, VarUnbondingMax).Int64()
}

func (s *State) SetUnbondingMax(value int64) error {
	if value <= 0 {
		return errors.IllegalArgumentError.New("UnbondingMax must have positive value")
	}
	return setValue(s.store, VarUnbondingMax, value)
}

func (s *State) GetValidationPenaltyCondition() int64 {
	return getValue(s.store, VarValidationPenaltyCondition).Int64()
}

func (s *State) SetValidationPenaltyCondition(value int) error {
	if value <= 0 {
		return errors.IllegalArgumentError.New("ValidationPenaltyCondition must have positive value")
	}
	return setValue(s.store, VarValidationPenaltyCondition, value)
}

func (s *State) GetConsistentValidationPenaltyCondition() int64 {
	return getValue(s.store, VarConsistentValidationPenaltyCondition).Int64()
}

func (s *State) SetConsistentValidationPenaltyCondition(value int64) error {
	if value <= 0 {
		return errors.IllegalArgumentError.New("ConsistentValidationPenaltyCondition must have positive value")
	}
	return setValue(s.store, VarConsistentValidationPenaltyCondition, value)
}

func (s *State) GetConsistentValidationPenaltyMask() int {
	return int(getValue(s.store, VarConsistentValidationPenaltyMask).Int64())
}

func (s *State) SetConsistentValidationPenaltyMask(value int64) error {
	if value <= 0 || value > 30 {
		return errors.IllegalArgumentError.New("ConsistentValidationPenaltyMask over range(1~30)")
	}
	return setValue(s.store, VarConsistentValidationPenaltyMask, value)
}

func (s *State) getConsistentValidationPenaltySlashRate() icmodule.Rate {
	v := getValue(s.store, VarConsistentValidationPenaltySlashRate).Int64()
	return icmodule.ToRate(v)
}

func (s *State) setConsistentValidationPenaltySlashRate(value icmodule.Rate) error {
	if !value.IsValid() {
		return errors.IllegalArgumentError.New("Invalid range")
	}
	return setValue(s.store, VarConsistentValidationPenaltySlashRate, value.Percent())
}

func (s *State) GetDelegationSlotMax() int {
	value := getValue(s.store, VarDelegationSlotMax).Int64()
	return int(value)
}

func (s *State) SetDelegationSlotMax(value int64) error {
	return setValue(s.store, VarDelegationSlotMax, value)
}

func (s *State) getNonVotePenaltySlashRate() icmodule.Rate {
	v := getValue(s.store, VarNonVotePenaltySlashRate).Int64()
	return icmodule.ToRate(v)
}

func (s *State) setNonVotePenaltySlashRate(value icmodule.Rate) error {
	if !value.IsValid() {
		return errors.IllegalArgumentError.New("Invalid range")
	}
	return setValue(s.store, VarNonVotePenaltySlashRate, value.Percent())
}

func (s *State) GetSlashingRate(revision int, penaltyType icmodule.PenaltyType) (icmodule.Rate, error) {
	if revision < icmodule.RevisionIISS4R0 {
		switch penaltyType {
		case icmodule.PenaltyAccumulatedValidationFailure:
			return s.getConsistentValidationPenaltySlashRate(), nil
		case icmodule.PenaltyMissedNetworkProposalVote:
			return s.getNonVotePenaltySlashRate(), nil
		case icmodule.PenaltyPRepDisqualification:
			return icmodule.ToRate(100), nil
		}
	}
	return s.getSlashingRate(penaltyType)
}

func (s *State) getSlashingRate(penaltyType icmodule.PenaltyType) (icmodule.Rate, error) {
	if !penaltyType.IsValid() {
		return 0, scoreresult.InvalidParameterError.Errorf("InvalidPenaltyType(%d)", penaltyType)
	}
	rate := icmodule.Rate(0)
	db := s.getDictDB(DictSlashingRate)
	if v := db.Get(int(penaltyType)); v != nil {
		rate = icmodule.Rate(v.Int64())
	} else if penaltyType == icmodule.PenaltyPRepDisqualification {
		rate = icmodule.ToRate(100)
	}
	return rate, nil
}

func (s *State) SetSlashingRate(revision int, penaltyType icmodule.PenaltyType, rate icmodule.Rate) error {
	if revision < icmodule.RevisionIISS4R0 {
		switch penaltyType {
		case icmodule.PenaltyAccumulatedValidationFailure:
			return s.setConsistentValidationPenaltySlashRate(rate)
		case icmodule.PenaltyMissedNetworkProposalVote:
			return s.setNonVotePenaltySlashRate(rate)
		}
	}
	return s.setSlashingRate(penaltyType, rate)
}

func (s *State) setSlashingRate(penaltyType icmodule.PenaltyType, rate icmodule.Rate) error {
	if !penaltyType.IsValid() {
		return scoreresult.InvalidParameterError.Errorf("InvalidPenaltyType(%d)", penaltyType)
	}
	if !rate.IsValid() {
		return scoreresult.InvalidParameterError.Errorf("RateOutOfRange(%d)", rate)
	}
	db := s.getDictDB(DictSlashingRate)
	return db.Set(int(penaltyType), rate.NumInt64())
}

// GetMinimumBond returns the minimum bond related to minimum wage
// It returns nil before RevisionIISS4R0
func (s *State) GetMinimumBond() *big.Int {
	ret := getValue(s.store, VarMinBond).BigInt()
	if ret == nil {
		ret = icmodule.BigIntZero
	}
	return ret
}

func (s *State) SetMinimumBond(bond *big.Int) error {
	if bond == nil {
		return scoreresult.InvalidParameterError.Errorf("MinimumBondIsNil")
	}
	if bond.Sign() < 0 {
		return scoreresult.InvalidParameterError.Errorf("NegativeMinimumBond")
	}
	return setValue(s.store, VarMinBond, bond)
}

func (s *State) GetNetworkInfoInJSON(revision int) (map[string]interface{}, error) {
	br := s.GetBondRequirement(revision)
	jso := make(map[string]interface{})
	jso["mainPRepCount"] = s.GetMainPRepCount()
	jso["extraMainPRepCount"] = s.GetExtraMainPRepCount()
	jso["subPRepCount"] = s.GetSubPRepCount()
	jso["totalStake"] = s.GetTotalStake()
	jso["iissVersion"] = int64(s.GetIISSVersion())
	jso["termPeriod"] = s.GetTermPeriod()
	jso["bondRequirement"] = br.Percent()
	jso["lockMinMultiplier"] = s.GetLockMinMultiplier()
	jso["lockMaxMultiplier"] = s.GetLockMaxMultiplier()
	jso["rewardFund"] = s.GetRewardFund(revision).ToJSON()
	if revision == icmodule.RevisionIISS4R0 {
		// Under RevisionIISS4R0, RewardFundV1 and RewardFundV2 coexist
		// jso["rewardFund"] contains RewardFundV1
		// jso["rewardFund2"] contains RewardFundV2
		jso["rewardFund2"] = s.GetRewardFundV2().ToJSON()
	}
	jso["unbondingMax"] = s.GetUnbondingMax()
	jso["unbondingPeriodMultiplier"] = s.GetUnbondingPeriodMultiplier()
	jso["validationPenaltyCondition"] = s.GetValidationPenaltyCondition()
	jso["consistentValidationPenaltyCondition"] = s.GetConsistentValidationPenaltyCondition()
	jso["consistentValidationPenaltyMask"] = s.GetConsistentValidationPenaltyMask()
	jso["unstakeSlotMax"] = s.GetUnstakeSlotMax()
	jso["delegationSlotMax"] = s.GetDelegationSlotMax()

	if revision < icmodule.RevisionIISS4R0 {
		jso["irep"] = s.GetIRep()
		jso["rrep"] = s.GetRRep()
		rate, _ := s.GetSlashingRate(revision, icmodule.PenaltyAccumulatedValidationFailure)
		jso["consistentValidationPenaltySlashRatio"] = rate.Percent()
		rate, _ = s.GetSlashingRate(revision, icmodule.PenaltyMissedNetworkProposalVote)
		jso["proposalNonVotePenaltySlashRatio"] = rate.Percent()
	} else {
		jso["minimumBond"] = s.GetMinimumBond()
	}

	if preps := s.GetPReps(true); preps != nil {
		totalBonded := new(big.Int)
		totalDelegated := new(big.Int)
		totalPower := new(big.Int)

		for _, prep := range preps {
			totalBonded.Add(totalBonded, prep.Bonded())
			totalDelegated.Add(totalDelegated, prep.Delegated())
			totalPower.Add(totalPower, prep.GetPower(br))
		}

		jso["totalBonded"] = totalBonded
		jso["totalDelegated"] = totalDelegated
		jso["totalPower"] = totalPower
		jso["preps"] = len(preps)
	}
	return jso, nil
}
