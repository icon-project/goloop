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
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
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
var networkScoreTimerDictPrefix = containerdb.ToKey(
	containerdb.HashBuilder, scoredb.DictDBPrefix, "timer_network",
)

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

type timerData struct {
	addresses []*common.Address
}

func (t *timerData) equal(t2 *timerData) bool {
	if t == t2 {
		return true
	}
	if len(t.addresses) != len(t2.addresses) {
		return false
	}
	for i, a := range t.addresses {
		if !a.Equal(t2.addresses[i]) {
			return false
		}
	}
	return true
}

func (t timerData) clone() timerData {
	addrs := make([]*common.Address, len(t.addresses))
	copy(addrs, t.addresses)
	return timerData{
		addresses: addrs,
	}
}

func (t timerData) IsEmpty() bool {
	return len(t.addresses) == 0
}

func (t timerData) IndexOf(addr module.Address) int {
	for i, a := range t.addresses {
		if a.Equal(addr) {
			return i
		}
	}
	return -1
}

func (t timerData) Contains(addr module.Address) bool {
	return t.IndexOf(addr) >= 0
}

func (t *timerData) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "timerData{addresses=%+v}", t.addresses)
		} else {
			fmt.Fprintf(f, "timerData{%v}", t.addresses)
		}
	}
}

type TimerIterator interface {
	Get() (module.Address, bool)
	Next()
	Has() bool
}

type timerIteratorImpl struct {
	addresses []*common.Address
	idx       int
}

func (t *timerIteratorImpl) Get() (module.Address, bool) {
	if t.idx >= len(t.addresses) {
		return nil, false
	}
	return t.addresses[t.idx], true
}

func (t *timerIteratorImpl) Next() {
	if t.idx < len(t.addresses) {
		t.idx += 1
	}
}

func (t *timerIteratorImpl) Has() bool {
	return t.idx < len(t.addresses)
}

func (t timerData) Iterator() TimerIterator {
	return &timerIteratorImpl{
		addresses: t.addresses,
		idx:       0,
	}
}

type TimerSnapshot struct {
	icobject.NoDatabase
	timerData
}

func (t *TimerSnapshot) Version() int {
	return timerVersion
}

func (t *TimerSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.DecodeAll(
		&t.addresses,
	)
}

func (t *TimerSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		t.addresses,
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
	return t.timerData.equal(&tt.timerData)
}

type TimerState struct {
	snapshot *TimerSnapshot
	timerData
}

func (t *TimerState) Reset(ts *TimerSnapshot) *TimerState {
	if t.snapshot == ts {
		return t
	}
	t.snapshot = ts
	t.timerData = ts.timerData.clone()
	return t
}

func (t *TimerState) setDirty() {
	if t.snapshot != nil {
		t.snapshot = nil
	}
}

func (t *TimerState) GetSnapshot() *TimerSnapshot {
	if t.snapshot == nil {
		t.snapshot = &TimerSnapshot{timerData: t.timerData.clone()}
	}
	return t.snapshot
}

func (t *TimerState) Delete(address module.Address) {
	idx := t.IndexOf(address)
	if idx >= 0 {
		l := len(t.addresses)
		if idx+1 < l {
			copy(t.addresses[idx:], t.addresses[idx+1:])
		} else {
			t.addresses[idx] = nil
		}
		t.addresses = t.addresses[0 : l-1]
		t.setDirty()
	}
}

func (t *TimerState) Add(address module.Address) {
	if t.Contains(address) {
		return
	}
	t.addresses = append(t.addresses, common.AddressToPtr(address))
	t.setDirty()
}

var emptyTimerSnapshot = &TimerSnapshot{}

func NewTimerWithSnapshot(tss *TimerSnapshot) *TimerState {
	return new(TimerState).Reset(tss)
}

func newTimer() *TimerState {
	return new(TimerState).Reset(emptyTimerSnapshot)
}

func newTimerWithTag(_ icobject.Tag) *TimerSnapshot {
	return &TimerSnapshot{}
}

func ScheduleTimerJob(t *TimerState, info TimerJobInfo, address module.Address) {
	switch info.Type {
	case JobTypeAdd:
		t.Add(address)
	case JobTypeRemove:
		t.Delete(address)
	}
}
