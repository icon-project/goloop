package service

import (
	"container/list"
	"sync"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

type worldSnapshotCache struct {
	lock      sync.Mutex
	size      int
	database  db.Database
	stateList *list.List
	stateMap  map[string]*list.Element
}

type wsCacheItem struct {
	result            string
	transactionResult *transitionResult
	worldSnapshot     state.WorldSnapshot
	normalReceipts    module.ReceiptList
	patchReceipts     module.ReceiptList
}

func (c *worldSnapshotCache) getItemInLock(result []byte) (*wsCacheItem, error) {
	var item *wsCacheItem
	s := string(result)
	if e, ok := c.stateMap[s]; ok {
		c.stateList.MoveToBack(e)
		item = e.Value.(*wsCacheItem)
	} else {
		tr, err := newTransitionResultFromBytes(result)
		if err != nil {
			return nil, err
		}
		item = &wsCacheItem{
			result:            s,
			transactionResult: tr,
		}
		e := c.stateList.PushBack(item)
		c.stateMap[s] = e
	}
	return item, nil
}

func (c *worldSnapshotCache) reclaimInLock() {
	for c.stateList.Len() > c.size {
		f := c.stateList.Front()
		c.stateList.Remove(f)
		toDel := f.Value.(*wsCacheItem)
		delete(c.stateMap, toDel.result)
	}
}

func (c *worldSnapshotCache) GetPatchReceipts(result []byte) (module.ReceiptList, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	item, err := c.getItemInLock(result)
	if err != nil {
		return nil, err
	}
	if len(item.transactionResult.PatchReceiptHash) > 0 && item.patchReceipts == nil {
		item.patchReceipts = txresult.NewReceiptListFromHash(
			c.database,
			item.transactionResult.PatchReceiptHash)
	}
	return item.patchReceipts, nil
}

func (c *worldSnapshotCache) GetNormalReceipts(result []byte) (module.ReceiptList, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	item, err := c.getItemInLock(result)
	if err != nil {
		return nil, err
	}
	if len(item.transactionResult.NormalReceiptHash) > 0 && item.normalReceipts == nil {
		item.normalReceipts = txresult.NewReceiptListFromHash(
			c.database,
			item.transactionResult.NormalReceiptHash)
	}
	return item.normalReceipts, nil
}

func (c *worldSnapshotCache) GetWorldSnapshot(result []byte) (state.WorldSnapshot, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	item, err := c.getItemInLock(result)
	if err != nil {
		return nil, err
	}

	if item.worldSnapshot == nil && len(item.transactionResult.StateHash) > 0 {
		item.worldSnapshot = state.NewWorldSnapshot(
			c.database,
			item.transactionResult.StateHash,
			nil)
	}

	c.reclaimInLock()

	return item.worldSnapshot, nil
}

func newWorldSnapshotCache(database db.Database, size int) *worldSnapshotCache {
	return &worldSnapshotCache{
		database:  database,
		size:      size,
		stateList: list.New(),
		stateMap:  make(map[string]*list.Element),
	}
}
