package chain

import (
	"encoding/json"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/rpc"
	"github.com/icon-project/goloop/service"
)

type Config struct {
	NID      int             `json:"nid"`
	Channel  string          `json:"channel"`
	RPCAddr  string          `json:"rpc_addr"`
	SeedAddr string          `json:"seed_addr"`
	Role     uint            `json:"role"`
	Genesis  json.RawMessage `json:"genesis"`
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

func voteListDecoder([]byte) module.VoteList {
	return nil
}

func (c *singleChain) VoteListDecoder() module.VoteListDecoder {
	return c.vld
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
	c.database = db.NewMapDB()

	c.vld = voteListDecoder
	c.sm = service.NewManager(c)
	c.bm = block.NewManager(c, c.sm)
	c.cs = consensus.NewConsensus(c.bm)
	c.sv = rpc.NewJsonRpcServer(c.bm, c.sm)

	c.nm = network.NewManager(c.cfg.Channel, c.nt, toRoles(c.cfg.Role)...)
	if c.cfg.SeedAddr != "" {
		c.nt.Dial(c.cfg.SeedAddr, c.cfg.Channel)
	}
	go c.cs.Start()
	c.sv.ListenAndServe(c.cfg.RPCAddr)
}

func NewChain(wallet module.Wallet, transport module.NetworkTransport, cfg *Config) *singleChain {
	return &singleChain{
		wallet: wallet,
		nt:     transport,
		cfg:    *cfg,
	}
}
