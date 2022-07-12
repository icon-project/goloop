package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/chain/gs"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie/cache"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/server"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/eeproxy"
)

type State int

const (
	Created State = iota
	Initializing
	InitializeFailed
	Starting
	Started
	Stopping
	Failed
	Finished
	Stopped
	Terminating
	Terminated
)

func (s State) String() string {
	switch s {
	case Created:
		return "created"
	case Initializing:
		return "initializing"
	case InitializeFailed:
		return "initialize failed"
	case Stopped:
		return "stopped"
	case Starting:
		return "starting"
	case Started:
		return "started"
	case Stopping:
		return "stopping"
	case Failed:
		return "failed"
	case Finished:
		return "finished"
	}
	return fmt.Sprintf("invalid(%d)", s)
}

type chainTask interface {
	String() string
	DetailOf(s State) string
	Start() error
	Stop()
	Wait() error
}

type singleChain struct {
	wallet module.Wallet

	database db.Database
	vld      module.CommitVoteSetDecoder
	pd       module.PatchDecoder
	sm       module.ServiceManager
	bm       module.BlockManager
	cs       module.Consensus
	srv      *server.Manager
	nt       module.NetworkTransport
	nm       module.NetworkManager
	plt      base.Platform

	cid int
	cfg Config
	pm  eeproxy.Manager

	logger log.Logger

	regulator *regulator

	state      State
	lastErr    error
	mtx        sync.RWMutex
	task       chainTask
	termWaiter *sync.Cond

	// monitor
	metricCtx context.Context
}

const (
	DefaultDBDir       = "db"
	DefaultWALDir      = "wal"
	DefaultContractDir = "contract"
	DefaultCacheDir    = "cache"
	DefaultTmpDBDir    = "tmp"
)

func (c *singleChain) Database() db.Database {
	return c.database
}

func (c *singleChain) Wallet() module.Wallet {
	return c.wallet
}

func (c *singleChain) NID() int {
	return c.cfg.NID
}

func (c *singleChain) CID() int {
	return c.cid
}

func (c *singleChain) NetID() int {
	return c.cfg.NetID()
}

func (c *singleChain) Channel() string {
	return c.cfg.Channel
}

func (c *singleChain) Genesis() []byte {
	return c.cfg.GenesisStorage.Genesis()
}

func (c *singleChain) GenesisStorage() module.GenesisStorage {
	return c.cfg.GenesisStorage
}

func (c *singleChain) CommitVoteSetDecoder() module.CommitVoteSetDecoder {
	return c.vld
}

func (c *singleChain) PatchDecoder() module.PatchDecoder {
	return c.pd
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

func (c *singleChain) DefaultWaitTimeout() time.Duration {
	if c.cfg.DefWaitTimeout > 0 {
		return time.Duration(c.cfg.DefWaitTimeout) * time.Millisecond
	}
	return 0
}

func (c *singleChain) MaxWaitTimeout() time.Duration {
	if c.cfg.DefWaitTimeout > 0 {
		if c.cfg.MaxWaitTimeout > c.cfg.DefWaitTimeout {
			return time.Duration(c.cfg.MaxWaitTimeout) * time.Millisecond
		}
		return time.Duration(c.cfg.DefWaitTimeout) * time.Millisecond
	}
	return 0
}

func (c *singleChain) TransactionTimeout() time.Duration {
	if c.cfg.TxTimeout > 0 {
		return time.Duration(c.cfg.TxTimeout) * time.Millisecond
	}
	return ConfigDefaultTxTimeout
}

func (c *singleChain) ChildrenLimit() int {
	if c.cfg.ChildrenLimit != nil && *c.cfg.ChildrenLimit >= 0 {
		return *c.cfg.ChildrenLimit
	}
	return ConfigDefaultChildrenLimit
}

func (c *singleChain) NephewsLimit() int {
	if c.cfg.NephewsLimit != nil && *c.cfg.NephewsLimit >= 0 {
		return *c.cfg.NephewsLimit
	}
	return ConfigDefaultNephewLimit
}

func (c *singleChain) ValidateTxOnSend() bool {
	return c.cfg.ValidateTxOnSend
}

func (c *singleChain) State() (string, int64, error) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	switch c.state {
	case Starting, Started, Stopping, Failed, Finished:
		var height int64
		if c.bm != nil {
			if blk, err := c.bm.GetLastBlock(); err == nil {
				height = blk.Height()
			}
		}
		return c.task.DetailOf(c.state), height, c.lastErr
	default:
		return c.state.String(), c.lastBlockHeight(), c.lastErr
	}
}

func (c *singleChain) lastBlockHeight() int64 {
	if c.database == nil {
		return 0
	}
	return block.GetLastHeightOf(c.database)
}

func (c *singleChain) IsStarted() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	if c.state == Started {
		if _, ok := c.task.(*taskConsensus); ok {
			return true
		}
	}
	return false
}

func (c *singleChain) IsStopped() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	switch c.state {
	case Stopped, Failed, Finished:
		return true
	default:
		return false
	}
}

func (c *singleChain) _transitOrTerminate(to State, err error, froms ...State) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c._transitInLock(to, err, froms...) {
		c._handleTerminateInLock()
	}
}

func (c *singleChain) _transit(to State, err error, froms ...State) bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c._transitInLock(to, err, froms...)
}

func (c *singleChain) _transitInLock(to State, err error, froms ...State) bool {
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
		return false
	}
	c.state = to
	c.lastErr = err
	return true
}

func (c *singleChain) _setStartingTask(task chainTask) error {
	defer c.mtx.Unlock()
	c.mtx.Lock()

	switch c.state {
	case Stopped, Failed, Finished:
		c.state = Starting
		c.lastErr = nil
		c.task = task
		c.logger.Infof("STARTING %s", task.String())
		return nil
	default:
		return errors.InvalidStateError.Errorf("InvalidState(state=%s)", c.state.String())
	}
}

func (c *singleChain) openDatabase(dbDir, dbType string) (db.Database, error) {
	if dbType != "mapdb" {
		c.logger.Infof("prepare a directory %s for database", dbDir)
		if err := os.MkdirAll(dbDir, 0700); err != nil {
			return nil, errors.Wrapf(err, "fail to make directory dir=%s", dbDir)
		}
	}
	DBName := strconv.FormatInt(int64(c.cfg.NID), 16)
	if cdb, err := db.Open(dbDir, dbType, DBName); err != nil {
		return nil, errors.Wrapf(err,
			"fail to open database dir=%s type=%s name=%s", dbDir, c.cfg.DBType, DBName)
	} else {
		return cdb, nil
	}
}

func (c *singleChain) ensureDatabase() {
	chainDir := c.cfg.AbsBaseDir()
	if err := c.prepareDatabase(chainDir); err != nil {
		c.logger.Panicf("Fail to open database chainDir=%s err=%+v",
			chainDir, err)
	}
	return
}

func (c *singleChain) prepareDatabase(chainDir string) error {
	DBDir := path.Join(chainDir, DefaultDBDir)
	cdb, err := c.openDatabase(DBDir, c.cfg.DBType)
	if err != nil {
		return err
	}
	if len(c.cfg.NodeCache) == 0 {
		c.cfg.NodeCache = NodeCacheDefault
	}
	mLevel, fLevel, stores, err := ParseNodeCacheOption(c.cfg.NodeCache)
	if err != nil {
		_ = cdb.Close()
		return errors.Wrapf(err, "UnknownCacheStrategy(%s)", c.cfg.NodeCache)
	}
	cacheDir := path.Join(chainDir, DefaultCacheDir)
	c.database = cache.AttachManager(cdb, cacheDir, mLevel, fLevel, stores)
	return nil
}

func (c *singleChain) releaseDatabase() {
	if c.database != nil {
		c.database.Close()
		c.database = nil
	}
}

func (c *singleChain) _init() error {
	if c.cfg.Channel == "" {
		c.cfg.Channel = strconv.FormatInt(int64(c.cfg.NID), 16)
	}
	if c.cfg.DBType == "" {
		c.cfg.DBType = string(db.GoLevelDBBackend)
	}
	if c.cfg.GenesisStorage == nil {
		if len(c.cfg.Genesis) == 0 {
			return errors.IllegalArgumentError.Errorf("FAIL to generate GenesisStorage")
		}
		c.cfg.GenesisStorage = gs.NewFromTx(c.cfg.Genesis)
	}

	chainDir := c.cfg.AbsBaseDir()
	log.Println("ConfigFilepath", c.cfg.FilePath, "BaseDir", c.cfg.BaseDir, "ChainDir", chainDir)

	if plt, err := NewPlatform(c.cfg.Platform, chainDir, c.cid); err != nil {
		return err
	} else {
		c.plt = plt
	}

	if err := c.prepareDatabase(chainDir); err != nil {
		return err
	}

	c.vld = c.plt.CommitVoteSetDecoder()
	if c.vld == nil {
		c.vld = consensus.NewCommitVoteSetFromBytes
	}
	c.pd = consensus.DecodePatch
	c.metricCtx = metric.GetMetricContextByCID(c.CID())
	return nil
}

func (c *singleChain) prepareManagers() error {
	pr := network.PeerRoleFlag(c.cfg.Role)
	c.nm = network.NewManager(c, c.nt, c.cfg.SeedAddr, pr.ToRoles()...)

	chainDir := c.cfg.AbsBaseDir()
	ContractDir := path.Join(chainDir, DefaultContractDir)
	var err error
	c.sm, err = service.NewManager(c, c.nm, c.pm, c.plt, ContractDir)
	if err != nil {
		return err
	}
	bhs := c.plt.NewBlockHandlers(c)
	c.bm, err = block.NewManager(c, nil, bhs)
	if err != nil {
		return err
	}
	WALDir := path.Join(chainDir, DefaultWALDir)
	c.cs, err = c.plt.NewConsensus(c, WALDir)
	if err != nil {
		return err
	}
	return nil
}

func (c *singleChain) releaseManagers() {
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

func (c *singleChain) _runTask(task chainTask, wait bool) error {
	if err := c._setStartingTask(task); err != nil {
		return err
	}
	if err := task.Start(); err != nil {
		c.logger.Infof("Fail to start %s err=%v",
			task.String(), err)
		c._transitOrTerminate(Failed, err, Starting)
		return err
	}
	c.logger.Infof("STARTED %s", task.String())
	if ok := c._transit(Started, nil, Starting); !ok {
		c.logger.Infof("TERMINATING %s", task.String())
		task.Stop()
	}
	if wait {
		return c._waitResultOf(task)
	} else {
		go c._waitResultOf(task)
		return nil
	}
}

func (c *singleChain) _waitResultOf(task chainTask) error {
	result := task.Wait()
	c.logger.Infof("DONE %s err=%+v", task.String(), result)

	if result == nil {
		c._transitOrTerminate(Finished, nil, Started, Stopping)
	} else if errors.InterruptedError.Equals(result) {
		c.task = nil
		c._transitOrTerminate(Stopped, nil, Stopping)
	} else {
		c._transitOrTerminate(Failed, result, Started, Stopping)
	}
	return result
}

func (c *singleChain) Init() error {
	if ok := c._transit(Initializing, nil, Created); !ok {
		return errors.InvalidStateError.Errorf("InvalidState(state=%s)", c.state)
	}
	err := c._init()
	if err != nil {
		c._transit(InitializeFailed, err)
	} else {
		c._transit(Stopped, nil)
	}
	return err
}

func (c *singleChain) Start() error {
	task := newTaskConsensus(c)
	return c._runTask(task, false)
}

func (c *singleChain) Stop() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	switch c.state {
	case Failed, Finished:
		c.state = Stopped
		c.lastErr = nil
		c.task = nil
		return nil
	case Started:
		c.state = Stopping
		c.lastErr = nil
		c.logger.Infof("STOP %s", c.task.String())
		c.task.Stop()
		return nil
	default:
		return errors.InvalidStateError.Errorf(
			"InvalidStateToStop(status=%s)", c.state.String())
	}
}

func (c *singleChain) Import(src string, height int64) error {
	task := newTaskImport(c, src, height)
	return c._runTask(task, false)
}

func (c *singleChain) Prune(gsfile string, dbtype string, height int64) error {
	if dbtype == "" {
		dbtype = c.cfg.DBType
	}
	task := newTaskPruning(c, gsfile, dbtype, height)
	return c._runTask(task, false)
}

func (c *singleChain) Backup(file string, extra []string) error {
	task := newTaskBackup(c, file, extra)
	return c._runTask(task, false)
}

type TaskFactory func(c *singleChain, params json.RawMessage) (chainTask, error)

var taskFactories = map[string]TaskFactory{}

func registerTaskFactory(name string, factory TaskFactory) {
	if _, ok := taskFactories[name]; ok {
		panic("duplicated task factory")
	}
	taskFactories[name] = factory
}

func (c *singleChain) RunTask(name string, params json.RawMessage) error {
	if factory, ok := taskFactories[name]; ok {
		if task, err := factory(c, params); err != nil {
			return err
		} else {
			return c._runTask(task, false)
		}
	}
	return errors.NotFoundError.Errorf("UnknownTask(name=%s)", name)
}

func (c *singleChain) _handleTerminateInLock() {
	if c.state != Terminating {
		c.logger.Panicf("InvalidStateForTerminate(state=%s)", c.state.String())
	}
	c._terminate()
	c.state = Terminated
	if c.termWaiter != nil {
		c.termWaiter.Broadcast()
	}
}

func (c *singleChain) _terminate() {
	c.releaseDatabase()
	c.plt.Term()
}

func (c *singleChain) Term() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	switch c.state {
	case Terminated:
		return errors.InvalidStateError.New("AlreadyTerminated")
	case Stopped, Failed, Finished, InitializeFailed:
		c._terminate()
		c.state = Terminated
		return nil
	case Started:
		c.task.Stop()
	}
	c.state = Terminating

	c.termWaiter = sync.NewCond(&c.mtx)
	c.termWaiter.Wait()
	return nil
}

func (c *singleChain) Verify() error {
	return errors.UnsupportedError.New("UnsupportedFeatureVerify")
}

func (c *singleChain) Reset(gs string, height int64, blockHash []byte) error {
	if len(gs) == 0 {
		chainDir := c.cfg.AbsBaseDir()
		const chainGenesisZipFileName = "genesis.zip"
		gs = path.Join(chainDir, chainGenesisZipFileName)
	}
	task := newTaskReset(c, gs, height, blockHash)
	return c._runTask(task, false)
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
	cid := cfg.CID()
	chainLogger := logger.WithFields(log.Fields{
		log.FieldKeyCID: strconv.FormatInt(int64(cid), 16),
	})
	c := &singleChain{
		wallet:    wallet,
		nt:        transport,
		srv:       srv,
		cid:       cid,
		cfg:       *cfg,
		pm:        pm,
		logger:    chainLogger,
		regulator: NewRegulator(chainLogger),
		metricCtx: metric.GetMetricContextByCID(cid),
	}
	return c
}
