package cache

import (
	"container/list"
	"sync"

	"github.com/icon-project/goloop/common"
)

type Create func([]byte) (interface{}, error)

type LRUCache struct {
	lock  sync.Mutex
	size  int
	lru   list.List
	items map[string]*list.Element

	create Create
}

type item struct {
	key   string
	value interface{}
}

func (c *LRUCache) putInLock(key string, value interface{}) {
	e := c.lru.PushBack(item{key: key, value: value})
	c.items[key] = e
	if c.lru.Len() > c.size {
		e := c.lru.Front()
		c.lru.Remove(e)
		i := e.Value.(item)
		delete(c.items, i.key)
	}
}

func (c *LRUCache) Get(key []byte) (interface{}, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	e, ok := c.items[string(key)]
	if ok {
		c.lru.MoveToBack(e)
		i := e.Value.(item)
		return i.value, nil
	}
	if c.create == nil {
		return nil, common.ErrNotFound
	}
	value, err := c.create(key)
	if err != nil {
		return nil, err
	}
	c.putInLock(string(key), value)
	return value, nil
}

func (c *LRUCache) Put(key string, value interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	e, ok := c.items[key]
	if ok {
		e.Value = item{key: key, value: value}
		return
	}
	c.putInLock(key, value)
}

func (c *LRUCache) Len() int {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.lru.Len()
}

func (c *LRUCache) Size() int {
	return c.size
}

func NewLRUCache(size int, create Create) *LRUCache {
	c := &LRUCache{
		size:   size,
		items:  make(map[string]*list.Element),
		create: create,
	}
	return c
}
