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

func (db *GoLevelDB) GetBucket(name string) (Bucket, error) {
	panic("implement me")
}

func (db *GoLevelDB) Close() error {
	panic("implement me")
}


//----------------------------------------
// GetBucket

var _ Bucket = (*goLevelBucket)(nil)

type goLevelBucket struct {
	id string
	db *leveldb.DB
}

func (bucket *goLevelBucket) Get(key []byte) ([]byte, error) {
	panic("implement me")
}

func (bucket *goLevelBucket) Has(key []byte) bool {
	panic("implement me")
}

func (bucket *goLevelBucket) Set(key []byte, value []byte) error {
	panic("implement me")
}

func (bucket *goLevelBucket) Delete(key []byte) error {
	panic("implement me")
}
