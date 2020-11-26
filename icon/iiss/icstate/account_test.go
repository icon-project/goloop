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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

var assTest = &Account{
	stake: big.NewInt(100),
	unstakes: []*Unstake{
{
	Amount:       big.NewInt(5),
	ExpireHeight: 10,
},
{
	Amount:       big.NewInt(10),
	ExpireHeight: 20,
},
},
	delegating: big.NewInt(20),
	delegations: []*Delegation{
{
	Address: common.NewAddressFromString("hx1"),
	Value:   common.NewHexInt(10),
},
{
	Address: common.NewAddressFromString("hx2"),
	Value:   common.NewHexInt(10),
},
},
	bonding: big.NewInt(20),
	bonds: []*Bond{
{
	Address: common.NewAddressFromString("hx3"),
	Value:   common.NewHexInt(10),
},
{
	Address: common.NewAddressFromString("hx4"),
	Value:   common.NewHexInt(10),
},
},
	unbonds: []*Unbond{
{
	Address: common.NewAddressFromString("hx5"),
	Value:   big.NewInt(10),
	Expire:  20,
},
{
	Address: common.NewAddressFromString("hx6"),
	Value:   big.NewInt(10),
	Expire:  30,
},
},
}

func TestAccount_Bytes(t *testing.T) {
	address, err := common.NewAddress(make([]byte, common.AddressBytes, common.AddressBytes))
	if err != nil {
		t.Errorf("Failed to create an address")
	}
	assTest.SetAddress(address)
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)

	o1 := icobject.New(TypeAccount, assTest.GetSnapshot())
	serialized := o1.Bytes()

	t.Logf("Serialized:%v", serialized)

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())

	ass2 := ToAccount(o2, address)
	assert.Equal(t, true, assTest.Equal(ass2))
}
