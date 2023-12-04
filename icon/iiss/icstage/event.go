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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
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

func (v *Vote) Amount() *big.Int {
	return v.value
}

func (v *Vote) SetAmount(amount *big.Int) {
	v.value = amount
}

func (v *Vote) RLPDecodeSelf(decoder codec.Decoder) error {
	return decoder.DecodeListOf(
		&v.address,
		&v.value,
	)
}

func (v *Vote) RLPEncodeSelf(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
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
	case 's':
		fmt.Fprintf(f, "address=%s value=%s", v.address, v.value)
	}
}

type VoteList []*Vote

func (vl VoteList) Has() bool {
	return len(vl) > 0
}

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
	var newVL VoteList

	map1 := vl.ToMap()
	map2 := vl2.ToMap()

	for _, vote := range *vl {
		_, ok := map2[icutils.ToKey(vote.To())]
		if !ok {
			newVL = append(newVL, vote)
		}
	}

	for _, vote2 := range vl2 {
		vote1, ok := map1[icutils.ToKey(vote2.To())]
		if ok {
			value := new(big.Int).Add(vote1.Amount(), vote2.Amount())
			if value.Sign() == 0 {
				continue
			}
			vote2 = NewVote(common.AddressToPtr(vote2.To()), value)
		}
		newVL = append(newVL, vote2)
	}
	*vl = newVL
}

func (vl VoteList) ToMap() map[string]*Vote {
	if !vl.Has() {
		return nil
	}
	m := make(map[string]*Vote, len(vl))

	for _, v := range vl {
		m[icutils.ToKey(v.To())] = v
	}
	return m
}

type EventVote struct {
	icobject.NoDatabase
	from  *common.Address
	votes VoteList
}

func (e *EventVote) Version() int {
	return 0
}

func (e *EventVote) From() *common.Address {
	return e.from
}

func (e *EventVote) Votes() VoteList {
	return e.votes
}

func (e *EventVote) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&e.from, &e.votes)
	return err
}

func (e *EventVote) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(e.from, e.votes)
}

func (e *EventVote) Equal(o icobject.Impl) bool {
	if ee2, ok := o.(*EventVote); ok {
		return e.from.Equal(ee2.from) && e.votes.Equal(ee2.votes)
	} else {
		return false
	}
}

func (e *EventVote) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "EventVote{address=%s value=%+v}", e.from, e.votes)
		} else {
			fmt.Fprintf(f, "EventVote{%s %v}", e.from, e.votes)
		}
	}
}

func newEventVote(_ icobject.Tag) *EventVote {
	return new(EventVote)
}

func NewEventVote(addr *common.Address, votes VoteList) *EventVote {
	return &EventVote{
		from:  addr,
		votes: votes,
	}
}

type EventDelegationV2 struct {
	icobject.NoDatabase
	from       *common.Address
	delegated  VoteList
	delegating VoteList
}

func (e *EventDelegationV2) Version() int {
	return 0
}

func (e *EventDelegationV2) From() *common.Address {
	return e.from
}

func (e *EventDelegationV2) Delegated() VoteList {
	return e.delegated
}

func (e *EventDelegationV2) Delegating() VoteList {
	return e.delegating
}

func (e *EventDelegationV2) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&e.from, &e.delegated, &e.delegating)
	return err
}

func (e *EventDelegationV2) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(e.from, e.delegated, e.delegating)
}

func (e *EventDelegationV2) Equal(o icobject.Impl) bool {
	if e2, ok := o.(*EventDelegationV2); ok {
		return e.from.Equal(e2.from) &&
			e.delegated.Equal(e2.delegated) &&
			e.delegating.Equal(e2.delegating)
	} else {
		return false
	}
}

func (e *EventDelegationV2) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "EventDelegationV2{from=%s delegated=%+v delegating=%+v}",
				e.from, e.delegated, e.delegating)
		} else {
			fmt.Fprintf(f, "EventDelegationV2{%s %v %v}", e.from, e.delegated, e.delegating)
		}
	}
}

func newEventDelegationV2(_ icobject.Tag) *EventDelegationV2 {
	return new(EventDelegationV2)
}

func NewEventDelegationV2(addr *common.Address, delegated VoteList, delegating VoteList) *EventDelegationV2 {
	return &EventDelegationV2{
		from:       addr,
		delegated:  delegated,
		delegating: delegating,
	}
}

type EventEnable struct {
	icobject.NoDatabase
	target *common.Address
	status icmodule.EnableStatus
}

func (ee *EventEnable) Version() int {
	return 0
}

func (ee *EventEnable) Target() *common.Address {
	return ee.target
}

func (ee *EventEnable) Status() icmodule.EnableStatus {
	return ee.status
}

func (ee *EventEnable) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&ee.target,
		&ee.status,
	)
	return err
}

func (ee *EventEnable) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		ee.target,
		ee.status,
	)
}

func (ee *EventEnable) Equal(o icobject.Impl) bool {
	if ee2, ok := o.(*EventEnable); ok {
		return ee.target.Equal(ee2.target) && ee.status == ee2.status
	} else {
		return false
	}
}

func (ee *EventEnable) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "EventEnable{target=%s status=%s}", ee.target, ee.status)
		} else {
			fmt.Fprintf(f, "EventEnable{%s %s}", ee.target, ee.status)
		}
	}
}

func newEventEnable(_ icobject.Tag) *EventEnable {
	return new(EventEnable)
}

func NewEventEnable(target *common.Address, status icmodule.EnableStatus) *EventEnable {
	return &EventEnable{
		target: target,
		status: status,
	}
}

type EventVotedReward struct {
	icobject.NoDatabase
}

func (ee *EventVotedReward) Version() int {
	return 0
}

func (ee *EventVotedReward) RLPDecodeFields(decoder codec.Decoder) error {
	return nil
}

func (ee *EventVotedReward) RLPEncodeFields(encoder codec.Encoder) error {
	return nil
}

func (ee *EventVotedReward) Equal(o icobject.Impl) bool {
	return true
}

func (ee *EventVotedReward) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "EventVotedRewward{}")
}

func newEventVotedReward(_ icobject.Tag) *EventVotedReward {
	return new(EventVotedReward)
}

func NewEventVotedReward() *EventVotedReward {
	return new(EventVotedReward)
}
