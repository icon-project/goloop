package block

import (
	"container/list"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/module"
)

type tBlock struct {
	module.Block
	height int64
}

func newTBlock(height int64) *tBlock {
	return &tBlock{height: height}
}

func (b *tBlock) Height() int64 {
	return b.height
}

func idForHeight(h int64) []byte {
	return []byte(fmt.Sprintf("%032d", h))
}

func (b *tBlock) ID() []byte {
	return idForHeight(b.height)
}

func assertListHeights(t *testing.T, l *list.List, h ...int64) {
	assert.Equal(t, l.Len(), len(h))
	i := 0
	for e := l.Front(); e != nil; e = e.Next() {
		assert.Equal(t, e.Value.(module.Block).Height(), h[i])
		i++
	}
}

func TestCache_Simple(t *testing.T) {
	c := newCache(3)
	c.Put(newTBlock(0))
	b := c.GetByHeight(0)
	assert.NotNil(t, b)
	b = c.Get(idForHeight(0))
	assert.NotNil(t, b)
}

func TestCache_Simple2(t *testing.T) {
	c := newCache(3)
	c.Put(newTBlock(0))
	c.Put(newTBlock(1))
	c.Put(newTBlock(2))
	c.GetByHeight(0)
	c.Put(newTBlock(3))
	assertListHeights(t, c._getMRU(), 3, 0, 2)
}
