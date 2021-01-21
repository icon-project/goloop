package icstate

import (
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

var activePRepArrayPrefix = containerdb.ToKey(containerdb.RawBuilder, "active_prep")

type activePRepCacheItem struct {
	owner module.Address
	idx   int
}

func (item *activePRepCacheItem) key() string {
	return icutils.ToKey(item.owner)
}

type ActivePRepCache struct {
	arraydb     *containerdb.ArrayDB
	items       []*activePRepCacheItem
	ownerToItem map[string]*activePRepCacheItem
}

// A new PRep is registered
func (c *ActivePRepCache) Add(owner module.Address) {
	if item := c.getByOwner(owner); item != nil {
		panic(errors.Errorf("ActivePRep already exists: %v", item))
	}

	item := &activePRepCacheItem{
		owner: owner,
		idx:   -1,
	}
	c.items = append(c.items, item)
	c.ownerToItem[item.key()] = item
}

// An active PRep is removed
func (c *ActivePRepCache) Remove(owner module.Address) {
	itemToRemove := c.getByOwner(owner)
	if itemToRemove == nil {
		panic(errors.Errorf("ActivePRep is not found: %v", itemToRemove))
	}

	idx := itemToRemove.idx
	lastIdx := c.Size() - 1
	if idx < lastIdx {
		c.items[idx] = c.items[lastIdx]
		c.items[idx].idx = idx
	}

	c.items = c.items[:lastIdx]
	delete(c.ownerToItem, itemToRemove.key())
}

func (c *ActivePRepCache) Size() int {
	return len(c.items)
}

func (c *ActivePRepCache) getByOwner(owner module.Address) *activePRepCacheItem {
	key := icutils.ToKey(owner)
	return c.ownerToItem[key]
}

func (c *ActivePRepCache) Get(i int) module.Address {
	if i < 0 || i >= c.Size() {
		return nil
	}
	return c.items[i].owner
}

func (c *ActivePRepCache) Clear() {
	c.items = nil
	c.ownerToItem = make(map[string]*activePRepCacheItem)
}

// Reset recovers data which is in the list of Map as of now
func (c *ActivePRepCache) Reset() {
	c.Clear()

	size := c.arraydb.Size()
	c.items = make([]*activePRepCacheItem, size)

	for i := 0; i < size; i++ {
		owner := c.arraydb.Get(i).Address()
		item := &activePRepCacheItem{owner, i}
		c.items[i] = item
		c.ownerToItem[item.key()] = item
	}
}

func (c *ActivePRepCache) Flush() {
	for i, item := range c.items {
		if i == item.idx {
			continue
		}

		o := icobject.NewBytesObject(item.owner.Bytes())
		if item.idx >= 0 {
			if err := c.arraydb.Set(i, o); err != nil {
				panic(errors.Errorf("ArrayDB.Set(%d, %s) is failed", i, item.owner))
			}
		} else {
			if err := c.arraydb.Put(o); err != nil {
				panic(errors.Errorf("ArrayDB.Put(%s) is failed", item.owner))
			}
		}

		item.idx = i
	}

	diff := c.arraydb.Size() - c.Size()
	for i := 0; i < diff; i++ {
		c.arraydb.Pop()
	}
}

func newActivePRepCache(store containerdb.ObjectStoreState) *ActivePRepCache {
	arraydb := containerdb.NewArrayDB(store, activePRepArrayPrefix)
	size := arraydb.Size()

	items := make([]*activePRepCacheItem, size)
	itemMap := make(map[string]*activePRepCacheItem)

	for i := 0; i < size; i++ {
		owner := arraydb.Get(i).Address()
		item := &activePRepCacheItem{owner, i}
		items[i] = item
		itemMap[item.key()] = item
	}

	return &ActivePRepCache{arraydb, items, itemMap}
}
