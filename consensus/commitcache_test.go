package consensus

import (
	"container/list"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertListHeights(t *testing.T, l *list.List, h ...int64) {
	assert.Equal(t, l.Len(), len(h))
	i := 0
	for e := l.Front(); e != nil; e = e.Next() {
		assert.Equal(t, e.Value.(*commit).height, h[i])
		i++
	}
}

func TestCache_Simple(t *testing.T) {
	c := newCommitCache(3)
	c.Put(&commit{height: 0})
	cm := c.GetByHeight(0)
	assert.NotNil(t, cm)
}

func TestCache_Simple2(t *testing.T) {
	c := newCommitCache(3)
	c.Put(&commit{height: 0})
	c.Put(&commit{height: 1})
	c.Put(&commit{height: 2})
	c.GetByHeight(0)
	c.Put(&commit{height: 3})
	assertListHeights(t, c._getMRU(), 3, 0, 2)
}
