package db

import "github.com/icon-project/goloop/common"

type nullDB struct {
}

func (*nullDB) GetBucket(id BucketID) (Bucket, error) {
	return &nullBucket{}, nil
}

func (*nullDB) Close() error {
	return nil
}

type nullBucket struct {
}

func (*nullBucket) Get(key []byte) ([]byte, error) {
	return nil, common.ErrNotFound
}

func (*nullBucket) Has(key []byte) bool {
	return false
}

func (*nullBucket) Set(key []byte, value []byte) error {
	panic("implement me")
	return common.ErrUnsupported
}

func (*nullBucket) Delete(key []byte) error {
	panic("implement me")
	return common.ErrUnsupported
}

func NewNullDB() *nullDB {
	return &nullDB{}
}
