package chain

import (
	"encoding/json"
	"log"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/server"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/eeproxy"
)

type Config struct {
	NID      int    `json:"nid"`
	Channel  string `json:"channel"`
	SeedAddr string `json:"seed_addr"`
	Role     uint   `json:"role"`

	DBDir  string `json:"db_dir"`
	DBType string `json:"db_type"`
	DBName string `json:"db_name"`

	WALDir      string `json:"wal_dir"`
	ContractDir string `json:"contract_dir"`

	GenesisStorage  GenesisStorage  `json:"-"`
	Genesis         json.RawMessage `json:"genesis"`
	GenesisDataPath string          `json:"genesis_data,omitempty"`

	ConcurrencyLevel int `json:"concurrency_level,omitempty"`

	ChainDir string `json:"chain_dir"`
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

	regulator *regulator

	state       State
	lastErr     error
	initialized bool
	mtx         sync.RWMutex
}

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

func (c *singleChain) State() string {
	return c._state().String()
}

func (c *singleChain) _state() State {
	defer c.mtx.RUnlock()
	c.mtx.RLock()

	return c.state
}

func (c *singleChain) _setState(s State, err error) {
	defer c.mtx.Unlock()
	c.mtx.Lock()

	c.state = s
	c.lastErr = err
}

func (c *singleChain) _transit(to State, froms ... State) error {
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
	if c.cfg.ChainDir == "" {
		c.cfg.ChainDir = path.Join(".", ".chain", c.wallet.Address().String())
	}
	if err := os.MkdirAll(c.cfg.ChainDir, 0700); err != nil {
		log.Panicf("Fail to create directory %s err=%+v", c.cfg.ChainDir, err)
	}

	if c.cfg.Channel == "" {
		c.cfg.Channel = strconv.FormatInt(int64(c.cfg.NID), 16)
	}

	if c.cfg.DBDir == "" {
		c.cfg.DBDir = path.Join(c.cfg.ChainDir, "db")
	}
	if c.cfg.DBType == "" {
		c.cfg.DBType = string(db.GoLevelDBBackend)
	}
	if c.cfg.DBType != "mapdb" {
		if err := os.MkdirAll(c.cfg.DBDir, 0700); err != nil {
			return err
		}
	}
	if c.cfg.DBName == "" {
		c.cfg.DBName = c.cfg.Channel
	}

	if c.cfg.WALDir == "" {
		c.cfg.WALDir = path.Join(c.cfg.ChainDir, "wal")
	}
	if c.cfg.ContractDir == "" {
		c.cfg.ContractDir = path.Join(c.cfg.ChainDir, "contract")
	}
	if c.cfg.GenesisStorage == nil {
		if gs, err := NewGenesisStorageWithDataDir(
			c.cfg.Genesis, c.cfg.GenesisDataPath); err != nil {
			return err
		} else {
			c.cfg.GenesisStorage = gs
		}
	}

	if cdb, err := db.Open(c.cfg.DBDir, c.cfg.DBType, c.cfg.DBName); err != nil {
		return err
	} else {
		c.database = cdb
	}

	c.vld = consensus.NewCommitVoteSetFromBytes
	return nil
}

func (c *singleChain) _prepare() {
	c.nm = network.NewManager(c.cfg.Channel, c.nt, c.cfg.SeedAddr, toRoles(c.cfg.Role)...)
	c.sm = service.NewManager(c, c.nm, c.pm, c.cfg.ContractDir)
	c.bm = block.NewManager(c, c.sm)
	c.cs = consensus.NewConsensus(c, c.bm, c.nm, c.cfg.WALDir)
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
			c._prepare()
		}
		c._setState(s, err)
	}
	return c._execute(sync, f)
}

func (c *singleChain) Start(sync bool) error {
	if err := c._transit(StateStarting, StateStopped); err != nil {
		return err
	}

	f := func(){
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
	if err := c._transit(StateStopping, StateStarted); err != nil {
		return err
	}
	f := func(){
		c._stop()
		c._prepare()
		c._setState(StateStopped, nil)
	}
	return c._execute(sync, f)
}

func (c *singleChain) Term(sync bool) error {
	err := c._transit(StateTerminating)
	if err != nil {
		return err
	}

	f := func(){
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
	return nil
}

func (c *singleChain) Verify(sync bool) error {
	if err := c._transit(StateVerifying, StateStopped); err != nil {
		return err
	}

	f := func(){
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
	if err := os.RemoveAll(c.cfg.DBDir); err != nil {
		return err
	}
	if err := os.RemoveAll(c.cfg.WALDir); err != nil {
		return err
	}
	if err := os.RemoveAll(c.cfg.ContractDir); err != nil {
		return err
	}
	return nil
}

func (c *singleChain) Reset(sync bool) error {
	if err := c._transit(StateReseting, StateStopped); err != nil {
		return err
	}

	f := func(){
		s := StateStopped
		err := c._reset()
		if err != nil {
			s = StateResetFailed
		}
		c._setState(s, err)
	}
	return c._execute(sync, f)
}

func NewChain(
	wallet module.Wallet,
	transport module.NetworkTransport,
	srv *server.Manager,
	pm eeproxy.Manager,
	cfg *Config,
) *singleChain {
	c := &singleChain{
		wallet:    wallet,
		nt:        transport,
		srv:       srv,
		cfg:       *cfg,
		pm:        pm,
		regulator: NewRegulator(time.Second, 1000),
	}
	return c
}
