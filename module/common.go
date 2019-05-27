package module

type Address interface {
	String() string
	Bytes() []byte
	ID() []byte
	IsContract() bool
	Equal(Address) bool
}

type Validator interface {
	Address() Address

	// PublicKey returns public key of the validator.
	// If it doesn't have, then it return nil
	PublicKey() []byte

	Bytes() []byte
}

type ValidatorList interface {
	Hash() []byte
	Bytes() []byte
	Flush() error
	IndexOf(Address) int
	Len() int
	Get(i int) (Validator, bool)
}

type MemberIterator interface {
	Has() bool
	Next() error
	Get() (Address, error)
}

type MemberList interface {
	IsEmpty() bool
	Equal(MemberList) bool
	Iterator() MemberIterator
}

type CommitVoteSet interface {
	Verify(block Block, validators ValidatorList) error
	Bytes() []byte
	Hash() []byte
	Timestamp() int64
}

type CommitVoteSetDecoder func([]byte) CommitVoteSet

type LogBloom interface {
	String() string
	Bytes() []byte
	CompressedBytes() []byte
	LogBytes() []byte
	Contain(lb2 LogBloom) bool
	Merge(lb2 LogBloom)
	Equal(lb2 LogBloom) bool
}

type TransactionGroup int

const (
	TransactionGroupPatch TransactionGroup = iota
	TransactionGroupNormal
)

const (
	TransactionVersion2 = 2
	TransactionVersion3 = 3
)
