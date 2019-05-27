package block

import (
	"container/list"

	"github.com/icon-project/goloop/module"
)

type cache struct {
	cap       int
	heightMap map[int64]module.Block
	idMap     map[string]module.Block
	mru       *list.List
}

func newCache(cap int) *cache {
	return &cache{
		cap:       cap,
		heightMap: make(map[int64]module.Block),
		idMap:     make(map[string]module.Block),
		mru:       list.New(),
	}
}

func (c *cache) Put(b module.Block) {
	if c.mru.Len() == c.cap {
		b := c.mru.Remove(c.mru.Back()).(module.Block)
		delete(c.heightMap, b.Height())
		delete(c.idMap, string(b.ID()))
	}
	c.mru.PushFront(b)
	c.heightMap[b.Height()] = b
	c.idMap[string(b.ID())] = b
}

func (c *cache) Get(id []byte) module.Block {
	if b, ok := c.idMap[string(id)]; ok {
		for e := c.mru.Front(); e != nil; e = e.Next() {
			if e.Value == b {
				c.mru.MoveToFront(e)
				break
			}
		}
		return b
	}
	return nil
}

func (c *cache) GetByHeight(h int64) module.Block {
	if b, ok := c.heightMap[h]; ok {
		for e := c.mru.Front(); e != nil; e = e.Next() {
			if e.Value == b {
				c.mru.MoveToFront(e)
				break
			}
		}
		return b
	}
	return nil
}

// for test
func (c *cache) _getMRU() *list.List {
	return c.mru
}

