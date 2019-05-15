package node

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/server"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/service/eeproxy"
)

type Node struct {
	w   module.Wallet
	nt  module.NetworkTransport
	srv *server.Manager
	pm  eeproxy.Manager
	cfg NodeConfig

	mtx      sync.RWMutex
	//TODO add module.Chain.Channel() then remove channels and change chains map[string] to map[int]
	chains   map[string]module.Chain
	channels map[int]string

	cliSrv *UnixDomainSockHttpServer
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
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
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

func (n *Node) _add(cfg *chain.Config) (module.Chain, error) {
	nid := cfg.NID
	channel := cfg.Channel
	if channel == "" {
		channel = strconv.FormatInt(int64(nid), 16)
	}

	if _, ok := n.channels[nid]; ok {
		return nil, fmt.Errorf("already joined chain nid:%d %v", nid, cfg)
	}

	if _, ok := n.chains[channel]; ok {
		return nil, fmt.Errorf("already joined chain channel:%s %v", channel, cfg)
	}

	c := chain.NewChain(n.w, n.nt, n.srv, n.pm, cfg)
	if err := c.Init(true); err != nil {
		return nil, err
	}
	n.channels[nid] = channel
	n.chains[channel] = c
	return n.chains[channel], nil
}

func (n *Node) _remove(c module.Chain) error {
	if err := c.Term(true); err != nil {
		return err
	}

	chainPath := n.ChainDir(c.NID())
	if err := os.RemoveAll(chainPath); err != nil {
		return fmt.Errorf("fail to remove dir %s err=%+v", chainPath, err)
	}

	delete(n.chains, n.channels[c.NID()])
	delete(n.channels, c.NID())
	metric.ResetMetricViews()
	return nil
}

func (n *Node) ChainDir(nid int) string {
	nodeDir := n.cfg.ResolveAbsolute(n.cfg.BaseDir)
	chainDir := path.Join(nodeDir, strconv.FormatInt(int64(nid), 16))
	return chainDir
}

func (n *Node) _get(nid int) (module.Chain, error) {
	channel, ok := n.channels[nid]
	if !ok {
		return nil, fmt.Errorf("not joined chain %d", nid)
	}
	c, ok := n.chains[channel]
	if !ok {
		return nil, fmt.Errorf("not joined chain %d", nid)
	}
	return c, nil
}

func (n *Node) Start() {
	err := n.nt.Listen()
	if err != nil {
		log.Panicf("FAIL to P2P listen err=%+v", err)
	}

	go n.srv.Start()
	go func() {
		if err := n.pm.Loop(); err != nil {
			log.Panic(err)
		}
	}()

	if err := n.cliSrv.Start(); err != nil {
		log.Panic(err)
	}

}

func (n *Node) Stop() {
	if err := n.nt.Close(); err != nil {
		log.Panicf("FAIL to P2P close err=%+v", err)
	}
	n.srv.Stop()
	if err := n.cliSrv.Stop(); err != nil {
		log.Panic(err)
	}
}

// TODO [TBD] using JoinChainParam struct
func (n *Node) JoinChain(
	nid int,
	seed string,
	role uint,
	dbType string,
	concurrencyLevel int,
	channel string,
	genesis []byte,
) (module.Chain, error) {
	defer n.mtx.Unlock()
	n.mtx.Lock()

	if _, ok := n.channels[nid]; ok {
		return nil, fmt.Errorf("already joined chain nid:%d", nid)
	}

	if channel == "" {
		channel = strconv.FormatInt(int64(nid), 16)
	}

	if _, ok := n.chains[channel]; ok {
		return nil, fmt.Errorf("already joined chain channel:%s", channel)
	}

	gs, err := chain.NewGenesisStorage(genesis)
	if err != nil {
		return nil, err
	}

	chainDir := n.ChainDir(nid)
	log.Println("ChainDir", chainDir)
	if err := os.MkdirAll(chainDir, 0700); err != nil {
		log.Panicf("Fail to create directory %s err=%+v", chainDir, err)
	}

	cfgFile, _ := filepath.Abs(path.Join(chainDir, ChainConfigFileName))

	cfg := &chain.Config{
		NID:            nid,
		DBType:         dbType,
		Channel:        channel,
		SeedAddr:       seed,
		Role:           role,
		GenesisStorage: gs,
		// GenesisDataPath: path.Join(chainDir, "genesis"),
		ConcurrencyLevel: concurrencyLevel,
		FilePath:         cfgFile,
	}

	if err := n.saveChainConfig(cfg, cfgFile); err != nil {
		_ = os.RemoveAll(chainDir)
		return nil, err
	}

	gsFile := path.Join(chainDir, ChainGenesisZipFileName)
	if err := ioutil.WriteFile(gsFile, genesis, 0755); err != nil {
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

func (n *Node) LeaveChain(nid int) error {
	defer n.mtx.Unlock()
	n.mtx.Lock()

	c, err := n._get(nid)
	if err != nil {
		return err
	}
	return n._remove(c)
}

func (n *Node) StartChain(nid int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(nid)
	if err != nil {
		return err
	}
	return c.Start(false)
}

func (n *Node) StopChain(nid int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(nid)
	if err != nil {
		return err
	}
	return c.Stop(false)
}

func (n *Node) ResetChain(nid int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(nid)
	if err != nil {
		return err
	}
	return c.Reset(true)
}

func (n *Node) VerifyChain(nid int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(nid)
	if err != nil {
		return err
	}
	return c.Verify(false)
}

func (n *Node) GetChains() []module.Chain {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	l := make([]module.Chain, 0)
	for _, v := range n.chains {
		l = append(l, v)
	}
	sort.Slice(l, func(i, j int) bool {
		return l[i].NID() > l[j].NID()
	})
	return l
}

func (n *Node) GetChain(nid int) module.Chain {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	return n.chains[n.channels[nid]]
}

func (n *Node) GetChainByChannel(channel string) module.Chain {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	return n.chains[channel]
}

func NewNode(
	w module.Wallet,
	cfg *NodeConfig,
) *Node {
	metric.Initialize(w)

	cfg.FillEmpty(w.Address())

	nt := network.NewTransport(cfg.P2PAddr, w)
	if cfg.P2PListenAddr != "" {
		_ = nt.SetListenAddress(cfg.P2PListenAddr)
	}
	srv := server.NewManager(cfg.RPCAddr, w)

	ee, err := eeproxy.NewPythonEE()
	if err != nil {
		log.Panicf("FAIL to create PythonEE err=%+v", err)
	}
	eeSocket := cfg.ResolveAbsolute(cfg.EESocket)
	pm, err := eeproxy.NewManager("unix", eeSocket, ee)
	if err != nil {
		log.Panicln("FAIL to start EEManager")
	}
	if err := pm.SetInstances(cfg.EEInstances, cfg.EEInstances, cfg.EEInstances); err != nil {
		log.Panicf("FAIL to EEManager.SetInstances err=%+v", err)
	}

	cliSrv := NewUnixDomainSockHttpServer(cfg.ResolveAbsolute(cfg.CliSocket), echo.New())

	n := &Node{
		w:        w,
		nt:       nt,
		srv:      srv,
		pm:       pm,
		cfg:      *cfg,
		chains:   make(map[string]module.Chain),
		channels: make(map[int]string),
		cliSrv:   cliSrv,
	}

	// Load chains
	nodeDir := cfg.ResolveAbsolute(cfg.BaseDir)
	if err := os.MkdirAll(nodeDir, 0700); err != nil {
		log.Panicf("Fail to create directory %s err=%+v", cfg.BaseDir, err)
	}
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

			gsFile := path.Join(chainDir, ChainGenesisZipFileName)
			genesis, err := ioutil.ReadFile(gsFile)
			if err != nil {
				log.Panicf("Fail to read chain genesis zip file %s err=%+v", gsFile, err)
			}
			gs, err := chain.NewGenesisStorage(genesis)
			if err != nil {
				log.Panicf("Fail to parse chain genesis zip file %s err=%+v", gsFile, err)
			}
			ccfg.GenesisStorage = gs
			if _, err := n._add(ccfg); err != nil {
				log.Panicf("Fail to join chain %v err=%+v", ccfg, err)
			}
		}
	}

	RegisterRest(n)
	return n
}
