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
	"math/big"
	"sort"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
)

type Vote struct {
	Address *common.Address
	Value   *big.Int
}

func NewVote() *Vote {
	return &Vote{
		Address: new(common.Address),
		Value:   new(big.Int),
	}
}

func (v *Vote) To() module.Address {
	return v.Address
}

func (v *Vote) Amount() *big.Int {
	return v.Value
}

func (v *Vote) Equal(v2 *Vote) bool {
	return v.Address.Equal(v2.Address) && v.Value.Cmp(v2.Value) == 0
}

func (v *Vote) Clone() *Vote {
	n := NewVote()
	n.Address.Set(v.Address)
	n.Value.Set(v.Value)
	return n
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
				vote.Amount().Add(vote.Amount(), vote2.Amount())
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

type EventDelegation struct {
	icobject.NoDatabase
	From        *common.Address
	Delegations icstate.Delegations
}

func (ed *EventDelegation) Version() int {
	return 0
}

func (ed *EventDelegation) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&ed.From, &ed.Delegations)
	return err
}

func (ed *EventDelegation) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(ed.From, ed.Delegations)
}

func (ed *EventDelegation) Equal(o icobject.Impl) bool {
	if ee2, ok := o.(*EventDelegation); ok {
		return ed.From.Equal(ee2.From) && ed.Delegations.Equal(ee2.Delegations)
	} else {
		return false
	}
}

func (ed *EventDelegation) Clear() {
	ed.From = nil
	ed.Delegations = nil
}

func (ed *EventDelegation) IsEmpty() bool {
	return ed.From == nil && ed.Delegations == nil
}

func newEventDelegation(tag icobject.Tag) *EventDelegation {
	return new(EventDelegation)
}

type EventBond struct {
	icobject.NoDatabase
	From  *common.Address
	Bonds icstate.Bonds
}

func (b *EventBond) Version() int {
	return 0
}

func (b *EventBond) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&b.From, &b.Bonds)
	return err
}

func (b *EventBond) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(b.From, b.Bonds)
}

func (b *EventBond) Equal(o icobject.Impl) bool {
	if ee2, ok := o.(*EventBond); ok {
		return b.From.Equal(ee2.From) && b.Bonds.Equal(ee2.Bonds)
	} else {
		return false
	}
}

func (b *EventBond) Clear() {
	b.From = nil
	b.Bonds = nil
}

func (b *EventBond) IsEmpty() bool {
	return b.From == nil && b.Bonds == nil
}

func newEventBond(tag icobject.Tag) *EventBond {
	return new(EventBond)
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

type EventSize struct {
	icobject.ObjectBigInt
}

func (e *EventSize) Version() int {
	return 0
}

func (e *EventSize) Equal(o icobject.Impl) bool {
	if e2, ok := o.(*EventSize); ok {
		return e.Value.Cmp(e2.Value) == 0
	} else {
		return false
	}
}

func newEventSize(tag icobject.Tag) *EventSize {
	return &EventSize{
		*icobject.NewObjectBigInt(tag),
	}
}
