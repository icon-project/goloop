package db

import (
	"container/list"
	"sync"

	"github.com/icon-project/goloop/common/errors"
)

type layerBucketItem struct {
	bk    *layerBucket
	key   string
	value []byte
}

type layerBucketItems struct {
	lock sync.Mutex
	list.List
}

func (l *layerBucketItems) PushBack(v *layerBucketItem) *list.Element {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.List.PushBack(v)
}

func (l *layerBucketItems) MoveToBack(element *list.Element) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.List.MoveToBack(element)
}

type layerBucket struct {
	lock sync.Mutex
	data map[string]*list.Element
	list *layerBucketItems
	real Bucket
}

func (bk *layerBucket) Get(key []byte) ([]byte, error) {
	bk.lock.Lock()
	defer bk.lock.Unlock()

	if bk.data != nil {
		if element, ok := bk.data[string(key)]; ok {
			return element.Value.(*layerBucketItem).value, nil
		}
	}
	return bk.real.Get(key)
}

func (bk *layerBucket) Has(key []byte) (bool, error) {
	bk.lock.Lock()
	defer bk.lock.Unlock()

	if bk.data != nil {
		if element, ok := bk.data[string(key)]; ok {
			return element.Value.(*layerBucketItem).value != nil, nil
		}
	}
	return bk.real.Has(key)
}

func (bk *layerBucket) Set(key []byte, value []byte) error {
	bk.lock.Lock()
	defer bk.lock.Unlock()

	if bk.data != nil {
		v2 := make([]byte, len(value))
		copy(v2, value)
		if element, ok := bk.data[string(key)] ; ok {
			if element != nil {
				bk.list.MoveToBack(element)
				element.Value.(*layerBucketItem).value = v2
				return nil
			}
		}
		item := &layerBucketItem{bk, string(key), v2 }
		bk.data[item.key] = bk.list.PushBack(item)
		return nil
	} else {
		return bk.real.Set(key, value)
	}
}

func (bk *layerBucket) Delete(key []byte) error {
	bk.lock.Lock()
	defer bk.lock.Unlock()

	if bk.data != nil {
		if element, ok := bk.data[string(key)] ; ok {
			bk.list.MoveToBack(element)
			element.Value.(*layerBucketItem).value = nil
		} else {
			item := &layerBucketItem{bk, string(key), nil }
			bk.data[item.key] = bk.list.PushBack(item)
		}
		return nil
	} else {
		return bk.real.Delete(key)
	}
}

type layerDB struct {
	lock sync.Mutex

	flushed bool
	real    Database
	buckets map[string]*layerBucket
	list    layerBucketItems
}

func (ldb *layerDB) GetBucket(id BucketID) (Bucket, error) {
	ldb.lock.Lock()
	defer ldb.lock.Unlock()

	if bk, ok := ldb.buckets[string(id)]; ok {
		return bk, nil
	}

	realbk, err := ldb.real.GetBucket(id)
	if err != nil {
		return nil, err
	}
	if ldb.flushed {
		return realbk, nil
	}
	bk := &layerBucket{
		data: make(map[string]*list.Element),
		list: &ldb.list,
		real: realbk,
	}
	ldb.buckets[string(id)] = bk
	return bk, nil
}

func (ldb *layerDB) Flush(write bool) error {
	ldb.lock.Lock()
	defer ldb.lock.Unlock()

	if ldb.flushed {
		if !write {
			return errors.InvalidStateError.New("DirectFlushMode")
		}
		return nil
	}

	for _, bk := range ldb.buckets {
		bk.lock.Lock()
	}
	ldb.list.lock.Lock()
	defer func() {
		ldb.list.lock.Unlock()
		for _, bk := range ldb.buckets {
			bk.lock.Unlock()
		}
	}()

	if write {
		for element := ldb.list.Front() ; element != nil ; element = element.Next() {
			item := element.Value.(*layerBucketItem)

			if item.value != nil {
				if err := item.bk.real.Set([]byte(item.key), item.value ); err != nil {
					return err
				}
			} else {
				if err := item.bk.real.Delete([]byte(item.key)); err != nil {
					return err
				}
			}
		}
		for _, bk := range ldb.buckets {
			bk.data = nil
		}
	} else {
		for _, bk := range ldb.buckets {
			bk.data = make(map[string]*list.Element)
		}
	}
	ldb.list.Init()
	ldb.flushed = write
	return nil
}

func (ldb *layerDB) Close() error {
	return nil
}

type layerDBContext struct {
	LayerDB
	flags Flags
}

func (c *layerDBContext) WithFlags(flags Flags) Context {
	newFlags := c.flags.Merged(flags)
	return &layerDBContext{c.LayerDB, newFlags}
}

func (c *layerDBContext) GetFlag(name string) interface{} {
	return c.flags.Get(name)
}

func (c *layerDBContext) Flags() Flags {
	return c.flags.Clone()
}

func (ldb *layerDB) WithFlags(flags Flags) Context {
	return &layerDBContext{ldb, flags}
}

func (ldb *layerDB) Unwrap() Database {
	return ldb.real
}

func NewLayerDB(database Database) LayerDB {
	ldb := &layerDB{
		real:    database,
		buckets: make(map[string]*layerBucket),
	}

	if ctx, ok := database.(Context); ok {
		return &layerDBContext{ldb, ctx.Flags()}
	} else {
		return ldb
	}
}

func Unwrap(database Database) Database {
	type unwrapper interface {
		Unwrap() Database
	}
	if layeredDB, ok := database.(unwrapper); ok {
		return layeredDB.Unwrap()
	} else {
		return database
	}
}
