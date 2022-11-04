package trie_manager

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/merkle"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

var testPool = map[string]string{
	"doe":          "reindeer",
	"dog":          "puppy",
	"dogglesworth": "cat",
}

func TestCommit(t *testing.T) {
	manager := New(db.NewMapDB())
	trie := manager.NewMutable(nil)
	rootHash := make([]string, 3)
	items := 0

	poolKey := []string{
		"doe", "dog", "dogglesworth",
	}
	for i, k := range poolKey {
		updateString(trie, k, testPool[k])
		snapshot := trie.GetSnapshot()
		snapshot.Flush()
		rootHash[i] = fmt.Sprintf("%x", snapshot.Hash())
		items++
	}

	i := items
	for i > 0 {
		i--
		snapshot := trie.GetSnapshot()
		root := fmt.Sprintf("%x", snapshot.Hash())
		if strings.Compare(root, rootHash[i]) != 0 {
			t.Errorf("%s vs %s", root, rootHash[i])
		}
		trie.Delete([]byte(poolKey[i]))
	}
}

func TestInsert(t *testing.T) {
	manager := New(db.NewMapDB())
	tree := manager.NewMutable(nil)

	for k, v := range testPool {
		updateString(tree, k, v)
	}

	hashHex := "8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3"
	if !trie.ConfigUseKeccak256 {
		hashHex = "c33dc4124235214a96518fc8bfdef30a6c1462a08e423c29f7b08513829c551f"
	}
	immutable := tree.GetSnapshot()
	strRoot := fmt.Sprintf("%x", immutable.Hash())
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}
	immutable.Flush()

	doeV, _ := immutable.Get([]byte("doe"))
	if strings.Compare(testPool["doe"], string(doeV)) != 0 {
		t.Errorf("%s vs %s", testPool["doe"], string(doeV))
	}

	tree = manager.NewMutable(nil)
	updateString(tree, "A", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	immutable = tree.GetSnapshot()
	hashHex = "d23786fb4a010da3ce639d66d5e904a11dbc02746d1ce25029e53290cabf28ab"
	if !trie.ConfigUseKeccak256 {
		hashHex = "84ec0052caf94dc2df953a1a77ed86c20b401cff8e9b85c1c68511bc3d9259a6"
	}
	strRoot = fmt.Sprintf("%x", immutable.Hash())
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}
}

func TestDelete1(t *testing.T) {
	manager := New(db.NewMapDB())
	tree := manager.NewMutable(nil)

	updateString(tree, "doe", "reindeer")
	immutable := tree.GetSnapshot() // SNAPSHOT 1 - doe
	solution1 := fmt.Sprintf("%x", immutable.Hash())
	updateString(tree, "dog", "puppy")
	immutable = tree.GetSnapshot() // SNAPSHOT 2 - doe, dog
	solution2 := fmt.Sprintf("%x", immutable.Hash())
	updateString(tree, "dogglesworth", "cat")

	hashHex := "8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3"
	if !trie.ConfigUseKeccak256 {
		hashHex = "c33dc4124235214a96518fc8bfdef30a6c1462a08e423c29f7b08513829c551f"
	}
	immutable = tree.GetSnapshot() // SNAPSHOT 3 - doe, dog, dogglesworth
	strRoot := fmt.Sprintf("%x", immutable.Hash())
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}

	tree.Delete([]byte("dogglesworth"))
	immutable = tree.GetSnapshot() // SNAPSHOT 4 - doe, dog
	resultRoot := fmt.Sprintf("%x", immutable.Hash())
	if strings.Compare(solution2, resultRoot) != 0 {
		t.Errorf("solution %s, result %s", solution2, resultRoot)
	}
	tree.Delete([]byte("dog"))
	immutable = tree.GetSnapshot() // SNAPSHOT 4 - doe

	resultRoot = fmt.Sprintf("%x", immutable.Hash())
	if strings.Compare(solution1, resultRoot) != 0 {
		t.Errorf("solution %s, result %s", solution1, resultRoot)
	}
}

func TestDelete2(t *testing.T) {
	manager := New(db.NewMapDB())
	tree := manager.NewMutable(nil)
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
			updateString(tree, val.k, val.v)
		} else {
			deleteString(tree, val.k)
		}
	}

	snapshot := tree.GetSnapshot()
	strRoot := fmt.Sprintf("%x", snapshot.Hash())
	hashHex := "5991bb8c6514148a29db676a14ac506cd2cd5775ace63c30a4fe457715e9ac84"
	if !trie.ConfigUseKeccak256 {
		hashHex = "422a17a872ce2e062b0998b85bee742dfa98f429c49bb22b81495d23fd3ce841"
	}
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}
}

func TestCache(t *testing.T) {
	manager := New(db.NewMapDB())
	mutable := manager.NewMutable(nil)

	for k, v := range testPool {
		updateString(mutable, k, v)
	}

	hashHex := "8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3"
	if !trie.ConfigUseKeccak256 {
		hashHex = "c33dc4124235214a96518fc8bfdef30a6c1462a08e423c29f7b08513829c551f"
	}
	snapshot := mutable.GetSnapshot()
	root := snapshot.Hash()
	strRoot := fmt.Sprintf("%x", root)
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}

	snapshot.Flush()
	// check : Does db in Snapshot have to be passed to Mutable?
	// cacheTrie := mpt.NewCache(nil)
	// cacheTrie.Load(db, root)
	immutable := manager.NewImmutable(root)
	for k, v := range testPool {
		value, _ := immutable.Get([]byte(k))
		if !bytes.Equal(value, []byte(v)) {
			t.Errorf("Wrong value. expected [%x] but [%x]", v, value)
		}
	}

}

func TestDeleteSnapshot(t *testing.T) {
	// delete, snapshot, write
	manager := New(db.NewMapDB())
	tree := manager.NewMutable(nil)

	updateString(tree, "doe", "reindeer")
	updateString(tree, "dog", "puppy")
	snapshot := tree.GetSnapshot() // SNAPSHOT - doe, dog
	solution2 := fmt.Sprintf("%x", snapshot.Hash())
	updateString(tree, "dogglesworth", "cat")

	snapshot = tree.GetSnapshot() // SNAPSHOT - doe, dog, dogglesworth
	hashHex := "8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3"
	if !trie.ConfigUseKeccak256 {
		hashHex = "c33dc4124235214a96518fc8bfdef30a6c1462a08e423c29f7b08513829c551f"
	}
	strRoot := fmt.Sprintf("%x", snapshot.Hash())
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}

	snapshot.Flush()
	trie2 := manager.NewMutable(nil)
	trie2.Reset(snapshot) // have doe, dog, dogglesworth
	snapshot2 := trie2.GetSnapshot()
	strRoot = fmt.Sprintf("%x", snapshot2.Hash())
	if strings.Compare(strRoot, hashHex) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}

	deleteString(trie2, "dogglesworth")

	snapshot2 = trie2.GetSnapshot()               // SNAPSHOT = doe, dog
	strRoot = fmt.Sprintf("%x", snapshot2.Hash()) // have doe, dog
	if strings.Compare(strRoot, solution2) != 0 {
		t.Errorf("exp %s got %s", solution2, strRoot)
	}

	// Get snapshot after delete dogglesworth
	snapshot = trie2.GetSnapshot()
	snapshot.Flush()

	hashAfterDelete := fmt.Sprintf("%x", snapshot.Hash())
	trie2.Reset(snapshot)
	snapshot = trie2.GetSnapshot()
	strRoot = fmt.Sprintf("%x", snapshot.Hash())
	if strings.Compare(strRoot, hashAfterDelete) != 0 {
		t.Errorf("exp %s got %s", hashHex, strRoot)
	}
	if strings.Compare(solution2, hashAfterDelete) != 0 {
		t.Errorf("exp %s got %s", solution2, strRoot)
	}
}

func TestLateFlush(t *testing.T) {
	manager := New(db.NewMapDB())
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
		hashList[i] = fmt.Sprintf("%x", ssList[i].Hash())
	}

	for i, _ := range poolList {
		ssList[i].Flush()
		rootHash := fmt.Sprintf("%x", ssList[i].Hash())
		if strings.Compare(hashList[i], rootHash) != 0 {
			t.Errorf("exp %s got %s", hashList[i], rootHash)
		}
	}
}

func TestNoHashed(t *testing.T) {
	manager := New(db.NewMapDB())
	tr := manager.NewMutable(nil)

	unchanged := byte(0xFD)
	tr.Set([]byte{0x00}, []byte{0xFF})
	tr.Set([]byte{0x00, 0x01}, []byte{0xFE})
	tr.Set([]byte{0x00, 0x01, 0x00}, []byte{unchanged})

	immutable := tr.GetSnapshot()
	immutable.Hash()
	immutable.Flush()
	v, _ := immutable.Get([]byte{0x00, 0x01, 0x00})
	if v[0] != unchanged {
		t.Errorf("%d : %d", v[0], unchanged)
	}
	changed := byte(0xFA)
	tr.Set([]byte{0x00, 0x01, 0x00}, []byte{changed})
	immutable = tr.GetSnapshot()
	immutable.Hash()
	immutable.Flush()
	v, _ = immutable.Get([]byte{0x00, 0x01, 0x00})

	if v[0] != changed {
		t.Errorf("%d : %d", v[0], changed)
	}
}

func TestNull(t *testing.T) {
	manager := New(db.NewMapDB())
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
	manager := New(db.NewMapDB())
	tr := manager.NewMutable(nil)

	key := []byte("test")
	value := []byte("test")
	tr.Set(key, value)
	v, _ := tr.Get(key)
	if !bytes.Equal(v, value) {
		t.Fatal("wrong value")
	}
	s := tr.GetSnapshot()
	fmt.Println(s.GetProof(key))
}

func TestMissingNode(t *testing.T) {
	manager := New(db.NewMapDB())
	trie := manager.NewMutable(nil)

	testMap := map[string][]byte{
		"120000": []byte("qwerqwerqwerqwerqwerqwerqwerqwer"),
		"123456": []byte("asdfasdfasdfasdfasdfasdfasdfasdf"),
	}

	updateString(trie, "120000", "qwerqwerqwerqwerqwerqwerqwerqwer")
	updateString(trie, "123456", "asdfasdfasdfasdfasdfasdfasdfasdf")
	snapshot := trie.GetSnapshot()
	snapshot.Flush()
	root := snapshot.Hash()

	trie = manager.NewMutable(root)
	v, _ := trie.Get([]byte("120000"))
	if bytes.Equal(v, testMap["120000"]) == false {
		t.Errorf("Wrong value. v = %x", v)
	}

	trie = manager.NewMutable(root)
	v, _ = trie.Get([]byte("120099"))
	if bytes.Equal(v, testMap["120099"]) == false {
		t.Errorf("Wrong value. v = %x", v)
	}

	trie = manager.NewMutable(root)
	v, _ = trie.Get([]byte("123456"))
	if bytes.Equal(v, testMap["123456"]) == false {
		t.Errorf("Wrong value. v = %x", v)
	}

	trie = manager.NewMutable(root)
	_, err := trie.Set([]byte("120099"), []byte("zxcvzxcvzxcvzxcvzxcvzxcvzxcvzxcv"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	trie = manager.NewMutable(root)
	_, err = trie.Delete([]byte("123456"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	snapshot = trie.GetSnapshot()
	rootHash := snapshot.Hash()
	fmt.Printf("%x\n", rootHash)

	// hash := common.HexToHash("0xe1d943cc8f061a0c0b98162830b970395ac9315654824bf21b73b891365262f9")

	// if memonly {
	// 	delete(triedb.nodes, hash)
	// } else {
	// 	diskdb.Delete(hash[:])
	// }

	/*
		trie, _ = New(root, triedb)
		_, err = trie.TryGet([]byte("120000"))
		if _, ok := err.(*MissingNodeError); !ok {
			t.Errorf("Wrong error: %v", err)
		}
		trie, _ = New(root, triedb)
		_, err = trie.TryGet([]byte("120099"))
		if _, ok := err.(*MissingNodeError); !ok {
			t.Errorf("Wrong error: %v", err)
		}
		trie, _ = New(root, triedb)
		_, err = trie.TryGet([]byte("123456"))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		trie, _ = New(root, triedb)
		err = trie.TryUpdate([]byte("120099"), []byte("zxcv"))
		if _, ok := err.(*MissingNodeError); !ok {
			t.Errorf("Wrong error: %v", err)
		}
		trie, _ = New(root, triedb)
		err = trie.TryDelete([]byte("123456"))
		if _, ok := err.(*MissingNodeError); !ok {
			t.Errorf("Wrong error: %v", err)
		}
	*/
}

func updateString(trie trie.Mutable, k, v string) {
	trie.Set([]byte(k), []byte(v))
}

func deleteString(trie trie.Mutable, k string) {
	trie.Delete([]byte(k))
}

type testEntry struct {
	k, v []byte
}

type testSetter interface {
	Set([]byte, []byte) ([]byte, error)
	Delete([]byte) ([]byte, error)
}

type testGetter interface {
	Get([]byte) ([]byte, error)
}

func applyTestEntries(m testSetter, entries []testEntry, t *testing.T) bool {
	ret := true
	for _, e := range entries {
		var err error
		if e.v != nil {
			_, err = m.Set(e.k, e.v)
		} else {
			_, err = m.Delete(e.k)
		}
		if err != nil {
			ret = false
			t.Errorf("Fail to Set(%x,%x)", e.k, e.v)
		}
	}
	return ret
}

func checkTestEntries(m testGetter, entries []testEntry, t *testing.T) bool {
	ret := true
	for _, e := range entries {
		v, err := m.Get(e.k)
		if err != nil {
			ret = false
			t.Errorf("Fail to Get(%x)", e.k)
		}
		if !bytes.Equal(v, e.v) {
			ret = false
			t.Errorf("Invalid data from Get(%x) exp=(%x) ret=(%x)", e.k, e.v, v)
		}
	}
	return ret
}

func Test_NewMutable(t *testing.T) {
	type args struct {
		h []byte
		e []testEntry
	}
	type result struct {
		e []testEntry
		h []byte
	}
	tests := []struct {
		name string
		args args
		want result
	}{
		{
			name: "Small1",
			args: args{
				nil,
				[]testEntry{
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
				},
			},
			want: result{
				[]testEntry{
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
				},
				[]byte{},
			},
		},
		{
			name: "AddRemove1",
			args: args{
				nil,
				[]testEntry{
					{[]byte{1, 2, 3}, []byte{1}},
					{[]byte{1, 2, 3}, []byte{2}},
					{[]byte{1, 2, 3}, []byte{0x11, 0x22, 0x33}},
					{[]byte{1, 2, 4}, []byte{0x11, 0x22, 0x44}},
					{[]byte{1, 2, 3, 4}, []byte{0x11, 0x22, 0x33, 0x44}},
					{[]byte{1, 2, 4}, nil},
					{[]byte{1, 2, 3, 4}, nil},
				},
			},
			want: result{
				[]testEntry{
					{[]byte{1, 2, 3}, []byte{0x11, 0x22, 0x33}},
					{[]byte{1, 2, 3, 4}, nil},
					{[]byte{1, 2, 4}, nil},
				},
				[]byte{},
			},
		},
		{
			name: "AddRemove2",
			args: args{
				nil,
				[]testEntry{
					{[]byte{0x01}, []byte{0x01}},
					{[]byte{0x01, 0x22}, []byte{0x01, 0x22}},
					{[]byte{0x01, 0x23}, []byte{0x01, 0x23}},
					{[]byte{0x01, 0x23, 0x44}, []byte{0x01, 0x23, 0x44}},
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
					{[]byte{0x01, 0x23}, nil},
					{[]byte{0x01}, nil},
					{[]byte{0x01, 0x23}, nil},
					{[]byte{0x01, 0x23, 0x44}, nil},
				},
			},
			want: result{
				[]testEntry{
					{[]byte{0x01}, nil},
					{[]byte{0x01, 0x22}, []byte{0x01, 0x22}},
					{[]byte{0x01, 0x23}, nil},
					{[]byte{0x01, 0x23, 0x44}, nil},
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
					{[]byte{0x01, 0x23, 0x46}, nil},
				},
				[]byte{},
			},
		},
		{
			name: "AddRemove3",
			args: args{
				nil,
				[]testEntry{
					{[]byte{0x01}, []byte{0x01}},
					{[]byte{0x01, 0x22}, []byte{0x01, 0x22}},
					{[]byte{0x01, 0x23}, []byte{0x01, 0x23}},
					{[]byte{0x01, 0x23, 0x44}, []byte{0x01, 0x23, 0x44}},
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
				},
			},
			want: result{
				[]testEntry{
					{[]byte{0x01}, []byte{0x01}},
					{[]byte{0x01, 0x22}, []byte{0x01, 0x22}},
					{[]byte{0x01, 0x23}, []byte{0x01, 0x23}},
					{[]byte{0x01, 0x23, 0x44}, []byte{0x01, 0x23, 0x44}},
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
				},
				[]byte{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := New(db.NewMapDB())

			log.Print("Makes new MPT with Manager")
			got := mgr.NewMutable(tt.args.h)
			if got == nil {
				t.Errorf("NewMutable() = %v, want non nil", got)
				return
			}
			applyTestEntries(got, tt.args.e, t)
			s := got.GetSnapshot()
			hash := s.Hash()
			log.Printf("Snapshot Hash:%x", hash)
			log.Println("Flush")
			s.Flush()

			s2 := mgr.NewImmutable(hash)
			if !checkTestEntries(s2, tt.want.e, t) {
				log.Printf("FAIL verification")
			} else {
				log.Printf("OKAY verification")
			}
		})
	}
}

func Test_Snapshot(t *testing.T) {
	type snapshot struct {
		tx []testEntry
		r  []testEntry
	}
	tests := []struct {
		name      string
		snapshots []snapshot
	}{
		{"Scenario1", []snapshot{
			{
				[]testEntry{
					{[]byte{0x01, 0x23, 0x45, 0x67}, []byte{0x01, 0x23, 0x45, 0x67}},
					{[]byte{0x01, 0x23, 0x54, 0x68}, []byte{0x01, 0x23, 0x54, 0x68}},
				},
				[]testEntry{
					{[]byte{0x01, 0x23, 0x45, 0x67}, []byte{0x01, 0x23, 0x45, 0x67}},
					{[]byte{0x01, 0x23, 0x54, 0x68}, []byte{0x01, 0x23, 0x54, 0x68}},
				},
			},
			{
				[]testEntry{
					{[]byte{0x01, 0x23, 0x45, 0x67}, nil},
					{[]byte{0x01, 0x23, 0x44}, []byte{0x01, 0x23, 0x44}},
					{[]byte{0x01, 0x23, 0x44, 0x55}, []byte{0x01, 0x23, 0x44, 0x55}},
				},
				[]testEntry{
					{[]byte{0x01, 0x23, 0x45, 0x67}, nil},
					{[]byte{0x01, 0x23, 0x54, 0x68}, []byte{0x01, 0x23, 0x54, 0x68}},
					{[]byte{0x01, 0x23, 0x44}, []byte{0x01, 0x23, 0x44}},
					{[]byte{0x01, 0x23, 0x44, 0x55}, []byte{0x01, 0x23, 0x44, 0x55}},
				},
			},
			{
				[]testEntry{
					{[]byte{0x01, 0x23, 0x44, 0x67}, nil},
					{[]byte{0x01, 0x23, 0x44}, nil},
					{[]byte{0x01}, []byte{0x01}},
				},
				[]testEntry{
					{[]byte{0x01, 0x23, 0x45, 0x67}, nil},
					{[]byte{0x01, 0x23, 0x54, 0x68}, []byte{0x01, 0x23, 0x54, 0x68}},
					{[]byte{0x01, 0x23, 0x44, 0x55}, []byte{0x01, 0x23, 0x44, 0x55}},
					{[]byte{0x01, 0x23, 0x44}, nil},
					{[]byte{0x01}, []byte{0x01}},
				},
			},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := New(db.NewMapDB())
			m := mgr.NewMutable(nil)

			ss := make([]trie.Snapshot, len(tt.snapshots))
			for sidx, s := range tt.snapshots {
				log.Printf("Apply Snapshot(%d) and check", sidx)
				t.Run(fmt.Sprintf("Apply_Snapshot(%d)", sidx), func(t *testing.T) {
					applyTestEntries(m, s.tx, t)
				})
				ss[sidx] = m.GetSnapshot()

				func(sidx int) {
					log.Printf("Snapshot(~%d) Verify START", sidx)
					for i := 0; i <= sidx; i++ {
						s := tt.snapshots[i]
						sx := ss[i]
						t.Run(fmt.Sprintf("Check_Snapshot(%d/%d)", i, sidx), func(t *testing.T) {
							checkTestEntries(sx, s.r, t)
						})
					}
					log.Printf("Snapshot(~%d) Verify DONE", sidx)
				}(sidx)
			}
			log.Println("Verifying Snapshot from Hashes after Flush in reverse")
			log.Printf("Verify Snapshots")
			for sidx := len(tt.snapshots) - 1; sidx >= 0; sidx-- {
				log.Printf("Snapshot(%d) Verify", sidx)
				ss[sidx].Flush()
				h := ss[sidx].Hash()
				sx := mgr.NewImmutable(h)
				s := tt.snapshots[sidx]
				t.Run(fmt.Sprintf("Verify_Snapshot(%d/%x)", sidx, h), func(t *testing.T) {
					checkTestEntries(sx, s.r, t)
				})

				if sidx < len(tt.snapshots)-1 {
					sidx := sidx + 1
					h := ss[sidx-1].Hash()
					s := tt.snapshots[sidx]
					log.Printf("Snapshot(%d) Verify from Snapshot(%x)", sidx, h)
					mutable := mgr.NewMutable(h)
					t.Run(fmt.Sprintf("Apply Snapshot(%d) from Snapshot(%x)", sidx, h), func(t *testing.T) {
						applyTestEntries(mutable, s.tx, t)
					})
					sx := mutable.GetSnapshot()
					t.Run(fmt.Sprintf("Verify Snapshot(%d) from Snapshot(%x)", sidx, h), func(t *testing.T) {
						checkTestEntries(sx, s.r, t)
					})
				}
			}
		})
	}
}

type Obj struct {
	value []byte
}

func (o *Obj) Bytes() []byte {
	return o.value
}

func (o *Obj) Equal(trie.Object) bool {
	return false
}

func (o *Obj) Flush() error {
	return nil
}

func (o *Obj) Reset(s db.Database, k []byte) error {
	return nil
}

func (o *Obj) Resolve(bd merkle.Builder) error {
	return nil
}

func (o *Obj) ClearCache() {
	// do nothing
}

func TestObject(t *testing.T) {
	manager := New(db.NewMapDB())
	mutable := manager.NewMutable(nil)
	mutableObj := manager.NewMutableForObject(nil, reflect.TypeOf(Obj{}))
	mutableSnaps := make([]trie.Snapshot, 3)
	mutableObjSnaps := make([]trie.SnapshotForObject, 3)
	i := 0
	for k, v := range testPool {
		mutable.Set([]byte(k), []byte(v))
		mutableSnaps[i] = mutable.GetSnapshot()
		mutableObj.Set([]byte(k), &Obj{value: []byte(v)})
		mutableObjSnaps[i] = mutableObj.GetSnapshot()
		i++
	}

	for i, v := range mutableSnaps {
		hash1 := v.Hash()
		hash2 := mutableObjSnaps[i].Hash()
		if !bytes.Equal(hash1, hash2) {
			t.Errorf("expected %x but got %x", hash1, hash2)
		}
	}
}

type testObject struct {
	s          string
	flushCount int
}

func (e *testObject) Bytes() []byte {
	return []byte(e.s)
}
func (e *testObject) Reset(d db.Database, b []byte) error {
	e.s = string(b)
	return nil
}
func (e *testObject) Flush() error {
	e.flushCount++
	return nil
}
func (e *testObject) Equal(o trie.Object) bool {
	e2, ok := o.(*testObject)
	return ok && e.s == e2.s
}

func (e *testObject) Resolve(bd merkle.Builder) error {
	return nil
}

func (e *testObject) ClearCache() {
	// do nothing
}

func TestObjectFlush(t *testing.T) {
	tests := [][]string{
		[]string{"test", "hello", "puha"},
		[]string{"apple", "pear", "strawberry"},
		[]string{"black", "blue", "red"},
	}

	db := db.NewMapDB()
	mgr := New(db)
	m1 := mgr.NewMutableForObject(nil, reflect.TypeOf((*testObject)(nil)))

	objs := []*testObject{}
	snapshots := make([]trie.SnapshotForObject, len(tests))
	for i, tt := range tests {
		for _, s := range tt {
			to := &testObject{s, 0}
			m1.Set([]byte(s), to)
			objs = append(objs, to)
		}
		snapshots[i] = m1.GetSnapshot()
	}

	for _, to := range objs {
		if to.flushCount != 0 {
			t.Errorf("Flush count is not zero, s='%s' count=%d", to.s, to.flushCount)
		}
	}

	for _, s := range snapshots {
		s.Flush()
	}

	for _, to := range objs {
		if to.flushCount == 0 {
			t.Errorf("Flush count is zero, s='%s' count=%d", to.s, to.flushCount)
		}
	}

	for i, tt := range tests {
		m2 := mgr.NewImmutableForObject(snapshots[i].Hash(), reflect.TypeOf((*testObject)(nil)))
		for _, s := range tt {
			o, err := m2.Get([]byte(s))
			if err != nil {
				t.Errorf("Fail to get '%s'", s)
			}
			if o == nil {
				t.Errorf("Fail to get proper object for '%s'", s)
				continue
			}
			to, ok := o.(*testObject)
			if !ok {
				t.Errorf("Type of object is different type = %T", o)
				continue
			}
			if to.s != s {
				t.Errorf("Returned object is invalid exp = '%s', ret = '%s'", s, to.s)
				continue
			}
		}
	}
}

func TestObjectIterate(t *testing.T) {
	tests := [][]string{
		[]string{"test", "hello", "puha"},
		[]string{"apple", "pear", "strawberry"},
		[]string{"black", "blue", "red"},
	}

	db := db.NewMapDB()
	mgr := New(db)
	m1 := mgr.NewMutableForObject(nil, reflect.TypeOf((*testObject)(nil)))

	snapshots := make([]trie.SnapshotForObject, len(tests))
	for i, tt := range tests {
		for _, s := range tt {
			to := &testObject{s, 0}
			m1.Set([]byte(s), to)
		}
		snapshots[i] = m1.GetSnapshot()
	}
	for _, s := range snapshots {
		s.Flush()
	}

	visited := map[string]bool{}

	for i, tt := range tests {
		m2 := mgr.NewImmutableForObject(snapshots[i].Hash(), reflect.TypeOf((*testObject)(nil)))

		for _, s := range tt {
			visited[s] = false
		}

		for itr := m2.Iterator(); itr.Has(); itr.Next() {
			o, k, err := itr.Get()
			if err != nil {
				t.Errorf("Fail to get item")
				continue
			}
			to, ok := o.(*testObject)
			if !ok {
				t.Errorf("Invalid object is retreived type=%T", o)
				continue
			}
			if to.s != string(k) {
				t.Errorf("Returned object(%s) is different from (%s)", to.s, string(k))
				continue
			}
			if yn, ok := visited[to.s]; ok {
				if yn {
					t.Errorf("Visit multiple for %s", to.s)
				} else {
					visited[to.s] = true
				}
			} else {
				t.Errorf("Should not exist %s", to.s)
			}
		}

		for s, yn := range visited {
			if !yn {
				t.Errorf("Missing element %s", s)
			}
		}
		for s, _ := range visited {
			visited[s] = false
		}
	}
}

func TestImmutableEqual(t *testing.T) {
	m := New(db.NewMapDB())
	mutable := m.NewMutable(nil)
	mutable.Set([]byte("test"), []byte("value"))
	s1 := mutable.GetSnapshot()
	s2 := mutable.GetSnapshot()
	if !s1.Equal(s1, false) {
		t.Errorf("Not same with same snapshot")
	}
	if !s1.Equal(s2, false) {
		t.Errorf("Not same with another snapshot after no change")
	}
}

type OldValue int

const (
	NoValue OldValue = -1
)

func OldValueOf(value []byte) OldValue {
	if value == nil {
		return NoValue
	} else {
		return OldValue(len(value))
	}
}

func TestOldValue_MutableSetDelete(t *testing.T) {
	type add struct {
		key   string
		value string
		old   OldValue
	}
	type remove struct {
		key string
		old OldValue
	}
	args := []struct {
		name    string
		adds    []add
		removes []remove
	}{
		{
			"Case1",
			[]add{
				{"1", "v1", NoValue},
				{"2", "v2", NoValue},
				{"3", "", NoValue},
				{"test1", "test1v1", NoValue},
				{"test1", "test1v2", 7},
				{"test2", "test2v1", NoValue},
				{"test2", "test2v2", 7},
				{"test", "value", NoValue},
				{"test", "value1", 5},
			},
			[]remove{
				{"test2", 7},
				{"test1", 7},
				{"test", 6},
				{"3", 0},
				{"4", NoValue},
				{"1", 2},
				{"2", 2},
			},
		},
		{
			"Case2",
			[]add{
				{"1", "v1", NoValue},
				{"1", "v22", 2},
				{"2", "v1", NoValue},
				{"2", "v22", 2},
				{"", "root1", NoValue},
				{"", "root2", 5},
			},
			[]remove{
				{"", 5},
				{"1", 3},
				{"2", 3},
			},
		},
	}

	for _, arg := range args {
		t.Run(arg.name, func(t *testing.T) {
			m := New(db.NewMapDB())
			mutable := m.NewMutable(nil)
			for _, item := range arg.adds {
				ov, err := mutable.Set([]byte(item.key), []byte(item.value))
				assert.NoError(t, err)
				assert.Equal(t, item.old, OldValueOf(ov), item)

				ov, err = mutable.Set([]byte(item.key), []byte(item.value))
				assert.NoError(t, err)
				assert.Equal(t, OldValueOf([]byte(item.value)), OldValueOf(ov), item)
			}

			for _, item := range arg.removes {
				ov, err := mutable.Delete([]byte(item.key))
				assert.NoError(t, err)
				assert.Equal(t, item.old, OldValueOf(ov))
			}

			mutable = m.NewMutable(nil)
			for _, item := range arg.adds {
				ov, err := mutable.Set([]byte(item.key), []byte(item.value))
				assert.NoError(t, err)
				assert.Equal(t, item.old, OldValueOf(ov))
			}

			ss := mutable.GetSnapshot()
			for _, item := range arg.removes {
				ov, err := mutable.Delete([]byte(item.key))
				assert.NoError(t, err)
				assert.Equal(t, item.old, OldValueOf(ov))
			}

			mutable.Reset(ss)
			for _, item := range arg.removes {
				ov, err := mutable.Delete([]byte(item.key))
				assert.NoError(t, err)
				assert.Equal(t, item.old, OldValueOf(ov))
			}
		})
	}
}
