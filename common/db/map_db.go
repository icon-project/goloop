package db

import (
	"fmt"
	"github.com/pkg/errors"
	"log"
)

const (
	configLogMapDB = false
)

func init() {
	dbCreator := func(name string, dir string) (Database, error) {
		return &mapDatabase{
			name: name,
			bks:  map[BucketID]*mapBucket{},
		}, nil
	}
	registerDBCreator(MapDBBackend, dbCreator, false)
}

func NewMapDB() Database {
	dbase := &mapDatabase{
		bks: map[BucketID]*mapBucket{},
	}
	dbase.name = fmt.Sprintf("%p", dbase)
	return dbase
}

//----------------------------------------
// DB

var _ Database = (*mapDatabase)(nil)

type mapDatabase struct {
	name string
	bks  map[BucketID]*mapBucket
}

func (t *mapDatabase) GetBucket(id BucketID) (Bucket, error) {
	if bk, ok := t.bks[id]; ok {
		return bk, nil
	}
	bk := &mapBucket{
		id:   fmt.Sprintf("%s:%s", t.name, id),
		real: make(map[string]string),
	}
	t.bks[id] = bk
	return bk, nil
}

func (t *mapDatabase) Close() error {
	return nil
}

//----------------------------------------
// Bucket

var _ Bucket = (*mapBucket)(nil)

type mapBucket struct {
	id   string
	real map[string]string
}

func (t *mapBucket) Get(k []byte) ([]byte, error) {
	v, ok := t.real[string(k)]
	if ok {
		bytes := []byte(v)
		if configLogMapDB {
			log.Printf("mapBucket[%s].Get(%x) -> [%x]", t.id, k, bytes)
		}
		return bytes, nil
	}
	if configLogMapDB {
		log.Printf("mapBucket[%s].Get(%x) -> FAIL", t.id, k)
	}
	return nil, nil
}

func (t *mapBucket) Has(k []byte) bool {
	_, ok := t.real[string(k)]
	if configLogMapDB {
		log.Printf("mapBucket[%s].Has(%x) -> %v", t.id, k, ok)
	}
	return ok
}

func (t *mapBucket) Set(k, v []byte) error {
	if len(k) == 0 {
		return errors.Errorf("Illegal Key:%x", k)
	}
	if configLogMapDB {
		log.Printf("mapBucket[%s].Set(%x,%x)", t.id, k, v)
	}
	t.real[string(k)] = string(v)
	return nil
}

func (t *mapBucket) Delete(k []byte) error {
	if configLogMapDB {
		log.Printf("mapBucket[%s].Delete(%x)", t.id, k)
	}
	delete(t.real, string(k))
	return nil
}
