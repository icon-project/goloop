package db

import (
	"path/filepath"

	"github.com/dgraph-io/badger"
)

func init() {
	dbCreator := func(name string, dir string) (DB, error) {
		return NewBadgerDB(name, dir)
	}
	registerDBCreator(BadgerDBBackend, dbCreator, false)
}

func NewBadgerDB(name string, dir string) (*BadgerDB, error) {

	dbPath := filepath.Join(dir, name)
	opts := badger.DefaultOptions
	opts.Dir = dbPath
	opts.ValueDir = dbPath

	// TODO : badger.Open() use os.Mkdir(). parent dirs must be created
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

var _ DB = (*BadgerDB)(nil)

type BadgerDB struct {
	db *badger.DB
}

func (db *BadgerDB) DB() *badger.DB {
	return db.db
}

func (db *BadgerDB) Get(key []byte) ([]byte, error) {
	key = nonNilBytes(key)
	var value []byte
	err := db.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
		}
		value, err = item.ValueCopy(nil)
		return  err
	})
	return value, err
}

func (db *BadgerDB) Has(key []byte) bool {
	value, err := db.Get(key)
	if !(value != nil && err == nil) {
		return false
	}
	return true
}

func (db *BadgerDB) Set(key []byte, value []byte) error {
	key = nonNilBytes(key)
	value = nonNilBytes(value)
	return db.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(key, value)
		return err
	})
}

func (db *BadgerDB) Delete(key []byte) error {
	key = nonNilBytes(key)
	return db.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (db *BadgerDB) Transaction() (Transaction, error) {
	txn := db.db.NewTransaction(true)
	return &badgerDBTx{ txn: txn }, nil
}

func (db *BadgerDB) Batch() (Batch) {
	wBatch := db.db.NewWriteBatch()
	return &badgerDBBatch{ batch: wBatch}
}

func (db *BadgerDB) Iterator() (Iterator) {
	txn := db.db.NewTransaction(false)
	itr := txn.NewIterator(badger.DefaultIteratorOptions)
	return &badgerDBIterator{
		txn: txn,
		iterator: itr,
	}
}

func (db *BadgerDB) Close() error {
	return db.db.Close()
}

//----------------------------------------
// Transaction

var _ Transaction = (*badgerDBTx)(nil)

type badgerDBTx struct {
	txn *badger.Txn
}

func (tx *badgerDBTx) Get(key []byte) ([]byte, error) {
	key = nonNilBytes(key)
	item, err := tx.txn.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
	}
	return item.ValueCopy(nil)
}

func (tx *badgerDBTx) Set(key []byte, value []byte) error {
	key = nonNilBytes(key)
	value = nonNilBytes(value)
	return tx.txn.Set(key, value)
}

func (tx *badgerDBTx) Delete(key []byte) error {
	key = nonNilBytes(key)
	return tx.txn.Delete(key)
}

func (tx *badgerDBTx) Commit() error {
	return tx.txn.Commit()
}

func (tx *badgerDBTx) Discard() {
	tx.txn.Discard()
}

//----------------------------------------
// Batch

var _ Batch = (*badgerDBBatch)(nil)

type badgerDBBatch struct {
	batch *badger.WriteBatch
}

func (batch *badgerDBBatch) Set(key []byte, value []byte) error {
	key = nonNilBytes(key)
	value = nonNilBytes(value)
	return batch.batch.Set(key, value, 0)
}

func (batch *badgerDBBatch) Delete(key []byte) error {
	key = nonNilBytes(key)
	return batch.batch.Delete(key)
}

func (batch *badgerDBBatch) Write() error {
	return batch.batch.Flush()
}

//----------------------------------------
// Iterator

var _ Iterator = (*badgerDBIterator)(nil)

type badgerDBIterator struct {
	txn *badger.Txn
	iterator *badger.Iterator
}

func (itr *badgerDBIterator) Seek(key []byte) {
	key = nonNilBytes(key)
	itr.iterator.Seek(key)
}

func (itr *badgerDBIterator) Next() {
	itr.iterator.Next()
}

func (itr *badgerDBIterator) Valid() bool {
	return itr.iterator.Valid()
}

func (itr *badgerDBIterator) Key() (key []byte) {
	item := itr.iterator.Item()
	return item.KeyCopy(nil)
}

func (itr *badgerDBIterator) Value() (value []byte) {
	item := itr.iterator.Item()
	val, err := item.ValueCopy(nil)
	if err != nil {
		panic(err)
	}
	return val
}

func (itr *badgerDBIterator) Close() {
	itr.iterator.Close()
	itr.txn.Discard()
}

