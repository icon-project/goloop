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
	CID() int
	NetID() int
	Channel() string
	ConcurrencyLevel() int
	NormalTxPoolSize() int
	PatchTxPoolSize() int
	MaxBlockTxBytes() int
	DefaultWaitTimeout() time.Duration
	MaxWaitTimeout() time.Duration
	Genesis() []byte
	GenesisStorage() GenesisStorage
	CommitVoteSetDecoder() CommitVoteSetDecoder
	PatchDecoder() PatchDecoder

	BlockManager() BlockManager
	Consensus() Consensus
	ServiceManager() ServiceManager
	NetworkManager() NetworkManager
	Regulator() Regulator

	Init(sync bool) error
	Start(sync bool) error
	Stop(sync bool) error
	Import(src string, height int64, sync bool) error
	Prune(gs string, dbt string, height int64, sync bool) error
	Term(sync bool) error
	State() string

	Reset(sync bool) error
	Verify(sync bool) error

	MetricContext() context.Context
	Logger() log.Logger
}

type Regulator interface {
	MaxTxCount() int
	OnPropose(now time.Time)
	CommitTimeout() time.Duration
	MinCommitTimeout() time.Duration
	OnTxExecution(count int, ed time.Duration, fd time.Duration)
	SetBlockInterval(i time.Duration, d time.Duration)
}

type GenesisType int

const (
	GenesisUnknown GenesisType = iota
	GenesisNormal
	GenesisPruned
)

type GenesisStorage interface {
	CID() (int, error)
	NID() (int, error)
	Type() (GenesisType, error)
	Genesis() []byte
	Get(key []byte) ([]byte, error)
}

type GenesisStorageWriter interface {
	WriteGenesis(gtx []byte) error
	WriteData(value []byte) ([]byte, error)
	Close() error
}
