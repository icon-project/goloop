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

type SimpleCache struct {
	cap int
	kv  map[string]*list.Element
	mru list.List
}

func NewSimpleCache(cap int) *SimpleCache {
	c := &SimpleCache{
		cap: cap,
		kv: make(map[string]*list.Element, cap),
	}
	c.mru.Init()
	return c
}

func (c *SimpleCache) Put(key []byte, value interface{}) {
	if c.mru.Len() == c.cap {
		c.mru.Remove(c.mru.Back())
		delete(c.kv, string(key))
	}
	e := c.mru.PushFront(item{
		key: string(key),
		value: value,
	})
	c.kv[string(key)] = e
}

func (c *SimpleCache) Get(key []byte) interface{} {
	if e, ok := c.kv[string(key)]; ok {
		c.mru.MoveToFront(e)
		return e.Value.(item).value
	}
	return nil
}
