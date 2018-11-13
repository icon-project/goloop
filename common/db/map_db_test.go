package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapDB_Database(t *testing.T) {

	testDB := openDatabase(MapDBBackend, "", "")
	defer testDB.Close()

	key := []byte("hello")
	value := []byte("world")

	bucket, _ := testDB.GetBucket("hello")
	bucket.Set(key, value)
	result, _ := bucket.Get(key)
	assert.Equal(t, value, result, "equal")
	assert.True(t, bucket.Has(key), "True")

	bucket.Delete(key)
	result, _ = bucket.Get(key)
	assert.Nil(t, result, "empty")
}
