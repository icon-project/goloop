package db

import (
	"sort"

	"github.com/pkg/errors"
)

type Database interface {
	GetBucket(id BucketID) (Bucket, error)
	Close() error
}

type LayerDB interface {
	Database
	Flush(write bool) error
}

type BackendType string

const (
	BadgerDBBackend  BackendType = "badgerdb"
	GoLevelDBBackend BackendType = "goleveldb"
	BoltDBBackend    BackendType = "boltdb"
	MapDBBackend     BackendType = "mapdb"
)

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
