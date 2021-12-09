package gs

import (
	"bytes"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type writerDatabase struct {
	w module.GenesisStorageWriter
}

func (d *writerDatabase) Get(key []byte) ([]byte, error) {
	return nil, nil
}

func (d *writerDatabase) Has(key []byte) (bool, error) {
	return false, nil
}

func (d *writerDatabase) Set(key []byte, value []byte) error {
	k2, err := d.w.WriteData(value)
	if err != nil {
		return err
	}
	if !bytes.Equal(key, k2) {
		return errors.CriticalHashError.Errorf(
			"InvalidHash(key=%x,hash=%x)", key, k2)
	}
	return nil
}

func (d *writerDatabase) Delete(key []byte) error {
	panic("unsupported")
}

func (d *writerDatabase) GetBucket(id db.BucketID) (db.Bucket, error) {
	if id == db.BytesByHash || id == db.MerkleTrie {
		return d, nil
	}
	return nil, errors.UnsupportedError.Errorf("GSWriterUnsupport(id=%s)", id)
}

func (d *writerDatabase) Close() error {
	return nil
}

func NewDatabaseWithWriter(w module.GenesisStorageWriter) db.Database {
	return &writerDatabase{w}
}

type readerDatabase struct {
	s module.GenesisStorage
}

func (d *readerDatabase) Close() error {
	// do nothing
	return nil
}

func (d *readerDatabase) Get(key []byte) ([]byte, error) {
	return d.s.Get(key)
}

func (d *readerDatabase) Has(key []byte) (bool, error) {
	v, err := d.s.Get(key)
	return len(v) > 0, err
}

func (d *readerDatabase) Set(key []byte, value []byte) error {
	return errors.UnsupportedError.Errorf("GenesisStorageIsReadOnly")
}

func (d *readerDatabase) Delete(key []byte) error {
	return errors.UnsupportedError.Errorf("GenesisStorageIsReadOnly")
}

func (d *readerDatabase) GetBucket(id db.BucketID) (db.Bucket, error) {
	if id == db.BytesByHash || id == db.MerkleTrie {
		return d, nil
	}
	return nil, errors.UnsupportedError.Errorf("GSWriterUnsupport(id=%s)", id)
}

func NewDatabaseWithStorage(s module.GenesisStorage) db.Database {
	return &readerDatabase{s}
}
