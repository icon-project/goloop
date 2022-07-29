package module

import (
	"context"
	"encoding/json"
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
	TransactionTimeout() time.Duration
	ChildrenLimit() int
	NephewsLimit() int
	ValidateTxOnSend() bool
	Genesis() []byte
	GenesisStorage() GenesisStorage
	CommitVoteSetDecoder() CommitVoteSetDecoder
	PatchDecoder() PatchDecoder

	BlockManager() BlockManager
	Consensus() Consensus
	ServiceManager() ServiceManager
	NetworkManager() NetworkManager
	Regulator() Regulator

	Init() error
	Start() error
	Stop() error
	Import(src string, height int64) error
	Prune(gs string, dbt string, height int64) error
	Backup(file string, extra []string) error
	RunTask(task string, params json.RawMessage) error
	Term() error
	State() (string, int64, error)
	IsStarted() bool
	IsStopped() bool

	// Reset resets chain. height must be 0 or greater than 1.
	// If height == 0, blockHash shall be nil or zero length
	// bytes and the function cleans up database and file systems for the chain.
	// If height > 1, blockHash shall be the hash of correct block with the
	// height and the function cleans up database and file systems for the chain
	// and prepare pruned genesis block of the height.
	Reset(gs string, height int64, blockHash []byte) error
	Verify() error

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
	Height() int64
	Type() (GenesisType, error)
	Genesis() []byte
	Get(key []byte) ([]byte, error)
}

type GenesisStorageWriter interface {
	WriteGenesis(gtx []byte) error
	WriteData(value []byte) ([]byte, error)
	Close() error
}
