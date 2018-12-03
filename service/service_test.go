package service_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icon-project/goloop/common/legacy"
	"github.com/icon-project/goloop/service"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/rpc"
)

type transitionCb struct {
	exeDone chan bool
}

func (ts *transitionCb) OnValidate(module.Transition, error) {
}

func (ts *transitionCb) OnExecute(module.Transition, error) {
	ts.exeDone <- true
}

type chain struct {
	wallet module.Wallet
	nid    int

	database db.Database
	sm       module.ServiceManager
	bm       module.BlockManager
	cs       module.Consensus
	sv       rpc.JsonRpcServer
}

func (c *chain) VoteListDecoder() module.VoteListDecoder {
	return nil
}

func (c *chain) Database() db.Database {
	log.Println("DATABASE")
	return c.database
}

func (c *chain) Wallet() module.Wallet {
	return c.wallet
}

func (c *chain) NID() int {
	return c.nid
}

func (c *chain) Genesis() []byte {
	genPath := ""
	if len(genPath) == 0 {
		file := "genesisTx.json"
		topDir := "goloop"
		path, _ := filepath.Abs(".")
		base := filepath.Base(path)
		switch {
		case strings.Compare(base, topDir) == 0:
			genPath = path + "/" + file
		case strings.Compare(base, "icon-project") == 0:
			genPath = path + "/" + topDir + "/" + file
		case strings.Compare(base, "service") == 0:
			genPath = strings.TrimSuffix(path, "/service") + "/" + file
		default:
			log.Panicln("Not considered case")
		}
	}
	log.Println("gen : ", genPath)
	gen, err := ioutil.ReadFile(genPath)
	if err != nil {
		log.Panicln("Failed to read genesisFile. err : ", err)
	}
	return gen
}

func sendTx(sm module.ServiceManager, done chan bool) {
	db, err := legacy.OpenDatabase("./data/testnet/block", "./data/testnet/score")
	if err != nil {
		log.Printf("Fail to open database err=%+v", err)
		return
	}

	for i := 1; i < 10000; i++ {
		blk, err := db.GetBlockByHeight(i)
		if err != nil {
			log.Printf("Fail to get block err=%+v", err)
			return
		}
		log.Printf("Block [%d] : %x", blk.Height(), blk.ID())
		txl := blk.NormalTransactions()
		txCnt := 0
		for i := txl.Iterator(); i.Has(); i.Next() {
			tx, _, err := i.Get()
			if err != nil {
				log.Printf("Failed to get transaction err=%+v", err)
				os.Exit(-1)
			}
			txCnt += 1
			//log.Printf("txCnt : %d\n", txCnt)

			if _, err := sm.SendTransaction(tx); err == service.ErrTransactionPoolOverFlow {
				log.Printf("Failed to send transaction err : %s\n", err)
				log.Printf("Waiting for 5 seconds...")
				time.Sleep(5 * time.Second)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	log.Println("SendTx is done!!!")
	done <- true
}
func TestUnitService(t *testing.T) {
	// request transactions
	c := new(chain)
	c.wallet = common.NewWallet()
	c.database = db.NewMapDB()
	leaderServiceManager := service.NewManager(c)
	it, _ := leaderServiceManager.CreateInitialTransition(nil, nil, -1)
	parentTrs, _ := leaderServiceManager.ProposeGenesisTransition(it)
	cb := &transitionCb{make(chan bool)}
	parentTrs.Execute(cb)
	<-cb.exeDone
	leaderServiceManager.Finalize(parentTrs, module.FinalizeNormalTransaction|module.FinalizeResult)

	// request SendTransaction
	sendDone := make(chan bool)
	go sendTx(leaderServiceManager, sendDone)

	//run service manager for leader
	txListChan := make(chan module.TransactionList)
	var validatorResult []byte
	// propose transition
	go func() {
		exeDone := make(chan bool)
		for {
			transition, err := leaderServiceManager.ProposeTransition(parentTrs)
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
			// get result then run below
			parentTrs = transition
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// validator
	validatorCh := new(chain)
	validatorCh.wallet = common.NewWallet()
	validatorCh.database = db.NewMapDB()
	validatorServiceManager := service.NewManager(validatorCh)
	vit, _ := leaderServiceManager.CreateInitialTransition(nil, nil, -1)
	parentVTransition, _ := leaderServiceManager.ProposeGenesisTransition(vit)
	parentVTransition.Execute(cb)
	<-cb.exeDone
	leaderServiceManager.Finalize(parentVTransition, module.FinalizeNormalTransaction|module.FinalizeResult)
	go func() {
		exeDone := make(chan bool)
		for {
			txList := <-txListChan
			vTransition, err := validatorServiceManager.CreateTransition(parentVTransition, txList)
			if err != nil {
				log.Panicf("Failed to create transition for validator : %s", err)
			}
			cb := &transitionCb{exeDone}
			vTransition.Execute(cb)
			<-cb.exeDone
			validatorResult = vTransition.Result()
			validatorServiceManager.Finalize(vTransition, module.FinalizeNormalTransaction|module.FinalizeResult)
			parentVTransition = vTransition
			txListChan <- nil
		}
	}()

	<-sendDone
	time.Sleep(5 * time.Second)
}
