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
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

type addresses []module.Address

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

type Timer struct {
	Addresses addresses
}

type TimerSnapshot struct {
	icobject.NoDatabase
	Timer
}

func (t *TimerSnapshot) Version() int {
	return timerVersion
}

func (t *TimerSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&t.Addresses,
	)
	return err
}

func (t *TimerSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		t.Addresses,
	)
}

func (t *TimerSnapshot) Equal(object icobject.Impl) bool {
	tt, ok := object.(*TimerSnapshot)
	if !ok {
		return false
	}
	if tt == t {
		return true
	}
	return t.Addresses.Equal(tt.Addresses)
}

func newTimerSnapshot(tag icobject.Tag) *TimerSnapshot {
	return &TimerSnapshot{}
}

type TimerState struct {
	Height int64
	Timer
}

func (t *TimerState) Reset(ts *TimerSnapshot) {
	t.Addresses = ts.Addresses
}

func (t *TimerState) Clear() {
	t.Height = 0
	t.Addresses = nil
}

func (t *TimerState) GetSnapshot() *TimerSnapshot {
	ts := &TimerSnapshot{}
	ts.Addresses = t.Addresses
	return ts
}

func (t TimerState) IsEmpty() bool {
	return len(t.Addresses) == 0
}

func (t *TimerState) Add(address module.Address) {
	t.Addresses = append(t.Addresses, address)
}

func (t *TimerState) Delete(address module.Address) error {
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

func newTimerState(height int64) *TimerState {
	return &TimerState{
		Height: height,
	}
}

func NewTimerStateWithSnapshot(h int64, ss *TimerSnapshot) *TimerState {
	ts := newTimerState(h)
	ts.Reset(ss)
	return ts
}

func ScheduleTimerJob(t *TimerState, info TimerJobInfo, address module.Address) error {
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
