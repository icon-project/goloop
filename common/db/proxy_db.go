package db

import "github.com/icon-project/goloop/common/errors"

type proxyBucket struct {
	database *proxyDB
	id       BucketID
	real     Bucket
}

func (bk *proxyBucket) Get(key []byte) ([]byte, error) {
	if bk.real != nil {
		return bk.real.Get(key)
	}
	return nil, errors.New("ProxyIsNotRealized")
}

func (bk *proxyBucket) Has(key []byte) (bool, error) {
	if bk.real != nil {
		return bk.real.Has(key)
	}
	return false, nil
}

func (bk *proxyBucket) Set(key []byte, value []byte) error {
	if bk.real != nil {
		return bk.real.Set(key, value)
	}
	return errors.New("ProxyIsNotRealized")
}

func (bk *proxyBucket) Delete(key []byte) error {
	if bk.real != nil {
		return bk.real.Delete(key)
	}
	return errors.New("ProxyIsNotRealized")
}

type proxyDB struct {
	real    Database
	buckets map[string]*proxyBucket
}

func (pdb *proxyDB) GetBucket(id BucketID) (Bucket, error) {
	bk, ok := pdb.buckets[string(id)]
	if ok {
		return bk, nil
	}
	if pdb.real != nil {
		return pdb.real.GetBucket(id)
	}
	bk = &proxyBucket{
		database: pdb,
		id:       id,
	}
	pdb.buckets[string(id)] = bk
	return bk, nil
}

func (pdb *proxyDB) Close() error {
	return nil
}

func (pdb *proxyDB) SetReal(database Database) error {
	pdb.real = database
	for _, bk := range pdb.buckets {
		var err error
		bk.real, err = database.GetBucket(bk.id)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewProxyDB() *proxyDB {
	return &proxyDB{
		buckets: make(map[string]*proxyBucket),
	}
}
