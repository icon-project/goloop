package module

import "github.com/icon-project/goloop/common/db"

type Wallet interface {
	Address() Address
	Sign(data []byte) ([]byte, error)
	PublicKey() []byte
}

type Chain interface {
	GetDatabase() db.Database
	GetWallet() Wallet
	GetNID() int
	VoteListDecoder() VoteListDecoder
	GetGenesisTxPath() string
}
