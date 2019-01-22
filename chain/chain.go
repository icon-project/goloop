package chain

import (
	"encoding/json"
	"log"

	"github.com/icon-project/goloop/service/eeproxy"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/service"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/rpc"
)

type Config struct {
	NID      int             `json:"nid"`
	Channel  string          `json:"channel"`
	RPCAddr  string          `json:"rpc_addr"`
	SeedAddr string          `json:"seed_addr"`
	Role     uint            `json:"role"`
	Genesis  json.RawMessage `json:"genesis"`

	DBDir  string `json:"db_dir"`
	DBType string `json:"db_type"`
	DBName string `json:"db_name"`

	WALDir      string `json:"wal_dir`
	ContractDir string `json:"contract_dir"`
}

type singleChain struct {
	wallet module.Wallet

	database db.Database
	vld      module.VoteListDecoder
	sm       module.ServiceManager
	bm       module.BlockManager
	cs       module.Consensus
	sv       rpc.JsonRpcServer
	nt       module.NetworkTransport
	nm       module.NetworkManager

	cfg Config
	pm  eeproxy.Manager
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
	return c.cfg.Genesis
}

func (c *singleChain) VoteListDecoder() module.VoteListDecoder {
	return c.vld
}

func (c *singleChain) EEProxyManager() eeproxy.Manager {
	return c.pm
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

func (c *singleChain) Start() {
	c.database = db.Open(c.cfg.DBDir, c.cfg.DBType, c.cfg.DBName)

	c.nm = network.NewManager(c.cfg.Channel, c.nt, c.cfg.SeedAddr, toRoles(c.cfg.Role)...)

	c.vld = consensus.NewVoteListFromBytes
	c.sm = service.NewManager(c, c.nm, c.pm, c.cfg.ContractDir)
	c.bm = block.NewManager(c, c.sm)

	c.cs = consensus.NewConsensus(c, c.bm, c.nm, c.cfg.WALDir)
	err := c.cs.Start()
	if err != nil {
		log.Panicf("singleChain.Start: %+v\n", err)
	}

	c.sv = rpc.NewJsonRpcServer(c, c.bm, c.sm, c.cs, c.nm)

	if err := c.sv.ListenAndServe(c.cfg.RPCAddr); err != nil {
		log.Printf("Fail to Listen on RPC server err=%+v", err)
	}
}

func NewChain(
	wallet module.Wallet,
	transport module.NetworkTransport,
	pm eeproxy.Manager,
	cfg *Config,
) *singleChain {
	chain := &singleChain{
		wallet: wallet,
		nt:     transport,
		cfg:    *cfg,
		pm:     pm,
	}
	if chain.cfg.DBName == "" {
		chain.cfg.DBName = chain.cfg.Channel
	}
	if chain.cfg.DBType == "" {
		chain.cfg.DBType = string(db.BadgerDBBackend)
	}
	return chain
}
