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

func (ee *EventDelegation) Version() int {
	return 0
}

func (ee *EventDelegation) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&ee.From, &ee.Delegations)
	return err
}

func (ee *EventDelegation) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(ee.From, ee.Delegations)
}

func (ee *EventDelegation) Equal(o icobject.Impl) bool {
	if ee2, ok := o.(*EventDelegation); ok {
		return ee.From.Equal(ee2.From) && ee.Delegations.Equal(ee2.Delegations)
	} else {
		return false
	}
}

func (ee *EventDelegation) Clear() {
	ee.From = nil
	ee.Delegations = nil
}

func (ee *EventDelegation) IsEmpty() bool {
	return ee.From == nil && ee.Delegations == nil
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
	Irep *big.Int
	Rrep *big.Int
}

func (ep *EventPeriod) Version() int {
	return 0
}

func (ep *EventPeriod) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&ep.Irep, &ep.Rrep)
	return err
}

func (ep *EventPeriod) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(ep.Irep, ep.Rrep)
}

func (ep *EventPeriod) Equal(o icobject.Impl) bool {
	if ep2, ok := o.(*EventPeriod); ok {
		return ep.Irep.Cmp(ep2.Irep) == 0 && ep2.Rrep.Cmp(ep2.Rrep) == 0
	} else {
		return false
	}
}

func (ep *EventPeriod) Clear() {
	ep.Irep = new(big.Int)
	ep.Rrep = new(big.Int)
}

func (ep *EventPeriod) IsEmpty() bool {
	return (ep.Irep == nil || ep.Irep.Sign() == 0) &&
		(ep.Rrep == nil || ep.Rrep.Sign() == 0)
}

func newEventPeriod(tag icobject.Tag) *EventPeriod {
	return new(EventPeriod)
}

type Validator struct {
	icobject.NoDatabase
	validators []*common.Address
}

func (v *Validator) Version() int {
	return 0
}

func (v *Validator) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&v.validators)
	return err
}

func (v *Validator) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(v.validators)
}

func (v *Validator) Equal(o icobject.Impl) bool {
	if v2, ok := o.(*Validator); ok {
		if len(v.validators) != len(v2.validators) {
			return false
		}
		for i, a := range v.validators {
			if a.Equal(v2.validators[i]) == false {
				return false
			}
		}
		return true
	} else {
		return false
	}
}

func (v *Validator) Clear() {
	v.validators = nil
}

func (v *Validator) IsEmpty() bool {
	return v.validators == nil
}

func newEventValidator(tag icobject.Tag) *Validator {
	return new(Validator)
}
