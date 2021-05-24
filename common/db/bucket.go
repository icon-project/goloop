package db

import "github.com/icon-project/goloop/common/errors"

// Bucket
type Bucket interface {
	Get(key []byte) ([]byte, error)
	Has(key []byte) bool
	Set(key []byte, value []byte) error
	Delete(key []byte) error
}

type BucketID string

//	Bucket ID
const (
	// MerkleTrie maps RLP encoded data from sha3(data)
	MerkleTrie BucketID = ""

	// BytesByHash maps data except merkle trie nodes from sha3(data)
	BytesByHash BucketID = "S"

	// TransactionLocatorByHash maps transaction locator from transaction hash.
	TransactionLocatorByHash BucketID = "T"

	// BlockHeaderHashByHeight maps hash of encoded block header from height.
	BlockHeaderHashByHeight BucketID = "H"

	// ChainProperty is general key value map for chain property.
	ChainProperty BucketID = "C"
)

// internalKey returns key prefixed with the bucket's id.
func internalKey(id BucketID, key []byte) []byte {
	buf := make([]byte, len(key)+len(id))
	copy(buf, id)
	copy(buf[len(id):], key)
	return buf
}

// nonNilBytes returns empty []byte if bz is nil
func nonNilBytes(bz []byte) []byte {
	if bz == nil {
		return []byte{}
	}
	return bz
}

func DoGet(bk Bucket, key []byte) ([]byte, error) {
	v, err := bk.Get(key)
	if v==nil && err==nil {
		return nil, errors.NotFoundError.New("NotFound")
	}
	return v, err
}
