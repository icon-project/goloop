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

package cache

import (
	"container/list"
)

type Coster interface {
	Cost() int
}

type entry[K comparable, V Coster] struct {
	key   K
	value V
}

type CosterLRU[K comparable, V Coster] struct {
	costCap int
	costSum int
	kv      map[K]*list.Element
	mru     list.List
}

func MakeCosterLRU[K comparable, V Coster](cap int) CosterLRU[K, V] {
	return CosterLRU[K, V]{
		costCap: cap,
		kv:      make(map[K]*list.Element),
	}
}

func NewCosterLRU[K comparable, V Coster](cap int) *CosterLRU[K, V] {
	return &CosterLRU[K, V]{
		costCap: cap,
		kv:      make(map[K]*list.Element),
	}
}

func (c *CosterLRU[K, V]) Put(key K, value V) {
	if value.Cost() > c.costCap {
		return
	}
	e, ok := c.kv[key]
	if ok {
		c.costSum -= e.Value.(entry[K, V]).value.Cost()
		e.Value = entry[K, V]{key, value}
		c.mru.MoveToFront(e)
	} else {
		e = c.mru.PushFront(entry[K, V]{
			key:   key,
			value: value,
		})
		c.kv[key] = e
	}
	c.costSum += value.Cost()
	for c.costSum > c.costCap {
		back := c.mru.Back()
		c.costSum -= back.Value.(entry[K, V]).value.Cost()
		delete(c.kv, back.Value.(entry[K, V]).key)
		c.mru.Remove(back)
	}
}

func (c *CosterLRU[K, V]) Get(key K) (val V, ok bool) {
	if e, ok := c.kv[key]; ok {
		c.mru.MoveToFront(e)
		return e.Value.(entry[K, V]).value, true
	}
	var zero V
	return zero, false
}
