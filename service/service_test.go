package service_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/big"
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

type txTest struct {
	group module.TransactionGroup

	version   int
	from      *common.Address
	to        *common.Address
	value     *big.Int
	stepLimit *big.Int
	timestamp int64
	nid       int
	nonce     int64
	signature []byte

	hash  []byte
	bytes []byte
}

type txTestV2 struct {
	From      common.Address  `json:"from"`
	To        common.Address  `json:"to"`
	Value     common.HexInt   `json:"value"`
	Fee       common.HexInt   `json:"fee"`
	Timestamp common.HexInt64 `json:"timestamp"`
	Nonce     common.HexInt64 `json:"nonce"`
	Signature common.HexBytes `json:"signature"`
}

type txTestV3 struct {
	Version   common.HexInt16 `json:"version"`
	From      common.Address  `json:"from"`
	To        common.Address  `json:"to"`
	Value     common.HexInt   `json:"value"`
	StepLimit common.HexInt   `json:"stepLimit"`
	Timestamp common.HexInt64 `json:"timestamp"`
	Nid       common.HexInt16 `json:"nid"`
	Nonce     common.HexInt64 `json:"nonce"`
	Signature common.HexBytes `json:"signature"`
}

func (tx *txTest) ToJSON(version int) (interface{}, error) {
	panic("implement me")
}

func (tx *txTest) Group() module.TransactionGroup {
	return tx.group
}

func (tx *txTest) ID() []byte {
	return tx.hash
}

func (tx *txTest) Version() int {
	return tx.version
}

func (tx *txTest) Bytes() []byte {
	return tx.bytes
}

func (tx *txTest) Verify() error {
	return nil
}

func (tx *txTest) From() module.Address {
	return tx.from
}

func (tx *txTest) To() module.Address {
	return tx.to
}

func (tx *txTest) Value() *big.Int {
	return tx.value
}

func (tx *txTest) StepLimit() *big.Int {
	return tx.stepLimit
}

func (tx *txTest) Timestamp() int64 {
	return tx.timestamp
}

func (tx *txTest) NID() int {
	return tx.nid
}

func (tx *txTest) Nonce() int64 {
	return tx.nonce
}

func (tx *txTest) Hash() []byte {
	return tx.hash
}

func (tx *txTest) Signature() []byte {
	return tx.signature
}

type transitionCb struct {
	exeDone chan bool
}

func (ts *transitionCb) OnValidate(module.Transition, error) {
}

func (ts *transitionCb) OnExecute(module.Transition, error) {
	ts.exeDone <- true
}

// test case
const (
	TEST_ACCOUNTS_NUM         = 10
	TEST_VALID_REQUEST_TX_NUM = 100
)

var resultMap = make(map[string]*big.Int)
var deposit = int64(1000000)
var testAddresses [TEST_ACCOUNTS_NUM]module.Address
var testWallets [TEST_ACCOUNTS_NUM]module.Wallet

//func createTxInst(wallet module.Wallet, to module.Address, value *big.Int, timestamp int64) module.Transaction {
//	r := rand.Int63()
//	ver := int((r % 2) + 2)
//
//	tx := txTest{}
//	tx.group = module.TransactionGroupNormal
//	tx.version = ver
//	tx.from = wallet.Address().(*common.Address)
//	tx.to = to.(*common.Address)
//	tx.value = value
//	tx.stepLimit = new(big.Int).SetInt64(r % 0xffff)
//	tx.timestamp = timestamp
//	tx.nid = 0
//	tx.nonce = r
//
//	// TODO find a better way to JSON serialization
//	// create a signature
//	m := make(map[string]interface{})
//	switch tx.version {
//	case 2:
//		m["from"] = tx.from.String()
//		m["to"] = tx.to.String()
//		m["value"] = common.HexInt{*tx.value}.String()
//		m["fee"] = common.HexInt{*tx.stepLimit}.String()
//		m["timestamp"] = common.HexInt64{tx.timestamp}.String()
//		m["nonce"] = common.HexInt64{tx.nonce}.String()
//	case 3:
//		m["version"] = common.HexInt16{int16(tx.version)}.String()
//		m["from"] = tx.from.String()
//		m["to"] = tx.to.String()
//		m["value"] = common.HexInt{*tx.value}.String()
//		m["stepLimit"] = common.HexInt{*tx.stepLimit}.String()
//		m["timestamp"] = common.HexInt64{tx.timestamp}.String()
//		m["nid"] = common.HexInt16{int16(tx.nid)}.String()
//		m["nonce"] = common.HexInt64{tx.nonce}.String()
//	default:
//		log.Fatalln("unknown transaction version:", tx.version)
//	}
//	bs, err := SerializeMap(m, map[string]bool(nil), map[string]bool(nil))
//	if err != nil {
//		log.Fatalln("fail to create transaction bytes")
//	}
//	bs = append([]byte("icx_sendTransaction."), bs...)
//	h := crypto.SHA3Sum256(bs)
//	sig, err := wallet.Sign(h)
//	if err != nil {
//		log.Fatalln("fail to create a signature")
//	}
//	tx.signature = sig
//
//	// create bytes
//	tx.bytes = marshalTx(&tx)
//	return &tx
//}

func marshalTx(tx *txTest) []byte {
	var i interface{}
	switch tx.version {
	case 2:
		ti := txTestV2{
			From:      *tx.from,
			To:        *tx.to,
			Value:     common.HexInt{*tx.value},
			Fee:       common.HexInt{*tx.stepLimit},
			Timestamp: common.HexInt64{tx.timestamp},
			Nonce:     common.HexInt64{tx.nonce},
			Signature: tx.signature,
		}
		i = ti
	case 3:
		ti := txTestV3{
			Version:   common.HexInt16{int16(tx.version)},
			From:      *tx.from,
			To:        *tx.to,
			Value:     common.HexInt{*tx.value},
			StepLimit: common.HexInt{*tx.stepLimit},
			Timestamp: common.HexInt64{tx.timestamp},
			Nid:       common.HexInt16{int16(tx.nid)},
			Nonce:     common.HexInt64{tx.nonce},
			Signature: tx.signature,
		}
		i = ti
	}
	b, _ := json.Marshal(&i)
	return b
}

//const txLiveDuration = int64(60 * time.Second / time.Millisecond) // 60 seconds in millisecond
//// true if valid transaction
//func createRandTx(valid bool, time int64, validNum int) module.Transaction {
//	idFrom := rand.Int() % TEST_ACCOUNTS_NUM
//	//tx.hash = []byte{id}
//	// valid 하도록 만든다. 기존에 없는 ID, time 등을 이용하도록.
//	// insert transaction to valid transaction (expected txPool).
//	// ID map, time map 사용.
//	// 중복될 경우 새로운 ID, time을 생성한다.
//	idTo := rand.Int() % TEST_ACCOUNTS_NUM
//	for idTo == idFrom {
//		idTo = rand.Int() % TEST_ACCOUNTS_NUM
//	}
//	//tx.to = addresses[toId]
//	stringFrom := string(testAddresses[idFrom].Bytes())
//	stringTo := string(testAddresses[idTo].Bytes())
//	walletFrom := testWallets[idFrom]
//	to := testAddresses[idTo]
//	value := big.NewInt(int64(rand.Int() % 300000))
//
//	var timestamp int64
//	if valid {
//		// TODO: 먼저 from에서 이체 가능금액 확인 & 이체
//		balance := resultMap[stringFrom]
//		if balance != nil && balance.Cmp(value) > 0 {
//			resultMap[stringFrom] = balance.Mul(balance, value)
//			if _, ok := resultMap[stringTo]; ok == false {
//				resultMap[stringTo] = big.NewInt(0)
//			}
//			resultMap[stringTo].Add(resultMap[stringTo], value)
//		}
//
//		timestamp = time + 1000 + int64(rand.Int()%100)
//		// TODO: check value type. no
//	} else {
//		timestamp = time - txLiveDuration - 1000 - int64(rand.Int()%10)
//	}
//
//	return createTxInst(walletFrom, to, value, timestamp)
//}

//func requestTx(validTxNum int, manager module.ServiceManager, done chan bool) {
//	txMap := map[bool]int{}
//	for validTxNum > 0 {
//		curTime := makeTimestamp()
//		validTx := rand.Int()%2 == 0
//		tx := createRandTx(validTx, curTime, validTxNum)
//
//		txMap[validTx]++
//		if validTx {
//			validTxNum--
//		}
//
//		manager.SendTransaction(tx)
//		//time.Sleep(time.Millisecond * 3) // 0.003 seconds
//	}
//	fmt.Println("invalid tx Num : ", txMap[false], ", valid tx Num : ", txMap[true])
//	done <- true
//
//	// TODO: send signal for end of request
//}

// create wallet(private/public keys & address) and set balance
// then set addresses and accounts to trie
//func initTestWallets(testWalletNum int, db db.Database, mpts ...trie.Mutable) {
//	for i := 0; i < testWalletNum; i++ {
//		testWallets[i] = common.NewWallet()
//		testAddresses[i] = testWallets[i].Address()
//		accountState := service.newAccountState(db, nil)
//		accountState.SetBalance(big.NewInt(deposit))
//		serializedAccount, _ := codec.MP.MarshalToBytes(accountState.GetSnapshot())
//		for _, mpt := range mpts {
//			mpt.Set(testAddresses[i].Bytes(), serializedAccount)
//		}
//	}
//}

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

	for i := 1; i < 1000; i++ {
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
