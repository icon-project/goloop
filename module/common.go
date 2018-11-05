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

type ValidatorList interface {
	Hash() []byte
}

type VoteList interface {
	Verify(block Block, validators ValidatorList) bool
	Bytes() []byte
	Hash() []byte
}

type VoteListDecoder func([]byte) VoteList
