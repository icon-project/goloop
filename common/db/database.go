package db

import (
	"sort"

	"github.com/icon-project/goloop/common/errors"
)

type Database interface {
	GetBucket(id BucketID) (Bucket, error)
	Close() error
}

type LayerDB interface {
	Database
	Flush(write bool) error
	Unwrap() Database
}

type BackendType string

type dbCreator func(name string, dir string) (Database, error)

var backends = map[BackendType]dbCreator{}

func registerDBCreator(backend BackendType, creator dbCreator, force bool) {
	_, ok := backends[backend]
	if !force && ok {
		return
	}
	backends[backend] = creator
}

func RegisteredBackendTypes() []string {
	l := make([]string, 0)
	for k := range backends {
		l = append(l, string(k))
	}
	sort.Strings(l)
	return l
}

func Open(dir, dbtype, name string) (Database, error) {
	return openDatabase(BackendType(dbtype), name, dir)
}

func openDatabase(backend BackendType, name string, dir string) (Database, error) {
	dbCreator, ok := backends[backend]
	if !ok {
		keys := make([]string, len(backends))
		i := 0
		for k := range backends {
			keys[i] = string(k)
			i++
		}
		return nil, errors.Errorf("UnknownBackend(type=%s)", backend)
	}

	return dbCreator(name, dir)
}

func GetSupportedTypes() []string {
	types := make([]string, 0, len(backends))
	for be := range backends {
		types = append(types, string(be))
	}
	return types
}

type errorBucket struct {
	error
}

func (e *errorBucket) Get(key []byte) ([]byte, error)     { return nil, e.error }
func (e *errorBucket) Has(key []byte) (bool, error)       { return false, e.error }
func (e *errorBucket) Set(key []byte, value []byte) error { return e.error }
func (e *errorBucket) Delete(key []byte) error            { return e.error }

// BucketOf returns valid bucket always, but it
func BucketOf(database Database, id BucketID) Bucket {
	if bk, err := database.GetBucket(id); err != nil {
		return &errorBucket{err}
	} else {
		return bk
	}
}