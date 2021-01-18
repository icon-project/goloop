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
	"github.com/icon-project/goloop/common/errors"
	"math/big"

	"github.com/icon-project/goloop/common/containerdb"
)

const (
	VarIRep            = "irep"
	VarRRep            = "rrep"
	VarMainPRepCount   = "main_prep_count"
	VarSubPRepCount    = "sub_prep_count"
	VarTotalStake      = "total_stake"
	VarTermPeriod      = "term_period"
	VarCalculatePeriod = "calculate_period"
	VarBondRequirement = "bond_requirement"
	VarLockMin         = "lockMin"
	VarLockMax         = "lockMax"
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

func GetTermPeriod(store containerdb.ObjectStoreState) int64 {
	return getValue(store, VarTermPeriod).Int64()
}

func SetTermPeriod(s *State, value int64) error {
	return setValue(s.store, VarTermPeriod, value)
}

func GetCalculatePeriod(s *State) int64 {
	return getValue(s.store, VarCalculatePeriod).Int64()
}

func SetCalculatePeriod(s *State, value int64) error {
	return setValue(s.store, VarCalculatePeriod, value)
}

func GetIRep(s *State) *big.Int {
	return getValue(s.store, VarIRep).BigInt()
}

func SetIRep(s *State, value *big.Int) error {
	return setValue(s.store, VarIRep, value)
}

func GetRRep(s *State) *big.Int {
	return getValue(s.store, VarRRep).BigInt()
}

func SetRRep(s *State, value *big.Int) error {
	return setValue(s.store, VarRRep, value)
}

func (s *State) GetMainPRepCount() int {
	return int(GetMainPRepCount(s))
}

func GetMainPRepCount(s *State) int64 {
	return getValue(s.store, VarMainPRepCount).Int64()
}

func SetMainPRepCount(s *State, value int64) error {
	return setValue(s.store, VarMainPRepCount, value)
}

func (s *State) GetSubPRepCount() int {
	return int(GetSubPRepCount(s))
}

func GetSubPRepCount(s *State) int64 {
	return getValue(s.store, VarSubPRepCount).Int64()
}

func SetSubPRepCount(s *State, value int64) error {
	return setValue(s.store, VarSubPRepCount, value)
}

func GetPRepCount(s *State) int64 {
	return GetMainPRepCount(s) + GetSubPRepCount(s)
}

func GetTotalStake(s *State) *big.Int {
	value := getValue(s.store, VarTotalStake).BigInt()
	if value == nil {
		value = new(big.Int)
	}
	return value
}

func SetTotalStake(s *State, value *big.Int) error {
	return setValue(s.store, VarTotalStake, value)
}

func (s *State) GetBondRequirement() int {
	return int(GetBondRequirement(s))
}

func GetBondRequirement(s *State) int64 {
	return getValue(s.store, VarBondRequirement).Int64()
}

func (s *State) SetBondRequirement(value int64) {
	SetBondRequirement(s, value)
}

func SetBondRequirement(s *State, value int64) error {
	if value < 1 || value > 100 {
		return errors.IllegalArgumentError.New("Bond Requirement should range from 1 to 100")
	}
	return setValue(s.store, VarBondRequirement, value)
}

func GetLockMin(s *State) *big.Int {
	value := getValue(s.store, VarLockMin).BigInt()
	return value
}

func setLockMin(s *State, value *big.Int) error {
	if value.Sign() != 1 {
		return errors.IllegalArgumentError.New("LockMin must have positive value")
	}
	return setValue(s.store, VarLockMin, value)
}

func GetLockMax(s *State) *big.Int {
	value := getValue(s.store, VarLockMax).BigInt()
	return value
}

func setLockMax(s *State, value *big.Int) error {
	if value.Sign() != 1 {
		return errors.IllegalArgumentError.New("LockMax must have positive value")
	}
	return setValue(s.store, VarLockMin, value)
}

func SetLockVariables(s *State, lockMin *big.Int, lockMax *big.Int) error {
	if lockMax.Cmp(lockMin) == -1 {
		return errors.IllegalArgumentError.New("LockMax < LockMin")
	}
	if err := setLockMin(s, lockMin); err != nil {
		return err
	}
	if err := setLockMax(s, lockMax); err != nil {
		return err
	}
	return nil
}
