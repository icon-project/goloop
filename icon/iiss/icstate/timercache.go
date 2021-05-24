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
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type TimerCache struct {
	dict   *containerdb.DictDB
	timers map[int64]*TimerState
}

func (c *TimerCache) Get(height int64) *TimerState {
	timer := c.timers[height]
	if timer != nil {
		return timer
	}

	o := c.dict.Get(height)
	if o == nil {
		timer = newTimer()
	} else {
		timer = NewTimerWithSnapshot(ToTimer(o.Object()))
	}
	c.timers[height] = timer
	return timer
}

func (c *TimerCache) GetSnapshot(height int64) *TimerSnapshot {
	timer := c.timers[height]
	if timer != nil {
		return timer.GetSnapshot()
	}
	o := c.dict.Get(height)
	if o == nil {
		return nil
	} else {
		return ToTimer(o.Object())
	}
}

func (c *TimerCache) Clear() {
	c.Flush()
	c.timers = make(map[int64]*TimerState)
}

func (c *TimerCache) Reset() {
	for key, timer := range c.timers {
		value := c.dict.Get(key)

		if value == nil {
			timer.Reset(emptyTimerSnapshot)
		} else {
			timer.Reset(ToTimer(value.Object()))
		}
	}
}

func (c *TimerCache) Flush() {
	for height, timer := range c.timers {
		if timer.IsEmpty() {
			if err := c.dict.Delete(height); err != nil {
				log.Errorf("Failed to delete Timer on %d, err+%+v", height, err)
			}
		} else {
			o := icobject.New(TypeTimer, timer.GetSnapshot())
			if err := c.dict.Set(height, o); err != nil {
				log.Errorf("Failed to set Timer for %d, err+%+v", height, err)
			}
		}
	}
}

func newTimerCache(store containerdb.ObjectStoreState, prefix containerdb.KeyBuilder) *TimerCache {
	return &TimerCache{
		timers: make(map[int64]*TimerState),
		dict:   containerdb.NewDictDB(store, 1, prefix),
	}
}
