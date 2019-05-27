package block

import (
	"container/list"

	"github.com/icon-project/goloop/module"
)

type cache struct {
	cap       int
	heightMap map[int64]*list.Element
	idMap     map[string]*list.Element
	mru       *list.List
}

func newCache(cap int) *cache {
	return &cache{
		cap:       cap,
		heightMap: make(map[int64]*list.Element),
		idMap:     make(map[string]*list.Element),
		mru:       list.New(),
	}
}

func (c *cache) Put(b module.Block) {
	if c.mru.Len() == c.cap {
		b := c.mru.Remove(c.mru.Back()).(module.Block)
		delete(c.heightMap, b.Height())
		delete(c.idMap, string(b.ID()))
	}
	e := c.mru.PushFront(b)
	c.heightMap[b.Height()] = e
	c.idMap[string(b.ID())] = e
}

func (c *cache) Get(id []byte) module.Block {
	if e, ok := c.idMap[string(id)]; ok {
		c.mru.MoveToFront(e)
		return e.Value.(module.Block)
	}
	return nil
}

func (c *cache) GetByHeight(h int64) module.Block {
	if e, ok := c.heightMap[h]; ok {
		c.mru.MoveToFront(e)
		return e.Value.(module.Block)
	}
	return nil
}

// for test
func (c *cache) _getMRU() *list.List {
	return c.mru
}
