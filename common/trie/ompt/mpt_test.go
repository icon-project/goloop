package ompt

import (
	"bytes"
	"encoding/hex"
	"log"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/merkle"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

func TestNewMPT(t *testing.T) {
	type entry struct {
		k, v []byte
	}
	type args struct {
		d db.Database
		h []byte
		t reflect.Type
		e []entry
	}
	type result struct {
		e []entry
		h []byte
	}
	tests := []struct {
		name string
		args args
		want result
	}{
		{
			name: "AddRemove1",
			args: args{
				db.NewMapDB(),
				nil,
				reflect.TypeOf(bytesObject(nil)),
				[]entry{
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
				[]entry{
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
				db.NewMapDB(),
				nil,
				reflect.TypeOf(bytesObject(nil)),
				[]entry{
					{[]byte{0x01}, []byte{0x01}},
					{[]byte{0x01, 0x22}, []byte{0x01, 0x22}},
					{[]byte{0x01, 0x33}, []byte{0x01, 0x33}},
					{[]byte{0x01, 0x23, 0x44}, []byte{0x01, 0x23, 0x44}},
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
					{[]byte{0x01}, nil},
					{[]byte{0x01}, nil},
					{[]byte{0x01, 0x33}, nil},
					{[]byte{0x01, 0x23, 0x44}, nil},
				},
			},
			want: result{
				[]entry{
					{[]byte{0x01, 0x22}, []byte{0x01, 0x22}},
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
					{[]byte{0x01}, nil},
					{[]byte{0x01, 0x33}, nil},
					{[]byte{0x01, 0x23, 0x44}, nil},
					{[]byte{0x01, 0x23, 0x46}, nil},
				},
				[]byte{},
			},
		},
		{
			name: "AddRemove3",
			args: args{
				db.NewMapDB(),
				nil,
				reflect.TypeOf(bytesObject(nil)),
				[]entry{
					{[]byte{0x01}, []byte{0x01}},
					{[]byte{0x01, 0x22}, []byte{0x01, 0x22}},
					{[]byte{0x01, 0x33}, []byte{0x01, 0x33}},
					{[]byte{0x01}, []byte{0x03}},
					{[]byte{0x01, 0x23, 0x44}, []byte{0x01, 0x23, 0x44}},
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
				},
			},
			want: result{
				[]entry{
					{[]byte{0x01}, []byte{0x03}},
					{[]byte{0x01, 0x22}, []byte{0x01, 0x22}},
					{[]byte{0x01, 0x33}, []byte{0x01, 0x33}},
					{[]byte{0x01, 0x23, 0x44}, []byte{0x01, 0x23, 0x44}},
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
				},
				[]byte{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Println("Makes new MPT")
			got := NewMPT(tt.args.d, tt.args.h, tt.args.t)
			if got == nil {
				t.Errorf("NewMPT() = %v, want non nil", got)
				return
			}
			for _, e := range tt.args.e {
				var err error
				if e.v != nil {
					log.Printf("Set(%x,%x)", e.k, e.v)
					_, err = got.Set(e.k, bytesObject(e.v))
				} else {
					log.Printf("Delete(%x)", e.k)
					_, err = got.Delete(e.k)
				}
				if err != nil {
					t.Errorf("FAIL to set key to value")
					return
				}
			}
			s := got.GetSnapshot()
			h := s.Hash()
			// if !bytes.Equal(h, tt.want.h) {
			// 	s.(*mpt).Dump()
			// 	t.Errorf("Hash() = %#x, want %#x", h, tt.want.h)
			// }
			log.Println("Flush")
			s.Flush()

			s2 := NewMPT(tt.args.d, h, tt.args.t)
			log.Printf("Dump current snapshot from hash")
			s2.Dump()
			log.Printf("Verify results")
			for _, e := range tt.want.e {
				obj, err := s2.Get(e.k)
				if err != nil {
					t.Errorf("Key(%s) return error=%v",
						hex.EncodeToString(e.k), err)
					continue
				}
				if obj == nil {
					if e.v == nil {
						continue
					} else {
						t.Errorf("Key(%x) expected %x result is nil", e.k, e.v)
						continue
					}
				}
				if !bytes.Equal(obj.Bytes(), e.v) {
					t.Errorf("Key(%x) expected %x result %x", e.k, e.v, obj.Bytes())
					s2.Dump()
					break
				}
			}
		})
	}
}

func Test_GetPoof(t *testing.T) {
	type entry struct {
		k, v []byte
	}
	type args struct {
		e []entry
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Case1",
			args: args{
				[]entry{
					{[]byte{0x01}, []byte{0x01}},
					{[]byte{0x01, 0x22}, []byte{0x01, 0x22}},
					{[]byte{0x01, 0x23}, []byte{0x01, 0x23}},
					{[]byte{0x01, 0x23, 0x66}, nil},
					{[]byte{0x01, 0x23, 0x44}, []byte{0x01, 0x23, 0x44}},
					{[]byte{0x01, 0x23, 0x45}, []byte{0x01, 0x23, 0x45}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d1 := db.NewMapDB()
			m1 := NewMPTForBytes(d1, nil)
			for _, e := range tt.args.e {
				if e.v != nil {
					m1.Set(e.k, e.v)
				}
			}
			s1 := m1.GetSnapshot()
			h := s1.Hash()
			log.Println("Flush snapshot 1")
			s1.Flush()
			s1r := NewMPTForBytes(d1, h)

			d2 := db.NewMapDB()
			s2 := NewMPTForBytes(d2, h)
			for _, e := range tt.args.e {
				log.Printf("Take Proof for [%x]", e.k)
				proof1 := s1r.GetProof(e.k)
				proof2 := s1.GetProof(e.k)
				if proof1 == nil {
					if e.v != nil {
						t.Errorf("Get proof for [%x] returns nil", e.k)
					}
					continue
				}
				if !reflect.DeepEqual(proof1, proof2) {
					t.Errorf("Proofs from snapshot and snapshot from hash are different")
				}
				log.Printf("Prove for [%x] proof=%v", e.k, proof1)
				obj, err := s2.Prove(e.k, proof1)
				if err != nil {
					t.Errorf("Fail to prove key [%x] err=%v", e.k, err)
				} else {
					log.Printf("Proved value [%x] expected [%x]", obj, e.v)
					if !bytes.Equal(obj, e.v) {
						t.Errorf("Fail to prove key [%x] exptected=[%x] returned=[%x]", e.k, e.v, obj)
					}
					s2.Flush()
				}
			}

			log.Println("Flush snapshot 2")
			s2.Flush()
		})
	}
}

// func TestIterateInOrder(t *testing.T) {
// 	mp := new(codec.MsgpackHandle)
// 	mp.Canonical = true
// 	mp.StructToArray = true
//
// 	db1 := db.NewMapDB()
//
// 	m := NewMPTForBytes(db1, nil)
// 	for i := int(0); i < 5000; i++ {
// 		var b []byte
// 		e := codec.NewEncoderBytes(&b, mp)
// 		e.Encode(i)
// 		m.Set(b, b)
// 	}
// 	m.Flush()
//
// 	m2 := NewMPTForBytes(db1, m.Hash())
//
// 	var idx int = 0
// 	for itr := m2.Iterator(); itr.Has(); itr.Next() {
// 		k, v, err := itr.Get()
// 		if err != nil {
// 			t.Errorf("it fails to get value from iterator")
// 			break
// 		}
// 		// log.printf("iter[%x] key[%x] value %v\n", idx, k, v)
//
// 		var k2, v2 int
// 		d := codec.NewDecoderBytes([]byte(k), mp)
// 		d.Decode(&k2)
// 		d = codec.NewDecoderBytes(v.Bytes(), mp)
// 		d.Decode(&v2)
//
// 		if k2 != v2 {
// 			t.Errorf("key(%d) and value(%d) are different", k2, v2)
// 		}
// 		if k2 != idx {
// 			t.Errorf("expected (%d) but key is (%d)", idx, k2)
// 		}
// 		idx++
// 	}
// }

func TestNullHash(t *testing.T) {
	m := NewMPTForBytes(db.NewMapDB(), nil)
	if m.Hash() != nil {
		t.Errorf("NewMPTForBytes(nil).Hash() should return nil")
	}

	m2 := NewMPT(db.NewMapDB(), nil, reflect.TypeOf(bytesObject(nil)))
	if m2.Hash() != nil {
		t.Errorf("NewMPT(nil).Hash() should return nil")
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
func (e *testObject) Resolve(builder merkle.Builder) error {
	return nil
}
func (e *testObject) ClearCache() {
	// do nothing
}

func TestObjectTest(t *testing.T) {
	tests := [][]string{
		{"test", "hello", "puha"},
		{"apple", "pear", "strawberry"},
		{"black", "blue", "red"},
	}

	db := db.NewMapDB()
	mgr := NewManager(db)
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
		m2 := NewMPT(db, snapshots[i].Hash(), reflect.TypeOf((*testObject)(nil)))
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
		[]string{"black", "blue", "red", "re", "reb"},
	}

	db := db.NewMapDB()
	mgr := NewManager(db)
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

func Test_mpt_Filter(t *testing.T) {
	tests := []struct {
		name   string
		data   []string
		prefix []byte
		want   []string
	}{
		{"C1", []string{"a", "b", "c"},
			nil, []string{"a", "b", "c"}},
		{"C2", []string{"a", "b", "c"},
			[]byte("b"), []string{"b"}},
		{"C3", []string{"a", "b", "bc", "bae", "bcf"},
			[]byte("bc"), []string{"bc", "bcf"}},
		{"C4", []string{"abc", "b", "bca", "bae", "bcf"},
			[]byte("bc"), []string{"bca", "bcf"}},
		{"C5", []string{"abc", "b", "bcdefg", "bae", "bcdefh"},
			[]byte("bc"), []string{"bcdefg", "bcdefh"}},
		{"C6",
			[]string{
				"\x12\x34",
				"\x23\x45\x67",
				"\x21\x34",
				"\x23\x45\x68",
			},
			[]byte{0x23},
			[]string{
				"\x23\x45\x67",
				"\x23\x45\x68",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbase := db.NewMapDB()
			m := NewMPTForBytes(dbase, nil)
			for _, s := range tt.data {
				_, err := m.Set([]byte(s), []byte(s))
				assert.NoError(t, err)
			}

			idx := 0
			for itr := m.Filter(tt.prefix); itr.Has(); itr.Next() {
				key, value, err := itr.Get()
				assert.NoError(t, err)
				assert.True(t, bytes.Equal(key, value))
				assert.Equal(t, tt.want[idx], string(key))
				idx += 1
			}
			assert.Equal(t, len(tt.want), idx)
		})
	}
}
