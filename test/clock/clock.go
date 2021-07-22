/*
 * Copyright 2021 ICON Foundation
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

package clock

import (
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
)

type afterFuncTimer struct {
	cl *Clock
	t  time.Time
	f  func()
}

func (timer *afterFuncTimer) Stop() bool {
	timer.cl.Lock()
	defer timer.cl.Unlock()

	if timer.f == nil {
		return false
	}
	timer.f = nil

	for i, tm := range timer.cl.afterFuncTimers {
		if timer == tm {
			last := len(timer.cl.afterFuncTimers) - 1
			timer.cl.afterFuncTimers[i] = timer.cl.afterFuncTimers[last]
			timer.cl.afterFuncTimers[last] = nil
			timer.cl.afterFuncTimers = timer.cl.afterFuncTimers[:last]
		}
	}
	return true
}

type Clock struct {
	sync.Mutex
	now             time.Time
	afterFuncTimers []*afterFuncTimer
}

func (cl *Clock) Now() time.Time {
	cl.Lock()
	defer cl.Unlock()

	return cl.now
}

func (cl *Clock) AfterFunc(d time.Duration, f func()) common.Timer {
	cl.Lock()
	defer cl.Unlock()

	t := cl.now.Add(d)
	aft := &afterFuncTimer{cl, t, f}
	cl.afterFuncTimers = append(cl.afterFuncTimers, aft)
	return common.NewTimer(aft)
}

func (cl *Clock) NewTimer(d time.Duration) common.Timer {
	cl.Lock()
	defer cl.Unlock()

	t := cl.now.Add(d)
	c := make(chan time.Time)
	aft := &afterFuncTimer{cl, t, func() {
		c <- cl.Now()
	}}
	cl.afterFuncTimers = append(cl.afterFuncTimers, aft)
	return common.NewTimerWithChan(aft, c)
}

func (cl *Clock) Sleep(d time.Duration) {
	timer := cl.NewTimer(d)
	<-timer.C
}

func (cl *Clock) PassTime(d time.Duration) {
	cl.SetTime(cl.now.Add(d))
}

func (cl *Clock) SetTime(t time.Time) {
	var timers []*afterFuncTimer

	cl.Lock()
	defer func() {
		for _, tm := range timers {
			tm.f()
		}
	}()
	defer cl.Unlock()

	if t.Before(cl.now) {
		return
	}
	cl.now = t
	for i := 0; i < len(cl.afterFuncTimers); {
		tm := cl.afterFuncTimers[i]
		if cl.now.Equal(tm.t) || cl.now.After(tm.t) {
			last := len(cl.afterFuncTimers) - 1
			cl.afterFuncTimers[i] = cl.afterFuncTimers[last]
			cl.afterFuncTimers[last] = nil
			cl.afterFuncTimers = cl.afterFuncTimers[:last]
			timers = append(timers, tm)
			continue
		}
		i++
	}
}
