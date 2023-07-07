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
	assert.Equal(t, votedVersion1, t2.Version())
	assert.Equal(t, 0, t1.Delegated().Cmp(t2.Delegated()))
	assert.Equal(t, 0, t1.BondedDelegation().Cmp(t2.BondedDelegation()))

	// v1 -> v2
	commissionRate := big.NewInt(1000)
	t2.SetCommissionRate(commissionRate)
	assert.True(t, t2.Version() == votedVersion2)

	o2 = icobject.New(type_, t2)
	o3 := new(icobject.Object)
	if err := o3.Reset(database, o2.Bytes()); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	t3 := ToVoted(o3)
	assert.False(t, t3.Equal(t2))
	assert.Equal(t, votedVersion2, t3.Version())
	assert.Equal(t, 0, t3.Delegated().Cmp(t2.Delegated()))
	assert.Equal(t, 0, t3.BondedDelegation().Sign())
	assert.Equal(t, 0, t3.CommissionRate().Cmp(t2.CommissionRate()))
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
			t1.UpdateBondedDelegation(in.bondRequirement)

			assert.Equal(t, tt.want, t1.BondedDelegation().Int64())
		})
	}
}
