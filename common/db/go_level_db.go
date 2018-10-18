package db

import (
	"path/filepath"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

func init() {
	dbCreator := func(name string, dir string) (DB, error) {
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
// DB

var _ DB = (*GoLevelDB)(nil)

type GoLevelDB struct {
	db *leveldb.DB
}

func (db *GoLevelDB) DB() *leveldb.DB {
	return db.db
}

func (db *GoLevelDB) Get(key []byte) ([]byte, error) {
	key = nonNilBytes(key)
	res, err := db.db.Get(key, nil)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, nil
		}
	}
	return res, err
}

func (db *GoLevelDB) Has(key []byte) bool {
	value, err := db.Get(key)
	if !(value != nil && err == nil) {
		return false
	}
	return true
}

func (db *GoLevelDB) Set(key []byte, value []byte) error{
	key = nonNilBytes(key)
	value = nonNilBytes(value)
	return db.db.Put(key, value, nil)

}

func (db *GoLevelDB) Delete(key []byte) error {
	key = nonNilBytes(key)
	return db.db.Delete(key, nil)

}

func (db *GoLevelDB) Transaction() (Transaction, error) {
	txn, err := db.db.OpenTransaction()
	return &goLevelDBTx{txn: txn}, err
}

func (db *GoLevelDB) Batch() (Batch) {
	b := new(leveldb.Batch)
	return &goLevelDBBatch{db: db.db, batch:b}
}

func (db *GoLevelDB) Iterator() (Iterator) {
	itr := db.db.NewIterator(nil, nil)
	return &goLevelDBIterator{ itr: itr}
}

func (db *GoLevelDB) Close() error {
	return db.db.Close()
}

//----------------------------------------
// Transaction

var _ Transaction = (*goLevelDBTx)(nil)

type goLevelDBTx struct {
	txn *leveldb.Transaction
}

func (tx *goLevelDBTx) Get(key []byte) ([]byte, error) {
	key = nonNilBytes(key)
	value, err := tx.txn.Get(key, nil)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, nil
		}
	}
	return value, err
}

func (tx *goLevelDBTx) Set(key []byte, value []byte) error {
	key = nonNilBytes(key)
	value = nonNilBytes(value)
	return tx.txn.Put(key, value, nil)
}

func (tx *goLevelDBTx) Delete(key []byte) error {
	key = nonNilBytes(key)
	return tx.txn.Delete(key, nil)
}

func (tx *goLevelDBTx) Commit() error {
	return tx.txn.Commit()
}

func (tx *goLevelDBTx) Discard() {
	tx.txn.Discard()
}

//----------------------------------------
// Batch

var _ Batch = (*goLevelDBBatch)(nil)

type goLevelDBBatch struct {
	db *leveldb.DB
	batch *leveldb.Batch
}

func (batch *goLevelDBBatch) Set(key []byte, value []byte) error {
	key = nonNilBytes(key)
	value = nonNilBytes(value)
	batch.batch.Put(key, value)
	return nil
}

func (batch *goLevelDBBatch) Delete(key []byte) error {
	key = nonNilBytes(key)
	batch.batch.Delete(key)
	return nil
}

func (batch *goLevelDBBatch) Write() error {
	return batch.db.Write(batch.batch, nil)
}

//----------------------------------------
// Iterator

var _ Iterator = (*goLevelDBIterator)(nil)

type goLevelDBIterator struct {
	itr iterator.Iterator
}

func (itr *goLevelDBIterator) Seek(key []byte) {
	panic("implement me")
}

func (itr *goLevelDBIterator) Next() {
	panic("implement me")
}

func (itr *goLevelDBIterator) Valid() bool {
	panic("implement me")
}

func (itr *goLevelDBIterator) Key() (key []byte) {
	panic("implement me")
}

func (itr *goLevelDBIterator) Value() (value []byte) {
	panic("implement me")
}

func (itr *goLevelDBIterator) Close() {
	panic("implement me")
}

