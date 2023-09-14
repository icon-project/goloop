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
	"sort"
	"sync"

	"github.com/icon-project/goloop/common/log"
)

type nodeCacheItem struct {
	id    string
	count int
	cache *NodeCache
}

type nodeCacheList struct {
	lock     sync.Mutex
	sample   int
	limit    int
	last     int
	hitList  list.List
	idToItem map[string]*nodeCacheItem
	sorted   []*nodeCacheItem
	factory  func(id string) *NodeCache
}

func (l *nodeCacheList) updateSorted(removed, added *nodeCacheItem) {
	needSort := false
	if removed != nil && removed.cache != nil && removed.count < l.last {
		needSort = true
	}
	if added.cache == nil && added.count > l.last && len(l.sorted) >= l.limit {
		needSort = true
	}
	if needSort {
		pointOf := func(item *nodeCacheItem) int {
			point := item.count << 2
			if item.cache != nil {
				point |= 1 << 1
			}
			if item == added {
				point |= 1
			}
			return point
		}
		sort.Slice(l.sorted, func(i, j int) bool {
			return pointOf(l.sorted[i]) > pointOf(l.sorted[j])
		})

		// remove items from samples
		for idx := len(l.sorted) - 1; idx >= l.limit; idx -= 1 {
			item := l.sorted[idx]
			if item.count == 0 {
				if l.idToItem[item.id] == item {
					if logCacheEvents {
						if item.cache != nil {
							log.Warnf("RemoveCacheFor(%#x)", item.id)
						}
					}
					delete(l.idToItem, item.id)
				}
				l.sorted = l.sorted[:idx]
				continue
			}
			break
		}
		// update cache status
		for idx, item := range l.sorted {
			if idx < l.limit {
				if item.cache == nil {
					item.cache = l.factory(item.id)
					if logCacheEvents {
						log.Warnf("AddCacheFor(%#x)", item.id)
					}
				}
			} else {
				if item.cache != nil {
					if logCacheEvents {
						log.Warnf("RemoveCacheFor(%#x)", item.id)
					}
					item.cache = nil
				}
			}
		}

		// update last value
		if len(l.sorted) >= l.limit {
			l.last = l.sorted[l.limit-1].count
		} else {
			l.last = -1
		}
	} else {
		if len(l.sorted) <= l.limit {
			if added.cache == nil {
				added.cache = l.factory(added.id)
				if logCacheEvents {
					log.Warnf("AddCacheFor(%#x)", added.id)
				}
			}
		}
	}
}

func (l *nodeCacheList) Get(id string) *NodeCache {
	l.lock.Lock()
	defer l.lock.Unlock()

	item, ok := l.idToItem[id]
	if !ok && l.limit > 0 {
		item = &nodeCacheItem{id: id}
		l.idToItem[id] = item
		l.sorted = append(l.sorted, item)
	} else if !ok {
		return nil
	}
	if item.count == -1 {
		return item.cache
	}
	item.count += 1
	l.hitList.PushBack(item)
	var removed *nodeCacheItem
	if l.hitList.Len() > l.sample {
		removed = l.hitList.Remove(l.hitList.Front()).(*nodeCacheItem)
		removed.count -= 1
	}
	l.updateSorted(removed, item)
	return item.cache
}

func (l *nodeCacheList) SetCache(id string, cache *NodeCache) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if item, ok := l.idToItem[id]; ok {
		if item.count == -1 {
			return
		}
	}
	l.idToItem[id] = &nodeCacheItem{
		id:    id,
		count: -1,
		cache: cache,
	}
}

func NewNodeCacheList(sample, limit int, factory func(id string) *NodeCache) *nodeCacheList {
	return &nodeCacheList{
		sample:   sample,
		limit:    limit,
		last:     -1,
		idToItem: make(map[string]*nodeCacheItem),
		sorted:   make([]*nodeCacheItem, 0, limit),
		factory:  factory,
	}
}
