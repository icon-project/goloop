package service

import (
	"sync"
	"sync/atomic"

	"github.com/icon-project/goloop/common/db"
)

type databaseAdaptor struct {
	lock    sync.Mutex
	origin  db.Database
	buckets map[db.BucketID]*bucketAdaptor
	size    int32
}

func (da *databaseAdaptor) GetBucket(id db.BucketID) (db.Bucket, error) {
	da.lock.Lock()
	defer da.lock.Unlock()

	if ba, ok := da.buckets[id]; ok {
		return ba, nil
	} else {
		bk, err := da.origin.GetBucket(id)
		if err != nil {
			return nil, err
		}
		ba := newBucketAdaptor(da, bk)
		return ba, nil
	}
}

func (da *databaseAdaptor) Close() error {
	panic("Not allowed to close database")
	return nil
}

func (da *databaseAdaptor) OnRead(size int) {
	atomic.AddInt32(&da.size, int32(size))
}

func (da *databaseAdaptor) Size() int {
	return int(atomic.LoadInt32(&da.size))
}

func newDatabaseAdaptor(database db.Database) *databaseAdaptor {
	return &databaseAdaptor{
		origin:  database,
		buckets: make(map[db.BucketID]*bucketAdaptor),
	}
}

type bucketAdaptor struct {
	database *databaseAdaptor
	bucket   db.Bucket
}

func (ba *bucketAdaptor) Get(key []byte) ([]byte, error) {
	value, err := ba.bucket.Get(key)
	if err == nil {
		ba.database.OnRead(len(value))
	}
	return value, err
}

func (ba *bucketAdaptor) Has(key []byte) (bool, error) {
	return ba.bucket.Has(key)
}

func (ba *bucketAdaptor) Set(key []byte, value []byte) error {
	panic("Now allowed")
}

func (ba *bucketAdaptor) Delete(key []byte) error {
	panic("Now allowed")
}

func newBucketAdaptor(da *databaseAdaptor, bk db.Bucket) *bucketAdaptor {
	return &bucketAdaptor{
		database: da,
		bucket:   bk,
	}
}
