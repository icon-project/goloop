/*
 * Copyright 2020 ICON Foundation
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/icon-project/goloop/common/intconv"
)

const (
	VarIRep            = "irep"
	VarRRep            = "rrep"
	VarMainPRepCount   = "main_prep_count"
	VarSubPRepCount    = "sub_prep_count"
	VarTotalStake      = "total_stake"
	VarIISSVersion     = "iiss_version"
	VarTermPeriod      = "term_period"
	VarCalculatePeriod = "calculate_period"
	VarBondRequirement = "bond_requirement"
	VarLockMin         = "lockMin"
	VarLockMax         = "lockMax"
	VarRewardFund      = "reward_fund"
)

const (
	IISSVersion1 int = iota
	IISSVersion2
)

func getValue(store containerdb.ObjectStoreState, key string) containerdb.Value {
	return containerdb.NewVarDB(
		store,
		containerdb.ToKey(containerdb.HashBuilder, key),
	)
}

func setValue(store containerdb.ObjectStoreState, key string, value interface{}) error {
	db := containerdb.NewVarDB(
		store,
		containerdb.ToKey(containerdb.HashBuilder, key),
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

func (s *State) GetCalculatePeriod() int64 {
	return getValue(s.store, VarCalculatePeriod).Int64()
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

// PrepCount
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

func (s *State) GetLockMin() *big.Int {
	value := getValue(s.store, VarLockMin).BigInt()
	return value
}

func (s *State) setLockMin(value *big.Int) error {
	if value.Sign() != 1 {
		return errors.IllegalArgumentError.New("LockMin must have positive value")
	}
	return setValue(s.store, VarLockMin, value)
}

func (s *State) GetLockMax() *big.Int {
	value := getValue(s.store, VarLockMax).BigInt()
	return value
}

func (s *State) setLockMax(value *big.Int) error {
	if value.Sign() != 1 {
		return errors.IllegalArgumentError.New("LockMax must have positive value")
	}
	return setValue(s.store, VarLockMax, value)
}

func (s *State) SetLockVariables(lockMin *big.Int, lockMax *big.Int) error {
	if lockMax.Cmp(lockMin) == -1 {
		return errors.IllegalArgumentError.New("LockMax < LockMin")
	}
	if err := s.setLockMin(lockMin); err != nil {
		return err
	}
	if err := s.setLockMax(lockMax); err != nil {
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

func NetworkValueToJSON(s *State) map[string]interface{} {
	jso := make(map[string]interface{})
	jso["irep"] = intconv.FormatBigInt(s.GetIRep())
	jso["rrep"] = intconv.FormatBigInt(s.GetRRep())
	jso["mainPRepCount"] = intconv.FormatInt(s.GetMainPRepCount())
	jso["subPRepCount"] = intconv.FormatInt(s.GetMainPRepCount())
	jso["totalStake"] = intconv.FormatBigInt(s.GetTotalStake())
	jso["iissVersion"] = intconv.FormatInt(int64(s.GetIISSVersion()))
	jso["termPeriod"] = intconv.FormatInt(s.GetTermPeriod())
	jso["calculationPeriod"] = intconv.FormatInt(s.GetCalculatePeriod())
	jso["bondRequirement"] = intconv.FormatInt(s.GetBondRequirement())
	jso["lockMin"] = intconv.FormatBigInt(s.GetLockMin())
	jso["lockMAX"] = intconv.FormatBigInt(s.GetLockMax())
	jso["rewardFund"] = s.GetRewardFund().ToJSON()
	return jso
}
