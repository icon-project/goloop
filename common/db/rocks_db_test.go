// +build rocksdb

package db

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRocksDB_Database(t *testing.T) {

	dir, err := ioutil.TempDir("", "rocksdb")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	testDB, _ := openDatabase(RocksDBBackend, "test", dir)
	defer testDB.Close()

	key := []byte("hello")
	value := []byte("world")

	bucket, _ := testDB.GetBucket("hello")

	assert.False(t, bucket.Has(key), "False")

	bucket.Set(key, value)
	result, _ := bucket.Get(key)
	assert.Equal(t, value, result, "equal")
	assert.True(t, bucket.Has(key), "True")

	bucket.Delete(key)
	result, _ = bucket.Get(key)
	assert.Nil(t, result, "empty")
}