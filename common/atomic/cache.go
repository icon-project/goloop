/*
 * Copyright 2022 ICON Foundation
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

package atomic

import "sync/atomic"

type Cache[T any] struct {
	val atomic.Value
}

func MakeCache[T any](val T) Cache[T] {
	var c Cache[T]
	c.val.Store(val)
	return c
}

func (c *Cache[T]) Get(create func() T) T {
	val := c.val.Load()
	if val == nil {
		v := create()
		c.val.Store(v)
		return v
	}
	return val.(T)
}

func (c *Cache[T]) TryGet(create func() (T, error)) (T, error) {
	val := c.val.Load()
	if val == nil {
		v, err := create()
		if err != nil {
			var zero T
			return zero, err
		}
		c.val.Store(v)
		return v, nil
	}
	return val.(T), nil
}

func (c *Cache[T]) Set(val T) {
	c.val.Store(val)
}

// UnsafePurge resets cache value. The function is not goroutine safe
// and shall not be used in production code
func (c *Cache[T]) UnsafePurge() {
	var zero atomic.Value
	c.val = zero
}
