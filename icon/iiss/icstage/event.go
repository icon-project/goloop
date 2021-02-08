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
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

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

type EventEnable struct {
	icobject.NoDatabase
	Target *common.Address
	Enable bool
}

func (ee *EventEnable) Version() int {
	return 0
}

func (ee *EventEnable) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&ee.Target,
		&ee.Enable,
	)
	return err
}

func (ee *EventEnable) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		ee.Target,
		ee.Enable,
	)
}

func (ee *EventEnable) Equal(o icobject.Impl) bool {
	if ee2, ok := o.(*EventEnable); ok {
		return ee.Target.Equal(ee2.Target) && ee.Enable == ee2.Enable
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

type EventPeriod struct {
	icobject.NoDatabase
	Irep          *big.Int
	Rrep          *big.Int
	MainPRepCount *big.Int
	PRepCount     *big.Int
}

func (ep *EventPeriod) Version() int {
	return 0
}

func (ep *EventPeriod) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&ep.Irep, &ep.Rrep, &ep.MainPRepCount, &ep.PRepCount)
	return err
}

func (ep *EventPeriod) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(ep.Irep, ep.Rrep, &ep.MainPRepCount, &ep.PRepCount)
}

func (ep *EventPeriod) Equal(o icobject.Impl) bool {
	if ep2, ok := o.(*EventPeriod); ok {
		return ep.Irep.Cmp(ep2.Irep) == 0 &&
			ep.Rrep.Cmp(ep2.Rrep) == 0 &&
			ep.MainPRepCount.Cmp(ep2.MainPRepCount) == 0 &&
			ep.PRepCount.Cmp(ep2.PRepCount) == 0
	} else {
		return false
	}
}

func (ep *EventPeriod) Clear() {
	ep.Irep = new(big.Int)
	ep.Rrep = new(big.Int)
	ep.MainPRepCount = new(big.Int)
	ep.PRepCount = new(big.Int)
}

func (ep *EventPeriod) IsEmpty() bool {
	return (ep.Irep == nil || ep.Irep.Sign() == 0) &&
		(ep.Rrep == nil || ep.Rrep.Sign() == 0)
}

func newEventPeriod(tag icobject.Tag) *EventPeriod {
	return NewEventPeriod()
}

func NewEventPeriod() *EventPeriod {
	return &EventPeriod{
		Irep:          new(big.Int),
		Rrep:          new(big.Int),
		MainPRepCount: new(big.Int),
		PRepCount:     new(big.Int),
	}
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
