package cache

import (
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	fullCacheMigrationThreshold = 150
)

func indexByNibs(nibs []byte) int {
	idx := 0
	for _, nib := range nibs {
		idx = idx*16 + int(nib) + 1
	}
	return idx
}

func sizeByDepth(d int) int {
	return ((1 << uint(4*d)) - 1) / 15
}

type cacheImpl interface {
	Get(nibs []byte, h []byte) ([]byte, bool)
	Put(nibs []byte, h []byte, serialized []byte)
	OnAttach(cnt int32, id []byte) cacheImpl
}

type NodeCache struct {
	lock     sync.Mutex
	impl     cacheImpl
	countGet int32
}

func (c *NodeCache) Get(nibs []byte, h []byte) ([]byte, bool) {
	if c == nil {
		return nil, false
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	atomic.AddInt32(&c.countGet, 1)
	return c.impl.Get(nibs, h)
}

func (c *NodeCache) String() string {
	return fmt.Sprintf("NodeCache{%v}", c.impl)
}

func (c *NodeCache) Put(nibs []byte, h []byte, serialized []byte) {
	if c == nil {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.impl.Put(nibs, h, serialized)
}

func (c *NodeCache) OnAttach(id []byte) *NodeCache {
	if c != nil {
		cnt := atomic.SwapInt32(&c.countGet, 0)
		c.impl = c.impl.OnAttach(cnt, id)
	}
	return c
}

func NewNodeCache(depth int, fdepth int, path string) *NodeCache {
	bc := NewBranchCache(depth, fdepth, path)
	return &NodeCache{
		impl: bc,
	}
}
