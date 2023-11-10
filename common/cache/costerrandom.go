/*
 * Copyright 2023 ICON Foundation
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
	"math/rand"
	"time"
)

// CosterRandom is a key value cache with random eviction.
// The cache holds key values under cost cap and evicts random items if the
// cache holds cache items beyond the cost cap.
type CosterRandom[K comparable, V Coster] struct {
	costCap int
	costSum int
	kv      map[K]V
	keys    []K
	rnd     *rand.Rand
}

// MakeCosterRandom returns a new CosterRandom object.
func MakeCosterRandom[K comparable, V Coster](cap int, src rand.Source) CosterRandom[K, V] {
	if src == nil {
		src = rand.NewSource(time.Now().UnixMilli())
	}
	return CosterRandom[K, V]{
		costCap: cap,
		kv:      make(map[K]V),
		rnd:     rand.New(src),
	}
}

func NewCosterRandom[K comparable, V Coster](cap int, src rand.Source) *CosterRandom[K, V] {
	if src == nil {
		src = rand.NewSource(time.Now().UnixMilli())
	}
	return &CosterRandom[K, V]{
		costCap: cap,
		kv:      make(map[K]V),
		rnd:     rand.New(src),
	}
}

func (c *CosterRandom[K, V]) Put(key K, value V) {
	if value.Cost() > c.costCap {
		return
	}
	v, ok := c.kv[key]
	if ok {
		c.costSum -= v.Cost()
	} else {
		c.keys = append(c.keys, key)
	}
	c.costSum += value.Cost()
	c.kv[key] = value
	for c.costSum > c.costCap {
		ei := c.rnd.Uint64() % uint64(len(c.keys)-1)
		ek := c.keys[ei]
		if ek == key {
			ek = c.keys[len(c.keys)-1]
		} else {
			c.keys[ei] = c.keys[len(c.keys)-1]
		}
		var zk K
		c.keys[len(c.keys)-1] = zk
		c.keys = c.keys[:len(c.keys)-1]

		ev := c.kv[ek]
		c.costSum -= ev.Cost()
		delete(c.kv, ek)
	}
}

func (c *CosterRandom[K, V]) Get(key K) (val V, ok bool) {
	val, ok = c.kv[key]
	return val, ok
}
