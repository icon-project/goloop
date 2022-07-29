package db

import (
	"sync"

	"github.com/pkg/errors"
)

type layerBucket struct {
	lock sync.Mutex
	data map[string][]byte
	real Bucket
}

func (bk *layerBucket) Get(key []byte) ([]byte, error) {
	bk.lock.Lock()
	defer bk.lock.Unlock()

	if bk.data != nil {
		if value, ok := bk.data[string(key)]; ok {
			return value, nil
		}
	}
	return bk.real.Get(key)
}

func (bk *layerBucket) Has(key []byte) (bool, error) {
	bk.lock.Lock()
	defer bk.lock.Unlock()

	if bk.data != nil {
		if value, ok := bk.data[string(key)]; ok {
			return value != nil, nil
		}
	}
	return bk.real.Has(key)
}

func (bk *layerBucket) Set(key []byte, value []byte) error {
	if value == nil {
		return errors.New("IllegalArgument")
	}

	bk.lock.Lock()
	defer bk.lock.Unlock()

	if bk.data != nil {
		v2 := make([]byte, len(value))
		copy(v2, value)
		bk.data[string(key)] = v2
		return nil
	} else {
		return bk.real.Set(key, value)
	}
}

func (bk *layerBucket) Delete(key []byte) error {
	bk.lock.Lock()
	defer bk.lock.Unlock()

	if bk.data != nil {
		bk.data[string(key)] = nil
		return nil
	} else {
		return bk.real.Delete(key)
	}
}

func (bk *layerBucket) Flush(write bool) error {
	bk.lock.Lock()
	defer bk.lock.Unlock()

	if write && bk.data != nil {
		for k, v := range bk.data {
			if v == nil {
				if err := bk.real.Delete([]byte(k)); err != nil {
					return err
				}
			} else {
				if err := bk.real.Set([]byte(k), v); err != nil {
					return err
				}
			}
		}
	}
	bk.data = nil
	return nil
}

type layerDB struct {
	lock sync.Mutex

	flushed bool
	real    Database
	buckets map[string]*layerBucket
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
		data: make(map[string][]byte),
		real: realbk,
	}
	ldb.buckets[string(id)] = bk
	return bk, nil
}

func (ldb *layerDB) Flush(write bool) error {
	ldb.lock.Lock()
	defer ldb.lock.Unlock()

	for _, bk := range ldb.buckets {
		if err := bk.Flush(write); err != nil {
			return err
		}
	}
	ldb.flushed = true
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
