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

package lcstore

import (
	"sync"
	"time"
)

type tpsRegulator struct {
	lock      sync.Mutex
	max       int
	idx, cnt  int
	intervals []time.Duration
	total     time.Duration
	last      time.Time
}

func (m *tpsRegulator) Wait() {
	if m.max <= 0 {
		return
	}
	m.lock.Lock()
	defer m.lock.Unlock()

	now := time.Now()
	if m.cnt == 0 {
		m.last = now
		m.cnt += 1
		return
	}
	interval := now.Sub(m.last)
	total := m.total + interval - m.intervals[m.idx]

	if m.cnt < m.max {
		m.cnt += 1
	} else {
		if total < time.Second {
			delay := time.Second - total
			time.Sleep(delay)
			now = time.Now()
			interval = now.Sub(m.last)
			total = m.total + interval - m.intervals[m.idx]
		}
	}
	m.total = total
	m.last = now
	m.intervals[m.idx] = interval
	m.idx = (m.idx + 1) % len(m.intervals)
}

func (m *tpsRegulator) Init(max int) *tpsRegulator {
	if max > 0 {
		m.intervals = make([]time.Duration, max)
	}
	m.max = max
	return m
}
