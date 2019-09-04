package db

import (
	"path/filepath"

	"github.com/dgraph-io/badger"
)

func init() {
	dbCreator := func(name string, dir string) (Database, error) {
		return NewBadgerDB(name, dir)
	}
	registerDBCreator(BadgerDBBackend, dbCreator, false)
}

func NewBadgerDB(name string, dir string) (*BadgerDB, error) {

	dbPath := filepath.Join(dir, name)
	opts := badger.DefaultOptions
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	// TODO : badger.openDatabase() use os.Mkdir(). parent dirs must be created
	db, err := badger.Open(opts)

	if err != nil {
		return nil, err
	}

	database := &BadgerDB{
		db: db,
	}

	return database, nil
}

//----------------------------------------
// DB

var _ Database = (*BadgerDB)(nil)

type BadgerDB struct {
	db *badger.DB
}

func (db *BadgerDB) GetBucket(id BucketID) (Bucket, error) {
	return &badgerBucket{
		id: id,
		db: db.db,
	}, nil
}

func (db *BadgerDB) Close() error {
	err := db.db.Close()
	return err
}

//----------------------------------------
// Bucket

var _ Bucket = (*badgerBucket)(nil)

type badgerBucket struct {
	id BucketID
	db *badger.DB
}

func (bucket *badgerBucket) Get(key []byte) ([]byte, error) {
	ikey := internalKey(bucket.id, key)
	var value []byte
	err := bucket.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(ikey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
		}
		value, err = item.ValueCopy(nil)
		return err
	})
	return value, err
}

func (bucket *badgerBucket) Has(key []byte) bool {
	value, err := bucket.Get(key)
	if !(value != nil && err == nil) {
		return false
	}
	return true
}

func (bucket *badgerBucket) Set(key []byte, value []byte) error {
	ikey := internalKey(bucket.id, key)
	return bucket.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(ikey, value)
		return err
	})
}

func (bucket *badgerBucket) Delete(key []byte) error {
	ikey := internalKey(bucket.id, key)
	return bucket.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(ikey)
	})
}
