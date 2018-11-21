package main

import (
	"flag"
	"fmt"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common"
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
	rpc    string

	database db.Database
	sm       module.ServiceManager
	bm       module.BlockManager
	cs       module.Consensus
	sv       rpc.JsonRpcServer
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

	c.sm = service.NewManager(c.database)
	c.bm = block.NewManager(c, c.sm)
	c.cs = consensus.NewConsensus(c.bm)
	c.sv = rpc.NewJsonRpcServer(c.bm, c.sm)
	//
	channel := fmt.Sprintf("%x", c.nid)
	c.nm = network.GetNetworkManager(channel)
	l := network.GetListener()
	l.Listen()
	defer l.Close()

	go c.cs.Start()
	c.sv.ListenAndServe(c.rpc)
}

func main() {
	c := new(singleChain)

	config := network.GetConfig()

	flag.IntVar(&c.nid, "nid", 1, "Chain Network ID")
	flag.StringVar(&config.ListenAddress, "listen", "127.0.0.1:8080", "Network address")
	flag.StringVar(&config.SeedAddress, "seed", "127.0.0.1:8080", "Seed address")
	flag.StringVar(&c.rpc, "rpc", ":9080", "JSON RPC address")
	flag.Parse()

	c.wallet, _ = common.WalletFromPrivateKey(config.PrivateKey)

	c.start()
}
