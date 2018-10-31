package module

type Address interface {
	String() string
	Bytes() []byte
}

type Vote interface {
	Voter() Address
	Bytes() []byte
}

type Validator Address

func GetID() Validator {
	return nil
}

type VoteList interface {
	Verify(block Block, validators []Validator) bool
	Bytes() []byte
}

type VoteListDecoder func([]byte) VoteList
