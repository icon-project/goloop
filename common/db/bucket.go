package db

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

	// BlockV1ByHash maps block V1 from block V1 hash.
	BlockV1ByHash BucketID = "B"

	// ReceiptV1ByHash maps receipt V1 from tx V3 hash.
	ReceiptV1ByHash BucketID = "R"

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