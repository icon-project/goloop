package module

import (
	"time"

	"github.com/icon-project/goloop/common/db"
)

type Wallet interface {
	Address() Address
	Sign(data []byte) ([]byte, error)
	PublicKey() []byte
}

type Chain interface {
	Database() db.Database
	Wallet() Wallet
	NID() int
	ConcurrencyLevel() int
	Genesis() []byte
	GetGenesisData(key []byte) ([]byte, error)
	CommitVoteSetDecoder() CommitVoteSetDecoder

	BlockManager() BlockManager
	Consensus() Consensus
	ServiceManager() ServiceManager
	NetworkManager() NetworkManager
	Regulator() Regulator
}

type Regulator interface {
	MaxTxCount() int
	CommitTimeout() time.Duration
	OnTxExecution(count int, ed time.Duration, fd time.Duration)
	SetCommitTimeout(d time.Duration)
}
