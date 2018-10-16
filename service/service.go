package service

type Transaction interface {
	GetID() []byte
	GetHash() []byte
	GetVersion() int
	Verify() error
}

func NewTransaction(b []byte) (Transaction, error) {
	return NewTransactionV3(b)
}
