package chain

import (
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/chain/gs"
	"github.com/icon-project/goloop/chain/imports"
	"github.com/icon-project/goloop/common"
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

	cid int
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
	DefaultCacheDir    = "cache"
	DefaultTmpDBDir    = "tmp"
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
	StateVerifyStarting
	StateVerifyStarted
	StateVerifyFailed
	StateImportStarting
	StateImportStarted
	StateImportStopping
	StateImportFailed
	StateResetting
	StateResetFailed
	StatePruneStarting
	StatePruneStarted
	StatePruneStopping
	StatePruneFailed
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
	case StateVerifyStarting:
		return "verify Starting"
	case StateVerifyStarted:
		return "verify started"
	case StateVerifyFailed:
		return "verify failed"
	case StateImportStarting:
		return "import starting"
	case StateImportStarted:
		return "import started"
	case StateImportStopping:
		return "import stopping"
	case StateImportFailed:
		return "import failed"
	case StateResetting:
		return "resetting"
	case StateResetFailed:
		return "reset failed"
	case StatePruneStarting:
		return "prune starting"
	case StatePruneStarted:
		return "prune started"
	case StatePruneFailed:
		return "prune failed"
	case StatePruneStopping:
		return "prune stopping"
	default:
		return fmt.Sprintf("unknown %d", s)
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

func (c *singleChain) CID() int {
	return c.cid
}

func (c *singleChain) NetID() int {
	if c.cfg.NIDForP2P {
		return c.NID()
	} else {
		return c.CID()
	}
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

func (c *singleChain) _setStateIf(to State, err error, froms ...State) (State, bool) {
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
		return c.state, false
	}
	prev := c.state
	c.state = to
	c.lastErr = err
	return prev, true
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

func (c *singleChain) _openDatabase(dbDir, dbType string) (db.Database, error) {
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

func (c *singleChain) _prepareDatabase(chainDir string) error {
	DBDir := path.Join(chainDir, DefaultDBDir)
	cdb, err := c._openDatabase(DBDir, c.cfg.DBType)
	if err != nil {
		return err
	}
	if len(c.cfg.NodeCache) == 0 {
		c.cfg.NodeCache = NodeCacheDefault
	}
	var mLevel, fLevel int
	switch c.cfg.NodeCache {
	case NodeCacheNone:
	case NodeCacheSmall:
		mLevel = 5
	case NodeCacheLarge:
		mLevel = 5
		fLevel = 1
	default:
		cdb.Close()
		return errors.Errorf("Unknown cache strategy:%s", c.cfg.NodeCache)
	}
	if mLevel > 0 || fLevel > 0 {
		cacheDir := path.Join(chainDir, DefaultCacheDir)
		c.database = cache.AttachManager(cdb, cacheDir, mLevel, fLevel)
	} else {
		c.database = cdb
	}
	return nil
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

	if err := c._prepareDatabase(chainDir); err != nil {
		return err
	}

	c.vld = consensus.NewCommitVoteSetFromBytes
	c.pd = consensus.DecodePatch
	c.metricCtx = metric.GetMetricContextByCID(c.CID())
	return nil
}

func (c *singleChain) _prepare() error {
	pr := network.PeerRoleFlag(c.cfg.Role)
	c.nm = network.NewManager(c, c.nt, c.cfg.SeedAddr, false, pr.ToRoles()...)
	//TODO [TBD] is service/contract.ContractManager owner of ContractDir ?
	chainDir := c.cfg.AbsBaseDir()
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

type importCallback struct {
	c          *singleChain
	lastHeight int64
}

func (ic *importCallback) OnError(err error) {
	if err := ic.c._transit(StateImportStopping, StateImportStarted); err != nil {
		return
	}
	ic.c._stop()
	log.Errorf("Import failed : %+v\n", err)
	ic.c._setState(StateImportFailed, err)
}

func (ic *importCallback) OnEnd(errCh <-chan error) {
	if err := ic.c._transit(StateStopping, StateImportStarted); err != nil {
		return
	}

	if ic.c.cs != nil {
		ic.c.cs.Term()
		ic.c.cs = nil
	}
	err := <-errCh
	if err != nil {
		ic.c._stop()
		log.Errorf("Import failed : %+v\n", err)
		ic.c._setState(StateImportFailed, err)
	}
	ic.c._stop()
	ic.c._prepare()
	ic.c._setState(StateStopped, nil)
}

func (c *singleChain) _import(src string, height int64) error {
	pr := network.PeerRoleFlag(c.cfg.Role)
	c.nm = network.NewManager(c, c.nt, c.cfg.SeedAddr, false, pr.ToRoles()...)
	//TODO [TBD] is service/contract.ContractManager owner of ContractDir ?
	chainDir := c.cfg.AbsBaseDir()
	ContractDir := path.Join(chainDir, DefaultContractDir)
	var err error
	var ts module.Timestamper
	c.sm, ts, err = imports.NewServiceManagerForImport(c, c.nm, c.pm, ContractDir, src, height, &importCallback{c, height})
	if err != nil {
		return err
	}
	c.bm, err = block.NewManager(c, ts)
	if err != nil {
		return err
	}
	blk, err := c.bm.GetLastBlock()
	if err != nil {
		return err
	}
	if blk.Height() > height {
		return errors.Errorf("chain already have height %d\n", blk.Height())
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
	if err := c._transit(StateStopping, StateStarted, StateStartFailed, StateImportStarted); err != nil {
		return err
	}
	f := func() {
		c._stop()
		c._prepare()
		c._setState(StateStopped, nil)
	}
	return c._execute(sync, f)
}

func (c *singleChain) Import(src string, height int64, sync bool) error {
	if err := c._transit(StateImportStarting, StateStopped); err != nil {
		return err
	}
	log.Infof("Import src:%s height:%d\n", src, height)
	f := func() {
		c._stop()
		err := c._import(src, height)
		s := StateImportStarted
		if err != nil {
			c._stop()
			s = StateImportFailed
			log.Errorf("Import failed %+v\n", err)
		}
		c._setState(s, err)
	}
	return c._execute(sync, f)
}

func (c *singleChain) _exportGenesis(blk module.Block, votes module.CommitVoteSet, gsfile string) (rerr error) {
	os.RemoveAll(gsfile)
	fd, err := os.OpenFile(gsfile, os.O_CREATE|os.O_WRONLY|os.O_EXCL|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	gsw := gs.NewGenesisStorageWriter(fd)
	defer func() {
		gsw.Close()
		fd.Close()
		if rerr != nil {
			os.Remove(gsfile)
		}
	}()
	if err := c.bm.ExportGenesis(blk, gsw); err != nil {
		return errors.Wrap(err, "fail on exporting genesis storage")
	}
	return nil
}

func (c *singleChain) _copyDatabase(dbpath, dbtype string, from, to int64) (rerr error) {
	os.RemoveAll(dbpath)
	dbase, err := c._openDatabase(dbpath, dbtype)
	if err != nil {
		return err
	}
	defer func() {
		dbase.Close()
		if rerr != nil {
			os.RemoveAll(dbpath)
		}
	}()
	return c.bm.ExportBlocks(from, to, dbase)
}

func (c *singleChain) _prune(gsfile, dbtype string, height int64) (rerr error) {
	chainDir := c.cfg.ResolveAbsolute(c.cfg.BaseDir)
	dbpath := path.Join(chainDir, DefaultTmpDBDir)
	gsTmp := gsfile + ".tmp"

	blk, err := c.bm.GetBlockByHeight(height)
	if err != nil {
		return err
	}

	if cid, err := c.sm.GetChainID(blk.Result()); err != nil {
		return errors.InvalidStateError.New("No ChainID is recorded (require Revision 8)")
	} else {
		if cid != int64(c.CID()) {
			return errors.InvalidStateError.Errorf("Invalid chain ID real=%d exp=%d", cid, c.CID())
		}
	}

	nblk, err := c.bm.GetBlockByHeight(height + 1)
	if err != nil {
		return errors.InvalidStateError.Errorf("No next block height=%d", height)
	}

	c.logger.Infof("Export Genesis to=%s from=%d", gsTmp, height)
	if err := c._exportGenesis(blk, nblk.Votes(), gsTmp); err != nil {
		return err
	}
	defer func() {
		if rerr != nil {
			os.Remove(gsTmp)
		}
	}()

	lb, err := c.bm.GetLastBlock()
	if err != nil {
		return err
	}
	targetHeight := lb.Height()
	c.logger.Infof("Copy Database path=%s type=%s from=%d to=%d",
		dbpath, dbtype, height, targetHeight)
	err = c._copyDatabase(dbpath, dbtype, height, targetHeight)
	if err != nil {
		return err
	}
	defer func() {
		if rerr != nil {
			os.RemoveAll(dbpath)
		}
	}()

	c._stop()
	target := path.Join(chainDir, DefaultDBDir)
	c.database.Close()
	c.database = nil

	dbbk := target + ".bk"
	gsbk := gsfile + ".bk"

	c.logger.Infof("Replace DB %s -> %s", dbpath, target)
	os.RemoveAll(dbbk)
	if err := os.Rename(target, dbbk); err != nil {
		return errors.UnknownError.Errorf("file on backup %s to %s",
			target, dbbk)
	}
	defer func() {
		if rerr != nil {
			os.RemoveAll(target)
			os.Rename(dbbk, target)
		} else {
			os.RemoveAll(dbbk)
		}
	}()
	if err := os.Rename(dbpath, target); err != nil {
		return errors.UnknownError.Errorf("fail on rename %s to %s",
			dbpath, target)
	}

	c.logger.Infof("Replace GS %s -> %s", gsTmp, gsfile)
	os.RemoveAll(gsbk)
	if err := os.Rename(gsfile, gsbk); err != nil {
		return errors.UnknownError.Errorf("fail on backup %s to %s",
			gsfile, gsbk)
	}
	defer func() {
		if rerr != nil {
			os.RemoveAll(gsfile)
			os.Rename(gsbk, gsfile)
		} else {
			os.RemoveAll(gsbk)
		}
	}()
	if err := os.Rename(gsTmp, gsfile); err != nil {
		return errors.UnknownError.Errorf("fail to rename %s to %s",
			gsTmp, gsfile)
	}

	// replace genesis
	fd, err := os.Open(gsfile)
	if err != nil {
		return errors.UnknownError.Wrapf(err, "fail to open file=%s", gsfile)
	}
	g, err := gs.NewFromFile(fd)
	if err != nil {
		return errors.UnknownError.Wrapf(err, "fail to parse gs=%s", gsfile)
	}

	c.logger.Infof("Reopen DB %s", chainDir)
	c.cfg.DBType = dbtype
	if err := c._prepareDatabase(chainDir); err != nil {
		c.logger.Panicf("_prepareDatabase() fails err=%+v", err)
	}

	c.cfg.GenesisStorage = g
	c.cfg.Genesis = g.Genesis()
	if err := c._prepare(); err != nil {
		c.logger.Panicf("_prepare() fails err=%+v", err)
	}

	return nil
}

func (c *singleChain) Prune(gsfile, dbtype string, height int64, sync bool) error {
	if s, ok := c._setStateIf(StatePruneStarting, nil, StateStopped); !ok {
		return errors.InvalidStateError.Errorf("InvalidState(state=%s)", s)
	}
	if dbtype == "" {
		dbtype = c.cfg.DBType
	}
	log.Infof("Prune gsfile=%s dbtype=%s height:%d",
		gsfile, dbtype, height)
	f := func() {
		c._setState(StatePruneStarted, nil)
		err := c._prune(gsfile, dbtype, height)
		if err != nil {
			if _, ok := c._setStateIf(StatePruneFailed, err, StatePruneStarted); ok {
				log.Errorf("Prune failed err=%+v", err)
			} else {
				c._setStateIf(StateStopped, nil, StatePruneStopping)
			}
		} else {
			c._setState(StateStopped, nil)
		}
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
	// TODO start verifying operation
	return fmt.Errorf("not implemented")
}

func (c *singleChain) Verify(sync bool) error {
	if err := c._transit(StateVerifyStarting, StateStopped); err != nil {
		return err
	}

	f := func() {
		s := StateVerifyStarted
		err := c._verify()
		if err != nil {
			// TODO need to cleanup on failure.
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
	c.database = nil
	chainDir := c.cfg.AbsBaseDir()
	DBDir := path.Join(chainDir, DefaultDBDir)
	if err := os.RemoveAll(DBDir); err != nil {
		return err
	}
	CacheDir := path.Join(chainDir, DefaultCacheDir)
	if err := os.RemoveAll(CacheDir); err != nil {
		return err
	}

	if err := c._prepareDatabase(chainDir); err != nil {
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
	if err := c._transit(StateResetting, StateStopped); err != nil {
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

func IsNodeCacheOption(s string) bool {
	for _, k := range NodeCacheOptions {
		if k == s {
			return true
		}
	}
	return false
}
