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

package icreward

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestVoted(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)

	type_ := TypeVoted
	version := 0
	v1 := int64(100)

	t1 := newVoted(icobject.MakeTag(type_, version))
	t1.SetDelegated(big.NewInt(v1))
	t1.SetBondedDelegation(big.NewInt(v1))

	o1 := icobject.New(type_, t1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())
	assert.Equal(t, type_, o2.Tag().Type())
	assert.Equal(t, version, o2.Tag().Version())

	t2 := ToVoted(o2)
	assert.Equal(t, true, t1.Equal(t2))
	assert.Equal(t, VotedVersion1, t2.Version())
	assert.Equal(t, 0, t1.Delegated().Cmp(t2.Delegated()))
	assert.Equal(t, 0, t1.BondedDelegation().Cmp(t2.BondedDelegation()))

	// v1 -> v2
	commissionRate := icmodule.Rate(1000)
	t2.SetCommissionRate(commissionRate)

	o2 = icobject.New(type_, t2)
	o3 := new(icobject.Object)
	if err := o3.Reset(database, o2.Bytes()); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	t3 := ToVoted(o3)
	assert.True(t, t3.Equal(t2))
	assert.Equal(t, VotedVersion2, t3.Version())
	assert.Equal(t, 0, t3.Delegated().Cmp(t2.Delegated()))
	assert.Equal(t, 0, t3.BondedDelegation().Sign())
	assert.Equal(t, t2.CommissionRate(), t3.CommissionRate())
}

func makeVotedFotTest(delegated int64, bonded int64) *Voted {
	voted := NewVoted()
	voted.SetDelegated(big.NewInt(delegated))
	voted.SetBonded(big.NewInt(bonded))
	return voted

}

func TestVoted_UpdateBondedDelegation(t *testing.T) {
	type args struct {
		delegated       int64
		bonded          int64
		bondRequirement int
	}

	tests := []struct {
		name string
		in   args
		want int64
	}{
		{
			"IISSVersion 1",
			args{
				100, 0, 0,
			},
			100,
		},
		{
			"IISSVersion 2 - exact fulfil",
			args{
				9500, 500, 5,
			},
			10000,
		},
		{
			"IISSVersion 2 - not enough",
			args{
				9600, 400, 5,
			},
			8000,
		},
		{
			"IISSVersion 2 - overbonded",
			args{
				1000, 100, 5,
			},
			1100,
		},
		{
			"IISSVersion 2 - Zero bond requirement",
			args{
				10000, 1000, 0,
			},
			11000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			t1 := makeVotedFotTest(in.delegated, in.bonded)
			t1.UpdateBondedDelegation(icmodule.ToRate(int64(in.bondRequirement)))

			assert.Equal(t, tt.want, t1.BondedDelegation().Int64())
		})
	}
}

func TestVoted_Equal(t *testing.T) {
	// VotedVersion1
	voted11 := &Voted{
		version:          VotedVersion1,
		status:           icmodule.ESDisablePermanent,
		delegated:        new(big.Int),
		bonded:           new(big.Int),
		bondedDelegation: new(big.Int),
		commissionRate:   11,
	}
	voted12 := &Voted{
		version:          VotedVersion1,
		status:           icmodule.ESDisablePermanent,
		delegated:        new(big.Int),
		bonded:           new(big.Int),
		bondedDelegation: new(big.Int),
		commissionRate:   12,
	}
	// does not compare commissionRate at VotedVersion1
	assert.True(t, voted11.Equal(voted12))
	// compare bondedDelegation at VotedVersion1
	voted11.SetBondedDelegation(big.NewInt(11))
	voted12.SetBondedDelegation(big.NewInt(12))
	assert.False(t, voted11.Equal(voted12))

	// VotedVersion2
	voted21 := NewVotedV2()
	voted22 := NewVotedV2()
	// does not compare bondedDelegation
	voted21.SetBondedDelegation(big.NewInt(21))
	voted22.SetBondedDelegation(big.NewInt(22))
	assert.True(t, voted21.Equal(voted22))
	// compare commissionRate at VotedVersion1
	voted21.SetCommissionRate(21)
	voted22.SetCommissionRate(22)
	assert.False(t, voted21.Equal(voted22))

	// invalid version
	voted1 := NewVoted()
	voted2 := NewVoted()
	assert.True(t, voted1.Equal(voted2))
	voted1.SetVersion(1000)
	voted2.SetVersion(1000)
	assert.False(t, voted1.Equal(voted2))
}

func TestVoted_SetCommissionRate(t *testing.T) {
	voted := NewVoted()
	assert.Equal(t, VotedVersion1, voted.Version())

	rate := icmodule.Rate(100)
	voted.SetCommissionRate(rate)
	assert.Equal(t, VotedVersion2, voted.Version())
	assert.Equal(t, rate, voted.CommissionRate())
}
