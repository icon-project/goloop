package module

type Address interface {
	String() string
	Bytes() []byte
	ID() []byte
	IsContract() bool
}

type Vote interface {
	Voter() Validator
	Bytes() []byte
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
}

type VoteList interface {
	Verify(block Block, validators ValidatorList) error
	Bytes() []byte
	Hash() []byte
}

type VoteListDecoder func([]byte) VoteList

type TransactionGroup int

const (
	TransactionGroupPatch TransactionGroup = iota
	TransactionGroupNormal
)

const (
	TransactionVersion2 = 2
	TransactionVersion3 = 3
)
