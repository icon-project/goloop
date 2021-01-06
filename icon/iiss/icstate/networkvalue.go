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
)

const (
	VarIRep          = "irep"
	VarRRep          = "rrep"
	VarMainPRepCount = "main_prep_count"
	VarSubPRepCount  = "sub_prep_count"
	VarTotalStake    = "total_stake"
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

func GetMainPRepCount(s *State) int64 {
	return getValue(s.store, VarMainPRepCount).Int64()
}

func SetMainPRepCount(s *State, value int64) error {
	return setValue(s.store, VarMainPRepCount, value)
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
	return getValue(s.store, VarTotalStake).BigInt()
}

func SetTotalStake(s *State, value *big.Int) error {
	return setValue(s.store, VarTotalStake, value)
}
