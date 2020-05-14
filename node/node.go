package node

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

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

func (n *Node) loadChainConfig(filename string) (*chain.Config, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	cfg := &chain.Config{}
	if err = json.Unmarshal(b, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (n *Node) saveChainConfig(cfg *chain.Config, filename string) error {
	f, err := os.OpenFile(filename,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&cfg); err != nil {
		return err
	}
	return nil
}

func (n *Node) _checkNetID(nid int) error {
	if n.cfg.NIDForP2P {
		for _, c := range n.chains {
			if c.NetID() == nid {
				return errors.Wrapf(ErrAlreadyExists, "P2PNetworkConflict(channel=%d)", nid)
			}
		}
	}
	return nil
}

func (n *Node) _add(cfg *chain.Config) (module.Chain, error) {
	nid := cfg.NID
	cid := cfg.CID()
	channel := cfg.Channel
	if channel == "" {
		channel = strconv.FormatInt(int64(nid), 16)
	}

	if _, ok := n.channels[cid]; ok {
		return nil, errors.Wrapf(ErrAlreadyExists, "Network(cid=%#x) already exists", cid)
	}

	if _, ok := n.chains[channel]; ok {
		return nil, errors.Wrapf(ErrAlreadyExists, "Network(channel=%s) already exists", channel)
	}

	if err := n._checkNetID(nid); err != nil {
		return nil, err
	}

	if err := n.nt.SetSecureSuites(channel, cfg.SecureSuites); err != nil {
		return nil, err
	}
	if err := n.nt.SetSecureAeads(channel, cfg.SecureAeads); err != nil {
		return nil, err
	}

	c := &Chain{chain.NewChain(n.w, n.nt, n.srv, n.pm, n.logger, cfg), cfg, false}
	if err := c.Init(true); err != nil {
		return nil, err
	}
	n.channels[cid] = channel
	n.chains[channel] = c
	return c, nil
}

func (n *Node) _remove(c module.Chain) error {
	if err := c.Term(true); err != nil {
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
		if cfg, lerr := n.loadChainConfig(c.cfg.FilePath); lerr != nil {
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

func (n *Node) ChainDir(cid int) string {
	nodeDir := n.cfg.ResolveAbsolute(n.cfg.BaseDir)
	chainDir := path.Join(nodeDir, strconv.FormatInt(int64(cid), 16))
	return chainDir
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

	if _, ok := n.channels[cid]; ok {
		return nil, errors.Wrapf(ErrAlreadyExists, "Network(id=%#x) already exists", nid)
	}

	channel := p.Channel
	if channel == "" {
		channel = strconv.FormatInt(int64(nid), 16)
	}

	if _, ok := n.chains[channel]; ok {
		return nil, errors.Wrapf(ErrAlreadyExists, "Network(channel=%s) already exists", channel)
	}

	if err := n._checkNetID(nid); err != nil {
		return nil, err
	}

	chainDir := n.ChainDir(cid)
	log.Println("ChainDir", chainDir)
	if err := os.MkdirAll(chainDir, 0700); err != nil {
		log.Panicf("Fail to create directory %s err=%+v", chainDir, err)
	}

	cfgFile, _ := filepath.Abs(path.Join(chainDir, ChainConfigFileName))

	cfg := &chain.Config{
		NID:              nid,
		DBType:           p.DBType,
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
		FilePath:         cfgFile,
		NIDForP2P:        n.cfg.NIDForP2P,
	}

	if err := n.saveChainConfig(cfg, cfgFile); err != nil {
		_ = os.RemoveAll(chainDir)
		return nil, err
	}

	gsFile := path.Join(chainDir, ChainGenesisZipFileName)
	if err := ioutil.WriteFile(gsFile, genesis, 0644); err != nil {
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
	return c.Start(false)
}

func (n *Node) StopChain(cid int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(cid)
	if err != nil {
		return err
	}
	return c.Stop(false)
}

func (n *Node) ResetChain(cid int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(cid)
	if err != nil {
		return err
	}
	return c.Reset(true)
}

func (n *Node) VerifyChain(cid int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(cid)
	if err != nil {
		return err
	}
	return c.Verify(true)
}

func (n *Node) ImportChain(cid int, s string, height int64) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(cid)
	if err != nil {
		return err
	}
	return c.Import(s, height, true)
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
	if c.State() == chain.StateStarted.String() {
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
		default:
			return errors.ErrInvalidState
		}
		hit = true
	}
	if c.State() == chain.StateStopped.String() {
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
		case "channel":
			if _, ok := n.chains[value]; ok {
				return errors.Wrapf(ErrAlreadyExists, "Network(channel=%s) already exists", value)
			}
			c.cfg.Channel = value
			refreshNow = true
		case "secureSuites":
			if err := n.nt.SetSecureSuites(c.cfg.Channel, value); err != nil {
				return err
			}
			c.cfg.SecureSuites = value
		case "secureAeads":
			if err := n.nt.SetSecureAeads(c.cfg.Channel, value); err != nil {
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
		if err = n.saveChainConfig(c.cfg, c.cfg.FilePath); err != nil {
			return err
		}
		return nil
	} else {
		return errors.ErrInvalidState
	}
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

func (n *Node) GetChain(nid int) *Chain {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	return n.chains[n.channels[nid]]
}

func (n *Node) GetChainByChannel(channel string) *Chain {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	return n.chains[channel]
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
	srv := server.NewManager(cfg.RPCAddr, cfg.RPCDump, rcfg.RPCIncludeDebug, rcfg.RPCDefaultChannel, w, l)

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
	fs, err := ioutil.ReadDir(nodeDir)
	if err != nil {
		log.Panicf("Fail to read directory %s err=%+v", cfg.BaseDir, err)
	}
	for _, f := range fs {
		if f.IsDir() {
			chainDir := path.Join(nodeDir, f.Name())
			log.Println("Load from ChainDir", chainDir)
			cfgFile := path.Join(chainDir, ChainConfigFileName)
			ccfg, err := n.loadChainConfig(cfgFile)
			if err != nil {
				log.Panicf("Fail to load chain config %s err=%+v", cfgFile, err)
			}
			ccfg.FilePath = cfgFile
			ccfg.NIDForP2P = n.cfg.NIDForP2P

			gsFile := path.Join(chainDir, ChainGenesisZipFileName)
			genesis, err := ioutil.ReadFile(gsFile)
			if err != nil {
				log.Panicf("Fail to read chain genesis zip file %s err=%+v", gsFile, err)
			}
			genesisStorage, err := gs.New(genesis)
			if err != nil {
				log.Panicf("Fail to parse chain genesis zip file %s err=%+v", gsFile, err)
			}
			ccfg.GenesisStorage = genesisStorage
			if _, err := n._add(ccfg); err != nil {
				log.Panicf("Fail to join chain %v err=%+v", ccfg, err)
			}
		}
	}

	RegisterRest(n)
	return n
}
