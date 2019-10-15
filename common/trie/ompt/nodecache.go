package ompt

import (
	"bytes"
	"sync"
)

const (
	dataMaxSize   = 532
	cacheItemSize = hashSize + dataMaxSize
)

type NodeCache struct {
	lock  sync.Mutex
	nodes [][2][]byte
}

func indexByNibs(nibs []byte) int {
	if len(nibs) == 0 {
		return 0
	}
	idx := 0
	for _, nib := range nibs {
		idx = idx*16 + int(nib) + 1
	}
	return idx
}

func sizeByDepth(d int) int {
	return ((1 << uint(4*d)) - 1) / 15
}

func (c *NodeCache) get(nibs []byte, h []byte) ([]byte, bool) {
	if c == nil || nibs == nil {
		return nil, false
	}
	idx := indexByNibs(nibs)

	c.lock.Lock()
	defer c.lock.Unlock()
	if idx >= len(c.nodes) {
		return nil, false
	}
	node := c.nodes[idx]
	if bytes.Equal(node[0], h) {
		return node[1], true
	}
	return nil, true
}

func (c *NodeCache) put(nibs []byte, h []byte, serialized []byte) {
	if c == nil || nibs == nil || len(serialized) > dataMaxSize {
		return
	}
	idx := indexByNibs(nibs)

	c.lock.Lock()
	defer c.lock.Unlock()
	if idx >= len(c.nodes) {
		return
	}
	node := c.nodes[idx]
	node[0] = h
	node[1] = serialized
	c.nodes[idx] = node
}

func NewNodeCache(depth int) *NodeCache {
	size := sizeByDepth(depth)
	return &NodeCache{
		nodes: make([][2][]byte, size),
	}
}
