package module

import "github.com/icon-project/goloop/common/db"

type Wallet interface {
	Address() Address
	Sign(data []byte) ([]byte, error)
	PublicKey() []byte
}

type Chain interface {
	Database() db.Database
	Wallet() Wallet
	NID() int
	Genesis() []byte
	CommitVoteSetDecoder() CommitVoteSetDecoder
	GetPreInstalledScore(contentID string) []byte
}
