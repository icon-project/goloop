package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBadgerDB_DB(t *testing.T) {

	dir, err := ioutil.TempDir("", "badger")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	testDB := NewDB(BadgerDBBackend,"test", dir)
	defer testDB.Close()

	key := []byte("hello")
	value := []byte("world")

	testDB.Set(key, value)
	result, _ := testDB.Get(key)
	assert.Equal(t, value, result, "equal")
	assert.True(t, testDB.Has(key), "True")

	testDB.Delete(key)
	result, _ = testDB.Get(key)
	assert.Nil(t, result, "empty")
}

func TestBadgerDB_Transaction(t *testing.T) {

	dir, err := ioutil.TempDir("", "badger")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	testDB := NewDB(BadgerDBBackend,"test", dir)
	defer testDB.Close()

	key := []byte("hello")
	value := []byte("world")

	tx, err := testDB.Transaction()
	tx.Set(key, value)
	tx.Commit()
	result, _ := testDB.Get(key)
	assert.Equal(t, value, result, "equal")

	tx, err = testDB.Transaction()
	tx.Delete(key)
	result, _ = tx.Get(key)
	assert.Nil(t, result, "empty")
	tx.Discard()
	result, _ = testDB.Get(key)
	assert.Equal(t, value, result, "equal")
}

func TestBadgerDB_Batch(t *testing.T) {

	dir, err := ioutil.TempDir("", "badger")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	testDB := NewDB(BadgerDBBackend,"test", dir)
	defer testDB.Close()

	key := func(i int) []byte {
		return []byte(fmt.Sprintf("%09d", i))
	}
	value := func(i int) []byte {
		return []byte(fmt.Sprintf("%025d", i))
	}

	//fmt.Println("-- Batch")
	batch := testDB.Batch()
	n := 10
	for i := 0; i < n; i++ {
		batch.Set(key(i), value(i))
		//fmt.Println(key(i), value(i))
	}
	batch.Write()

	//fmt.Println("-- Iterator")
	itr := testDB.Iterator()
	defer itr.Close()

	var count int
	for itr.Seek(key(0)); itr.Valid(); itr.Next()  {
		//fmt.Println(itr.Key(), itr.Value())
		count++
	}
	assert.Equal(t, n, count, "equal")
}
