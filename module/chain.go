package module

import "github.com/icon-project/goloop/common/db"

type Wallet interface {
	GetAddress() Address
	Sign(data []byte) []byte
	PublicKey() []byte
}

type Chain interface {
	GetDatabase() db.Database
	GetWallet() Wallet
	GetNID() int
}
