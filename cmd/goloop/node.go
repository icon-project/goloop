package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo"

	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server"
	"github.com/icon-project/goloop/service/eeproxy"
)

const (
	ChainConfigFileName = "config.json"
	ChainGenesisZipFileName = "genesis.zip"
)

type NodeConfig struct {
	NodeDir string `json:"node_dir"`
	CliSocket string `json:"node_sock"`

	//chain.Config
	DBType           string `json:"db_type"`
	ConcurrencyLevel int    `json:"concurrency_level,omitempty"`
}

type Node struct {
	w   module.Wallet
	nt  module.NetworkTransport
	srv *server.Manager
	pm  eeproxy.Manager
	cfg NodeConfig

	mtx sync.RWMutex
	m   map[int]module.Chain

	cliSrv struct {
		srv http.Server
		e *echo.Echo
		l net.Listener
	}
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

	chainPath := path.Join(n.cfg.NodeDir, strconv.FormatInt(int64(c.NID()), 16))
	if err := os.RemoveAll(chainPath); err != nil {
		return fmt.Errorf("fail to remove dir %s err=%+v", chainPath, err)
	}

	delete(n.m, c.NID())
	return nil
}

func (n *Node) _get(NID int) (module.Chain, error) {
	c, ok := n.m[NID]
	if !ok {
		return nil, fmt.Errorf("not joined chain %d", NID)
	}
	return c, nil
}

func (n *Node) Start() {
	go n.srv.Start()

	if err := os.RemoveAll(n.cfg.CliSocket); err != nil {
		log.Panic(err)
	}
	l, err := net.Listen("unix", n.cfg.CliSocket)
	if err != nil {
		log.Panic(err)
	}
	n.cliSrv.l = l
	if err := n.cliSrv.srv.Serve(l); err != nil {
		log.Panic(err)
	}
}

func (n *Node) Stop() {
	n.srv.Stop()
	ctx, cf := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cf()
	if err := n.cliSrv.srv.Shutdown(ctx); err != nil {
		log.Panic(err)
	}
}

func (n *Node) JoinChain(
	NID int,
	seed string,
	role uint,
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

	chainPath := path.Join(n.cfg.NodeDir, strconv.FormatInt(int64(NID), 16))
	if err := os.MkdirAll(chainPath, 0700); err != nil {
		log.Panicf("Fail to create directory %s err=%+v", chainPath, err)
	}

	channel := strconv.FormatInt(int64(NID), 16)
	cfg := &chain.Config{
		NID:            NID,
		Channel:        channel,
		SeedAddr:       seed,
		Role:           role,
		DBType:         n.cfg.DBType,
		DBName:         channel,
		GenesisStorage: gs,
		//GenesisDataPath: path.Join(chainPath, "genesis"),
		ConcurrencyLevel: n.cfg.ConcurrencyLevel,
		ChainDir:         chainPath,
	}

	cfgFile := path.Join(chainPath, ChainConfigFileName)
	if err := n.saveChainConfig(cfg, cfgFile); err != nil {
		_ = os.RemoveAll(chainPath)
		return nil, err
	}

	gsFile := path.Join(chainPath, ChainGenesisZipFileName)
	if err := ioutil.WriteFile(gsFile, genesis, 0755); err != nil {
		_ = os.RemoveAll(chainPath)
		return nil, err
	}

	c, err := n._add(cfg)
	if err != nil {
		_ = os.RemoveAll(chainPath)
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
	nt module.NetworkTransport,
	srv *server.Manager,
	pm eeproxy.Manager,
	cfg *NodeConfig,
) *Node {
	if cfg.NodeDir == "" {
		cfg.NodeDir = path.Join(".", ".chain", w.Address().String())
	}
	if cfg.CliSocket == "" {
		cfg.CliSocket = path.Join(cfg.NodeDir, "cli.sock")
	}

	n := &Node{
		w:   w,
		nt:  nt,
		srv: srv,
		pm:  pm,
		cfg: *cfg,
		m:   make(map[int]module.Chain),
	}
	//Load chains
	if err := os.MkdirAll(cfg.NodeDir, 0700); err != nil {
		log.Panicf("Fail to create directory %s err=%+v", cfg.NodeDir, err)
	}
	fs, err := ioutil.ReadDir(cfg.NodeDir)
	if err != nil {
		log.Panicf("Fail to read directory %s err=%+v", cfg.NodeDir, err)
	}
	for _, f := range fs {
		if f.IsDir() {
			chainPath := path.Join(cfg.NodeDir, f.Name())
			cfgFile := path.Join(chainPath, ChainConfigFileName)
			ccfg, err := n.loadChainConfig(cfgFile)
			if err != nil {
				log.Panicf("Fail to load chain config %s err=%+v", cfgFile, err)
			}
			gsFile := path.Join(chainPath, ChainGenesisZipFileName)
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

	n.cliSrv.e = echo.New()
	n.cliSrv.srv.Handler = n.cliSrv.e
	n.cliSrv.srv.ErrorLog = n.cliSrv.e.StdLogger

	RegisterRest(n)
	return n
}
