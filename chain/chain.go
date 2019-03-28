package chain

import (
	"encoding/json"
	"log"

	"github.com/icon-project/goloop/block"
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

func (c *singleChain) Start() {
	var err error
	c.database, err = db.Open(c.cfg.DBDir, c.cfg.DBType, c.cfg.DBName)
	if err != nil {
		log.Panicf("singleChain.Start: %+v", err)
	}

	c.nm = network.NewManager(c.cfg.Channel, c.nt, c.cfg.SeedAddr, toRoles(c.cfg.Role)...)
	err = c.nm.Start()
	if err != nil {
		log.Panicf("singleChain.Start: %+v\n", err)
	}

	c.vld = consensus.NewCommitVoteSetFromBytes
	c.sm = service.NewManager(c, c.nm, c.pm, c.cfg.ContractDir)
	c.bm = block.NewManager(c, c.sm)

	c.cs = consensus.NewConsensus(c, c.bm, c.nm, c.cfg.WALDir)
	err = c.cs.Start()
	if err != nil {
		log.Panicf("singleChain.Start: %+v\n", err)
	}

	// server : SetChain()
	c.srv.SetChain(c.cfg.Channel, c)
}

func NewChain(
	wallet module.Wallet,
	transport module.NetworkTransport,
	srv *server.Manager,
	pm eeproxy.Manager,
	cfg *Config,
) *singleChain {
	chain := &singleChain{
		wallet: wallet,
		nt:     transport,
		srv:    srv,
		cfg:    *cfg,
		pm:     pm,
	}
	if chain.cfg.DBName == "" {
		chain.cfg.DBName = chain.cfg.Channel
	}
	if chain.cfg.DBType == "" {
		chain.cfg.DBType = string(db.BadgerDBBackend)
	}
	if chain.cfg.GenesisStorage == nil {
		if gs, err := NewGenesisStorageWithDataDir(
			chain.cfg.Genesis, chain.cfg.GenesisDataPath); err != nil {
			log.Panicf("Fail to create GenesisStorage with path=%s err=%+v",
				chain.cfg.GenesisDataPath, err)
			return nil
		} else {
			chain.cfg.GenesisStorage = gs
		}
	}
	return chain
}
