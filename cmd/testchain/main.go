package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/rpc"
	"github.com/icon-project/goloop/service"
)

type chain struct {
	wallet module.Wallet
	nid    int

	database db.Database
	sm       module.ServiceManager
	bm       module.BlockManager
	cs       module.Consensus
	sv       rpc.JsonRpcServer
}

func (c *chain) GetDatabase() db.Database {
	return c.database
}

func (c *chain) GetWallet() module.Wallet {
	return c.wallet
}

func (c *chain) GetNID() int {
	return c.nid
}

func voteListDecoder([]byte) module.VoteList {
	return &emptyVoteList{}
}

func (c *chain) VoteListDecoder() module.VoteListDecoder {
	return module.VoteListDecoder(voteListDecoder)
}

type emptyVoteList struct {
}

func (vl *emptyVoteList) Verify(block module.Block) error {
	return nil
}

func (vl *emptyVoteList) Bytes() []byte {
	return nil
}

func (vl *emptyVoteList) Hash() []byte {
	return crypto.SHA3Sum256(vl.Bytes())
}

type proposeOnlyConsensus struct {
	sm module.ServiceManager
	bm module.BlockManager
	ch chan<- []byte
}

func (c *proposeOnlyConsensus) Start() {
	blks, err := c.bm.FinalizeGenesisBlocks(
		common.NewAccountAddress(make([]byte, common.AddressBytes)),
		time.Unix(0, 0),
		&emptyVoteList{},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("Proposer: FinalizedGenesisBlocks")
	blk := blks[len(blks)-1]

	ch := make(chan module.Block)
	height := 1
	wallet := Wallet{"https://testwallet.icon.foundation/api/v3"}
	for {
		b, err := wallet.GetBlockByHeight(height)
		if err != nil {
			panic(err)
		}
		wblk, err := NewBlockV1(b)
		if err != nil {
			panic(err)
		}
		for itr := wblk.NormalTransactions().Iterator(); itr.Has(); itr.Next() {
			t, _, _ := itr.Get()
			c.sm.SendTransaction(t)
		}

		_, err = c.bm.Propose(blk.ID(), &emptyVoteList{}, func(b module.Block, e error) {
			if e != nil {
				panic(e)
			}
			ch <- b
		})
		if err != nil {
			panic(err)
		}
		blk = <-ch
		err = c.bm.Finalize(blk)
		if err != nil {
			panic(err)
		}
		buf := bytes.NewBuffer(nil)
		blk.MarshalHeader(buf)
		blk.MarshalBody(buf)
		fmt.Printf("Proposer: Finalized Block(%d) %x\n", blk.Height(), blk.ID())
		c.ch <- buf.Bytes()
		_, err = c.bm.GetBlockByHeight(int64(height) + 1)
		if err != nil {
			panic(err)
		}
		height++
	}
}

type importOnlyConsensus struct {
	bm module.BlockManager
	sm module.ServiceManager
	ch <-chan []byte
}

func (c *importOnlyConsensus) Start() {
	_, err := c.bm.FinalizeGenesisBlocks(
		common.NewAccountAddress(make([]byte, common.AddressBytes)),
		time.Unix(0, 0),
		&emptyVoteList{},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("Importer : FinalizedGenesisBlocks")

	ch := make(chan module.Block)
	for {
		bs := <-c.ch
		buf := bytes.NewBuffer(bs)
		_, err := c.bm.Import(buf, func(b module.Block, e error) {
			if e != nil {
				panic(e)
			}
			ch <- b
		})
		if err != nil {
			panic(err)
		}
		blk := <-ch
		err = c.bm.Finalize(blk)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Importer: Finalized Block(%d) %x\n", blk.Height(), blk.ID())
	}
}

func (c *chain) startAsProposer(ch chan<- []byte) {
	c.wallet = common.NewWallet()
	c.database = db.NewMapDB()
	c.sm = service.NewManager(c.database)
	c.bm = block.NewManager(c, c.sm)
	c.cs = &proposeOnlyConsensus{
		sm: c.sm,
		bm: c.bm,
		ch: ch,
	}
	sm = c.sm

	c.sv = rpc.NewJsonRpcServer(c.bm, c.sm)
	c.sv.Start()

	c.cs.Start()
}

func (c *chain) startAsImporter(ch <-chan []byte) {
	c.wallet = common.NewWallet()
	c.database = db.NewMapDB()
	c.sm = service.NewManager(c.database)
	c.bm = block.NewManager(c, c.sm)
	c.cs = &importOnlyConsensus{
		sm: c.sm,
		bm: c.bm,
		ch: ch,
	}
	sm = c.sm

	c.cs.Start()
}

type JSONRPCResponse struct {
	Version string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
}

type Wallet struct {
	url string
}

func (w *Wallet) Call(method string, params map[string]interface{}) ([]byte, error) {
	d := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		d["params"] = params
	}
	req, err := json.Marshal(d)
	if err != nil {
		log.Println("Making request fails")
		log.Println("Data", d)
		return nil, err
	}
	resp, err := http.Post(w.url, "application/json", bytes.NewReader(req))
	if resp.StatusCode != 200 {
		return nil, errors.New(
			fmt.Sprintf("FAIL to call res=%d", resp.StatusCode))
	}

	var buf = make([]byte, 2048*1024)
	var bufLen, readed int = 0, 0

	for true {
		readed, _ = resp.Body.Read(buf[bufLen:])
		if readed < 1 {
			break
		}
		bufLen += readed
	}
	var r JSONRPCResponse
	err = json.Unmarshal(buf[0:bufLen], &r)
	if err != nil {
		log.Println("JSON Parse Fail")
		log.Println("JSON=", string(buf[0:bufLen]))
		return nil, err
	}
	return r.Result.MarshalJSON()
}

func (w *Wallet) GetBlockByHeight(h int) ([]byte, error) {
	p := map[string]interface{}{
		"height": fmt.Sprintf("0x%x", h),
	}
	return w.Call("icx_getBlockByHeight", p)
}

var sm module.ServiceManager

type transaction struct {
	module.Transaction
}

func (t *transaction) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (t *transaction) UnmarshalJSON(b []byte) error {
	tr := sm.TransactionFromBytes(b, common.BlockVersion1)
	if tr == nil {
		return common.ErrUnknown
	}
	t.Transaction = tr
	return nil
}

func (t transaction) String() string {
	return fmt.Sprint(t.Transaction)
}

func main() {
	proposer := new(chain)
	importer := new(chain)

	ch := make(chan []byte)
	go proposer.startAsProposer(ch)
	importer.startAsImporter(ch)
}
