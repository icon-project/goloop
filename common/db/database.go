package db

// DB
type DB interface {
	Get(key []byte) ([]byte, error)
	Has(key []byte) bool
	Set(key []byte, value []byte) error
	Delete(key []byte) error
	Transaction() (Transaction, error)
	Batch() Batch
	Iterator() Iterator
	Close() error
}

// Transaction
type Transaction interface {
	Get(key []byte) ([]byte, error)
	Set(key []byte, value []byte) error
	Delete(key []byte) error
	Commit() error
	Discard()
}

// Batch
type Batch interface {
	Set(key []byte, value []byte) error
	Delete(key []byte) error
	Write() error
}

// Iterator
type Iterator interface {
	Seek(key []byte)
	Next()
	Valid() bool
	Key() (key []byte)
	Value() (value []byte)
	Close()
}

// We defensively turn nil keys or values into []byte{} for most operations.
func nonNilBytes(bz []byte) []byte {
	if bz == nil {
		return []byte{}
	}
	return bz
}