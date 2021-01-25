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
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type TimerCache struct {
	dict   *containerdb.DictDB
	timers map[int64]*Timer
}

func (c *TimerCache) Add(t *Timer) {
	c.timers[t.Height] = t
}

func (c *TimerCache) Remove(height int64) error {
	timer := c.Get(height)
	if timer == nil {
		return errors.Errorf("Timer not found on BH %d", height)
	}

	timer.Clear()
	return nil
}

func (c *TimerCache) Get(height int64) *Timer {
	timer := c.timers[height]
	if timer != nil {
		return timer
	}

	o := c.dict.Get(height)
	if o == nil {
		return nil
	}

	timer = ToTimer(o.Object())
	c.timers[height] = timer

	return timer
}

func (c *TimerCache) Clear() {
	c.timers = make(map[int64]*Timer)
}

func (c *TimerCache) Reset() {
	for key, timer := range c.timers {
		value := c.dict.Get(key)

		if value == nil {
			delete(c.timers, key)
		} else {
			timer.Set(ToTimer(value.Object()))
		}
	}
}

func (c *TimerCache) Flush() {
	for key, timer := range c.timers {
		if timer.IsEmpty() {
			if err := c.dict.Delete(key); err != nil {
				log.Errorf("Failed to delete Timer on %d, err+%+v", key, err)
			}
			delete(c.timers, key)
		} else {
			o := icobject.New(TypeTimer, timer.Clone())
			if err := c.dict.Set(timer.Height, o); err != nil {
				log.Errorf("Failed to set snapshotMap for %x, err+%+v", key, err)
			}
		}
	}
}

func newTimerCache(store containerdb.ObjectStoreState, prefix containerdb.KeyBuilder) *TimerCache {
	return &TimerCache{
		timers: make(map[int64]*Timer),
		dict:   containerdb.NewDictDB(store, 1, prefix),
	}
}
