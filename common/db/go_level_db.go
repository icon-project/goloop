package db

import (
	"path/filepath"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

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
		db: db,
	}
	return database, nil
}

//----------------------------------------
// Database

var _ Database = (*GoLevelDB)(nil)

type GoLevelDB struct {
	db *leveldb.DB
}

func (db *GoLevelDB) GetBucket(id BucketID) (Bucket, error) {
	return &goLevelBucket{
		id: id,
		db: db.db,
	}, nil
}

func (db *GoLevelDB) Close() error {
	return db.db.Close()
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

func (bucket *goLevelBucket) Has(key []byte) bool {
	ret, err := bucket.db.Has(internalKey(bucket.id, key), nil)
	if err != nil {
		return false
	}
	return ret
}

func (bucket *goLevelBucket) Set(key []byte, value []byte) error {
	return bucket.db.Put(internalKey(bucket.id, key), value, nil)
}

func (bucket *goLevelBucket) Delete(key []byte) error {
	return bucket.db.Delete(internalKey(bucket.id, key), nil)
}
