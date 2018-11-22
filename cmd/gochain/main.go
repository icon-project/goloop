package main

import (
	"flag"
	"fmt"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/rpc"
	"github.com/icon-project/goloop/service"
)

type singleChain struct {
	nid    int
	wallet module.Wallet

	database db.Database
	sm       module.ServiceManager
	bm       module.BlockManager
	cs       module.Consensus
	sv       rpc.JsonRpcServer
	nt       module.NetworkTransport
	nm       module.NetworkManager
}

func (c *singleChain) GetDatabase() db.Database {
	return c.database
}

func (c *singleChain) GetWallet() module.Wallet {
	return c.wallet
}

func (c *singleChain) GetNID() int {
	return c.nid
}

func voteListDecoder([]byte) module.VoteList {
	return nil
}

func (c *singleChain) VoteListDecoder() module.VoteListDecoder {
	return module.VoteListDecoder(voteListDecoder)
}

func (c *singleChain) start() {
	c.database = db.NewMapDB()

	c.sm = service.NewManager(c)
	c.bm = block.NewManager(c, c.sm)
	c.cs = consensus.NewConsensus(c.bm)
	c.sv = rpc.NewJsonRpcServer(c.bm, c.sm)
	channel := fmt.Sprintf("%x", c.nid)

	c.nm = network.NewManager(channel, c.nt, toRoles(role)...)
	if seedAddr != "" {
		c.nt.Dial(seedAddr, channel)
	}

	go c.cs.Start()
	c.sv.ListenAndServe(rpcAddr)
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

var (
	rpcAddr     string
	p2pAddr     string
	seedAddr    string
	role        uint
	asValidator bool
	asSeed      bool
)

func main() {
	c := new(singleChain)

	flag.IntVar(&c.nid, "nid", 1, "Chain Network ID")
	flag.StringVar(&rpcAddr, "rpc", ":9080", "Listen ip-port of JSON-RPC")
	flag.StringVar(&p2pAddr, "p2p", "127.0.0.1:8080", "Listen ip-port of P2P")
	flag.StringVar(&seedAddr, "seed", "", "Ip-port of Seed")
	flag.UintVar(&role, "role", 0, "[0:None, 1:Seed, 2:Validator, 3:Both]")
	flag.Parse()

	priK, _ := crypto.GenerateKeyPair()
	c.wallet, _ = common.WalletFromPrivateKey(priK)
	c.nt = network.NewTransport(p2pAddr, priK)
	c.nt.Listen()
	defer c.nt.Close()

	c.start()
}
