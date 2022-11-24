package db

import (
	"path/filepath"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const GoLevelDBBackend BackendType = "goleveldb"

func init() {
	dbCreator := func(name string, dir string) (Database, error) {
		return NewGoLevelDB(name, dir)
	}
	registerDBCreator(GoLevelDBBackend, dbCreator, false)
}

func NewGoLevelDB(name string, dir string) (*GoLevelDB, error) {
	return NewGoLevelDBWithOpts(name, dir, nil)
}

func NewGoLevelDBWithOpts(name string, dir string, o *opt.Options) (*GoLevelDB, error) {
	dbPath := filepath.Join(dir, name)
	db, err := leveldb.OpenFile(dbPath, o)
	if err != nil {
		return nil, err
	}
	database := &GoLevelDB{
		db:      db,
		buckets: make(map[BucketID]Bucket),
	}
	return database, nil
}

//----------------------------------------
// Database

var _ Database = (*GoLevelDB)(nil)

type GoLevelDB struct {
	lock    sync.Mutex
	db      *leveldb.DB
	buckets map[BucketID]Bucket
}

func (db *GoLevelDB) GetBucket(id BucketID) (Bucket, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.db == nil {
		return nil, leveldb.ErrClosed
	}

	if bk, ok := db.buckets[id]; ok {
		return bk, nil
	} else {
		bk = &goLevelBucket{
			id: id,
			db: db.db,
		}
		db.buckets[id] = bk
		return bk, nil
	}
}

func (db *GoLevelDB) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.db == nil {
		return leveldb.ErrClosed
	}
	if err := db.db.Close(); err != nil {
		return err
	}
	db.db = nil
	return nil
}

//----------------------------------------
// GetBucket

var _ Bucket = (*goLevelBucket)(nil)

type goLevelBucket struct {
	id BucketID
	db *leveldb.DB
}

func (bucket *goLevelBucket) Get(key []byte) ([]byte, error) {
	value, err := bucket.db.Get(internalKey(bucket.id, key), nil)
	if err == leveldb.ErrNotFound {
		return nil, nil
	} else {
		return value, err
	}
}

func (bucket *goLevelBucket) Has(key []byte) (bool, error) {
	return bucket.db.Has(internalKey(bucket.id, key), nil)
}

func (bucket *goLevelBucket) Set(key []byte, value []byte) error {
	return bucket.db.Put(internalKey(bucket.id, key), value, nil)
}

func (bucket *goLevelBucket) Delete(key []byte) error {
	return bucket.db.Delete(internalKey(bucket.id, key), nil)
}
