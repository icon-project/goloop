package block

import (
	"bytes"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
)

type bucket struct {
	dbBucket db.Bucket
	codec    codec.Codec
}

func newBucket(database db.Database, id db.BucketID, c codec.Codec) *bucket {
	b := &bucket{}
	dbb, err := database.GetBucket(id)
	if err != nil {
		return nil
	}
	b.dbBucket = dbb
	if c == nil {
		c = codec.BC
	}
	b.codec = c
	return b
}

type raw []byte

func (b *bucket) _marshal(obj interface{}) ([]byte, error) {
	if bs, ok := obj.(raw); ok {
		return bs, nil
	}
	buf := bytes.NewBuffer(nil)
	err := b.codec.Marshal(buf, obj)
	return buf.Bytes(), err
}

func (b *bucket) get(key interface{}, value interface{}) error {
	bs, err := b.getBytes(key)
	if err != nil {
		return err
	}
	return b.codec.Unmarshal(bytes.NewBuffer(bs), value)
}

func (b *bucket) getBytes(key interface{}) ([]byte, error) {
	keyBS, err := b._marshal(key)
	if err != nil {
		return nil, err
	}
	bs, err := b.dbBucket.Get(keyBS)
	if bs == nil && err == nil {
		err = errors.NotFoundError.Wrap(err, "Not found")
	}
	return bs, err
}

func (b *bucket) set(key interface{}, value interface{}) error {
	keyBS, err := b._marshal(key)
	if err != nil {
		return err
	}
	valueBS, err := b._marshal(value)
	if err != nil {
		return err
	}
	return b.dbBucket.Set(keyBS, valueBS)
}

func (b *bucket) put(value interface{}) error {
	valueBS, err := b._marshal(value)
	if err != nil {
		return err
	}
	keyBS := crypto.SHA3Sum256(valueBS)
	return b.dbBucket.Set(keyBS, valueBS)
}
