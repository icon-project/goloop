package node

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/chain/gs"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/server"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/service/eeproxy"
)

var (
	ErrAlreadyExists = errors.New("already exists")
	ErrNotExists     = errors.New("not exists")
)

type Node struct {
	w    module.Wallet
	nt   module.NetworkTransport
	srv  *server.Manager
	pm   eeproxy.Manager
	rsm  RestoreManager
	cfg  StaticConfig
	rcfg *RuntimeConfig

	logger log.Logger

	mtx sync.RWMutex

	chains   map[string]*Chain
	channels map[int]string

	cliSrv *UnixDomainSockHttpServer
}

type Chain struct {
	module.Chain
	cfg     *chain.Config
	refresh bool
}

func (n *Node) loadChainConfig(chainDir string) (*chain.Config, error) {
	cfgFile := path.Join(chainDir, ChainConfigFileName)
	if st, err := os.Stat(cfgFile); err != nil || !st.Mode().IsRegular() {
		return nil, errors.NotFoundError.Errorf(
			"NoConfigurationFile(name=%s)", cfgFile)
	}
	log.Println("Load channel config ", cfgFile)

	b, err := os.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}
	cfg := &chain.Config{}
	if err = json.Unmarshal(b, cfg); err != nil {
		return nil, err
	}

	cfg.FilePath = cfgFile
	cfg.NIDForP2P = n.cfg.NIDForP2P

	gsFile := path.Join(chainDir, ChainGenesisZipFileName)
	genesis, err := os.ReadFile(gsFile)
	if err != nil {
		return nil, errors.CriticalIOError.Wrapf(err,
			"Fail to read chain genesis zip file %s err=%+v", gsFile, err)
	}

	genesisStorage, err := gs.New(genesis)
	if err != nil {
		return nil, errors.CriticalIOError.Wrapf(err,
			"Fail to parse chain genesis zip file %s err=%+v", gsFile, err)
	}
	cfg.GenesisStorage = genesisStorage

	return cfg, nil
}

func (n *Node) CanAdd(cid, nid int, channel string, overwrite bool) error {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	return n._canAdd(cid, nid, channel, overwrite)
}

func (n *Node) _canAdd(cid, nid int, channel string, overwrite bool) error {
	var cidSame, ncSame, chnSame, sidSame *Chain
	sid, _ := cidOfSelector(channel)
	for _, c := range n.chains {
		if c.CID() == cid {
			cidSame = c
		}
		if n.cfg.NIDForP2P && nid == c.NID() {
			ncSame = c
		}
		if c.Channel() == channel {
			chnSame = c
		}
		if sid == c.CID() {
			sidSame = c
		}
	}
	if cidSame == nil {
		if ncSame != nil {
			return errors.IllegalArgumentError.Errorf(
				"NetIDConflicts(cid=%#x)", ncSame.CID())
		}
		if chnSame != nil {
			return errors.IllegalArgumentError.Errorf(
				"ChannelConflicts(cid=%#x)", chnSame.CID())
		}
		if sidSame != nil {
			return errors.IllegalArgumentError.Errorf(
				"ChannelConflicts(cid=%#x)", sidSame.CID())
		}
	} else if overwrite {
		if ncSame != nil && ncSame != cidSame {
			return errors.IllegalArgumentError.Errorf(
				"NetIDConflicts(cid=%#x)", ncSame.CID())
		}
		if chnSame != nil && chnSame != cidSame {
			return errors.IllegalArgumentError.Errorf(
				"ChannelConflicts(cid=%#x)", chnSame.CID())
		}
		if sidSame != nil && sidSame != cidSame {
			return errors.IllegalArgumentError.Errorf(
				"ChannelConflicts(cid=%#x)", sidSame.CID())
		}
	} else {
		return errors.IllegalArgumentError.Errorf(
			"CIDConflicts(channel=%s)", cidSame.Channel())
	}
	return nil
}

func (n *Node) restoreChain(tmpDir string, overwrite bool) (ret error) {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	cfg, err := n.loadChainConfig(tmpDir)
	if err != nil {
		return err
	}

	if err := n._canAdd(cfg.CID(), cfg.NID, cfg.Channel, overwrite); err != nil {
		return err
	}

	channel, exist := n.channels[cfg.CID()]
	if exist {
		c := n.chains[channel]
		if err := n._remove(c); err != nil {
			return err
		}
		defer func() {
			if ret != nil {
				n._add(c.cfg)
			}
		}()

		backupDir := path.Join(n.cfg.AbsBaseDir(), RestoreDirectoryPrefix)
		if err := os.RemoveAll(backupDir); err != nil {
			return err
		}
		n.logger.Debugf("Backup existing chain (%s -> %s)",
			c.cfg.AbsBaseDir(), backupDir)
		if err := os.Rename(c.cfg.AbsBaseDir(), backupDir); err != nil {
			return err
		}
		defer func() {
			if ret != nil {
				if err := os.Rename(backupDir, c.cfg.AbsBaseDir()); err != nil {
					n.logger.Panic(err)
				}
			} else {
				go os.RemoveAll(backupDir)
			}
		}()
	}
	chainDir, err := n._renameChainDir(tmpDir, cfg.CID())
	if err != nil {
		n.logger.Debugf("Fail to rename chain directory tmp=%s, err=%+v", tmpDir, err)
		return err
	}
	defer func() {
		if ret != nil {
			os.Rename(chainDir, tmpDir)
		}
	}()

	if cfg, err = n.loadChainConfig(chainDir); err != nil {
		return err
	}
	if _, err := n._add(cfg); err != nil {
		return err
	}
	return nil
}

func (n *Node) _add(cfg *chain.Config) (module.Chain, error) {
	nid := cfg.NID
	cid := cfg.CID()
	channel := cfg.GetChannel()
	nc := network.ChannelOfNetID(cfg.NetID())

	if err := n._canAdd(cid, nid, channel, false); err != nil {
		return nil, err
	}

	if err := n.nt.SetSecureSuites(nc, cfg.SecureSuites); err != nil {
		return nil, err
	}
	if err := n.nt.SetSecureAeads(nc, cfg.SecureAeads); err != nil {
		return nil, err
	}

	c := &Chain{chain.NewChain(n.w, n.nt, n.srv, n.pm, n.logger, cfg), cfg, false}
	if err := c.Init(); err != nil {
		return nil, err
	}
	n.channels[cid] = channel
	n.chains[channel] = c
	return c, nil
}

func (n *Node) _remove(c module.Chain) error {
	if err := c.Term(); err != nil {
		return err
	}

	delete(n.chains, n.channels[c.CID()])
	delete(n.channels, c.CID())
	metric.RemoveMetricContextByCID(c.CID())
	metric.ResetMetricViews()
	return nil
}

func (n *Node) _refresh(c *Chain) (*Chain, error) {
	if err := n._remove(c); err != nil {
		return nil, errors.Wrapf(err, "fail to refresh on remove")
	}
	if nc, err := n._add(c.cfg); err != nil {
		err = errors.Wrapf(err, "fail to recreate on add")
		if cfg, lerr := n.loadChainConfig(c.cfg.AbsBaseDir()); lerr != nil {
			err = errors.Wrapf(err, "fail to loadChainConfig on rollback err=%+v", lerr)
			return nil, err
		} else {
			if _, aerr := n._add(cfg); aerr != nil {
				err = errors.Wrapf(err, "fail to add on rollback err=%+v", aerr)
				return nil, err
			}
		}
		return nil, errors.Wrapf(err, "fail to refresh on add")
	} else {
		return nc.(*Chain), nil
	}
}

func (n *Node) _renameChainDir(dir string, cid int) (string, error) {
	nodeDir := n.cfg.AbsBaseDir()
	chainBase := path.Join(nodeDir, strconv.FormatInt(int64(cid), 16))
	chainDir := chainBase
	for idx := 0; idx < 1000; idx++ {
		if _, err := os.Stat(chainDir); os.IsNotExist(err) {
			if err := os.Rename(dir, chainDir); err != nil {
				return "", err
			} else {
				return chainDir, nil
			}
		} else {
			chainDir = fmt.Sprintf("%s.%d", chainBase, idx)
		}
	}
	return "", errors.CriticalIOError.New("Fail to rename chain directory")
}

func (n *Node) _mkChainDir(cid int) (string, error) {
	nodeDir := n.cfg.AbsBaseDir()
	chainDir := path.Join(nodeDir, strconv.FormatInt(int64(cid), 16))
	if _, err := os.Stat(chainDir); os.IsNotExist(err) {
		if err := os.Mkdir(chainDir, 0700); err == nil {
			return chainDir, nil
		}
	}
	return os.MkdirTemp(nodeDir, strconv.FormatInt(int64(cid), 16)+"_")
}

func (n *Node) _get(cid int) (*Chain, error) {
	channel, ok := n.channels[cid]
	if !ok {
		return nil, errors.Wrapf(ErrNotExists, "Network(cid=%#x) not exists", cid)
	}
	c, ok := n.chains[channel]
	if !ok {
		return nil, errors.Wrapf(ErrNotExists, "Network(channel=%s) not exists", channel)
	}
	return c, nil
}

func (n *Node) Start() {
	err := n.nt.Listen()
	if err != nil {
		log.Panicf("fail to P2P listen err=%+v", err)
	}

	go func() {
		for channel, chain := range n.chains {
			if chain.cfg.AutoStart {
				if err := chain.Start(); err != nil {
					n.logger.Warnf("fail to start chain channel=%s err=%+v",
						channel, err)
				}
			}
		}
	}()

	go func() {
		if err := n.srv.Start(); err != nil {
			log.Panicf("fail to server close err=%+v", err)
		}
	}()

	if err := n.cliSrv.Start(); err != nil {
		log.Panicf("fail to cli server start err=%+v", err)
	}

}

func (n *Node) Stop() {
	if err := n.nt.Close(); err != nil {
		log.Panicf("fail to P2P close err=%+v", err)
	}
	if err := n.srv.Stop(); err != nil {
		log.Panicf("fail to server close err=%+v", err)
	}
	if err := n.cliSrv.Stop(); err != nil {
		log.Panicf("fail to cli server close err=%+v", err)
	}
}

// TODO [TBD] using JoinChainParam struct
func (n *Node) JoinChain(
	p *ChainConfig,
	genesis []byte,
) (module.Chain, error) {
	defer n.mtx.Unlock()
	n.mtx.Lock()

	genesisStorage, err := gs.New(genesis)
	if err != nil {
		return nil, errors.Wrap(err, "fail to get genesis storage")
	}

	cid, err := genesisStorage.CID()
	if err != nil {
		return nil, errors.Wrap(err, "fail to get CID for genesis")
	}

	nid, err := genesisStorage.NID()
	if err != nil {
		return nil, errors.Wrap(err, "fail to get NID for genesis")
	}

	channel := chain.GetChannel(p.Channel, nid)

	if err := n._canAdd(cid, nid, channel, false); err != nil {
		return nil, err
	}

	chainDir, err := n._mkChainDir(cid)
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to create directory for cid=%d", cid)
	}
	log.Println("ChainDir", chainDir)

	cfgFile, _ := filepath.Abs(path.Join(chainDir, ChainConfigFileName))

	cfg := &chain.Config{
		NID:              nid,
		DBType:           p.DBType,
		Platform:         p.Platform,
		Channel:          channel,
		SecureSuites:     p.SecureSuites,
		SecureAeads:      p.SecureAeads,
		SeedAddr:         p.SeedAddr,
		Role:             p.Role,
		GenesisStorage:   genesisStorage,
		ConcurrencyLevel: p.ConcurrencyLevel,
		NormalTxPoolSize: p.NormalTxPoolSize,
		PatchTxPoolSize:  p.PatchTxPoolSize,
		MaxBlockTxBytes:  p.MaxBlockTxBytes,
		NodeCache:        p.NodeCache,
		DefWaitTimeout:   p.DefWaitTimeout,
		MaxWaitTimeout:   p.MaxWaitTimeout,
		TxTimeout:        p.TxTimeout,
		AutoStart:        p.AutoStart,
		FilePath:         cfgFile,
		NIDForP2P:        n.cfg.NIDForP2P,
		ChildrenLimit:    p.ChildrenLimit,
		NephewsLimit:     p.NephewsLimit,
		ValidateTxOnSend: p.ValidateTxOnSend,
	}

	if err := cfg.Save(); err != nil {
		_ = os.RemoveAll(chainDir)
		return nil, err
	}

	gsFile := path.Join(chainDir, ChainGenesisZipFileName)
	if err := os.WriteFile(gsFile, genesis, 0644); err != nil {
		_ = os.RemoveAll(chainDir)
		return nil, err
	}

	c, err := n._add(cfg)
	if err != nil {
		_ = os.RemoveAll(chainDir)
		return nil, err
	}
	return c, nil
}

func (n *Node) LeaveChain(cid int) error {
	defer n.mtx.Unlock()
	n.mtx.Lock()

	c, err := n._get(cid)
	if err != nil {
		return err
	}
	err = n._remove(c)
	if err != nil {
		return err
	}

	chainPath := c.cfg.AbsBaseDir()
	if err := os.RemoveAll(chainPath); err != nil {
		return errors.Wrapf(err, "fail to remove dir %s", chainPath)
	}
	return nil
}

func (n *Node) StartChain(cid int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(cid)
	if err != nil {
		return err
	}
	if c.refresh {
		if c, err = n._refresh(c); err != nil {
			return err
		}
	}
	return c.Start()
}

func (n *Node) StopChain(cid int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(cid)
	if err != nil {
		return err
	}
	return c.Stop()
}

func (n *Node) ResetChain(cid int, height int64, blockHash []byte) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(cid)
	if err != nil {
		return err
	}
	chainDir := c.cfg.AbsBaseDir()
	gs := path.Join(chainDir, ChainGenesisZipFileName)
	return c.Reset(gs, height, blockHash)
}

func (n *Node) VerifyChain(cid int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(cid)
	if err != nil {
		return err
	}
	return c.Verify()
}

func (n *Node) ImportChain(cid int, s string, height int64) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(cid)
	if err != nil {
		return err
	}
	return c.Import(s, height)
}

func (n *Node) PruneChain(cid int, dbt string, height int64) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(cid)
	if err != nil {
		return err
	}
	chainDir := c.cfg.AbsBaseDir()
	gs := path.Join(chainDir, ChainGenesisZipFileName)
	return c.Prune(gs, dbt, height)
}

func (n *Node) BackupChain(cid int, manual bool) (string, error) {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(cid)
	if err != nil {
		return "", err
	}

	if manual {
		return "manual", c.Backup("", nil)
	}
	backupDir := n.cfg.ResolveAbsolute(n.cfg.BackupDir)
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", errors.InvalidStateError.Wrapf(err,
			"Fail to make backup directory=%s", backupDir)
	}
	now := time.Now()
	name := fmt.Sprintf("%#x_%#x_%s_%s.zip", c.CID(), c.NID(), c.Channel(),
		now.Format("20060102-150405"))
	file := path.Join(backupDir, name)
	return name, c.Backup(file, []string{ChainGenesisZipFileName, ChainConfigFileName})
}

type BackupInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	chain.BackupInfo
}

func (n *Node) GetBackups() ([]BackupInfo, error) {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	backupDir := n.cfg.ResolveAbsolute(n.cfg.BackupDir)
	fis, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}
	infos := make([]BackupInfo, 0, len(fis))
	for _, fi := range fis {
		if fi.Type().IsRegular() {
			if strings.HasPrefix(fi.Name(), chain.TemporalBackupFile) {
				continue
			}
			info, err := chain.GetBackupInfoOf(path.Join(backupDir, fi.Name()))
			if err != nil {
				continue
			}
			state, err := fi.Info()
			if err != nil {
				continue
			}
			infos = append(infos, BackupInfo{
				Name:       fi.Name(),
				Size:       state.Size(),
				BackupInfo: *info,
			})
		}
	}
	return infos, nil
}

type RestoreView struct {
	State     string `json:"state"`
	Name      string `json:"name,omitempty"`
	Overwrite bool   `json:"overwrite,omitempty"`
	Error     string `json:"error,omitempty"`
}

// StartRestore start to restore chain.
func (n *Node) StartRestore(name string, overwrite bool) (ret error) {
	baseDir, backupDir := func() (string, string) {
		n.mtx.Lock()
		defer n.mtx.Unlock()

		return n.cfg.ResolveAbsolute(n.cfg.BaseDir),
			n.cfg.ResolveAbsolute(n.cfg.BackupDir)
	}()

	backupFile := path.Join(backupDir, name)

	return n.rsm.Start(n, backupFile, baseDir, overwrite)
}

// GetRestore returns state of latest restore operations.
func (n *Node) GetRestore() *RestoreView {
	status := n.rsm.GetStatus()
	if status == nil {
		return &RestoreView{
			State: "stopped",
		}
	}
	return &RestoreView{
		State:     status.State,
		Name:      path.Base(status.File),
		Overwrite: status.Overwrite,
		Error:     errors.ToString(status.Error),
	}
}

// StopRestore stops last restore operation.
// If there is no ongoing restore,then it clears already finished job.
func (n *Node) StopRestore() error {
	return n.rsm.Stop()
}

func (n *Node) ConfigureChain(cid int, key string, value string) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(cid)
	if err != nil {
		return err
	}

	hit := false
	refreshNow := false
	if c.IsStarted() {
		switch key {
		case "seedAddress":
			c.cfg.SeedAddr = value
			c.NetworkManager().SetTrustSeeds(c.cfg.SeedAddr)
		case "role":
			if uintVal, err := strconv.ParseUint(value, 0, 32); err != nil {
				return errors.Wrapf(err, "invalid value type")
			} else {
				c.cfg.Role = uint(uintVal)
			}
			pr := network.PeerRoleFlag(c.cfg.Role)
			c.NetworkManager().SetInitialRoles(pr.ToRoles()...)
		case "autoStart":
			if as, err := strconv.ParseBool(value); err != nil {
				return err
			} else {
				c.cfg.AutoStart = as
			}
		default:
			return errors.ErrInvalidState
		}
		hit = true
	}
	if c.IsStopped() {
		switch key {
		case "concurrencyLevel":
			if intVal, err := strconv.Atoi(value); err != nil {
				return errors.Wrapf(err, "invalid value type")
			} else {
				c.cfg.ConcurrencyLevel = intVal
			}
		case "normalTxPool":
			if intVal, err := strconv.Atoi(value); err != nil {
				return errors.Wrapf(err, "invalid value type")
			} else {
				c.cfg.NormalTxPoolSize = intVal
			}
		case "patchTxPool":
			if intVal, err := strconv.Atoi(value); err != nil {
				return errors.Wrapf(err, "invalid value type")
			} else {
				c.cfg.PatchTxPoolSize = intVal
			}
		case "maxBlockTxBytes":
			if intVal, err := strconv.Atoi(value); err != nil {
				return errors.Wrapf(err, "invalid value type")
			} else {
				c.cfg.MaxBlockTxBytes = intVal
			}
		case "nodeCache":
			if !chain.IsNodeCacheOption(value) {
				return errors.Errorf("InvalidNodeCacheOption(%s)", value)
			}
			c.cfg.NodeCache = value
		case "defaultWaitTimeout":
			if intVal, err := strconv.ParseInt(value, 0, 64); err != nil {
				return errors.Wrapf(err, "invalid value type")
			} else {
				c.cfg.DefWaitTimeout = intVal
			}
		case "maxWaitTimeout":
			if intVal, err := strconv.ParseInt(value, 0, 64); err != nil {
				return errors.Wrapf(err, "invalid value type")
			} else {
				c.cfg.MaxWaitTimeout = intVal
			}
		case "txTimeout":
			if intVal, err := strconv.ParseInt(value, 0, 64); err != nil {
				return errors.Wrapf(err, "invalid value type")
			} else {
				c.cfg.TxTimeout = intVal
			}
		case "channel":
			if err := n._canAdd(c.CID(), c.NID(), value, true); err != nil {
				return err
			}
			c.cfg.Channel = value
			refreshNow = true
		case "secureSuites":
			nc := network.ChannelOfNetID(c.cfg.NetID())
			if err := n.nt.SetSecureSuites(nc, value); err != nil {
				return err
			}
			c.cfg.SecureSuites = value
		case "secureAeads":
			nc := network.ChannelOfNetID(c.cfg.NetID())
			if err := n.nt.SetSecureAeads(nc, value); err != nil {
				return err
			}
			c.cfg.SecureAeads = value
		case "seedAddress":
			c.cfg.SeedAddr = value
		case "role":
			if uintVal, err := strconv.ParseUint(value, 0, 32); err != nil {
				return errors.Wrapf(err, "invalid value type")
			} else {
				c.cfg.Role = uint(uintVal)
			}
		case "autoStart":
			if as, err := strconv.ParseBool(value); err != nil {
				return err
			} else {
				c.cfg.AutoStart = as
			}
		case "childrenLimit":
			if intVal, err := strconv.Atoi(value); err != nil {
				return errors.Wrapf(err, "invalid value type")
			} else {
				c.cfg.ChildrenLimit = &intVal
			}
		case "nephewsLimit":
			if intVal, err := strconv.Atoi(value); err != nil {
				return errors.Wrapf(err, "invalid value type")
			} else {
				c.cfg.NephewsLimit = &intVal
			}
		case "validateTxOnSend":
			if bc, err := strconv.ParseBool(value); err != nil {
				return errors.Wrapf(err, "InvalidValueType(exp=bool,val=%s)", value)
			} else {
				c.cfg.ValidateTxOnSend = bc
			}
		default:
			return errors.Errorf("not found key %s", key)
		}
		hit = true
	}

	if hit {
		if refreshNow {
			if c, err = n._refresh(c); err != nil {
				return err
			}
		} else {
			c.refresh = true
		}
		if err = c.cfg.Save(); err != nil {
			return err
		}
		return nil
	} else {
		return errors.ErrInvalidState
	}
}

func (n *Node) RunChainTask(cid int, task string, params json.RawMessage) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(cid)
	if err != nil {
		return err
	}
	return c.RunTask(task, params)
}

func (n *Node) GetChains() []*Chain {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	l := make([]*Chain, 0)
	for _, v := range n.chains {
		l = append(l, v)
	}
	sort.Slice(l, func(i, j int) bool {
		return l[i].CID() > l[j].CID()
	})
	return l
}

func (n *Node) GetChain(cid int) *Chain {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	return n.chains[n.channels[cid]]
}

func (n *Node) GetChainByChannel(channel string) *Chain {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	return n.chains[channel]
}

func cidOfSelector(s string) (int, bool) {
	if cid, err := strconv.ParseInt(s, 0, 32); err == nil {
		return int(cid), true
	}
	return -1, false
}

func (n *Node) GetChainBySelector(s string) *Chain {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	if cid, ok := cidOfSelector(s); ok {
		if channel, ok := n.channels[int(cid)]; ok {
			s = channel
		}
	}
	return n.chains[s]
}

func (n *Node) Configure(key string, value string) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	switch key {
	case "eeInstances":
		if intVal, err := strconv.Atoi(value); err != nil {
			return errors.Wrapf(err, "invalid value type")
		} else {
			n.rcfg.EEInstances = intVal
		}
		if err := n.pm.SetInstances(n.rcfg.EEInstances, n.rcfg.EEInstances, n.rcfg.EEInstances); err != nil {
			return err
		}
	case "rpcDefaultChannel":
		n.rcfg.RPCDefaultChannel = value
		n.srv.SetDefaultChannel(n.rcfg.RPCDefaultChannel)
	case "rpcIncludeDebug":
		if boolVal, err := strconv.ParseBool(value); err != nil {
			return errors.Wrapf(err, "invalid value type")
		} else {
			n.rcfg.RPCIncludeDebug = boolVal
		}
		n.srv.SetIncludeDebug(n.rcfg.RPCIncludeDebug)
	case "rpcRosetta":
		if boolVal, err := strconv.ParseBool(value); err != nil {
			return errors.Wrapf(err, "invalid value type")
		} else {
			n.rcfg.RPCRosetta = boolVal
		}
		n.srv.SetRosetta(n.rcfg.RPCRosetta)
	case "disableRPC":
		if boolVal, err := strconv.ParseBool(value); err != nil {
			return errors.Wrapf(err, "invalid value type")
		} else {
			n.rcfg.DisableRPC = boolVal
		}
		n.srv.SetDisableRPC(n.rcfg.DisableRPC)
	case "rpcBatchLimit":
		if intVal, err := strconv.Atoi(value); err != nil {
			return errors.Wrapf(err, "invalid value type")
		} else {
			n.rcfg.RPCBatchLimit = intVal
		}
		n.srv.SetBatchLimit(n.rcfg.RPCBatchLimit)
	case "wsMaxSession":
		if intVal, err := strconv.Atoi(value); err != nil {
			return errors.Wrapf(err, "invalid value type")
		} else {
			n.rcfg.WSMaxSession = intVal
		}
		n.srv.SetWSMaxSession(n.rcfg.WSMaxSession)
	default:
		return errors.Errorf("not found key")
	}
	if err := n.rcfg.save(); err != nil {
		return err
	}
	return nil
}

func NewNode(
	w module.Wallet,
	cfg *StaticConfig,
	l log.Logger,
) *Node {
	metric.Initialize(w)

	cfg.FillEmpty(w.Address())
	nodeDir := cfg.ResolveAbsolute(cfg.BaseDir)
	if err := os.MkdirAll(nodeDir, 0700); err != nil {
		log.Panicf("Fail to create directory %s err=%+v", cfg.BaseDir, err)
	}
	log.Println("NodeDir :", nodeDir)
	rcfg, err := loadRuntimeConfig(nodeDir)
	if err != nil {
		log.Panicf("fail to load runtime config err=%+v", err)
	}

	nt := network.NewTransport(cfg.P2PAddr, w, l)
	if cfg.P2PListenAddr != "" {
		_ = nt.SetListenAddress(cfg.P2PListenAddr)
	}
	config := &server.Config{
		ServerAddress:         cfg.RPCAddr,
		JSONRPCDump:           cfg.RPCDump,
		JSONRPCIncludeDebug:   rcfg.RPCIncludeDebug,
		JSONRPCRosetta:        rcfg.RPCRosetta,
		DisableRPC:            rcfg.DisableRPC,
		JSONRPCDefaultChannel: rcfg.RPCDefaultChannel,
		JSONRPCBatchLimit:     rcfg.RPCBatchLimit,
		WSMaxSession:          rcfg.WSMaxSession,
	}
	srv := server.NewManager(config, w, l)

	ee, err := eeproxy.AllocEngines(l, strings.Split(cfg.Engines, ",")...)
	if err != nil {
		log.Panicf("fail to create engines err=%+v", err)
	}
	eeSocket := cfg.ResolveAbsolute(cfg.EESocket)
	pm, err := eeproxy.NewManager("unix", eeSocket, l, ee...)
	if err != nil {
		log.Panicf("fail to start EEManager err=%+v", err)
	}

	if err := pm.SetInstances(rcfg.EEInstances, rcfg.EEInstances, rcfg.EEInstances); err != nil {
		log.Panicf("fail to EEManager.SetInstances err=%+v", err)
	}
	go func() {
		if err := pm.Loop(); err != nil {
			log.Panic(err)
		}
	}()

	cliSrv := NewUnixDomainSockHttpServer(cfg.ResolveAbsolute(cfg.CliSocket), nil)
	cliSrv.e.Logger.SetOutput(l.WriterLevel(log.DebugLevel))

	n := &Node{
		w:        w,
		nt:       nt,
		srv:      srv,
		pm:       pm,
		logger:   l,
		cfg:      *cfg,
		rcfg:     rcfg,
		chains:   make(map[string]*Chain),
		channels: make(map[int]string),
		cliSrv:   cliSrv,
	}

	// Load chains
	fs, err := os.ReadDir(nodeDir)
	if err != nil {
		log.Panicf("Fail to read directory %s err=%+v", cfg.BaseDir, err)
	}
	for _, f := range fs {
		if f.IsDir() {
			chainDir := path.Join(nodeDir, f.Name())

			// remove temporal directories for restore
			if strings.HasPrefix(f.Name(), RestoreDirectoryPrefix) {
				log.Infof("Remove temporal directory: %s", chainDir)
				if err := os.RemoveAll(chainDir); err != nil {
					log.Warnf("Fail to remove directory: %s err=%+v", chainDir, err)
				}
				continue
			}

			chainCfg, err := n.loadChainConfig(chainDir)
			if err != nil {
				if errors.NotFoundError.Equals(err) {
					continue
				}
				log.Panicf("Fail to load chain config chainDir=%s err=%+v", chainDir, err)
			}
			if _, err := n._add(chainCfg); err != nil {
				log.Panicf("Fail to join chain %v err=%+v", chainCfg, err)
			}
		}
	}

	RegisterRest(n)
	return n
}
