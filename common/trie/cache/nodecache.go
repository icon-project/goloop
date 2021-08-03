package cache

import (
	"fmt"
	"sync"
)

const (
	fullCacheMigrationThreshold = 256
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
	OnAttach(id []byte) cacheImpl
}

type NodeCache struct {
	lock     sync.Mutex
	impl     cacheImpl
}

func (c *NodeCache) Get(nibs []byte, h []byte) ([]byte, bool) {
	if c == nil {
		return nil, false
	}
	c.lock.Lock()
	defer c.lock.Unlock()

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
	if c == nil {
		return nil
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.impl = c.impl.OnAttach(id)
	return c
}

func NewNodeCache(depth int, fdepth int, path string) *NodeCache {
	bc := NewBranchCache(depth, fdepth, path)
	return &NodeCache{
		impl: bc,
	}
}
