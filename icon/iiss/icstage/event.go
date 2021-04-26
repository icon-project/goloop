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

package icstage

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

type Vote struct {
	address *common.Address
	value   *big.Int
}

func NewVote(address *common.Address, value *big.Int) *Vote {
	return &Vote{
		address: address,
		value:   value,
	}
}

func (v *Vote) To() module.Address {
	return v.address
}

func (v *Vote) SetTo(addr module.Address) {
	v.address = common.AddressToPtr(addr)
}

func (v *Vote) Amount() *big.Int {
	return v.value
}

func (v *Vote) SetAmount(amount *big.Int) {
	v.value = amount
}

func (v *Vote) RLPDecodeSelf(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&v.address,
		&v.value,
	)
	return err
}

func (v *Vote) RLPEncodeSelf(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		v.address,
		v.value,
	)
}

func (v *Vote) Equal(v2 *Vote) bool {
	return v.address.Equal(v2.address) && v.value.Cmp(v2.value) == 0
}

func (v *Vote) Clone() *Vote {
	return NewVote(v.address, v.value)
}

func (v *Vote) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "Vote{address=%s value=%s}", v.address, v.value)
		} else {
			fmt.Fprintf(f, "Vote{%s %s}", v.address, v.value)
		}
	}
}

type VoteList []*Vote

func (vl VoteList) Equal(vl2 VoteList) bool {
	if len(vl) != len(vl2) {
		return false
	}
	for i, b := range vl {
		if !b.Equal(vl2[i]) {
			return false
		}
	}
	return true
}

func (vl VoteList) Clone() VoteList {
	if vl == nil {
		return nil
	}
	votes := make([]*Vote, len(vl))
	for i, vote := range vl {
		votes[i] = vote.Clone()
	}
	return votes
}

func (vl *VoteList) Delete(i int) error {
	if i < 0 || i >= len(*vl) {
		return errors.Errorf("Invalid index")
	}

	copy((*vl)[i:], (*vl)[i+1:])
	(*vl)[len(*vl)-1] = nil // or the zero value of T
	*vl = (*vl)[:len(*vl)-1]
	return nil
}

func (vl *VoteList) Update(vl2 VoteList) {
	newVL := vl.Clone()
	deleteIdx := make([]int, 0)
	for _, vote2 := range vl2 {
		find := false
		for idx, _ := range *vl {
			vote := newVL[idx]
			if vote.To().Equal(vote2.To()) {
				find = true
				vote.SetAmount(new(big.Int).Add(vote.Amount(), vote2.Amount()))
				if vote.Amount().Sign() == 0 {
					deleteIdx = append(deleteIdx, idx)
				}
				break
			}
		}
		if !find {
			newVL = append(newVL, vote2)
		}
	}
	sort.Ints(deleteIdx)
	for i, value := range deleteIdx {
		newVL.Delete(value - i)
	}
	*vl = newVL
}

type EventVote struct {
	icobject.NoDatabase
	From  *common.Address
	Votes VoteList
}

func (ed *EventVote) Version() int {
	return 0
}

func (ed *EventVote) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&ed.From, &ed.Votes)
	return err
}

func (ed *EventVote) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(ed.From, ed.Votes)
}

func (ed *EventVote) Equal(o icobject.Impl) bool {
	if ee2, ok := o.(*EventVote); ok {
		return ed.From.Equal(ee2.From) && ed.Votes.Equal(ee2.Votes)
	} else {
		return false
	}
}

func newEventVote(tag icobject.Tag) *EventVote {
	return new(EventVote)
}

type EnableFlag int

const (
	EfEnable EnableFlag = iota
	EfDisableTemp
	EfDisablePermanent
	EfMAX
)

func (ef EnableFlag) IsEnable() bool {
	return ef == EfEnable
}

func (ef EnableFlag) IsTemporarilyDisabled() bool {
	return ef == EfDisableTemp
}

type EventEnable struct {
	icobject.NoDatabase
	Target *common.Address
	Flag   EnableFlag
}

func (ee *EventEnable) Version() int {
	return 0
}

func (ee *EventEnable) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&ee.Target,
		&ee.Flag,
	)
	return err
}

func (ee *EventEnable) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		ee.Target,
		ee.Flag,
	)
}

func (ee *EventEnable) Equal(o icobject.Impl) bool {
	if ee2, ok := o.(*EventEnable); ok {
		return ee.Target.Equal(ee2.Target) && ee.Flag == ee2.Flag
	} else {
		return false
	}
}

func (ee *EventEnable) Clear() {
	ee.Target = nil
}

func (ee *EventEnable) IsEmpty() bool {
	return ee.Target == nil
}

func newEventEnable(tag icobject.Tag) *EventEnable {
	return new(EventEnable)
}