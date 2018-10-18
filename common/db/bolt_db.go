package db

import (
	"path/filepath"

	bolt "go.etcd.io/bbolt"
)

func init() {
	dbCreator := func(name string, dir string) (DB, error) {
		return NewBoltDB(name, dir)
	}
	registerDBCreator(BoltDBBackend, dbCreator, false)
}

func NewBoltDB(name string, dir string) (*BoltDB, error) {
	dbPath := filepath.Join(dir, name+".db")
	db, err := bolt.Open(dbPath, 0644, nil)

	err = db.Update(func(tx *bolt.Tx) error {
		// TODO : boltDB need to create bucket
		_, err := tx.CreateBucketIfNotExists([]byte("loopchain"))
		return err
	})

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

// TODO : BoltDB not implements
var _ DB = (*BoltDB)(nil)

type BoltDB struct {
	db *bolt.DB
}

func (db *BoltDB) Get(key []byte) ([]byte, error) {
	panic("implement me")
}

func (db *BoltDB) Has(key []byte) bool {
	panic("implement me")
}

func (db *BoltDB) Set(key []byte, value []byte) error {
	panic("implement me")
}

func (db *BoltDB) Delete(key []byte) error {
	panic("implement me")
}

func (db *BoltDB) Transaction() (Transaction, error) {
	panic("implement me")
}

func (db *BoltDB) Batch() Batch {
	panic("implement me")
}

func (db *BoltDB) Iterator() Iterator {
	panic("implement me")
}

func (db *BoltDB) Close() error {
	panic("implement me")
}

