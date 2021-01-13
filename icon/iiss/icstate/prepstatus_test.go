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
	"github.com/bmizerany/assert"
	"github.com/icon-project/goloop/common"
	"math/big"
	"testing"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestPRepStatus_Bytes(t *testing.T) {
	owner := common.NewAccountAddress(make([]byte, common.AddressIDBytes, common.AddressIDBytes))
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	ss1 := newPRepStatusWithTag(icobject.MakeTag(TypePRepStatus, prepStatusVersion))
	g := Candidate
	ss1.grade = g
	ss1.SetOwner(owner)

	o1 := icobject.New(TypePRepStatus, ss1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())

	ss2 := ToPRepStatus(o2, owner)
	assert.Equal(t, true, ss1.Equal(ss2))
	assert.Equal(t, false, ss2.readonly)
	assert.Equal(t, true, ss1.owner.Equal(owner))
	assert.Equal(t, true, ss2.owner.Equal(owner))
}

// test for GetBondedDelegation
func TestPRepManager_GetBondedDelegation(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	s := NewStateFromSnapshot(NewSnapshot(database, nil), false)

	addr1 := common.NewAddressFromString("hx1")

	status1 := NewPRepStatus(addr1)
	base := NewPRepBase(addr1)
	s.AddPRepBase(base)
	s.AddPRepStatus(status1)

	delegated := big.NewInt(int64(99))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded := big.NewInt(int64(1))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res := status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(20))))

	delegated = big.NewInt(int64(99))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(2))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(40))))

	delegated = big.NewInt(int64(93))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(7))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(100))))

	delegated = big.NewInt(int64(90))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(10))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(100))))

	// 0 input, exptected 0 output
	delegated = big.NewInt(int64(0))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(0))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(0))))

	// extreme
	delegated = big.NewInt(int64(99999999999))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(5)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(19980))))

	// different requirement
	delegated = big.NewInt(int64(99999))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(4)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(24975))))

	// 0 for bond requirement
	delegated = big.NewInt(int64(99999))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(0)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(0))))

	// 101 for bond requirement
	delegated = big.NewInt(int64(99999))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(101)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(0))))

	// 0 for bond requirement
	delegated = big.NewInt(int64(99999))
	s.GetPRepStatus(addr1).SetDelegated(delegated)
	bonded = big.NewInt(int64(999))
	s.GetPRepStatus(addr1).SetBonded(bonded)
	res = status1.GetBondedDelegation(100)
	assert.Equal(t, 0, res.Cmp(big.NewInt(int64(999))))
}