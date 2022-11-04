package db

import (
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
)

// Bucket
type Bucket interface {
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Set(key []byte, value []byte) error
	Delete(key []byte) error
}

type BucketID string

type Hasher interface {
	Name() string
	Hash(value []byte) []byte
}

type sha3Hasher struct{}

func (h sha3Hasher) Name() string {
	return "sha3"
}

func (h sha3Hasher) Hash(v []byte) []byte {
	return crypto.SHA3Sum256(v)
}

var hasherMap = map[BucketID]Hasher{
	MerkleTrie:  sha3Hasher{},
	BytesByHash: sha3Hasher{},
}

func RegisterHasher(bk BucketID, hasher Hasher) {
	if _, ok := hasherMap[bk]; ok {
		panic("Duplicate BucketID")
	}
	hasherMap[bk] = hasher
}

func (bk BucketID) Hasher() Hasher {
	return hasherMap[bk]
}

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

	// ListByMerkleRootBase is the base for the bucket that maps list
	// from network type dependent merkle root(list)
	ListByMerkleRootBase BucketID = "L"
)

// internalKey returns key prefixed with the bucket's id.
func internalKey(id BucketID, key []byte) []byte {
	buf := make([]byte, len(key)+len(id))
	copy(buf, id)
	copy(buf[len(id):], key)
	return buf
}

func DoGet(bk Bucket, key []byte) ([]byte, error) {
	v, err := bk.Get(key)
	if v == nil && err == nil {
		return nil, errors.NotFoundError.New("NotFound")
	}
	return v, err
}

func DoGetWithBucketID(dbase Database, bid BucketID, key []byte) ([]byte, error) {
	bk, err := dbase.GetBucket(bid)
	if err != nil {
		return nil, err
	}
	return DoGet(bk, key)
}
