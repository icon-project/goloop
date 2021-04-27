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
		for idx, vote := range newVL {
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
		from: addr,
		votes: votes,
	}
}

type EnableStatus int

const (
	ESEnable EnableStatus = iota
	ESDisableTemp
	ESDisablePermanent
	ESMax
)

func (ef EnableStatus) IsEnabled() bool {
	return ef == ESEnable
}

func (ef EnableStatus) IsDisabledTemporarily() bool {
	return ef == ESDisableTemp
}

func (ef EnableStatus) String() string {
	switch ef {
	case ESEnable:
		return "Enabled"
	case ESDisableTemp:
		return "DisabledTemporarily"
	case ESDisablePermanent:
		return "DisabledPermanently"
	default:
		return "Unknown"
	}
}

type EventEnable struct {
	icobject.NoDatabase
	target *common.Address
	status EnableStatus
}

func (ee *EventEnable) Version() int {
	return 0
}

func (ee *EventEnable) Target() *common.Address {
	return ee.target
}

func (ee *EventEnable) Status() EnableStatus {
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
			fmt.Fprintf(f, "EventVote{target=%s status=%s}", ee.target, ee.status)
		} else {
			fmt.Fprintf(f, "EventVote{%s %s}", ee.target, ee.status)
		}
	}
}

func newEventEnable(_ icobject.Tag) *EventEnable {
	return new(EventEnable)
}

func NewEventEnable(target *common.Address, status EnableStatus) *EventEnable {
	return &EventEnable{
		target: target,
		status: status,
	}
}
