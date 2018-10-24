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

