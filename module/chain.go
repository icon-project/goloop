package module

import (
	"context"
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
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
	Channel() string
	ConcurrencyLevel() int
	NormalTxPoolSize() int
	PatchTxPoolSize() int
	MaxBlockTxBytes() int
	Genesis() []byte
	GetGenesisData(key []byte) ([]byte, error)
	CommitVoteSetDecoder() CommitVoteSetDecoder

	BlockManager() BlockManager
	Consensus() Consensus
	ServiceManager() ServiceManager
	NetworkManager() NetworkManager
	Regulator() Regulator

	Init(sync bool) error
	Start(sync bool) error
	Stop(sync bool) error
	Import(src string, sync bool) error
	Term(sync bool) error
	State() string

	Reset(sync bool) error
	Verify(sync bool) error

	MetricContext() context.Context
	Logger() log.Logger
}

type Regulator interface {
	MaxTxCount() int
	CommitTimeout() time.Duration
	OnTxExecution(count int, ed time.Duration, fd time.Duration)
	SetCommitTimeout(d time.Duration)
}
