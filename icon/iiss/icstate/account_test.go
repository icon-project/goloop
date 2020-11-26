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

var assTest = &AccountSnapshot{
	staked: big.NewInt(100),
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
	delegated: big.NewInt(20),
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
	bonded: big.NewInt(20),
	bonds: []*Bond{
		{
			Target: common.NewAddressFromString("hx3"),
			Amount: big.NewInt(10),
		},
		{
			Target: common.NewAddressFromString("hx4"),
			Amount: big.NewInt(10),
		},
	},
	unbonds: []*Unbond{
		{
			Target:       common.NewAddressFromString("hx5"),
			Amount:       big.NewInt(10),
			ExpireHeight: 20,
		},
		{
			Target:       common.NewAddressFromString("hx6"),
			Amount:       big.NewInt(10),
			ExpireHeight: 30,
		},
	},
}

func TestAccountSnapshot_Bytes(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)

	o1 := icobject.New(TypeAccount, assTest)
	serialized := o1.Bytes()

	t.Logf("Serialized:%v", serialized)

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())

	ass2 := ToAccountSnapshot(o2)
	assert.Equal(t, true, assTest.Equal(ass2))
}
