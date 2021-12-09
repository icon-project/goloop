package db

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
	return nil, nil
}

func (*nullBucket) Has(key []byte) (bool, error) {
	return false, nil
}

func (*nullBucket) Set(key []byte, value []byte) error {
	panic("NullBucket.Set() Unsupported")
}

func (*nullBucket) Delete(key []byte) error {
	panic("NullBucket.Delete() Unsupported")
}

func NewNullDB() *nullDB {
	return &nullDB{}
}
