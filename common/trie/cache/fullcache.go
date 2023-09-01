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
	"bytes"
	"container/list"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/icon-project/goloop/common/log"
)

const (
	fullCacheBranchDepth = 5
	fullCacheBranchSize  = ((1 << uint(4*fullCacheBranchDepth)) - 1) / 15
	fullCacheLRUSize     = 80_000
)

type FullCache struct {
	lock   sync.Mutex
	nodes  [][2][]byte
	hash2e map[string]*list.Element
	lru    list.List
	hits   int32
	out    int32
	in     int32
}

type nodeItem struct {
	key   string
	value []byte
}

func (c *FullCache) getNode(h []byte) []byte {
	if e, ok := c.hash2e[string(h)]; ok {
		c.lru.MoveToBack(e)
		atomic.AddInt32(&c.hits, 1)
		return e.Value.(*nodeItem).value
	} else {
		return nil
	}
}

func (c *FullCache) putNode(h, v []byte) {
	if e, ok := c.hash2e[string(h)]; ok {
		c.lru.MoveToBack(e)
	} else {
		if c.lru.Len() >= fullCacheLRUSize {
			e = c.lru.Front()
			c.lru.Remove(e)
			atomic.AddInt32(&c.out, 1)
			delete(c.hash2e, e.Value.(*nodeItem).key)
		}
		key := string(h)
		item := &nodeItem{
			key:   key,
			value: v,
		}
		atomic.AddInt32(&c.in, 1)
		c.hash2e[key] = c.lru.PushBack(item)
	}
}

func (c *FullCache) Get(nibs []byte, h []byte) ([]byte, bool) {
	if nibs == nil {
		return nil, false
	}
	idx := indexByNibs(nibs)

	c.lock.Lock()
	defer c.lock.Unlock()

	if idx < fullCacheBranchSize {
		node := c.nodes[idx][:]
		if bytes.Equal(node[0], h) {
			atomic.AddInt32(&c.hits, 1)
			return node[1], true
		}
		return nil, true
	} else {
		return c.getNode(h), true
	}
}

func (c *FullCache) Put(nibs []byte, h, v []byte) {
	if nibs == nil {
		return
	}
	idx := indexByNibs(nibs)

	c.lock.Lock()
	defer c.lock.Unlock()

	if idx < fullCacheBranchSize {
		atomic.AddInt32(&c.in, 1)
		c.nodes[idx] = [2][]byte{h, v}
	} else {
		c.putNode(h, v)
	}
}

func (c *FullCache) String() string {
	return fmt.Sprintf("FullCache{%p}", c)
}

func (c *FullCache) OnAttach(count int32, id []byte) cacheImpl {
	in := atomic.SwapInt32(&c.in, 0)
	out := atomic.SwapInt32(&c.out, 0)
	hits := atomic.SwapInt32(&c.hits, 0)
	if logCacheEvents && out > 0 {
		log.Warnf("FullCacheOnAttach(id=%#x,count=%d,in=%d,out=%d,hits=%d)",
			id, count, in, out, hits)
	}
	return c
}

func NewFullCache() *FullCache {
	fc := &FullCache{
		nodes:  make([][2][]byte, fullCacheBranchSize),
		hash2e: make(map[string]*list.Element),
	}
	return fc
}

func NewFullCacheFromBranch(bc *BranchCache) *FullCache {
	var nodes [][2][]byte
	if bc.depth == fullCacheBranchDepth {
		nodes = bc.nodes
	} else {
		nodes = make([][2][]byte, fullCacheBranchSize)
		copy(nodes, bc.nodes)
	}
	if bc.f != nil {
		bc.f.Close()
	}
	fc := &FullCache{
		nodes:  nodes,
		hash2e: make(map[string]*list.Element),
	}
	return fc
}
