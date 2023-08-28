//go:build rocksdb
// +build rocksdb

/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package db

import (
	"errors"
	"os"
	"path"
	"reflect"
	"sync"
	"unsafe"

	"github.com/icon-project/goloop/common/log"
)

// #cgo LDFLAGS: -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy
// #include <stdlib.h>
// #include "rocksdb/c.h"
import "C"

const (
	RocksDBBackend BackendType = "rocksdb"
)

var ErrAlreadyClosed = errors.New("AlreadyClosed")

func init() {
	dbCreator := func(name string, dir string) (Database, error) {
		return NewRocksDB(name, dir)
	}
	registerDBCreator(RocksDBBackend, dbCreator, false)
}

type RocksDB struct {
	lock    sync.RWMutex
	bkLock  sync.Mutex
	buckets map[BucketID]*RocksBucket

	db *C.rocksdb_t
	ro *C.rocksdb_readoptions_t
	wo *C.rocksdb_writeoptions_t
}

func NewRocksDB(name string, dir string) (*RocksDB, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Errorln("fail to MkdirAll", err.Error())
		return nil, err
	}
	opts := C.rocksdb_options_create()
	C.rocksdb_options_set_create_if_missing(opts, C.uchar(1))
	C.rocksdb_options_set_create_missing_column_families(opts, C.uchar(1))

	var (
		cErr    *C.char
		cName   = C.CString(path.Join(dir, name))
		hdl     *C.rocksdb_t
		buckets = make(map[BucketID]*RocksBucket)
	)
	defer C.free(unsafe.Pointer(cName))

	var cfsLen C.size_t
	if cfs := C.rocksdb_list_column_families(opts, cName, &cfsLen, &cErr); cErr != nil {
		errMsg := C.GoString(cErr)
		C.rocksdb_free(unsafe.Pointer(cErr))
		log.Traceln("fail to rocksdb_list_column_families", errMsg)

		// ignore and try open
		cErr = nil
		hdl = C.rocksdb_open(opts, cName, &cErr)
		if cErr != nil {
			errMsg = C.GoString(cErr)
			defer C.rocksdb_free(unsafe.Pointer(cErr))
			log.Errorln("fail to rocksdb_open", errMsg)
			return nil, errors.New(errMsg)
		}
	} else {
		numOfCfs := int(cfsLen)
		log.Traceln("rocksdb_list_column_families returns num:", numOfCfs)

		cfOpts := make([]*C.rocksdb_options_t, numOfCfs)
		for i := 0; i < numOfCfs; i++ {
			cfOpts[i] = C.rocksdb_options_create()
		}
		cfhs := make([]*C.rocksdb_column_family_handle_t, numOfCfs)
		hdl = C.rocksdb_open_column_families(
			opts,
			cName,
			C.int(numOfCfs),
			cfs,
			&cfOpts[0],
			&cfhs[0],
			&cErr)
		if cErr != nil {
			errMsg := C.GoString(cErr)
			defer C.rocksdb_free(unsafe.Pointer(cErr))
			log.Errorln("fail to rocksdb_column_family_handle_t", errMsg)
			return nil, errors.New(errMsg)
		}
		if numOfCfs > 1 {
			cNamesArr := (*[(1 << 29) - 1]*C.char)(unsafe.Pointer(cfs))[:int(cfsLen):int(cfsLen)]
			for i := 1; i < numOfCfs; i++ {
				id := C.GoString(cNamesArr[i])
				bk := &RocksBucket{
					cf: cfhs[i],
				}
				buckets[BucketID(id)] = bk
			}
		}
		if cfsLen > 0 {
			C.rocksdb_column_family_handle_destroy(cfhs[0])
			C.rocksdb_list_column_families_destroy(cfs, cfsLen)
		}
	}

	ro := C.rocksdb_readoptions_create()
	wo := C.rocksdb_writeoptions_create()
	rdb := &RocksDB{
		db:      hdl,
		ro:      ro,
		wo:      wo,
		buckets: buckets,
	}
	if len(buckets) > 0 {
		for _, bk := range buckets {
			bk.db = rdb
		}
	}
	return rdb, nil
}

func (db *RocksDB) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if db.db == nil {
		return ErrAlreadyClosed
	}

	db.bkLock.Lock()
	defer db.bkLock.Unlock()

	for _, bk := range db.buckets {
		C.rocksdb_column_family_handle_destroy(bk.cf)
	}
	C.rocksdb_close(db.db)
	db.db = nil
	return nil
}

func (db *RocksDB) GetBucket(id BucketID) (Bucket, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.db == nil {
		return nil, ErrAlreadyClosed
	}

	db.bkLock.Lock()
	defer db.bkLock.Unlock()

	if bk, ok := db.buckets[id]; ok {
		return bk, nil
	}

	cName := C.CString(string(id))
	defer C.free(unsafe.Pointer(cName))
	var cErr *C.char

	opts := C.rocksdb_options_create()
	C.rocksdb_options_set_create_if_missing(opts, C.uchar(1))
	C.rocksdb_options_set_create_missing_column_families(opts, C.uchar(1))
	cf := C.rocksdb_create_column_family(db.db, opts, cName, &cErr)
	if cErr != nil {
		defer C.rocksdb_free(unsafe.Pointer(cErr))
		return nil, errors.New(C.GoString(cErr))
	}
	bk := &RocksBucket{
		cf: cf,
		db: db,
	}
	db.buckets[id] = bk
	return bk, nil
}

func unsafePointerOf(p []byte) unsafe.Pointer {
	if len(p) == 0 {
		return nil
	} else {
		return unsafe.Pointer(&p[0])
	}
}

func (db *RocksDB) getValue(cf *C.rocksdb_column_family_handle_t, k []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.db == nil {
		return nil, ErrAlreadyClosed
	}
	var (
		cErr    *C.char
		cValLen C.size_t
		cKey    = (*C.char)(unsafePointerOf(k))
	)
	cValue := C.rocksdb_get_cf(db.db, db.ro, cf, cKey, C.size_t(len(k)), &cValLen, &cErr)
	if cErr != nil {
		defer C.rocksdb_free(unsafe.Pointer(cErr))
		return nil, errors.New(C.GoString(cErr))
	}
	if cValue == nil {
		return nil, nil
	}
	var org []byte
	sH := (*reflect.SliceHeader)(unsafe.Pointer(&org))
	sH.Cap, sH.Len, sH.Data = int(cValLen), int(cValLen), uintptr(unsafe.Pointer(cValue))
	defer C.rocksdb_free(unsafe.Pointer(cValue))
	value := make([]byte, int(cValLen))
	copy(value, org)
	return value, nil
}

func (db *RocksDB) hasValue(cf *C.rocksdb_column_family_handle_t, k []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.db == nil {
		return false, ErrAlreadyClosed
	}
	var (
		cErr    *C.char
		cValLen C.size_t
		cKey    = (*C.char)(unsafePointerOf(k))
	)
	cValue := C.rocksdb_get_cf(db.db, db.ro, cf, cKey, C.size_t(len(k)), &cValLen, &cErr)
	if cErr != nil {
		defer C.rocksdb_free(unsafe.Pointer(cErr))
		return false, errors.New(C.GoString(cErr))
	}
	defer C.rocksdb_free(unsafe.Pointer(cValue))
	return cValue != nil, nil
}

func (db *RocksDB) setValue(cf *C.rocksdb_column_family_handle_t, k, v []byte) error {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.db == nil {
		return ErrAlreadyClosed
	}
	var (
		cErr   *C.char
		cKey   = (*C.char)(unsafePointerOf(k))
		cValue = (*C.char)(unsafePointerOf(v))
	)
	C.rocksdb_put_cf(db.db, db.wo, cf, cKey, C.size_t(len(k)), cValue, C.size_t(len(v)), &cErr)
	if cErr != nil {
		defer C.rocksdb_free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}
	return nil
}

func (db *RocksDB) deleteValue(cf *C.rocksdb_column_family_handle_t, k []byte) error {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if db.db == nil {
		return ErrAlreadyClosed
	}
	var (
		cErr *C.char
		cKey = (*C.char)(unsafePointerOf(k))
	)
	C.rocksdb_delete_cf(db.db, db.wo, cf, cKey, C.size_t(len(k)), &cErr)
	if cErr != nil {
		defer C.rocksdb_free(unsafe.Pointer(cErr))
		return errors.New(C.GoString(cErr))
	}
	return nil
}

type RocksBucket struct {
	cf *C.rocksdb_column_family_handle_t
	db *RocksDB
}

func (b *RocksBucket) Get(key []byte) ([]byte, error) {
	return b.db.getValue(b.cf, key)
}

func (b *RocksBucket) Has(key []byte) (bool, error) {
	return b.db.hasValue(b.cf, key)
}

func (b *RocksBucket) Set(key []byte, value []byte) error {
	return b.db.setValue(b.cf, key, value)
}

func (b *RocksBucket) Delete(key []byte) error {
	return b.db.deleteValue(b.cf, key)
}
