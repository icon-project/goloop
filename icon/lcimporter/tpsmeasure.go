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

package lcimporter

import (
	"math/big"
	"time"
)

type rateRecord struct {
	Duration time.Duration
	Count    int64
}

type RateMeasure struct {
	history []rateRecord
	index   int
	total   time.Duration
	count   int64
}

func (m *RateMeasure) Record(t time.Duration, c int64) {
	old := m.history[m.index]
	m.history[m.index] = rateRecord{t, c}
	m.total += t - old.Duration
	m.count += c - old.Count
	m.index = (m.index + 1) % (len(m.history))
}

func (m *RateMeasure) GetRate(t time.Duration) float32 {
	if m.total == 0 {
		return float32(m.count)
	}
	return float32(time.Duration(m.count)*t*100/m.total) / 100
}

func (m *RateMeasure) Init(cnt int) *RateMeasure {
	m.history = make([]rateRecord, cnt)
	return m
}

type TPSMeasure struct {
	RateMeasure
	lastTime time.Time
}

func (m *TPSMeasure) GetTPS() float32 {
	return m.GetRate(time.Second)
}

func (m *TPSMeasure) Init(cnt int) *TPSMeasure {
	m.RateMeasure.Init(cnt)
	m.lastTime = time.Now()
	return m
}

func (m *TPSMeasure) OnTransactions(c *big.Int) {
	now := time.Now()
	t := now.Sub(m.lastTime)
	m.lastTime = now
	m.Record(t, c.Int64())
}

type BPSMeasure struct {
	RateMeasure
	lastTime time.Time
}

func (m *BPSMeasure) GetBPS() float32 {
	return m.GetRate(time.Second)
}

func (m *BPSMeasure) Init(cnt int) *BPSMeasure {
	m.RateMeasure.Init(cnt)
	m.lastTime = time.Now()
	return m
}

func (m *BPSMeasure) OnBlock() {
	now := time.Now()
	t := now.Sub(m.lastTime)
	m.lastTime = now
	m.Record(t, int64(1))
}
