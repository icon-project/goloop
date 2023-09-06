package service

import (
	"container/list"
	"sync"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/cache"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

type trCacheItem struct {
	result            string
	database          *databaseAdaptor
	transactionResult *transitionResult
	worldSnapshot     state.WorldSnapshot
	worldContext      state.WorldContext
	normalReceipts    module.ReceiptList
	patchReceipts     module.ReceiptList
}

func (i *trCacheItem) Size() int {
	return i.database.Size()
}

type transitionResultCache struct {
	lock sync.Mutex

	entryCount int
	entrySize  int

	database  db.Database
	platform  base.Platform
	stateList *list.List
	stateMap  map[string]*list.Element

	vssCache *cache.LRUCache

	log log.Logger
}

func (c *transitionResultCache) GetValidatorSnapshot(vh []byte) (state.ValidatorSnapshot, error) {
	vhs := string(vh)
	if vs, err := c.vssCache.Get(vhs); err != nil {
		return nil, err
	} else {
		return vs.(state.ValidatorSnapshot), nil
	}
}

func (c *transitionResultCache) createValidatorSnapshot(vhs string) (interface{}, error) {
	vs, err := state.ValidatorSnapshotFromHash(c.database, []byte(vhs))
	return vs, err
}

func (c *transitionResultCache) getItemInLock(result []byte) (*trCacheItem, error) {
	var item *trCacheItem
	s := string(result)
	if e, ok := c.stateMap[s]; ok {
		item = e.Value.(*trCacheItem)
		if item.Size() < c.entrySize {
			c.stateList.MoveToBack(e)
			return item, nil
		}
		c.log.Tracef("TransitionResultCache() drop cache.size=%d by SIZE", item.Size())
		c.stateList.Remove(e)
		delete(c.stateMap, item.result)
	}

	tr, err := newTransitionResultFromBytes(result)
	if err != nil {
		return nil, err
	}
	item = &trCacheItem{
		result:            s,
		database:          newDatabaseAdaptor(c.database),
		transactionResult: tr,
	}
	e := c.stateList.PushBack(item)
	c.stateMap[s] = e

	return item, nil
}

func (c *transitionResultCache) reclaimInLock() {
	for c.stateList.Len() > c.entryCount {
		f := c.stateList.Front()
		c.stateList.Remove(f)
		toDel := f.Value.(*trCacheItem)
		delete(c.stateMap, toDel.result)
		c.log.Tracef("TransitionResultCache() drop cache.size=%d by ENTRY", toDel.Size())
	}
}

func (c *transitionResultCache) GetReceipts(result []byte, group module.TransactionGroup) (module.ReceiptList, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	item, err := c.getItemInLock(result)
	if err != nil {
		return nil, err
	}
	if group == module.TransactionGroupNormal {
		if item.normalReceipts == nil {
			item.normalReceipts = txresult.NewReceiptListFromHash(
				item.database,
				item.transactionResult.NormalReceiptHash)
		}
		return item.normalReceipts, nil
	} else {
		if item.patchReceipts == nil {
			item.patchReceipts = txresult.NewReceiptListFromHash(
				item.database,
				item.transactionResult.PatchReceiptHash)
		}
		return item.patchReceipts, nil
	}
}

func (c *transitionResultCache) GetWorldSnapshot(result []byte, vh []byte) (state.WorldSnapshot, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.getWorldSnapshotInLock(result, vh)
}

func (c *transitionResultCache) getWorldSnapshotInLock(result []byte, vh []byte) (state.WorldSnapshot, error) {
	item, err := c.getItemInLock(result)
	if err != nil {
		return nil, err
	}
	if item.worldSnapshot == nil {
		item.worldSnapshot = state.NewWorldSnapshot(
			item.database,
			item.transactionResult.StateHash,
			nil,
			c.platform.NewExtensionSnapshot(c.database, item.transactionResult.ExtensionData),
			item.transactionResult.BTPData,
		)
	}
	c.reclaimInLock()

	if len(vh) > 0 {
		vss, err := c.GetValidatorSnapshot(vh)
		if err != nil {
			return nil, err
		}
		return state.NewWorldSnapshotWithNewValidators(item.database, item.worldSnapshot, vss), nil
	}

	return item.worldSnapshot, nil
}

func (c *transitionResultCache) GetWorldContext(result []byte, vh []byte) (state.WorldContext, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	item, err := c.getItemInLock(result)
	if err != nil {
		return nil, err
	}
	if item.worldContext == nil {
		wss, err := c.getWorldSnapshotInLock(result, vh)
		if err != nil {
			return nil, err
		}
		ws := state.NewReadOnlyWorldState(wss)
		item.worldContext = state.NewWorldContext(ws, nil, nil, c.platform)
	}
	return item.worldContext, nil
}

func (c *transitionResultCache) Count() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.stateList.Len()
}

func (c *transitionResultCache) MaxCount() int {
	return c.entryCount
}

func (c *transitionResultCache) MaxSize() int {
	return c.entrySize
}

func (c *transitionResultCache) TotalBytes() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	var total int
	for item := c.stateList.Front(); item != nil; item = item.Next() {
		ci := item.Value.(*trCacheItem)
		total += ci.Size()
	}
	return total
}

func newTransitionResultCache(database db.Database, plt base.Platform, count int, size int, log log.Logger) *transitionResultCache {
	trc := &transitionResultCache{
		database:   database,
		platform:   plt,
		entryCount: count,
		entrySize:  size,
		stateList:  list.New(),
		stateMap:   make(map[string]*list.Element),
		log:        log,
	}
	trc.vssCache = cache.NewLRUCache(size, trc.createValidatorSnapshot)
	return trc
}
