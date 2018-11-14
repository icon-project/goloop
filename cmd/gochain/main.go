package main

import (
	"flag"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/rpc"
	"github.com/icon-project/goloop/service"
)

type singleChain struct {
	nid int

	database db.Database
	sm       module.ServiceManager
	bm       module.BlockManager
	cs       module.Consensus
	sv       rpc.JsonRpcServer
}

func (c *singleChain) GetDatabase() db.Database {
	return c.database
}

func (c *singleChain) GetWallet() module.Wallet {
	// TODO Implement wallet.
	return nil
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

	go c.cs.Start()
	c.sv.Start()
}

func main() {
	c := new(singleChain)

	flag.IntVar(&c.nid, "nid", 1, "Chain Network ID")
	flag.Parse()

	c.start()
}
