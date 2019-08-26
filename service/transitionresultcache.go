package service

import (
	"container/list"
	"github.com/icon-project/goloop/common/log"
	"sync"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

type trCacheItem struct {
	result            string
	database          *databaseAdaptor
	transactionResult *transitionResult
	worldSnapshot     state.WorldSnapshot
	validatorSnapshot state.ValidatorSnapshot
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
	stateList *list.List
	stateMap  map[string]*list.Element

	log log.Logger
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

	item, err := c.getItemInLock(result)
	if err != nil {
		return nil, err
	}

	if item.validatorSnapshot == nil && len(vh) > 0 {
		vl, err := state.ValidatorSnapshotFromHash(item.database, vh)
		if err != nil {
			return nil, err
		}
		item.validatorSnapshot = vl

		if item.worldSnapshot != nil {
			item.worldSnapshot = state.UpdateWorldSnapshotValidators(item.database, item.worldSnapshot, item.validatorSnapshot)
		}
	}

	if item.worldSnapshot == nil && len(item.transactionResult.StateHash) > 0 {
		item.worldSnapshot = state.NewWorldSnapshot(
			item.database,
			item.transactionResult.StateHash,
			item.validatorSnapshot)
	}

	c.reclaimInLock()

	return item.worldSnapshot, nil
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

func newTransitionResultCache(database db.Database, count int, size int, log log.Logger) *transitionResultCache {
	return &transitionResultCache{
		database:   database,
		entryCount: count,
		entrySize:  size,
		stateList:  list.New(),
		stateMap:   make(map[string]*list.Element),
		log:        log,
	}
}
