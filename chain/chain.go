package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/chain/imports"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/server"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/eeproxy"
)

const (
	ConfigDefaultNormalTxPoolSize = 5000
	ConfigDefaultPatchTxPoolSize  = 1000
	ConfigDefaultMaxBlockTxBytes  = 1024 * 1024
)

type Config struct {
	//fixed
	NID    int    `json:"nid"`
	DBType string `json:"db_type"`

	//static
	SeedAddr         string `json:"seed_addr"`
	Role             uint   `json:"role"`
	ConcurrencyLevel int    `json:"concurrency_level,omitempty"`
	NormalTxPoolSize int    `json:"normal_tx_pool,omitempty"`
	PatchTxPoolSize  int    `json:"patch_tx_pool,omitempty"`
	MaxBlockTxBytes  int    `json:"max_block_tx_bytes,omitempty"`

	//runtime
	Channel      string `json:"channel"`
	SecureSuites string `json:"secureSuites"`
	SecureAeads  string `json:"secureAeads"`

	GenesisStorage GenesisStorage  `json:"-"`
	Genesis        json.RawMessage `json:"genesis"`

	BaseDir  string `json:"chain_dir"`
	FilePath string `json:"-"` //absolute path
}

func (c *Config) ResolveAbsolute(targetPath string) string {
	if filepath.IsAbs(targetPath) {
		return targetPath
	}
	if c.FilePath == "" {
		r, _ := filepath.Abs(targetPath)
		return r
	}
	return filepath.Clean(path.Join(filepath.Dir(c.FilePath), targetPath))
}

func (c *Config) ResolveRelative(targetPath string) string {
	absPath, _ := filepath.Abs(targetPath)
	base := filepath.Dir(c.FilePath)
	base, _ = filepath.Abs(base)
	r, _ := filepath.Rel(base, absPath)
	return r
}

type singleChain struct {
	wallet module.Wallet

	database db.Database
	vld      module.CommitVoteSetDecoder
	sm       module.ServiceManager
	bm       module.BlockManager
	cs       module.Consensus
	srv      *server.Manager
	nt       module.NetworkTransport
	nm       module.NetworkManager

	cfg Config
	pm  eeproxy.Manager

	logger log.Logger

	regulator *regulator

	state       State
	lastErr     error
	initialized bool
	mtx         sync.RWMutex

	// monitor
	metricCtx context.Context
}

const (
	DefaultDBDir       = "db"
	DefaultWALDir      = "wal"
	DefaultContractDir = "contract"
)

const (
	StateCreated State = iota
	StateInitializing
	StateInitializeFailed
	StateStarting
	StateStartFailed
	StateStarted
	StateStopping
	StateStopped
	StateTerminating
	StateTerminated
	StateVerifying
	StateVerifyFailed
	StateImporting
	StateImportFailed
	StateReseting
	StateResetFailed
)

type State int

func (s State) String() string {
	switch s {
	case StateCreated:
		return "created"
	case StateInitializing:
		return "initializing"
	case StateInitializeFailed:
		return "initialize failed"
	case StateStarting:
		return "starting"
	case StateStartFailed:
		return "start failed"
	case StateStarted:
		return "started"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	case StateTerminating:
		return "terminating"
	case StateTerminated:
		return "terminated"
	case StateImporting:
		return "importing"
	case StateImportFailed:
		return "import failed"
	default:
		return "unknown"
	}
}

func (c *singleChain) Database() db.Database {
	return c.database
}

func (c *singleChain) Wallet() module.Wallet {
	return c.wallet
}

func (c *singleChain) NID() int {
	return c.cfg.NID
}

func (c *singleChain) Channel() string {
	return c.cfg.Channel
}

func (c *singleChain) Genesis() []byte {
	return c.cfg.GenesisStorage.Genesis()
}

func (c *singleChain) GetGenesisData(key []byte) ([]byte, error) {
	return c.cfg.GenesisStorage.Get(key)
}

func (c *singleChain) CommitVoteSetDecoder() module.CommitVoteSetDecoder {
	return c.vld
}

func (c *singleChain) EEProxyManager() eeproxy.Manager {
	return c.pm
}

func (c *singleChain) BlockManager() module.BlockManager {
	return c.bm
}

func (c *singleChain) ServiceManager() module.ServiceManager {
	return c.sm
}

func (c *singleChain) Consensus() module.Consensus {
	return c.cs
}

func (c *singleChain) NetworkManager() module.NetworkManager {
	return c.nm
}

func (c *singleChain) Regulator() module.Regulator {
	return c.regulator
}

func (c *singleChain) MetricContext() context.Context {
	return c.metricCtx
}

func toRoles(r uint) []module.Role {
	roles := make([]module.Role, 0)
	switch r {
	case 1:
		roles = append(roles, module.ROLE_SEED)
	case 2:
		roles = append(roles, module.ROLE_VALIDATOR)
	case 3:
		roles = append(roles, module.ROLE_VALIDATOR)
		roles = append(roles, module.ROLE_SEED)
	}
	return roles
}

func (c *singleChain) ConcurrencyLevel() int {
	if c.cfg.ConcurrencyLevel > 1 {
		return c.cfg.ConcurrencyLevel
	} else {
		return 1
	}
}

func (c *singleChain) NormalTxPoolSize() int {
	if c.cfg.NormalTxPoolSize > 0 {
		return c.cfg.NormalTxPoolSize
	}
	return ConfigDefaultNormalTxPoolSize
}

func (c *singleChain) PatchTxPoolSize() int {
	if c.cfg.PatchTxPoolSize > 0 {
		return c.cfg.PatchTxPoolSize
	}
	return ConfigDefaultPatchTxPoolSize
}

func (c *singleChain) MaxBlockTxBytes() int {
	if c.cfg.MaxBlockTxBytes > 0 {
		return c.cfg.MaxBlockTxBytes
	}
	return ConfigDefaultMaxBlockTxBytes
}

func (c *singleChain) State() string {
	return c._state().String()
}

func (c *singleChain) _state() State {
	defer c.mtx.RUnlock()
	c.mtx.RLock()

	return c.state
}

func (c *singleChain) LastError() error {
	return c.lastErr
}

func (c *singleChain) _setState(s State, err error) {
	defer c.mtx.Unlock()
	c.mtx.Lock()

	c.state = s
	c.lastErr = err
}

func (c *singleChain) _transit(to State, froms ...State) error {
	defer c.mtx.Unlock()
	c.mtx.Lock()

	invalid := true
	if len(froms) < 1 {
		if c.state != to {
			invalid = false
		}
	} else {
		for _, f := range froms {
			if f == c.state {
				invalid = false
				break
			}
		}
	}
	if invalid {
		return common.ErrInvalidState
	}
	c.state = to
	return nil
}

func (c *singleChain) _init() error {
	if c.cfg.Channel == "" {
		c.cfg.Channel = strconv.FormatInt(int64(c.cfg.NID), 16)
	}
	chainDir := c.cfg.ResolveAbsolute(c.cfg.BaseDir)
	log.Println("ConfigFilepath", c.cfg.FilePath, "BaseDir", c.cfg.BaseDir, "ChainDir", chainDir)
	DBDir := path.Join(chainDir, DefaultDBDir)
	if c.cfg.DBType == "" {
		c.cfg.DBType = string(db.GoLevelDBBackend)
	}
	if c.cfg.DBType != "mapdb" {
		if err := os.MkdirAll(DBDir, 0700); err != nil {
			return err
		}
	}
	DBName := strconv.FormatInt(int64(c.cfg.NID), 16)

	if c.cfg.GenesisStorage == nil {
		if len(c.cfg.Genesis) == 0 {
			return errors.IllegalArgumentError.Errorf("FAIL to generate GenesisStorage")
		}
		c.cfg.GenesisStorage = &genesisStorageWithDataDir{
			genesis:  c.cfg.Genesis,
			dataMap:  nil,
			dataPath: "",
		}
	}

	if cdb, err := db.Open(DBDir, c.cfg.DBType, DBName); err != nil {
		return err
	} else {
		c.database = cdb
	}

	c.vld = consensus.NewCommitVoteSetFromBytes
	c.metricCtx = metric.GetMetricContextByNID(c.NID())
	return nil
}

func (c *singleChain) _prepare() error {
	c.nm = network.NewManager(c, c.nt, c.cfg.SeedAddr, toRoles(c.cfg.Role)...)
	//TODO [TBD] is service/contract.ContractManager owner of ContractDir ?
	chainDir := c.cfg.ResolveAbsolute(c.cfg.BaseDir)
	ContractDir := path.Join(chainDir, DefaultContractDir)
	var err error
	var ts module.Timestamper
	c.sm, err = service.NewManager(c, c.nm, c.pm, ContractDir)
	if err != nil {
		return err
	}
	c.bm, err = block.NewManager(c, ts)
	if err != nil {
		return err
	}
	WALDir := path.Join(chainDir, DefaultWALDir)
	c.cs = consensus.NewConsensus(c, WALDir, ts)
	return nil
}

func (c *singleChain) _start() error {
	if err := c.nm.Start(); err != nil {
		return err
	}
	c.sm.Start()
	if err := c.cs.Start(); err != nil {
		return err
	}
	c.srv.SetChain(c.cfg.Channel, c)

	return nil
}

func (c *singleChain) _stop() {
	c.srv.RemoveChain(c.cfg.Channel)

	if c.cs != nil {
		c.cs.Term()
		c.cs = nil
	}
	if c.bm != nil {
		c.bm.Term()
		c.bm = nil
	}
	if c.sm != nil {
		c.sm.Term()
		c.sm = nil
	}
	if c.nm != nil {
		c.nm.Term()
		c.nm = nil
	}
}

func (c *singleChain) _import(src string) error {
	c.nm = network.NewManager(c, c.nt, c.cfg.SeedAddr, toRoles(c.cfg.Role)...)
	//TODO [TBD] is service/contract.ContractManager owner of ContractDir ?
	chainDir := c.cfg.ResolveAbsolute(c.cfg.BaseDir)
	ContractDir := path.Join(chainDir, DefaultContractDir)
	var err error
	var ts module.Timestamper
	c.sm, ts, err = imports.NewManagerForMigration(c, c.nm, c.pm, ContractDir, src)
	if err != nil {
		return err
	}
	c.bm, err = block.NewManager(c, ts)
	if err != nil {
		return err
	}
	WALDir := path.Join(chainDir, DefaultWALDir)
	c.cs = consensus.NewConsensus(c, WALDir, ts)

	if err := c.nm.Start(); err != nil {
		return err
	}
	if err := c.cs.Start(); err != nil {
		return err
	}

	return nil
}

func (c *singleChain) _execute(sync bool, f func()) error {
	if sync {
		f()
		return c.lastErr
	} else {
		go f()
		return nil
	}
}

func (c *singleChain) Init(sync bool) error {
	if err := c._transit(StateInitializing, StateCreated, StateStopped); err != nil {
		return err
	}

	f := func() {
		s := StateStopped
		err := c._init()
		if err != nil {
			s = StateInitializeFailed
		} else {
			err = c._prepare()
			if err != nil {
				s = StateInitializeFailed
			}
		}
		c._setState(s, err)
	}
	return c._execute(sync, f)
}

func (c *singleChain) Start(sync bool) error {
	if err := c._transit(StateStarting, StateStopped); err != nil {
		return err
	}

	f := func() {
		s := StateStarted
		err := c._start()
		if err != nil {
			s = StateStartFailed
			c._stop()
			c._prepare()
		}
		c._setState(s, err)
	}
	return c._execute(sync, f)
}

func (c *singleChain) Stop(sync bool) error {
	if err := c._transit(StateStopping, StateStarted, StateImporting); err != nil {
		return err
	}
	f := func() {
		c._stop()
		c._prepare()
		c._setState(StateStopped, nil)
	}
	return c._execute(sync, f)
}

func (c *singleChain) Import(src string, sync bool) error {
	if err := c._transit(StateImporting, StateStopped); err != nil {
		return err
	}
	f := func() {
		c._stop()
		s := StateStopped
		err := c._import(src)
		if err != nil {
			s = StateImportFailed
			c._stop()
			c._prepare()
		}
		c._setState(s, err)
	}
	return c._execute(sync, f)
}

func (c *singleChain) Term(sync bool) error {
	err := c._transit(StateTerminating)
	if err != nil {
		return err
	}

	f := func() {
		c._stop()
		c.vld = nil
		if c.database != nil {
			err = c.database.Close()
		}
		c._setState(StateTerminated, err)
	}
	return c._execute(sync, f)
}

func (c *singleChain) _verify() error {
	//verify code here
	return fmt.Errorf("not implemented")
}

func (c *singleChain) Verify(sync bool) error {
	if err := c._transit(StateVerifying, StateStopped); err != nil {
		return err
	}

	f := func() {
		s := StateStopped
		err := c._verify()
		if err != nil {
			s = StateVerifyFailed
		}
		c._setState(s, err)
	}
	return c._execute(sync, f)
}

func (c *singleChain) _reset() error {
	if err := c.database.Close(); err != nil {
		return err
	}
	chainDir := c.cfg.ResolveAbsolute(c.cfg.BaseDir)
	DBDir := path.Join(chainDir, DefaultDBDir)
	if err := os.RemoveAll(DBDir); err != nil {
		return err
	}

	WALDir := path.Join(chainDir, DefaultWALDir)
	if err := os.RemoveAll(WALDir); err != nil {
		return err
	}

	ContractDir := path.Join(chainDir, DefaultContractDir)
	if err := os.RemoveAll(ContractDir); err != nil {
		return err
	}
	return nil
}

func (c *singleChain) Reset(sync bool) error {
	if err := c._transit(StateReseting, StateStopped); err != nil {
		return err
	}

	f := func() {
		//TODO [TBD] if each module has Reset(), then doesn't need c._stop(), c._prepare()
		s := StateStopped
		c._stop()
		err := c._reset()
		if err != nil {
			s = StateResetFailed
		}
		c._prepare()
		c._setState(s, err)
	}
	return c._execute(sync, f)
}

func (c *singleChain) Logger() log.Logger {
	return c.logger
}

func NewChain(
	wallet module.Wallet,
	transport module.NetworkTransport,
	srv *server.Manager,
	pm eeproxy.Manager,
	logger log.Logger,
	cfg *Config,
) *singleChain {
	chainLogger := logger.WithFields(log.Fields{
		log.FieldKeyNID: strconv.FormatInt(int64(cfg.NID), 16),
	})
	c := &singleChain{
		wallet:    wallet,
		nt:        transport,
		srv:       srv,
		cfg:       *cfg,
		pm:        pm,
		logger:    chainLogger,
		regulator: NewRegulator(time.Second, 1000, chainLogger),
		metricCtx: metric.GetMetricContextByNID(cfg.NID),
	}
	return c
}
