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
	return &goLevelBucket{
		id: name,
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
	id string
	db *leveldb.DB
}

func makeKey(id string, key []byte) []byte {
	nkey := make([]byte, len(id)+len(key))
	copy(nkey, id)
	copy(nkey[len(id):], key)
	return nkey
}

func (bucket *goLevelBucket) Get(key []byte) ([]byte, error) {
	value, err := bucket.db.Get(makeKey(bucket.id, key), nil)
	if err == leveldb.ErrNotFound {
		return nil, nil
	} else {
		return value, err
	}
}

func (bucket *goLevelBucket) Has(key []byte) bool {
	ret, err := bucket.db.Has(makeKey(bucket.id, key), nil)
	if err != nil {
		return false
	}
	return ret
}

func (bucket *goLevelBucket) Set(key []byte, value []byte) error {
	return bucket.db.Put(makeKey(bucket.id, key), value, nil)
}

func (bucket *goLevelBucket) Delete(key []byte) error {
	return bucket.db.Delete(makeKey(bucket.id, key), nil)
}
