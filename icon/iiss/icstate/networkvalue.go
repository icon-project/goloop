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
	"github.com/icon-project/goloop/service/scoredb"
	"math/big"

	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
)

const (
	VarIRep                                  = "irep"
	VarRRep                                  = "rrep"
	VarMainPRepCount                         = "main_prep_count"
	VarSubPRepCount                          = "sub_prep_count"
	VarTotalStake                            = "total_stake"
	VarIISSVersion                           = "iiss_version"
	VarTermPeriod                            = "term_period"
	VarBondRequirement                       = "bond_requirement"
	VarUnbondingPeriodMultiplier             = "unbonding_period_multiplier"
	VarLockMinMultiplier                     = "lockMinMultiplier"
	VarLockMaxMultiplier                     = "lockMaxMultiplier"
	VarRewardFund                            = "reward_fund"
	VarUnbondingMax                          = "unbonding_max"
	VarValidationPenaltyCondition            = "validation_penalty_condition"
	VarConsistentValidationPenaltyCondition  = "consistent_validation_penalty_condition"
	VarConsistentValidationPenaltyMask       = "consistent_validation_penalty_mask"
	VarConsistentValidationPenaltySlashRatio = "consistent_validation_penalty_slashRatio"
)

const (
	IISSVersion1 int = iota // IISS 2.0
	IISSVersion2            // IISS 3.1
)

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
	return getValue(s.store, VarIRep).BigInt()
}

func (s *State) SetIRep(value *big.Int) error {
	return setValue(s.store, VarIRep, value)
}

func (s *State) GetRRep() *big.Int {
	return getValue(s.store, VarRRep).BigInt()
}

func (s *State) SetRRep(value *big.Int) error {
	return setValue(s.store, VarRRep, value)
}

// MainPrepCount
func (s *State) GetMainPRepCount() int64 {
	return getValue(s.store, VarMainPRepCount).Int64()
}

func (s *State) SetMainPRepCount(value int64) error {
	return setValue(s.store, VarMainPRepCount, value)
}

// SubPrepCount
func (s *State) GetSubPRepCount() int64 {
	return getValue(s.store, VarSubPRepCount).Int64()
}

func (s *State) SetSubPRepCount(value int64) error {
	return setValue(s.store, VarSubPRepCount, value)
}

// GetPRepCount returns the number of mainPReps and subPReps based on ICON Network Value
func (s *State) GetPRepCount() int64 {
	return s.GetMainPRepCount() + s.GetSubPRepCount()
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

func (s *State) GetBondRequirement() int64 {
	return getValue(s.store, VarBondRequirement).Int64()
}

func (s *State) SetBondRequirement(value int64) error {
	if value < 0 || value > 100 {
		return errors.IllegalArgumentError.New("Bond Requirement should range from 0 to 100")
	}
	return setValue(s.store, VarBondRequirement, value)
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

func (s *State) GetRewardFund() *RewardFund {
	bs := getValue(s.store, VarRewardFund).Bytes()
	rc, _ := newRewardFundFromByte(bs)
	return rc
}

func (s *State) SetRewardFund(rc *RewardFund) error {
	return setValue(s.store, VarRewardFund, rc.Bytes())
}

func (s *State) GetUnbondingMax() *big.Int {
	value := getValue(s.store, VarUnbondingMax).BigInt()
	return value
}

func (s *State) SetUnbondingMax(value *big.Int) error {
	if value.Sign() != 1 {
		return errors.IllegalArgumentError.New("UnbondingMax must have positive value")
	}
	return setValue(s.store, VarUnbondingMax, value)
}

func (s *State) GetValidationPenaltyCondition() *big.Int {
	value := getValue(s.store, VarValidationPenaltyCondition).BigInt()
	return value
}

func (s *State) SetValidationPenaltyCondition(value *big.Int) error {
	if value.Sign() != 1 {
		return errors.IllegalArgumentError.New("ValidationPenaltyCondition must have positive value")
	}
	return setValue(s.store, VarValidationPenaltyCondition, value)
}

func (s *State) GetConsistentValidationPenaltyCondition() *big.Int {
	value := getValue(s.store, VarConsistentValidationPenaltyCondition).BigInt()
	return value
}

func (s *State) SetConsistentValidationPenaltyCondition(value *big.Int) error {
	if value.Sign() != 1 {
		return errors.IllegalArgumentError.New("ConsistentValidationPenaltyCondition must have positive value")
	}
	return setValue(s.store, VarConsistentValidationPenaltyCondition, value)
}

func (s *State) GetConsistentValidationPenaltyMask() *big.Int {
	value := getValue(s.store, VarConsistentValidationPenaltyMask).BigInt()
	return value
}

func (s *State) SetConsistentValidationPenaltyMask(value *big.Int) error {
	if value.Sign() != 1 || value.Int64() > 30 {
		return errors.IllegalArgumentError.New("ConsistentValidationPenaltyMask over range(1~30)")
	}
	return setValue(s.store, VarConsistentValidationPenaltyMask, value)
}

func (s *State) GetConsistentValidationPenaltySlashRatio() *big.Int {
	value := getValue(s.store, VarConsistentValidationPenaltySlashRatio).BigInt()
	return value
}

func (s *State) SetConsistentValidationPenaltySlashRatio(value *big.Int) error {
	if value.Sign() != 1 {
		return errors.IllegalArgumentError.New("ConsistentValidationPenaltySlashRatio must have positive value")
	}
	return setValue(s.store, VarConsistentValidationPenaltySlashRatio, value)
}

func NetworkValueToJSON(s *State) map[string]interface{} {
	jso := make(map[string]interface{})
	jso["irep"] = s.GetIRep()
	jso["rrep"] = s.GetRRep()
	jso["mainPRepCount"] = s.GetMainPRepCount()
	jso["subPRepCount"] = s.GetSubPRepCount()
	jso["totalStake"] = s.GetTotalStake()
	jso["iissVersion"] = int64(s.GetIISSVersion())
	jso["termPeriod"] = s.GetTermPeriod()
	jso["bondRequirement"] = s.GetBondRequirement()
	jso["lockMinMultiplier"] = s.GetLockMinMultiplier()
	jso["lockMaxMultiplier"] = s.GetLockMaxMultiplier()
	jso["rewardFund"] = s.GetRewardFund().ToJSON()
	jso["unbondingMax"] = s.GetUnbondingMax()
	jso["unbondingPeriodMultiplier"] = s.GetUnbondingPeriodMultiplier()
	jso["validationPenaltyCondition"] = s.GetValidationPenaltyCondition()
	jso["consistentValidationPenaltyCondition"] = s.GetConsistentValidationPenaltyCondition()
	jso["consistentValidationPenaltyMask"] = s.GetConsistentValidationPenaltyMask()
	jso["consistentValidationPenaltySlashRatio"] = s.GetConsistentValidationPenaltySlashRatio()
	return jso
}
