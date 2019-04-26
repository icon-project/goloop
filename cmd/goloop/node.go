package main

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

const (
	ChainConfigFileName     = "config.json"
	ChainGenesisZipFileName = "genesis.zip"
	DefaultNodeCliSock      = "cli.sock"
)

type NodeConfig struct {
	//static
	CliSocket     string `json:"node_sock"` //relative path
	P2PAddr       string `json:"p2p"`
	P2PListenAddr string `json:"p2p_listen"`
	RPCAddr       string `json:"rpc_addr"`
	EESocket      string `json:"ee_socket"`
	EEInstances   int    `json:"ee_instances"`

	BaseDir  string `json:"node_dir"`
	FilePath string `json:"-"` //absolute path
}

func (c *NodeConfig) ResolveAbsolute(targetPath string) string {
	if filepath.IsAbs(targetPath) {
		return targetPath
	}
	if c.FilePath == "" {
		r, _ := filepath.Abs(targetPath)
		return r
	}
	return filepath.Clean(path.Join(filepath.Dir(c.FilePath), targetPath))
}

func (c *NodeConfig) ResolveRelative(targetPath string) string {
	absPath, _ := filepath.Abs(targetPath)
	base := filepath.Dir(c.FilePath)
	base, _ = filepath.Abs(base)
	r, _ := filepath.Rel(base, absPath)
	return r
}

type Node struct {
	w   module.Wallet
	nt  module.NetworkTransport
	srv *server.Manager
	pm  eeproxy.Manager
	cfg NodeConfig

	mtx sync.RWMutex
	m   map[int]module.Chain

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
	if _, ok := n.m[cfg.NID]; ok {
		return nil, fmt.Errorf("already joined chain %v", cfg)
	}

	c := chain.NewChain(n.w, n.nt, n.srv, n.pm, cfg)
	if err := c.Init(true); err != nil {
		return nil, err
	}
	n.m[cfg.NID] = c
	return n.m[cfg.NID], nil
}

func (n *Node) _remove(c module.Chain) error {
	if err := c.Term(true); err != nil {
		return err
	}

	chainPath := n.ChainDir(c.NID())
	if err := os.RemoveAll(chainPath); err != nil {
		return fmt.Errorf("fail to remove dir %s err=%+v", chainPath, err)
	}

	delete(n.m, c.NID())
	metric.ResetMetricViews()
	return nil
}

func (n *Node) ChainDir(NID int) string {
	nodeDir := n.cfg.ResolveAbsolute(cfg.BaseDir)
	chainDir := path.Join(nodeDir, strconv.FormatInt(int64(NID), 16))
	return chainDir
}

func (n *Node) _get(NID int) (module.Chain, error) {
	c, ok := n.m[NID]
	if !ok {
		return nil, fmt.Errorf("not joined chain %d", NID)
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

//TODO [TBD] using JoinChainParam struct
func (n *Node) JoinChain(
	NID int,
	seed string,
	role uint,
	dbType string,
	concurrencyLevel int,
	genesis []byte,
) (module.Chain, error) {
	defer n.mtx.Unlock()
	n.mtx.Lock()

	if _, ok := n.m[NID]; ok {
		return nil, fmt.Errorf("already joined chain %d", NID)
	}

	gs, err := chain.NewGenesisStorage(genesis)
	if err != nil {
		return nil, err
	}

	chainDir := n.ChainDir(NID)
	log.Println("ChainDir", chainDir)
	if err := os.MkdirAll(chainDir, 0700); err != nil {
		log.Panicf("Fail to create directory %s err=%+v", chainDir, err)
	}

	cfgFile, _ := filepath.Abs(path.Join(chainDir, ChainConfigFileName))
	channel := strconv.FormatInt(int64(NID), 16)
	cfg := &chain.Config{
		NID:            NID,
		DBType:         dbType,
		Channel:        channel,
		SeedAddr:       seed,
		Role:           role,
		GenesisStorage: gs,
		//GenesisDataPath: path.Join(chainDir, "genesis"),
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

func (n *Node) LeaveChain(NID int) error {
	defer n.mtx.Unlock()
	n.mtx.Lock()

	c, err := n._get(NID)
	if err != nil {
		return err
	}
	return n._remove(c)
}

func (n *Node) StartChain(NID int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(NID)
	if err != nil {
		return err
	}
	return c.Start(false)
}

func (n *Node) StopChain(NID int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(NID)
	if err != nil {
		return err
	}
	return c.Stop(false)
}

func (n *Node) ResetChain(NID int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(NID)
	if err != nil {
		return err
	}
	return c.Reset(true)
}

func (n *Node) VerifyChain(NID int) error {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	c, err := n._get(NID)
	if err != nil {
		return err
	}
	return c.Verify(false)
}

func (n *Node) GetChains() []module.Chain {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	l := make([]module.Chain, 0)
	for _, v := range n.m {
		l = append(l, v)
	}
	sort.Slice(l, func(i, j int) bool {
		return l[i].NID() > l[j].NID()
	})
	return l
}

func (n *Node) GetChain(NID int) module.Chain {
	defer n.mtx.RUnlock()
	n.mtx.RLock()

	return n.m[NID]
}

func NewNode(
	w module.Wallet,
	cfg *NodeConfig,
) *Node {
	metric.Initialize(w)

	if cfg.BaseDir == "" {
		cfg.BaseDir = path.Join(".", ".chain", w.Address().String())
	}
	if cfg.CliSocket == "" {
		cfg.CliSocket = path.Join(cfg.BaseDir, DefaultNodeCliSock)
	}
	if cfg.EESocket == "" {
		cfg.EESocket = path.Join(cfg.BaseDir, "ee.sock")
	}

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
		w:      w,
		nt:     nt,
		srv:    srv,
		pm:     pm,
		cfg:    *cfg,
		m:      make(map[int]module.Chain),
		cliSrv: cliSrv,
	}

	//Load chains
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
