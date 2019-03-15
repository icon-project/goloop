package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/transaction"

	"github.com/pkg/errors"

	"github.com/icon-project/goloop/network"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

const (
	testTransactionNum = 10000
	startBlk           = 23434 // version3
)

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
	if err != nil || resp.StatusCode != 200 {
		if err != nil {
			return nil, err
		}
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

type blockV1Impl struct {
	Version            string                    `json:"version"`
	PrevBlockHash      common.RawHexBytes        `json:"prev_block_hash"`
	MerkleTreeRootHash common.RawHexBytes        `json:"merkle_tree_root_hash"`
	Transactions       []transaction.Transaction `json:"confirmed_transaction_list"`
	BlockHash          common.RawHexBytes        `json:"block_hash"`
	Height             int64                     `json:"height"`
	PeerID             string                    `json:"peer_id"`
	TimeStamp          uint64                    `json:"time_stamp"`
	Signature          common.Signature          `json:"signature"`
}

func ParseLegacy(b []byte) (module.TransactionList, error) {
	var blk = new(blockV1Impl)
	err := json.Unmarshal(b, blk)
	if err != nil {
		return nil, err
	}
	trs := make([]module.Transaction, len(blk.Transactions))
	for i, tx := range blk.Transactions {
		trs[i] = tx
	}
	return transaction.NewTransactionListV1FromSlice(trs), nil
}

type transitionCb struct {
	exeDone chan bool
}

func (ts *transitionCb) OnValidate(module.Transition, error) {
}

func (ts *transitionCb) OnExecute(module.Transition, error) {
	ts.exeDone <- true
}

type serviceChain struct {
	wallet module.Wallet
	nid    int

	database db.Database
	sm       module.ServiceManager
	bm       module.BlockManager
	cs       module.Consensus
}

func (c *serviceChain) GetGenesisData(key []byte) ([]byte, error) {
	panic("implement me")
}

func (c *serviceChain) BlockManager() module.BlockManager {
	return c.bm
}

func (c *serviceChain) Consensus() module.Consensus {
	return c.cs
}

func (c *serviceChain) ServiceManager() module.ServiceManager {
	return c.sm
}

func (c *serviceChain) NetworkManager() module.NetworkManager {
	return nil
}

func (c *serviceChain) CommitVoteSetDecoder() module.CommitVoteSetDecoder {
	return nil
}

func (c *serviceChain) Database() db.Database {
	return c.database
}

func (c *serviceChain) Wallet() module.Wallet {
	return c.wallet
}

func (c *serviceChain) NID() int {
	return c.nid
}

func (c *serviceChain) Genesis() []byte {
	genesis :=
		`{
		  "accounts": [
			{
			  "name": "god",
			  "address": "hx5a05b58a25a1e5ea0f1d5715e1f655dffc1fb30a",
			  "balance": "0x2961fff8ca4a623278000000000000000"
			},
			{
			  "name": "treasury",
			  "address": "hx1000000000000000000000000000000000000000",
			  "balance": "0x0"
			}
		  ],
		  "message": "A rhizome has no beginning or end; it is always in the middle, between things, interbeing, intermezzo. The tree is filiation, but the rhizome is alliance, uniquely alliance. The tree imposes the verb \"to be\" but the fabric of the rhizome is the conjunction, \"and ... and ...and...\"This conjunction carries enough force to shake and uproot the verb \"to be.\" Where are you going? Where are you coming from? What are you heading for? These are totally useless questions.\n\n - Mille Plateaux, Gilles Deleuze & Felix Guattari\n\n\"Hyperconnect the world\"",
		  "validatorlist": [
			"hx100000000000000000000000000000000001234",
			"hx100000000000000000000000000000000012345"
		  ]
		}`
	return []byte(genesis)
}

func eeProxy() eeproxy.Manager {
	pm, err := eeproxy.NewManager("unix", "/tmp/ee.socket")
	if err != nil {
		log.Panicln("FAIL to start EEManager")
	}
	go pm.Loop()

	ee, err := eeproxy.NewPythonEE()
	if err != nil {
		log.Panicf("FAIL to create PythonEE err=%+v", err)
	}
	pm.SetEngine("python", ee)
	pm.SetInstances("python", 1)
	return pm
}

func TestUnitService(t *testing.T) {
	// request transactions
	if testTransactionNum == 0 {
		return
	}
	c := new(serviceChain)
	c.wallet = wallet.New()
	c.database = db.NewMapDB()
	nt := network.NewTransport("127.0.0.1:8081", c.wallet)
	nt.Listen()
	defer nt.Close()
	//eem, err := eeproxy.NewManager("unix", "/tmp/ee.socket")
	//if err != nil {
	//	log.Panicln("FAIL to start EEManager")
	//}
	//go eem.Loop()
	em := eeProxy()
	leaderServiceManager := NewManager(c, nil, em, "./contract")
	//leaderServiceManager := NewManager(c, network.NewManager("default", nt, "", module.ROLE_VALIDATOR), eem, "./contract")
	it, _ := leaderServiceManager.CreateInitialTransition(nil, nil)
	bi := newBlockInfo(0, 0)
	genesisTx, _ := transaction.NewTransactionFromJSON(c.Genesis())
	txs := transaction.NewTransactionListFromSlice(c.database, []module.Transaction{genesisTx})
	parentTrs, _ := leaderServiceManager.CreateTransition(it, txs, bi)
	cb := &transitionCb{make(chan bool)}
	parentTrs.Execute(cb)
	<-cb.exeDone
	leaderServiceManager.Finalize(parentTrs, module.FinalizeNormalTransaction|module.FinalizeResult)

	// request SendTransaction
	sendDone := make(chan bool)

	client := Wallet{"https://testwallet.icon.foundation/api/v3"}
	blockDone := make(chan bool)

	go func() {
		for i := startBlk; i < startBlk+testTransactionNum; i++ {
			fmt.Printf("block height = %d\n", i)
			b, err := client.GetBlockByHeight(i)
			if err != nil {
				panic(err)
			}
			tl, err := ParseLegacy(b)
			if err != nil {
				panic(err)
			}
			for itr := tl.Iterator(); itr.Has(); itr.Next() {
				t, _, _ := itr.Get()
				leaderServiceManager.SendTransaction(t)
			}
			<-blockDone
		}
		sendDone <- true
	}()

	//run service manager for leader
	txListChan := make(chan module.TransactionList)
	var validatorResult []byte
	// propose transition
	go func() {
		exeDone := make(chan bool)
		h := int64(1)
		for {
			transition, err := leaderServiceManager.ProposeTransition(parentTrs, newBlockInfo(h, time.Now().UnixNano()/int64(time.Millisecond)))
			if err != nil {
				log.Panicf("Failed to propose transition!, err = %s\n", err)
			}
			txList := transition.NormalTransactions()
			txListChan <- txList
			<-txListChan
			cb := &transitionCb{exeDone}
			transition.Execute(cb)
			<-cb.exeDone
			if bytes.Compare(transition.Result(), validatorResult) != 0 {
				panic("Failed to compare result ")
			}
			leaderServiceManager.Finalize(transition, module.FinalizeNormalTransaction|module.FinalizeResult)
			blockDone <- true
			// get result then run below
			parentTrs = transition
			h++
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// validator
	validatorCh := new(serviceChain)
	validatorCh.wallet = wallet.New()
	validatorCh.database = db.NewMapDB()

	nt2 := network.NewTransport("127.0.0.1:8082", c.wallet)
	nt2.Listen()
	defer nt2.Close()

	if err := nt.Dial("127.0.0.1:8081", "default"); err != nil {
		log.Panic("Failed")
	}
	validatorServiceManager := NewManager(validatorCh, nil, em, "./contract")
	//validatorServiceManager := NewManager(validatorCh, network.NewManager("default", nt2, "", module.ROLE_VALIDATOR), eem, "./contract")
	vit, _ := leaderServiceManager.CreateInitialTransition(nil, nil)
	parentVTransition, _ := leaderServiceManager.CreateTransition(vit, txs, bi)
	parentVTransition.Execute(cb)
	<-cb.exeDone
	leaderServiceManager.Finalize(parentVTransition, module.FinalizeNormalTransaction|module.FinalizeResult)
	go func() {
		exeDone := make(chan bool)
		h := int64(1)
		for {
			txList := <-txListChan
			// Just make a similar BlockInfo and set it.
			vTransition, err := validatorServiceManager.CreateTransition(parentVTransition, txList, newBlockInfo(h, time.Now().UnixNano()/int64(time.Millisecond)))
			if err != nil {
				log.Panicf("Failed to create transition for validator : %s", err)
			}
			cb := &transitionCb{exeDone}
			vTransition.Execute(cb)
			<-cb.exeDone
			validatorResult = vTransition.Result()
			validatorServiceManager.Finalize(vTransition, module.FinalizeNormalTransaction|module.FinalizeResult)
			parentVTransition = vTransition
			h++
			txListChan <- nil
		}
	}()
	<-sendDone
}
