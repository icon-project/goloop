package trie_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/mpt"
)

type testDB struct {
	pool map[string][]byte
}

func (db *testDB) Get(k []byte) ([]byte, error) {
	return db.pool[string(k)], nil
}

func (db *testDB) Set(k, v []byte) error {
	db.pool[string(k)] = v
	return nil
}

func (db *testDB) Batch() db.Batch {

	return nil
}
func (db *testDB) Has(key []byte) bool {
	return false
}

func (db *testDB) Delete(key []byte) error {

	return nil
}

func (db *testDB) Transaction() (db.Transaction, error) {
	return nil, nil
}

func (db *testDB) Iterator() db.Iterator {
	return nil
}

func (db *testDB) Close() error {
	return nil
}

func newDB() *testDB {
	return &testDB{pool: make(map[string][]byte)}

}

var testPool = map[string]string{
	"doe":          "reindeer",
	"dog":          "puppy",
	"dogglesworth": "cat",
}

func TestInsert(t *testing.T) {
	trie := mpt.NewMutable(nil)

	updateString(trie, "doe", "reindeer")
	updateString(trie, "dog", "puppy")
	updateString(trie, "dogglesworth", "cat")

	hashHex := "8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3"
	strRoot := fmt.Sprintf("%x", trie.RootHash())
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}

	db := newDB()
	immutable := trie.GetSnapshot()
	immutable.Flush(db)

	mutable := mpt.NewMutable(nil)
	mutable.Reset(immutable)
	doeV, _ := mutable.Get([]byte("doe"))
	if strings.Compare(testPool["doe"], string(doeV)) != 0 {
		t.Errorf("%s vs %s", testPool["doe"], string(doeV))
	}

	trie = mpt.NewMutable(nil)
	updateString(trie, "A", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	hashHex = "d23786fb4a010da3ce639d66d5e904a11dbc02746d1ce25029e53290cabf28ab"
	strRoot = fmt.Sprintf("%x", trie.RootHash())
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}
}

func TestDelete(t *testing.T) {
	trie := mpt.NewMutable(nil)

	updateString(trie, "doe", "reindeer")
	solution1 := fmt.Sprintf("%x", trie.RootHash())
	updateString(trie, "dog", "puppy")
	solution2 := fmt.Sprintf("%x", trie.RootHash())
	updateString(trie, "dogglesworth", "cat")

	hashHex := "8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3"
	strRoot := fmt.Sprintf("%x", trie.RootHash())
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}

	trie.Delete([]byte("dogglesworth"))
	resultRoot := fmt.Sprintf("%x", trie.RootHash())
	if strings.Compare(solution2, resultRoot) != 0 {
		t.Errorf("solution %s, result %s", solution2, resultRoot)
	}
	trie.Delete([]byte("dog"))
	resultRoot = fmt.Sprintf("%x", trie.RootHash())
	if strings.Compare(solution1, resultRoot) != 0 {
		t.Errorf("solution %s, result %s", solution1, resultRoot)
	}
}

/*
func TestSet(t *testing.T) {
	mutable := mpt.NewMutable(nil)
	mutable.Set([]byte{10, 11, 12}, []byte{1, 2, 3, 4, 5})
	mutable.Set([]byte{10, 21, 22}, []byte{21, 22, 23, 24, 25})
	mutable.Set([]byte{20, 21, 22}, []byte{11, 12, 13, 14, 15})

	v := mutable.Get([]byte{10, 11, 12})
	fmt.Println("v : ", v)

	mutable.RootHash()

	snapShot := mutable.GetSnapshot()

	db := &testDB{pool: make(map[string][]byte)}
	snapShot.Flush(db)
	fmt.Println("db : ", db.pool)

	mutable2 := mpt.NewMutable(nil)
	mutable2.Reset(snapShot)
	fmt.Println("new mutable object & set snapshot & get something")
	v2 := mutable2.Get([]byte{10, 11, 12})
	v3 := mutable2.Get([]byte{20, 21, 22})
	fmt.Println("result of getting . : ", v2, ", v3 : ", v3)
}
*/

func updateString(trie trie.Mutable, k, v string) {
	trie.Set([]byte(k), []byte(v))
}
