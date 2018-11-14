package main

import (
	"bytes"
	"flag"
	"net/http"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common/db"
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

type consensus struct {
	c  module.Chain
	bm module.BlockManager
	ch chan module.Block
}

func newConsensus(c module.Chain, bm module.BlockManager) *consensus {
	return &consensus{
		c:  c,
		bm: bm,
		ch: make(chan module.Block),
	}
}

func (c *consensus) Start() {
	blk, err := c.bm.GetLastBlock()
	if err != nil {
		panic(err)
	}
	for {
		_, err := c.bm.Propose(blk.ID(), nil, func(b module.Block, e error) {
			c.ch <- b
		})
		if err != nil {
			panic(err)
		}
		blk := <-c.ch
		buf := bytes.NewBuffer(nil)
		blk.MarshalHeader(buf)
		blk.MarshalBody(buf)
		_, err = c.bm.Import(buf, func(b module.Block, e error) {
			c.ch <- b
		})
		if err != nil {
			panic(err)
		}
		blk = <-c.ch
	}
}

func (c *singleChain) start() {
	c.database = db.NewMapDB()
	c.sm = service.NewManager(c.database)
	c.bm = block.NewManager(c, c.sm)
	c.cs = newConsensus(c, c.bm)

	go c.cs.Start()
}

func main() {
	c := new(singleChain)

	flag.IntVar(&c.nid, "nid", 1, "Chain Network ID")
	flag.Parse()

	c.start()

	http.ListenAndServe(":8080", rpc.JsonRpcHandler())
}
