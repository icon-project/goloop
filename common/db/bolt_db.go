package db

import (
	"path/filepath"

	bolt "go.etcd.io/bbolt"
)

func init() {
	dbCreator := func(name string, dir string) (Database, error) {
		return NewBoltDB(name, dir)
	}
	registerDBCreator(BoltDBBackend, dbCreator, false)
}

func NewBoltDB(name string, dir string) (*BoltDB, error) {
	dbPath := filepath.Join(dir, name+".db")
	db, err := bolt.Open(dbPath, 0644, nil)
	if err != nil {
		return nil, err
	}
	database := &BoltDB{
		db: db,
	}
	return database, nil
}

func (db *BoltDB) DB() *bolt.DB {
	return db.db
}

//----------------------------------------
// DB

var _ Database = (*BoltDB)(nil)

type BoltDB struct {
	db *bolt.DB
}

func (db *BoltDB) GetBucket(name string) (Bucket, error) {
	// create bucket
	err := db.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(name))
		return err
	})
	return &boltBucket{db: db.db, id: name}, err
}

func (db *BoltDB) Close() error {
	err := db.db.Close()
	return err
}

//----------------------------------------
// Bucket

var _ Bucket = (*boltBucket)(nil)

type boltBucket struct {
	id string
	db *bolt.DB
}

func (bucket *boltBucket) Get(key []byte) ([]byte, error) {
	var value []byte
	err := bucket.db.View(func(tx *bolt.Tx) error {
		value = tx.Bucket([]byte(bucket.id)).Get(key)
		if value == nil {
			// TODO : return ErrNotFound
			return nil
		}
		return nil
	})
	return value, err
}

func (bucket *boltBucket) Has(key []byte) bool {
	value, err := bucket.Get(key)
	if !(value != nil && err == nil) {
		return false
	}
	return true
}

func (bucket *boltBucket) Set(key []byte, value []byte) error {
	err := bucket.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucket.id))
		err := bucket.Put(key, value)
		return err
	})
	return err
}

func (bucket *boltBucket) Delete(key []byte) error {
	err := bucket.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucket.id))
		err := bucket.Delete(key)
		return err
	})
	return err
}
