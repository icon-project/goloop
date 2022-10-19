package ompt

import (
	"bytes"
	"log"
	"reflect"
	"testing"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
)

type TestObject struct {
	dirty  bool
	bucket db.Bucket
	hash   []byte
	data   []byte
}

func (o *TestObject) Bytes() []byte {
	return o.Hash()
}

func (o *TestObject) Reset(s db.Database, k []byte) error {
	if bucket, err := s.GetBucket(db.BytesByHash); err != nil {
		return err
	} else {
		o.bucket = bucket
	}
	o.hash = k
	return nil
}

func (o *TestObject) Data() ([]byte, error) {
	if o.data == nil {
		if v, err := o.bucket.Get(o.hash); err == nil {
			o.data = v
		} else {
			return nil, err
		}
	}
	return o.data, nil
}

func (o *TestObject) Flush() error {
	if o.dirty {
		if err := o.bucket.Set(o.Hash(), o.data); err != nil {
			return err
		}
		o.dirty = false
	}
	return nil
}

func (o *TestObject) Hash() []byte {
	if o.hash == nil {
		if o.data == nil {
			o.hash = []byte{}
		} else {
			o.hash = crypto.SHA3Sum256(o.data)
		}
	}
	return o.hash
}

func (o *TestObject) Equal(obj trie.Object) bool {
	o2, ok := obj.(*TestObject)
	if !ok {
		return false
	}
	return bytes.Equal(o.Hash(), o2.Hash())
}

func (o *TestObject) Resolve(builder merkle.Builder) error {
	builder.RequestData(db.BytesByHash, o.hash, o)
	return nil
}

func (o *TestObject) OnData(bs []byte, builder merkle.Builder) error {
	o.data = bs
	return nil
}

func (o *TestObject) ClearCache() {
	// do nothing
}

func NewTestObject(dbase db.Database, bs []byte) *TestObject {
	bucket, err := dbase.GetBucket(db.BytesByHash)
	if err != nil {
		log.Panicf("Fail to get Bucket")
	}
	return &TestObject{
		bucket: bucket,
		dirty:  true,
		data:   bs,
	}
}

func TestMerkleBuild(t *testing.T) {
	type entry struct {
		k, v []byte
	}
	entries := []entry{
		{[]byte{0x00, 0x11, 0x22}, []byte("hello")},
		{[]byte{0x00, 0x12, 0x22}, []byte("foo")},
		{[]byte{0x00, 0x13, 0x22}, []byte("bar")},
		{[]byte{0x00, 0x13, 0x24}, []byte("test")},
		{[]byte{0x00, 0x13, 0x25}, []byte("test")},
	}

	dbase := db.NewMapDB()
	m1 := NewMutableForObject(dbase, nil, reflect.TypeOf((*TestObject)(nil)))
	for _, e := range entries {
		if _, err := m1.Set(e.k, NewTestObject(dbase, e.v)); err != nil {
			t.Errorf("Fail to Set(%x,'%s')", e.k, e.v)
			return
		}
	}
	ss := m1.GetSnapshot()
	ss.Flush()

	log.Printf("Built trie hash=<%x>", ss.Hash())

	dbase2 := db.NewMapDB()
	builder := merkle.NewBuilder(dbase2)
	ss2 := NewImmutableForObject(builder.Database(), ss.Hash(), reflect.TypeOf((*TestObject)(nil)))
	ss2.Resolve(builder)

	bkTrie, err := dbase.GetBucket(db.MerkleTrie)
	if err != nil {
		t.Errorf("Fail to get bucket for MerkleTrie err=%+v", err)
		return
	}
	bkBytes, err := dbase.GetBucket(db.BytesByHash)
	if err != nil {
		t.Errorf("Fail to get bucket for BytesForHash err=%+v", err)
		return
	}

	for builder.UnresolvedCount() > 0 {
		log.Printf("UnresolvedCount() = %d", builder.UnresolvedCount())
		req := builder.Requests()

		for req.Next() {
			key := req.Key()
			log.Printf("Get Value for Key=<%x>", key)
			if v, err := bkTrie.Get(key); err == nil && v != nil {
				builder.OnData(db.MerkleTrie, v)
				continue
			}
			if v, err := bkBytes.Get(key); err == nil && v != nil {
				builder.OnData(db.BytesByHash, v)
				continue
			}
			t.Errorf("Fail to get bytes for key=<%x>", key)
			return
		}
	}

	for _, e := range entries {
		vobj, err := ss2.Get(e.k)
		if err != nil {
			t.Errorf("Fail to get value for key=<%x>", e.k)
			return
		}
		to := vobj.(*TestObject)
		data, err := to.Data()
		if err != nil {
			t.Errorf("Fail to get Data err=%+v", err)
			return
		}
		if !bytes.Equal(data, e.v) {
			t.Errorf("Returned '%s' is different from exp='%s'",
				data, e.v)
			return
		}
	}

	if err := builder.Flush(true); err != nil {
		t.Errorf("Fail to flush builder err=%+v", err)
		return
	}

	ss3 := NewImmutableForObject(dbase2, ss.Hash(), reflect.TypeOf((*TestObject)(nil)))
	for _, e := range entries {
		vobj, err := ss3.Get(e.k)
		if err != nil {
			t.Errorf("Fail to get value for key=<%x>", e.k)
			return
		}
		to := vobj.(*TestObject)
		data, err := to.Data()
		if err != nil {
			t.Errorf("Fail to get Data err=%+v", err)
			return
		}
		if !bytes.Equal(data, e.v) {
			t.Errorf("Returned '%s' is different from exp='%s'",
				data, e.v)
			return
		}
	}
}
