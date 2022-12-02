package consensus

import (
	"container/list"

	"github.com/icon-project/goloop/module"
)

type commit struct {
	height       int64
	commitVotes  module.CommitVoteSet
	votes        *VoteList
	blockPartSet PartSet
}

type commitCache struct {
	cap       int
	heightMap map[int64]*list.Element
	mru       *list.List
}

func newCommitCache(cap int) *commitCache {
	return &commitCache{
		cap:       cap,
		heightMap: make(map[int64]*list.Element),
		mru:       list.New(),
	}
}

func (c *commitCache) Put(cm *commit) {
	if c.mru.Len() == c.cap {
		cm := c.mru.Remove(c.mru.Back()).(*commit)
		delete(c.heightMap, cm.height)
	}
	e := c.mru.PushFront(cm)
	c.heightMap[cm.height] = e
}

func (c *commitCache) GetByHeight(h int64) *commit {
	if e, ok := c.heightMap[h]; ok {
		c.mru.MoveToFront(e)
		return e.Value.(*commit)
	}
	return nil
}

// for test
func (c *commitCache) _getMRU() *list.List {
	return c.mru
}
