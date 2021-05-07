package icstate

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/module"
)

func newDummyNodeOwnerCache(readonly bool) *NodeOwnerCache {
	store := newDummyObjectStore(false)
	return newNodeOwnerCache(store)
}

func TestNodeOwnerCache_Clear(t *testing.T) {
	var err error
	cache := newDummyNodeOwnerCache(false)

	for i := 0; i < 5; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		err = cache.Add(node, owner)
		assert.NoError(t, err)
	}

	for i := 0; i < 5; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		assert.True(t, owner.Equal(cache.Get(node)))
	}

	cache.Clear()

	for i := 0; i < 5; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		assert.False(t, owner.Equal(cache.Get(node)))
	}

	for i := 0; i < 5; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		err = cache.Add(node, owner)
		assert.NoError(t, err)
	}

	cache.Flush()

	for i := 0; i < 5; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		assert.True(t, owner.Equal(cache.Get(node)))
	}

	cache.Clear()

	for i := 0; i < 5; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		assert.True(t, owner.Equal(cache.Get(node)))
	}
}

func TestNodeOwnerCache_Contains(t *testing.T) {
	var err error

	size := 10
	s := newDummyState(false)

	for i := 0; i < size; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		err = s.nodeOwnerCache.Add(node, owner)
		assert.NoError(t, err)
		s.nodeOwnerCache.Flush()
	}

	for i := 0; i < size; i++ {
		node := newDummyAddress(i + 100)
		assert.True(t, s.nodeOwnerCache.Contains(node))
	}

	for i := size; i < size; i++ {
		node := newDummyAddress(i + 100 + 100)
		assert.False(t, s.nodeOwnerCache.Contains(node))
		assert.True(t, node.Equal(s.nodeOwnerCache.Get(node)))
	}
}

func TestNodeOwnerCache_Add(t *testing.T) {
	var err error
	var node module.Address
	s := newDummyState(false)

	for i := 0; i < 2; i++ {
		owner := newDummyAddress(i)
		node = newDummyAddress(i + 100)
		err = s.nodeOwnerCache.Add(node, owner)
		assert.NoError(t, err)
		s.nodeOwnerCache.Flush()
	}
	s.nodeOwnerCache.Clear()

	for i := 0; i < 2; i++ {
		// Node address is already in use
		node = newDummyAddress(i + 100)
		err = s.nodeOwnerCache.Add(node, node)
		assert.Error(t, err)

		// owner is the same as node
		node = newDummyAddress(i + 100 + 3)
		err = s.nodeOwnerCache.Add(node, node)
		assert.NoError(t, err)
	}
}
