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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
)

func TestDelegating(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)

	type_ := TypeDelegating
	version := 0

	t1 := newDelegating(icobject.MakeTag(type_, version))
	d := icstate.NewDelegation(common.MustNewAddressFromString("hx1"), big.NewInt(10))
	t1.Delegations = append(t1.Delegations, d)

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

	t2 := ToDelegating(o2)
	assert.Equal(t, true, t1.Equal(t2))
}

func TestDelegating_ApplyVotes(t *testing.T) {
	addr1 := "hx1"
	addr2 := "hx2"
	addr3 := "hx3"
	addr4 := "hx4"
	val1 := int64(1)
	val2 := int64(2)
	val3 := int64(3)
	vBig := int64(100)
	d1 := icstate.NewDelegation(common.MustNewAddressFromString(addr1), big.NewInt(val1))
	v1Delete := icstage.NewVote(common.MustNewAddressFromString(addr1), big.NewInt(-val1))
	v1TooBig := icstage.NewVote(common.MustNewAddressFromString(addr1), big.NewInt(-vBig))
	d2 := icstate.NewDelegation(common.MustNewAddressFromString(addr2), big.NewInt(val2))
	v2 := icstage.NewVote(common.MustNewAddressFromString(addr2), big.NewInt(val2))
	d2Double := icstate.NewDelegation(common.MustNewAddressFromString(addr2), big.NewInt(val2*2))
	d3 := icstate.NewDelegation(common.MustNewAddressFromString(addr3), big.NewInt(val3))
	dNew := icstate.NewDelegation(common.MustNewAddressFromString(addr4), big.NewInt(val3))
	vNew := icstage.NewVote(common.MustNewAddressFromString(addr4), big.NewInt(val3))
	vNewNegative := icstage.NewVote(common.MustNewAddressFromString(addr4), big.NewInt(-val3))
	delegating := Delegating{
		Delegations: icstate.Delegations{d1, d2, d3},
	}

	tests := []struct {
		name string
		in   icstage.VoteList
		err  bool
		want icstate.Delegations
	}{
		{"Success", icstage.VoteList{v1Delete, v2, vNew}, false, icstate.Delegations{d3, d2Double, dNew}},
		{"New with negative value", icstage.VoteList{vNewNegative}, true, icstate.Delegations{}},
		{"Update result value is negative", icstage.VoteList{v1TooBig}, true, icstate.Delegations{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := delegating.Clone()
			err := test.ApplyVotes(tt.in)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, tt.want.Equal(test.Delegations), "%v\n%v", tt.want, test.Delegations)
			}
		})
	}
}
