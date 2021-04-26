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

package icstate

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

var unstakingTimerDictPrefix = containerdb.ToKey(
	containerdb.HashBuilder, scoredb.DictDBPrefix, "timer_unstaking",
)
var unbondingTimerDictPrefix = containerdb.ToKey(
	containerdb.HashBuilder, scoredb.DictDBPrefix, "timer_unbonding",
)

type addresses []*common.Address

const timerVersion = iota + 1

const (
	JobTypeAdd JobType = iota + 1
	JobTypeRemove
)

type JobType int

type TimerJobInfo struct {
	Type   JobType
	Height int64
}

func (a addresses) Equal(a2 addresses) bool {
	if len(a) != len(a2) {
		return false
	}
	for i, b := range a {
		if !b.Equal(a2[i]) {
			return false
		}
	}
	return true
}

func (a addresses) Clone() addresses {
	if a == nil {
		return nil
	}
	c := make([]*common.Address, len(a))
	for i, address := range a {
		c[i] = address
	}
	return c
}

func (a addresses) Contains(address module.Address) bool {
	for _, addr := range a {
		if addr.Equal(address) {
			return true
		}
	}
	return false
}

type Timer struct {
	icobject.NoDatabase
	StateAndSnapshot

	Addresses addresses
}

func (t *Timer) Version() int {
	return timerVersion
}

func (t *Timer) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&t.Addresses,
	)
	return err
}

func (t *Timer) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		t.Addresses,
	)
}

func (t *Timer) Equal(object icobject.Impl) bool {
	tt, ok := object.(*Timer)
	if !ok {
		return false
	}
	if tt == t {
		return true
	}
	return t.Addresses.Equal(tt.Addresses)
}

func (t *Timer) Clear() {
	t.Addresses = nil
}

func (t Timer) IsEmpty() bool {
	return len(t.Addresses) == 0
}

func (t *Timer) Set(other *Timer) {
	t.checkWritable()
	t.Addresses = other.Addresses.Clone()
}

func (t *Timer) Add(address module.Address) {
	if !t.Addresses.Contains(address) {
		t.Addresses = append(t.Addresses, common.AddressToPtr(address))
	}
}

func (t *Timer) Delete(address module.Address) error {
	tmp := make(addresses, 0)
	for _, a := range t.Addresses {
		if !a.Equal(address) {
			tmp = append(tmp, a)
		}
	}

	if len(tmp) == len(t.Addresses) {
		return errors.Errorf("%s not in timer", address.String())
	}

	t.Addresses = tmp
	return nil
}
func (t *Timer) Clone() *Timer {
	return &Timer{
		Addresses: t.Addresses.Clone(),
	}
}

func newTimer() *Timer {
	return &Timer{}
}

func newTimerWithTag(_ icobject.Tag) *Timer {
	return &Timer{}
}

func ScheduleTimerJob(t *Timer, info TimerJobInfo, address module.Address) error {
	switch info.Type {
	case JobTypeAdd:
		t.Add(address)
	case JobTypeRemove:
		if err := t.Delete(address); err != nil {
			return err
		}
	}
	return nil
}
