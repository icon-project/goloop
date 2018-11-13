package db

import (
	"fmt"
	"strings"
)

type Database interface {
	GetBucket(id BucketID) (Bucket, error)
	Close() error
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

func Open(name string) Database {
	// TODO : configure Database options
	defaultBackend := BadgerDBBackend
	dir := "./data"
	return openDatabase(defaultBackend, name, dir)
}

func openDatabase(backend BackendType, name string, dir string) Database {
	dbCreator, ok := backends[backend]
	if !ok {
		keys := make([]string, len(backends))
		i := 0
		for k := range backends {
			keys[i] = string(k)
			i++
		}
		panic(fmt.Sprintf("Unknown db_backend %s, expected either %s", backend, strings.Join(keys, " or ")))
	}

	db, err := dbCreator(name, dir)
	if err != nil {
		panic(fmt.Sprintf("Error initializing Database: %v", err))
	}
	return db
}
