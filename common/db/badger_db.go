package db

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"

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

	err = database.readMeta(nil)
	if err == badger.ErrKeyNotFound {
		database.buckets = bucketMeta{
			buckets: make(map[string]bucketId),
		}
		err = database.writeMeta(nil)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	database.bucketSequence, err = database.db.GetSequence(bucketIdSequence, 1e3)
	if err != nil {
		return nil, err
	}

	return database, nil
}

//----------------------------------------
// DB

var _ Database = (*BadgerDB)(nil)

type BadgerDB struct {
	db *badger.DB
	buckets bucketMeta
	bucketMutex sync.RWMutex
	bucketSequence *badger.Sequence
}

func (db *BadgerDB) GetBucket(name string) (Bucket, error) {
	bucket, ok := db.bucket(name)
	if !ok {
		var err error
		bucket, err = db.createBucket(nil, name)
		if err != nil {
			return nil, err
		}
	}
	return bucket, nil
}

func (db *BadgerDB) Close() error {
	err := db.db.Close()
	return err
}

func (db *BadgerDB) bucket(name string) (Bucket, bool) {
	db.bucketMutex.RLock()
	meta, ok := db.buckets.buckets[name]
	db.bucketMutex.RUnlock()
	if !ok {
		return nil, false
	}
	bucket := &badgerBucket{
		id: meta,
		db: db.db,
	}
	return bucket, true
}

func (db *BadgerDB) createBucket(txn *badger.Txn, name string) (Bucket, error) {
	db.bucketMutex.Lock()
	defer db.bucketMutex.Unlock()

	meta, ok := db.buckets.buckets[name]
	if ok {
		return &badgerBucket{id: meta, db: db.db}, nil
	}

	nextId, err := db.bucketSequence.Next()
	if err != nil {
		return nil, err
	}
	// This increments the first byte of the bucket id by 8. The bucket id
	// prefixes records in the database, and since values 0 to 8 of the
	// first byte of keys are reserved for internal use, bucket ids can't
	// have their first byte between 0 and 8.
	nextId += 8 * 256
	if nextId > MaxBuckets {
		return nil, fmt.Errorf("bow.createBucket: reached maximum amount of buckets limit (%d)", MaxBuckets)
	}

	var id bucketId
	binary.BigEndian.PutUint16(id[:], uint16(nextId))
	db.buckets.buckets[name] = id
	err = db.writeMeta(txn)
	if err != nil {
		return nil, err
	}

	return &badgerBucket{id: id, db: db.db}, err
}

func (db *BadgerDB) readMeta(txn *badger.Txn) error {
	if txn == nil {
		txn = db.db.NewTransaction(false)
		defer func() {
			txn.Discard()
		}()
	}
	item, err := txn.Get(metaKey)
	if err != nil {
		return err
	}
	b, err := item.Value()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &db.buckets)
}

func (db *BadgerDB) writeMeta(txn *badger.Txn) (err error) {
	if txn == nil {
		txn = db.db.NewTransaction(true)
		defer func() {
			err = txn.Commit(nil)
		}()
	}
	b, err := json.Marshal(db.buckets)

	if err != nil {
		return err
	}
	err = txn.Set(metaKey, b)
	return
}

//----------------------------------------
// Bucket

var _ Bucket = (*badgerBucket)(nil)

type badgerBucket struct {
	id bucketId
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
		value, err = item.Value()
		return  err
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
