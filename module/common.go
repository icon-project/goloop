package module

type Address interface {
	String() string
	Bytes() []byte
	ID() []byte
	IsContract() bool
	Equal(Address) bool
}

type Validator interface {
	Member

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

type Member interface {
	Address() Address
}

type CommitVoteSet interface {
	Verify(block Block, validators ValidatorList) error
	Bytes() []byte
	Hash() []byte
}

type CommitVoteSetDecoder func([]byte) CommitVoteSet

type LogBloom interface {
	String() string
	Bytes() []byte
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
