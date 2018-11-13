package db

import "log"

func init() {
	dbCreator := func(name string, dir string) (Database, error) {
		return make(mapDatabase), nil
	}
	registerDBCreator(MapDBBackend, dbCreator, false)
}

func NewMapDB() Database {
	return make(mapDatabase)
}

//----------------------------------------
// DB

var _ Database = (*mapDatabase)(nil)

type mapDatabase map[BucketID]mapBucket

func (t mapDatabase) GetBucket(id BucketID) (Bucket, error) {
	if bk, ok := t[id]; ok {
		return bk, nil
	}
	bk := make(mapBucket)
	t[id] = bk
	return bk, nil
}

func (t mapDatabase) Close() error {
	return nil
}

//----------------------------------------
// Bucket

var _ Bucket = (*mapBucket)(nil)

type mapBucket map[string]string

func (t mapBucket) Get(k []byte) ([]byte, error) {
	v, ok := t[string(k)]
	if ok {
		bytes := []byte(v)
		log.Printf("mapBucket.Get(%x) -> [%x]", k, bytes)
		return bytes, nil
	}
	log.Printf("mapBucket.Get(%x) -> FAIL", k)
	return nil, nil
}

func (t mapBucket) Has(k []byte) bool {
	_, ok := t[string(k)]
	log.Printf("mapBucket.Has(%x) -> %v", k, ok)
	return ok
}

func (t mapBucket) Set(k, v []byte) error {
	log.Printf("mapBucket.Set(%x,%x)", k, v)
	t[string(k)] = string(v)
	return nil
}

func (t mapBucket) Delete(k []byte) error {
	log.Printf("mapBucket.Delete(%x)", k)
	delete(t, string(k))
	return nil
}
