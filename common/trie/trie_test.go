package trie_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/mpt"
)

var testPool = map[string]string{
	"doe":          "reindeer",
	"dog":          "puppy",
	"dogglesworth": "cat",
}

func TestCommit(t *testing.T) {
	manager := mpt.NewManager(db.NewMapDB())
	trie := manager.NewMutable(nil)
	rootHash := make([]string, 3)
	i := 0

	poolKey := []string{
		"doe", "dog", "dogglesworth",
	}
	for i, k := range poolKey {
		updateString(trie, k, testPool[k])
		snapshot := trie.GetSnapshot()
		snapshot.Flush()
		rootHash[i] = fmt.Sprintf("%x", snapshot.RootHash())
		i++
	}

	for i > 0 {
		i--
		snapshot := trie.GetSnapshot()
		root := fmt.Sprintf("%x", snapshot.RootHash())
		if strings.Compare(root, rootHash[i]) != 0 {
			t.Errorf("%s vs %s", root, rootHash[i])
		}
		trie.Delete([]byte(poolKey[i]))

	}
}

func TestInsert(t *testing.T) {
	manager := mpt.NewManager(db.NewMapDB())
	trie := manager.NewMutable(nil)

	for k, v := range testPool {
		updateString(trie, k, v)
	}

	hashHex := "8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3"
	immutable := trie.GetSnapshot()
	strRoot := fmt.Sprintf("%x", immutable.RootHash())
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}
	immutable.Flush()

	doeV, _ := immutable.Get([]byte("doe"))
	if strings.Compare(testPool["doe"], string(doeV)) != 0 {
		t.Errorf("%s vs %s", testPool["doe"], string(doeV))
	}

	trie = manager.NewMutable(nil)
	updateString(trie, "A", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	immutable = trie.GetSnapshot()
	hashHex = "d23786fb4a010da3ce639d66d5e904a11dbc02746d1ce25029e53290cabf28ab"
	strRoot = fmt.Sprintf("%x", immutable.RootHash())
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}
}

func TestDelete1(t *testing.T) {
	manager := mpt.NewManager(db.NewMapDB())
	trie := manager.NewMutable(nil)

	updateString(trie, "doe", "reindeer")
	immutable := trie.GetSnapshot() // SNAPSHOT 1 - doe
	solution1 := fmt.Sprintf("%x", immutable.RootHash())
	updateString(trie, "dog", "puppy")
	immutable = trie.GetSnapshot() // SNAPSHOT 2 - doe, dog
	solution2 := fmt.Sprintf("%x", immutable.RootHash())
	updateString(trie, "dogglesworth", "cat")

	hashHex := "8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3"
	immutable = trie.GetSnapshot() // SNAPSHOT 3 - doe, dog, dogglesworth
	strRoot := fmt.Sprintf("%x", immutable.RootHash())
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}

	trie.Delete([]byte("dogglesworth"))
	immutable = trie.GetSnapshot() // SNAPSHOT 4 - doe, dog
	resultRoot := fmt.Sprintf("%x", immutable.RootHash())
	if strings.Compare(solution2, resultRoot) != 0 {
		t.Errorf("solution %s, result %s", solution2, resultRoot)
	}
	trie.Delete([]byte("dog"))
	immutable = trie.GetSnapshot() // SNAPSHOT 4 - doe

	resultRoot = fmt.Sprintf("%x", immutable.RootHash())
	if strings.Compare(solution1, resultRoot) != 0 {
		t.Errorf("solution %s, result %s", solution1, resultRoot)
	}
}

func TestDelete2(t *testing.T) {
	manager := mpt.NewManager(db.NewMapDB())
	trie := manager.NewMutable(nil)
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		if val.v != "" {
			updateString(trie, val.k, val.v)
		} else {
			deleteString(trie, val.k)
		}
	}

	snapshot := trie.GetSnapshot()
	strRoot := fmt.Sprintf("%x", snapshot.RootHash())
	hashHex := "5991bb8c6514148a29db676a14ac506cd2cd5775ace63c30a4fe457715e9ac84"
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}
}

func TestCache(t *testing.T) {
	manager := mpt.NewManager(db.NewMapDB())
	mutable := manager.NewMutable(nil)

	for k, v := range testPool {
		updateString(mutable, k, v)
	}

	hashHex := "8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3"
	snapshot := mutable.GetSnapshot()
	root := snapshot.RootHash()
	strRoot := fmt.Sprintf("%x", root)
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}

	snapshot.Flush()
	// check : Does db in Snapshot have to be passed to Mutable?
	//cacheTrie := mpt.NewCache(nil)
	//cacheTrie.Load(db, root)
	immutable := manager.NewImmutable(root)
	for k, v := range testPool {
		value, _ := immutable.Get([]byte(k))
		if bytes.Compare(value, []byte(v)) != 0 {
			t.Errorf("Wrong value. expected [%x] but [%x]", v, value)
		}
	}

}

func TestDeleteSnapshot(t *testing.T) {
	// delete, snapshot, write
	manager := mpt.NewManager(db.NewMapDB())
	trie := manager.NewMutable(nil)

	updateString(trie, "doe", "reindeer")
	updateString(trie, "dog", "puppy")
	snapshot := trie.GetSnapshot() // SNAPSHOT - doe, dog
	solution2 := fmt.Sprintf("%x", snapshot.RootHash())
	updateString(trie, "dogglesworth", "cat")

	snapshot = trie.GetSnapshot() // SNAPSHOT - doe, dog, dogglesworth
	hashHex := "8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3"
	strRoot := fmt.Sprintf("%x", snapshot.RootHash())
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}

	snapshot.Flush()
	trie2 := manager.NewMutable(nil)
	trie2.Reset(snapshot) // have doe, dog, dogglesworth
	snapshot2 := trie2.GetSnapshot()
	strRoot = fmt.Sprintf("%x", snapshot2.RootHash())
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}

	deleteString(trie2, "dogglesworth")

	snapshot2 = trie2.GetSnapshot()                   // SNAPSHOT = doe, dog
	strRoot = fmt.Sprintf("%x", snapshot2.RootHash()) // have doe, dog
	if strings.Compare(strRoot, solution2) != 0 {
		t.Errorf("exp %s got %s", solution2, strRoot)
	}

	// Get snapshot after delete dogglesworth
	snapshot = trie2.GetSnapshot()
	snapshot.Flush()

	hashAfterDelete := fmt.Sprintf("%x", snapshot.RootHash())
	trie2.Reset(snapshot)
	snapshot = trie2.GetSnapshot()
	strRoot = fmt.Sprintf("%x", snapshot.RootHash())
	if strings.Compare(strRoot, hashAfterDelete) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}
	if strings.Compare(solution2, hashAfterDelete) != 0 {
		t.Errorf("exp %s got %s", solution2, strRoot)
	}
}

func TestLateFlush(t *testing.T) {
	manager := mpt.NewManager(db.NewMapDB())
	tr := manager.NewMutable(nil)
	poolList := []string{
		"doe",
		"dog",
		"dogglesworth",
	}

	var ssList [3]trie.Snapshot
	var hashList [3]string

	for i, k := range poolList {
		updateString(tr, k, testPool[k])
		ssList[i] = tr.GetSnapshot()
		hashList[i] = fmt.Sprintf("%x", ssList[i].RootHash())
	}

	for i, _ := range poolList {
		ssList[i].Flush()
		rootHash := fmt.Sprintf("%x", ssList[i].RootHash())
		if strings.Compare(hashList[i], rootHash) != 0 {
			t.Errorf("exp %s got %s", hashList[i], rootHash)
		}
	}
}

func TestNoHashed(t *testing.T) {
	manager := mpt.NewManager(db.NewMapDB())
	tr := manager.NewMutable(nil)

	unchanged := byte(0xFD)
	tr.Set([]byte{0x00}, []byte{0xFF})
	tr.Set([]byte{0x00, 0x01}, []byte{0xFE})
	tr.Set([]byte{0x00, 0x01, 0x00}, []byte{unchanged})

	immutalble := tr.GetSnapshot()
	immutalble.RootHash()
	immutalble.Flush()
	v, _ := immutalble.Get([]byte{0x0, 0x01, 0x00})
	if v[0] != unchanged {
		t.Errorf("%d : %d", v[0], unchanged)
	}
	changed := byte(0xFA)
	tr.Set([]byte{0x00, 0x01, 0x00}, []byte{changed})
	immutalble = tr.GetSnapshot()
	immutalble.RootHash()
	immutalble.Flush()
	v, _ = immutalble.Get([]byte{0x0, 0x01, 0x00})

	if v[0] != changed {
		t.Errorf("%d : %d", v[0], changed)
	}
}

func TestNull(t *testing.T) {
	manager := mpt.NewManager(db.NewMapDB())
	tr := manager.NewMutable(nil)

	key := make([]byte, 32)
	value := []byte("test")
	tr.Set(key, value)
	v, _ := tr.Get(key)
	if !bytes.Equal(v, value) {
		t.Fatal("wrong value")
	}
}

func TestProof(t *testing.T) {
	manager := mpt.NewManager(db.NewMapDB())
	tr := manager.NewMutable(nil)

	key := make([]byte, 32)
	value := []byte("test")
	tr.Set(key, value)
	v, _ := tr.Get(key)
	if !bytes.Equal(v, value) {
		t.Fatal("wrong value")
	}
	fmt.Println(tr.Proof(key))
}
func updateString(trie trie.Mutable, k, v string) {
	trie.Set([]byte(k), []byte(v))
}

func deleteString(trie trie.Mutable, k string) {
	trie.Delete([]byte(k))
}
