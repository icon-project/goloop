package icstate

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

type dummyObjectStore struct {
	Database db.Database
	Trie     trie.MutableForObject
	containerdb.ObjectStoreState
}

func newDummyObjectStore(_ bool) *dummyObjectStore {
	dbase := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	t := trie_manager.NewMutableForObject(dbase, nil, icobject.ObjectType)
	os := icobject.NewObjectStoreState(t)
	return &dummyObjectStore{
		Database:         dbase,
		Trie:             t,
		ObjectStoreState: os,
	}
}

func (os *dummyObjectStore) flushAndNewStore() *dummyObjectStore {
	ss := os.Trie.GetSnapshot()
	ss.Flush()
	trie := trie_manager.NewMutableForObject(os.Database, ss.Hash(),icobject.ObjectType)
	os2 := icobject.NewObjectStoreState(trie)
	return &dummyObjectStore{
		Database:         os.Database,
		Trie:             trie,
		ObjectStoreState: os2,
	}
}

func newDummyNodeOwnerCache(_ bool) (*NodeOwnerCache, *dummyObjectStore) {
	store := newDummyObjectStore(false)
	return newNodeOwnerCache(store), store
}

func TestNodeOwnerCache_Clear(t *testing.T) {
	var err error
	cache, store := newDummyNodeOwnerCache(false)

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

	cache.Flush()

	for i := 0; i < 5; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		assert.True(t, owner.Equal(cache.Get(node)))
	}

	store = store.flushAndNewStore()
	cache = newNodeOwnerCache(store)

	for i := 0; i < 5; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		assert.True(t, owner.Equal(cache.Get(node)))
	}

	for i := 5; i < 10; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		err = cache.Add(node, owner)
		assert.NoError(t, err)
	}

	cache.Clear()

	for i := 0; i < 10; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		assert.True(t, owner.Equal(cache.Get(node)))
	}

	store = store.flushAndNewStore()

	for i := 0; i < 10; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		assert.True(t, owner.Equal(cache.Get(node)))
	}
}

func TestNodeOwnerCache_Contains(t *testing.T) {
	var err error
	cache, store := newDummyNodeOwnerCache(false)

	for i := 0; i < 5; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		err = cache.Add(node, owner)
		assert.NoError(t, err)
	}

	for i := 0; i < 5; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		assert.True(t, cache.Contains(node))
		assert.False(t, cache.Contains(owner))
	}

	cache.Flush()

	store = store.flushAndNewStore()
	cache = newNodeOwnerCache(store)

	for i := 0; i < 5; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		assert.True(t, cache.Contains(node))
		assert.False(t, cache.Contains(owner))
	}
}

func TestNodeOwnerCache_Add(t *testing.T) {
	var err error
	var node module.Address

	cache, store := newDummyNodeOwnerCache(false)

	for i := 0; i < 2; i++ {
		owner := newDummyAddress(i)
		node = newDummyAddress(i + 100)
		err = cache.Add(node, owner)
		assert.NoError(t, err)
	}

	for i := 0; i < 2; i++ {
		// Node address is already in use
		node = newDummyAddress(i + 100)
		err = cache.Add(node, node)
		assert.Error(t, err)

		// owner is the same as node
		node = newDummyAddress(i + 100 + 3)
		err = cache.Add(node, node)
		assert.NoError(t, err)

		nodeB := newDummyAddress(i + 100 + 3)
		err = cache.Add(node, nodeB)
		assert.NoError(t, err)
	}

	cache.Clear()
	store = store.flushAndNewStore()
	cache = newNodeOwnerCache(store)

	for i := 0; i < 2; i++ {
		// Node address is already in use
		node = newDummyAddress(i + 100)
		err = cache.Add(node, node)
		assert.Error(t, err)

		// owner is the same as node (and new)
		node = newDummyAddress(i + 100 + 3)
		err = cache.Add(node, node)
		assert.NoError(t, err)
	}
}
