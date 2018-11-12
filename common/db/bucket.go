package db

import "math"

// Bucket
type Bucket interface {
	Get(key []byte) ([]byte, error)
	Has(key []byte) bool
	Set(key []byte, value []byte) error
	Delete(key []byte) error
}

const bucketIdSize = 2
const reserved byte = 0x00
const MaxBuckets = math.MaxUint16 - (8 * 256)

var bucketIdSequence = []byte{reserved, 0x00}
var metaKey = []byte{reserved, 0x01}

type bucketId [bucketIdSize]byte

type bucketMeta struct {
	buckets map[string]bucketId
}

// internalKey returns key prefixed with the bucket's id.
func internalKey(id bucketId, key []byte) []byte {
	buf := make([]byte, len(key)+bucketIdSize)
	copy(buf, id[:])
	copy(buf[bucketIdSize:], key)
	return buf
}

//	Bucket ID
const (
	// MerkleTrie maps RLP encoded data from sha3(data)
	MerkleTrie = ""

	// BytesByHash maps data except merkle trie nodes from sha3(data)
	BytesByHash = "S"

	// TransactionLocatorByHash maps transaction locator from transaction hash.
	TransactionLocatorByHash = "T"

	// BlockHeaderHashByHeight maps hash of encoded block header from height.
	BlockHeaderHashByHeight = "H"

	// BlockV1ByHash maps block V1 from block V1 hash.
	BlockV1ByHash = "B"

	// ReceiptV1ByHash maps receipt V1 from tx V3 hash.
	ReceiptV1ByHash = "R"

	// ChainProperty is general key value map for chain property.
	ChainProperty = "C"
)
