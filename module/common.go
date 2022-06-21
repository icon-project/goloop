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
	// VerifyBlock verifies a block with block votes
	VerifyBlock(block BlockData, validators ValidatorList) ([]bool, error)
	BlockVoteSetBytes() []byte
	Bytes() []byte
	Hash() []byte
	Timestamp() int64

	// VoteRound returns vote round if it is for block version >= 2. In other
	// case, the value is ignored.
	VoteRound() int32
	NTSDProofList
}

type CommitVoteSetDecoder func([]byte) CommitVoteSet

type LogsBloom interface {
	String() string
	Bytes() []byte
	CompressedBytes() []byte
	LogBytes() []byte
	Contain(lb2 LogsBloom) bool
	Merge(lb2 LogsBloom)
	Equal(lb2 LogsBloom) bool
}

type Timestamper interface {
	GetVoteTimestamp(h, ts int64) int64
	GetBlockTimestamp(h, ts int64) int64
}

type Canceler interface {
	Cancel() bool
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

type JSONVersion int

const (
	JSONVersion2 JSONVersion = iota
	JSONVersion3
	JSONVersion3Raw
	JSONVersionLast = JSONVersion3Raw
)
